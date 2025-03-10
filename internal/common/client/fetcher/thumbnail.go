package fetcher

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/jaxron/roapi.go/pkg/api"
	"github.com/jaxron/roapi.go/pkg/api/middleware/auth"
	"github.com/jaxron/roapi.go/pkg/api/resources/thumbnails"
	apiTypes "github.com/jaxron/roapi.go/pkg/api/types"
	"github.com/robalyx/rotector/internal/common/storage/database/types"
	"github.com/sourcegraph/conc/pool"
	"go.uber.org/zap"
)

const ThumbnailPlaceholder = "-"

// ThumbnailFetcher handles retrieval of user and group thumbnails from the Roblox API.
type ThumbnailFetcher struct {
	roAPI  *api.API
	logger *zap.Logger
}

// NewThumbnailFetcher creates a ThumbnailFetcher with the provided API client and logger.
func NewThumbnailFetcher(roAPI *api.API, logger *zap.Logger) *ThumbnailFetcher {
	return &ThumbnailFetcher{
		roAPI:  roAPI,
		logger: logger.Named("thumbnail_fetcher"),
	}
}

// GetImageURLs fetches thumbnails for a batch of users and returns a map of results.
func (t *ThumbnailFetcher) GetImageURLs(ctx context.Context, users map[uint64]*types.User) map[uint64]string {
	// Create batch request for headshots
	requests := thumbnails.NewBatchThumbnailsBuilder()
	for _, user := range users {
		requests.AddRequest(apiTypes.ThumbnailRequest{
			Type:      apiTypes.AvatarType,
			TargetID:  user.ID,
			RequestID: strconv.FormatUint(user.ID, 10),
			Size:      apiTypes.Size420x420,
			Format:    apiTypes.WEBP,
		})
	}

	// Process thumbnails
	results := t.ProcessBatchThumbnails(ctx, requests)

	t.logger.Debug("Finished fetching user thumbnails",
		zap.Int("totalUsers", len(users)),
		zap.Int("successfulFetches", len(results)))

	return results
}

// AddGroupImageURLs fetches thumbnails for groups and adds them to the group records.
func (t *ThumbnailFetcher) AddGroupImageURLs(
	ctx context.Context, groups map[uint64]*types.Group,
) map[uint64]*types.Group {
	// Create batch request for group icons
	requests := thumbnails.NewBatchThumbnailsBuilder()
	for _, group := range groups {
		requests.AddRequest(apiTypes.ThumbnailRequest{
			Type:      apiTypes.GroupIconType,
			TargetID:  group.ID,
			RequestID: strconv.FormatUint(group.ID, 10),
			Size:      apiTypes.Size420x420,
			Format:    apiTypes.WEBP,
		})
	}

	// Process thumbnails
	results := t.ProcessBatchThumbnails(ctx, requests)

	// Add thumbnail URLs to groups
	now := time.Now()
	updatedGroups := make(map[uint64]*types.Group, len(groups))
	for _, group := range groups {
		if thumbnailURL, ok := results[group.ID]; ok {
			group.ThumbnailURL = thumbnailURL
			group.LastThumbnailUpdate = now
			updatedGroups[group.ID] = group
		}
	}

	t.logger.Debug("Finished fetching group thumbnails",
		zap.Int("totalGroups", len(groups)),
		zap.Int("successfulFetches", len(updatedGroups)))

	return updatedGroups
}

// ProcessBatchThumbnails handles batched thumbnail requests, processing them in groups of 100.
// It returns a map of target IDs to their thumbnail URLs.
func (t *ThumbnailFetcher) ProcessBatchThumbnails(
	ctx context.Context, requests *thumbnails.BatchThumbnailsBuilder,
) map[uint64]string {
	ctx = context.WithValue(ctx, auth.KeyAddCookie, true)

	var (
		requestList   = requests.Build()
		thumbnailURLs = make(map[uint64]string)
		p             = pool.New().WithContext(ctx)
		mu            sync.Mutex
		batchSize     = 100
	)

	// Process batches concurrently
	for i := 0; i < len(requestList.Requests); i += batchSize {
		p.Go(func(ctx context.Context) error {
			end := min(i+batchSize, len(requestList.Requests))

			// Create new batch request
			batchRequests := thumbnails.NewBatchThumbnailsBuilder()
			for _, request := range requestList.Requests[i:end] {
				batchRequests.AddRequest(request)
			}

			// Send batch request to Roblox API
			thumbnailResponses, err := t.roAPI.Thumbnails().GetBatchThumbnails(ctx, batchRequests.Build())
			if err != nil {
				t.logger.Error("Error fetching batch thumbnails",
					zap.Error(err),
					zap.Int("batchStart", i))
				return nil // Don't fail the whole batch for one error
			}

			// Process responses and store URLs
			mu.Lock()
			for _, response := range thumbnailResponses.Data {
				if response.State == apiTypes.ThumbnailStateCompleted && response.ImageURL != nil {
					thumbnailURLs[response.TargetID] = *response.ImageURL
				} else {
					thumbnailURLs[response.TargetID] = ThumbnailPlaceholder
				}
			}
			mu.Unlock()

			return nil
		})
	}

	// Wait for all goroutines to complete
	if err := p.Wait(); err != nil {
		t.logger.Error("Error during thumbnail processing", zap.Error(err))
	}

	t.logger.Debug("Processed batch thumbnails",
		zap.Int("totalRequested", len(requestList.Requests)),
		zap.Int("successfulFetches", len(thumbnailURLs)))

	return thumbnailURLs
}
