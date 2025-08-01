package friend

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jaxron/roapi.go/pkg/api"
	"github.com/robalyx/rotector/internal/database"
	"github.com/robalyx/rotector/internal/database/types"
	"github.com/robalyx/rotector/internal/database/types/enum"
	"github.com/robalyx/rotector/internal/progress"
	"github.com/robalyx/rotector/internal/queue"
	"github.com/robalyx/rotector/internal/roblox/checker"
	"github.com/robalyx/rotector/internal/roblox/fetcher"
	"github.com/robalyx/rotector/internal/setup"
	"github.com/robalyx/rotector/internal/worker/core"
	"github.com/robalyx/rotector/pkg/utils"
	"go.uber.org/zap"
)

const (
	// MinFriendPercentage is the minimum percentage of friends that must be in the system for flagged users.
	MinFriendPercentage = 30.0
	// MinFriendsInSystem is the minimum number of friends that must be in the system for flagged users.
	MinFriendsInSystem = 10
)

// Worker processes user friend networks by checking each friend's
// status and analyzing their profiles for inappropriate content.
type Worker struct {
	db               database.Client
	roAPI            *api.API
	d1Client         *queue.D1Client
	bar              *progress.Bar
	userFetcher      *fetcher.UserFetcher
	userChecker      *checker.UserChecker
	friendFetcher    *fetcher.FriendFetcher
	reporter         *core.StatusReporter
	thresholdChecker *core.ThresholdChecker
	processingCache  *core.UserProcessingCache
	pendingFriends   []*types.ReviewUser
	logger           *zap.Logger
	batchSize        int
}

// New creates a new friend worker.
func New(app *setup.App, bar *progress.Bar, logger *zap.Logger) *Worker {
	userFetcher := fetcher.NewUserFetcher(app, logger)
	userChecker := checker.NewUserChecker(app, userFetcher, logger)
	friendFetcher := fetcher.NewFriendFetcher(app, logger)
	reporter := core.NewStatusReporter(app.StatusClient, "friend", logger)
	thresholdChecker := core.NewThresholdChecker(
		app.DB,
		app.Config.Worker.ThresholdLimits.FlaggedUsers,
		bar,
		reporter,
		logger.Named("friend_worker"),
		"friend worker",
	)
	processingCache := core.NewUserProcessingCache(app.RedisManager, logger)

	return &Worker{
		db:               app.DB,
		roAPI:            app.RoAPI,
		d1Client:         app.D1Client,
		bar:              bar,
		userFetcher:      userFetcher,
		userChecker:      userChecker,
		friendFetcher:    friendFetcher,
		reporter:         reporter,
		thresholdChecker: thresholdChecker,
		processingCache:  processingCache,
		pendingFriends:   make([]*types.ReviewUser, 0),
		logger:           logger.Named("friend_worker"),
		batchSize:        app.Config.Worker.BatchSizes.FriendUsers,
	}
}

// Start begins the friend worker's main loop.
func (w *Worker) Start(ctx context.Context) {
	w.logger.Info("Friend Worker started", zap.String("workerID", w.reporter.GetWorkerID()))

	w.reporter.Start(ctx)
	defer w.reporter.Stop()

	w.bar.SetTotal(100)

	for {
		// Check if context was cancelled
		if utils.ContextGuardWithLog(ctx, w.logger, "Context cancelled, stopping friend worker") {
			w.bar.SetStepMessage("Shutting down", 100)
			w.reporter.UpdateStatus("Shutting down", 100)

			return
		}

		w.bar.Reset()
		w.reporter.SetHealthy(true)

		// Check flagged users threshold
		thresholdExceeded, err := w.thresholdChecker.CheckThreshold(ctx)
		if err != nil {
			if !utils.ErrorSleep(ctx, 5*time.Minute, w.logger, "friend worker") {
				return
			}

			continue
		}

		if thresholdExceeded {
			continue
		}

		// Step 1: Process friends batch (40%)
		w.bar.SetStepMessage("Processing friends batch", 40)
		w.reporter.UpdateStatus("Processing friends batch", 40)

		userInfos, err := w.processFriendsBatch(ctx)
		if err != nil {
			w.reporter.SetHealthy(false)

			if !utils.ErrorSleep(ctx, 5*time.Minute, w.logger, "friend worker") {
				return
			}

			continue
		}

		// Step 2: Process users (60%)
		w.bar.SetStepMessage("Processing users", 60)
		w.reporter.UpdateStatus("Processing users", 60)
		processResult := w.userChecker.ProcessUsers(ctx, &checker.UserCheckerParams{
			Users:                     userInfos[:w.batchSize],
			InappropriateOutfitFlags:  nil,
			InappropriateProfileFlags: nil,
			InappropriateFriendsFlags: nil,
			InappropriateGroupsFlags:  nil,
		})

		// Mark processed users in cache to prevent reprocessing
		var processedUserIDs []uint64
		for _, user := range userInfos[:w.batchSize] {
			processedUserIDs = append(processedUserIDs, user.ID)
		}

		if err := w.processingCache.MarkUsersProcessed(ctx, processedUserIDs); err != nil {
			w.logger.Error("Failed to mark users as processed in cache", zap.Error(err))
		}

		// Step 3: Add flagged users to D1 database (80%)
		w.bar.SetStepMessage("Adding flagged users to D1", 80)
		w.reporter.UpdateStatus("Adding flagged users to D1", 80)

		if len(processResult.FlaggedUsers) > 0 {
			if err := w.d1Client.AddFlaggedUsers(ctx, processResult.FlaggedUsers); err != nil {
				w.logger.Error("Failed to add flagged users to D1", zap.Error(err))
			}
		}

		// Step 4: Prepare for next batch
		w.pendingFriends = userInfos[w.batchSize:]

		// Step 5: Completed (100%)
		w.bar.SetStepMessage("Completed", 100)
		w.reporter.UpdateStatus("Completed", 100)

		// Short pause before next iteration
		if !utils.IntervalSleep(ctx, 1*time.Second, w.logger, "friend worker") {
			return
		}
	}
}

