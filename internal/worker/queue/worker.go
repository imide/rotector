package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/jaxron/roapi.go/pkg/api"
	"github.com/robalyx/rotector/internal/database"
	"github.com/robalyx/rotector/internal/progress"
	"github.com/robalyx/rotector/internal/queue"
	"github.com/robalyx/rotector/internal/roblox/checker"
	"github.com/robalyx/rotector/internal/roblox/fetcher"
	"github.com/robalyx/rotector/internal/setup"
	"github.com/robalyx/rotector/internal/worker/core"
	"go.uber.org/zap"
)

// Worker handles items in the queues.
type Worker struct {
	db          database.Client
	roAPI       *api.API
	queue       *queue.Manager
	bar         *progress.Bar
	userFetcher *fetcher.UserFetcher
	userChecker *checker.UserChecker
	reporter    *core.StatusReporter
	logger      *zap.Logger
	batchSize   int
}

// New creates a new queue worker.
func New(app *setup.App, bar *progress.Bar, logger *zap.Logger) *Worker {
	userFetcher := fetcher.NewUserFetcher(app, logger)
	userChecker := checker.NewUserChecker(app, userFetcher, logger)
	reporter := core.NewStatusReporter(app.StatusClient, "queue", logger)

	return &Worker{
		db:          app.DB,
		roAPI:       app.RoAPI,
		queue:       app.Queue,
		bar:         bar,
		userFetcher: userFetcher,
		userChecker: userChecker,
		reporter:    reporter,
		logger:      logger.Named("queue_worker"),
		batchSize:   app.Config.Worker.BatchSizes.QueueItems,
	}
}

// Start begins the process worker's main loop.
func (w *Worker) Start() {
	w.logger.Info("Process Worker started", zap.String("workerID", w.reporter.GetWorkerID()))
	w.reporter.Start()
	defer w.reporter.Stop()

	w.bar.SetTotal(100)

	for {
		w.bar.Reset()
		w.reporter.SetHealthy(true)

		// Step 1: Get next batch of items (20%)
		w.bar.SetStepMessage("Getting next batch", 20)
		w.reporter.UpdateStatus("Getting next batch", 20)
		items, err := w.getNextBatch()
		if err != nil {
			w.logger.Error("Error getting next batch", zap.Error(err))
			w.reporter.SetHealthy(false)
			time.Sleep(5 * time.Minute)
			continue
		}

		// If no items to process, wait before checking again
		if len(items) == 0 {
			w.bar.SetStepMessage("No items to process, waiting", 0)
			w.reporter.UpdateStatus("No items to process, waiting", 0)
			time.Sleep(10 * time.Second)
			continue
		}

		// Step 2: Process items (80%)
		w.processItems(items)

		// Step 3: Completed (100%)
		w.bar.SetStepMessage("Completed", 100)
		w.reporter.UpdateStatus("Completed", 100)
	}
}

// getNextBatch retrieves items from queues based on priority order.
func (w *Worker) getNextBatch() ([]*queue.Item, error) {
	var items []*queue.Item

	// Check queues in priority order
	for _, priority := range []queue.Priority{
		queue.PriorityHigh,
		queue.PriorityNormal,
		queue.PriorityLow,
	} {
		// Get items from current priority queue
		key := fmt.Sprintf("queue:%s_priority", priority)
		itemsJSON, err := w.queue.GetQueueItems(context.Background(), key, w.batchSize-len(items))
		if err != nil {
			return nil, fmt.Errorf("failed to get items from queue: %w", err)
		}

		// Parse items from JSON
		for _, itemJSON := range itemsJSON {
			var item queue.Item
			if err := sonic.Unmarshal([]byte(itemJSON), &item); err != nil {
				w.logger.Error("Failed to unmarshal queue item",
					zap.Error(err),
					zap.String("itemJSON", itemJSON))
				continue
			}

			items = append(items, &item)
		}

		// Stop if batch is full
		if len(items) >= w.batchSize {
			break
		}
	}

	return items, nil
}

// processItems handles batches of queued items.
func (w *Worker) processItems(items []*queue.Item) {
	ctx := context.Background()
	itemCount := len(items)

	w.bar.SetStepMessage("Processing batch", 25)
	w.reporter.UpdateStatus(fmt.Sprintf("Processing batch of %d items", itemCount), 25)

	// Update status to processing for all items
	for _, item := range items {
		if err := w.queue.SetQueueInfo(ctx, item.UserID, queue.StatusProcessing, item.Priority, 0); err != nil {
			w.logger.Error("Failed to update queue info",
				zap.Error(err),
				zap.Uint64("userID", item.UserID))
			w.reporter.SetHealthy(false)
		}
	}

	// Extract user IDs from items
	userIDs := make([]uint64, len(items))
	userIDToItem := make(map[uint64]*queue.Item)
	for i, item := range items {
		userIDs[i] = item.UserID
		userIDToItem[item.UserID] = item
	}

	// Fetch all users in batch
	w.bar.SetStepMessage("Fetching user information", 50)
	w.reporter.UpdateStatus("Fetching user information", 50)

	userInfos := w.userFetcher.FetchInfos(ctx, userIDs)

	// Process users with AI checker
	w.bar.SetStepMessage("Processing with AI", 75)
	w.reporter.UpdateStatus("Processing with AI", 75)
	w.userChecker.ProcessUsers(userInfos)

	// Update final status for all items
	w.bar.SetStepMessage("Updating queue status", 100)
	w.reporter.UpdateStatus("Updating queue status", 100)

	for _, userID := range userIDs {
		item := userIDToItem[userID]
		// Update final status and remove from queue for successful validations
		w.updateQueueStatus(ctx, item, queue.StatusComplete)
	}

	w.logger.Info("Finished processing batch",
		zap.Int("totalItems", len(items)))
}

// updateQueueStatus handles the final state of a queue item.
func (w *Worker) updateQueueStatus(ctx context.Context, item *queue.Item, status queue.Status) {
	// Update queue info with final status
	if err := w.queue.SetQueueInfo(ctx, item.UserID, status, item.Priority, 0); err != nil {
		w.logger.Error("Failed to update final queue info",
			zap.Error(err),
			zap.Uint64("userID", item.UserID))
	}

	// Remove item from queue
	key := fmt.Sprintf("queue:%s_priority", item.Priority)
	if err := w.queue.RemoveQueueItem(ctx, key, item); err != nil {
		w.logger.Error("Failed to remove item from queue",
			zap.Error(err),
			zap.Uint64("userID", item.UserID))
	}
}