// processFriendsBatch builds a list of validated friends to check.
func (w *Worker) processFriendsBatch(ctx context.Context) ([]*types.ReviewUser, error) {
	validFriends := w.pendingFriends

	for len(validFriends) < w.batchSize {
		// Get the next confirmed user
		user, err := w.db.Model().User().GetUserToScan(ctx)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				w.logger.Error("Error getting user to scan", zap.Error(err))
			} else {
				w.logger.Warn("No more users to scan", zap.Error(err))
			}

			return validFriends, err
		}

		// Fetch friends for the user
		userFriendIDs, err := w.friendFetcher.GetFriendIDs(ctx, user.ID)
		if err != nil {
			w.logger.Error("Error fetching friends", zap.Error(err), zap.Uint64("userID", user.ID))
			continue
		}

		// If the user has no friends, skip them
		if len(userFriendIDs) == 0 {
			continue
		}

		// Get all users that exist in our system
		existingUsers, err := w.db.Model().User().GetUsersByIDs(ctx, userFriendIDs, types.UserFieldID)
		if err != nil {
			w.logger.Error("Error checking existing users", zap.Error(err))
			continue
		}

		// For flagged users, check if they meet the friend criteria
		if user.Status == enum.UserTypeFlagged {
			existingCount := len(existingUsers)
			friendPercentage := (float64(existingCount) / float64(len(userFriendIDs))) * 100

			if existingCount < MinFriendsInSystem || friendPercentage < MinFriendPercentage {
				w.logger.Debug("Flagged user does not meet friend criteria",
					zap.Uint64("userID", user.ID),
					zap.Int("totalFriends", len(userFriendIDs)),
					zap.Int("existingFriends", existingCount),
					zap.Float64("friendPercentage", friendPercentage))

				continue
			}

			w.logger.Info("Processing flagged user",
				zap.Int("userFriends", len(userFriendIDs)),
				zap.Int("existingFriends", existingCount),
				zap.Float64("friendPercentage", friendPercentage),
				zap.Uint64("userID", user.ID))
		} else {
			w.logger.Info("Processing confirmed user",
				zap.Int("userFriends", len(userFriendIDs)),
				zap.Int("existingFriends", len(existingUsers)),
				zap.Uint64("userID", user.ID))
		}

		// Collect users that do not exist in our system
		var friendIDs []uint64

		for _, userID := range userFriendIDs {
			if _, exists := existingUsers[userID]; !exists {
				friendIDs = append(friendIDs, userID)
			}
		}

		// Filter out already processed users to prevent duplicate processing
		if len(friendIDs) > 0 {
			unprocessedFriendIDs, err := w.processingCache.FilterProcessedUsers(ctx, friendIDs)
			if err != nil {
				w.logger.Error("Error filtering processed users", zap.Error(err), zap.Uint64("userID", user.ID))

				unprocessedFriendIDs = friendIDs
			}

			// Fetch user info and validate friends
			if len(unprocessedFriendIDs) > 0 {
				userInfos := w.userFetcher.FetchInfos(ctx, unprocessedFriendIDs)
				validFriends = append(validFriends, userInfos...)

				w.logger.Debug("Added friends for processing",
					zap.Uint64("userID", user.ID),
					zap.Int("totalFriends", len(userFriendIDs)),
					zap.Int("existingFriends", len(existingUsers)),
					zap.Int("fetchedFriends", len(friendIDs)),
					zap.Int("unprocessedFriends", len(unprocessedFriendIDs)),
					zap.Int("validFriends", len(userInfos)),
					zap.Int("totalValidFriends", len(validFriends)))
			}
		}
	}

	return validFriends, nil
}
