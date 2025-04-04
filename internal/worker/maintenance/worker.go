package maintenance

import (
	"context"
	"time"

	"github.com/jaxron/roapi.go/pkg/api"
	"github.com/robalyx/rotector/internal/database"
	"github.com/robalyx/rotector/internal/progress"
	"github.com/robalyx/rotector/internal/roblox/checker"
	"github.com/robalyx/rotector/internal/roblox/fetcher"
	"github.com/robalyx/rotector/internal/setup"
	"github.com/robalyx/rotector/internal/worker/core"
	"go.uber.org/zap"
)

// Worker handles all maintenance operations.
type Worker struct {
	db                      database.Client
	roAPI                   *api.API
	bar                     *progress.Bar
	userFetcher             *fetcher.UserFetcher
	groupFetcher            *fetcher.GroupFetcher
	thumbnailFetcher        *fetcher.ThumbnailFetcher
	groupChecker            *checker.GroupChecker
	reporter                *core.StatusReporter
	logger                  *zap.Logger
	userBatchSize           int
	groupBatchSize          int
	trackBatchSize          int
	thumbnailUserBatchSize  int
	thumbnailGroupBatchSize int
	minGroupFlaggedUsers    int
	minFlaggedOverride      int
	minFlaggedPercent       float64
}

// New creates a new maintenance worker.
func New(app *setup.App, bar *progress.Bar, logger *zap.Logger) *Worker {
	userFetcher := fetcher.NewUserFetcher(app, logger)
	groupFetcher := fetcher.NewGroupFetcher(app.RoAPI, logger)
	thumbnailFetcher := fetcher.NewThumbnailFetcher(app.RoAPI, logger)
	reporter := core.NewStatusReporter(app.StatusClient, "maintenance", logger)
	groupChecker := checker.NewGroupChecker(app.DB, logger,
		app.Config.Worker.ThresholdLimits.MaxGroupMembersTrack,
		app.Config.Worker.ThresholdLimits.MinFlaggedOverride,
		app.Config.Worker.ThresholdLimits.MinFlaggedPercentage,
	)

	return &Worker{
		db:                      app.DB,
		roAPI:                   app.RoAPI,
		bar:                     bar,
		userFetcher:             userFetcher,
		groupFetcher:            groupFetcher,
		thumbnailFetcher:        thumbnailFetcher,
		groupChecker:            groupChecker,
		reporter:                reporter,
		logger:                  logger.Named("maintenance_worker"),
		userBatchSize:           app.Config.Worker.BatchSizes.PurgeUsers,
		groupBatchSize:          app.Config.Worker.BatchSizes.PurgeGroups,
		trackBatchSize:          app.Config.Worker.BatchSizes.TrackGroups,
		thumbnailUserBatchSize:  app.Config.Worker.BatchSizes.ThumbnailUsers,
		thumbnailGroupBatchSize: app.Config.Worker.BatchSizes.ThumbnailGroups,
		minGroupFlaggedUsers:    app.Config.Worker.ThresholdLimits.MinGroupFlaggedUsers,
		minFlaggedOverride:      app.Config.Worker.ThresholdLimits.MinFlaggedOverride,
		minFlaggedPercent:       app.Config.Worker.ThresholdLimits.MinFlaggedPercentage,
	}
}

// Start begins the maintenance worker's main loop.
func (w *Worker) Start() {
	w.logger.Info("Maintenance Worker started", zap.String("workerID", w.reporter.GetWorkerID()))
	w.reporter.Start()
	defer w.reporter.Stop()

	w.bar.SetTotal(100)

	for {
		w.bar.Reset()
		w.reporter.SetHealthy(true)

		// Step 1: Process banned users (12%)
		w.processBannedUsers()

		// Step 2: Process locked groups (24%)
		w.processLockedGroups()

		// Step 3: Process cleared users (36%)
		w.processClearedUsers()

		// Step 4: Process cleared groups (48%)
		w.processClearedGroups()

		// Step 5: Process group tracking (60%)
		w.processGroupTracking()

		// Step 6: Process user thumbnails (72%)
		w.processUserThumbnails()

		// Step 7: Process group thumbnails (84%)
		w.processGroupThumbnails()

		// Step 8: Process old Discord server members (96%)
		w.processOldServerMembers()

		// Step 9: Completed (100%)
		w.bar.SetStepMessage("Completed", 100)
		w.reporter.UpdateStatus("Completed", 100)

		// Short pause before next iteration
		time.Sleep(10 * time.Second)
	}
}

// processBannedUsers checks for and marks banned users.
func (w *Worker) processBannedUsers() {
	w.bar.SetStepMessage("Processing banned users", 12)
	w.reporter.UpdateStatus("Processing banned users", 12)

	// Get users to check
	users, currentlyBanned, err := w.db.Model().User().GetUsersToCheck(context.Background(), w.userBatchSize)
	if err != nil {
		w.logger.Error("Error getting users to check", zap.Error(err))
		w.reporter.SetHealthy(false)
		return
	}

	if len(users) == 0 {
		w.logger.Info("No users to check for bans")
		return
	}

	// Check for banned users
	bannedUserIDs, err := w.userFetcher.FetchBannedUsers(context.Background(), users)
	if err != nil {
		w.logger.Error("Error fetching banned users", zap.Error(err))
		w.reporter.SetHealthy(false)
		return
	}

	// Create map of newly banned users for O(1) lookup
	bannedMap := make(map[uint64]struct{}, len(bannedUserIDs))
	for _, id := range bannedUserIDs {
		bannedMap[id] = struct{}{}
	}

	// Find users that are no longer banned
	var unbannedUserIDs []uint64
	for _, id := range currentlyBanned {
		if _, ok := bannedMap[id]; !ok {
			unbannedUserIDs = append(unbannedUserIDs, id)
		}
	}

	// Mark banned users
	if len(bannedUserIDs) > 0 {
		err = w.db.Model().User().MarkUsersBanStatus(context.Background(), bannedUserIDs, true)
		if err != nil {
			w.logger.Error("Error marking banned users", zap.Error(err))
			w.reporter.SetHealthy(false)
			return
		}
		w.logger.Info("Marked banned users", zap.Int("count", len(bannedUserIDs)))
	}

	// Unmark users that are no longer banned
	if len(unbannedUserIDs) > 0 {
		err = w.db.Model().User().MarkUsersBanStatus(context.Background(), unbannedUserIDs, false)
		if err != nil {
			w.logger.Error("Error unmarking banned users", zap.Error(err))
			w.reporter.SetHealthy(false)
			return
		}
		w.logger.Info("Unmarked banned users", zap.Int("count", len(unbannedUserIDs)))
	}
}

// processLockedGroups checks for and marks locked groups.
func (w *Worker) processLockedGroups() {
	w.bar.SetStepMessage("Processing locked groups", 24)
	w.reporter.UpdateStatus("Processing locked groups", 24)

	// Get groups to check
	groups, currentlyLocked, err := w.db.Model().Group().GetGroupsToCheck(context.Background(), w.groupBatchSize)
	if err != nil {
		w.logger.Error("Error getting groups to check", zap.Error(err))
		w.reporter.SetHealthy(false)
		return
	}

	if len(groups) == 0 {
		w.logger.Info("No groups to check for locks")
		return
	}

	// Check for locked groups
	lockedGroupIDs, err := w.groupFetcher.FetchLockedGroups(context.Background(), groups)
	if err != nil {
		w.logger.Error("Error fetching locked groups", zap.Error(err))
		w.reporter.SetHealthy(false)
		return
	}

	// Create map of newly locked groups for O(1) lookup
	lockedMap := make(map[uint64]struct{}, len(lockedGroupIDs))
	for _, id := range lockedGroupIDs {
		lockedMap[id] = struct{}{}
	}

	// Find groups that are no longer locked
	var unlockedGroupIDs []uint64
	for _, id := range currentlyLocked {
		if _, ok := lockedMap[id]; !ok {
			unlockedGroupIDs = append(unlockedGroupIDs, id)
		}
	}

	// Mark locked groups
	if len(lockedGroupIDs) > 0 {
		err = w.db.Model().Group().MarkGroupsLockStatus(context.Background(), lockedGroupIDs, true)
		if err != nil {
			w.logger.Error("Error marking locked groups", zap.Error(err))
			w.reporter.SetHealthy(false)
			return
		}
		w.logger.Info("Marked locked groups", zap.Int("count", len(lockedGroupIDs)))
	}

	// Unmark groups that are no longer locked
	if len(unlockedGroupIDs) > 0 {
		err = w.db.Model().Group().MarkGroupsLockStatus(context.Background(), unlockedGroupIDs, false)
		if err != nil {
			w.logger.Error("Error unmarking locked groups", zap.Error(err))
			w.reporter.SetHealthy(false)
			return
		}
		w.logger.Info("Unmarked locked groups", zap.Int("count", len(unlockedGroupIDs)))
	}
}

// processClearedUsers removes old cleared users.
func (w *Worker) processClearedUsers() {
	w.bar.SetStepMessage("Processing cleared users", 36)
	w.reporter.UpdateStatus("Processing cleared users", 36)

	cutOffDate := time.Now().AddDate(0, 0, -30)
	affected, err := w.db.Model().User().PurgeOldClearedUsers(context.Background(), cutOffDate)
	if err != nil {
		w.logger.Error("Error purging old cleared users", zap.Error(err))
		w.reporter.SetHealthy(false)
		return
	}

	if affected > 0 {
		w.logger.Info("Purged old cleared users",
			zap.Int("affected", affected),
			zap.Time("cutOffDate", cutOffDate))
	}
}

// processClearedGroups removes old cleared groups.
func (w *Worker) processClearedGroups() {
	w.bar.SetStepMessage("Processing cleared groups", 48)
	w.reporter.UpdateStatus("Processing cleared groups", 48)

	cutOffDate := time.Now().AddDate(0, 0, -30)
	affected, err := w.db.Model().Group().PurgeOldClearedGroups(context.Background(), cutOffDate)
	if err != nil {
		w.logger.Error("Error purging old cleared groups", zap.Error(err))
		w.reporter.SetHealthy(false)
		return
	}

	if affected > 0 {
		w.logger.Info("Purged old cleared groups",
			zap.Int("affected", affected),
			zap.Time("cutOffDate", cutOffDate))
	}
}

// processGroupTracking manages group tracking data.
func (w *Worker) processGroupTracking() {
	w.bar.SetStepMessage("Processing group tracking", 60)
	w.reporter.UpdateStatus("Processing group tracking", 60)

	// Get groups to check
	groupsWithUsers, err := w.db.Model().Tracking().GetGroupTrackingsToCheck(
		context.Background(),
		w.trackBatchSize,
		w.minGroupFlaggedUsers,
		w.minFlaggedOverride,
	)
	if err != nil {
		w.logger.Error("Error checking group trackings", zap.Error(err))
		w.reporter.SetHealthy(false)
		return
	}

	// Check if there are any groups to check
	if len(groupsWithUsers) == 0 {
		w.logger.Info("No groups to check for tracking")
		return
	}

	// Extract group IDs for batch lookup
	groupIDs := make([]uint64, 0, len(groupsWithUsers))
	for groupID := range groupsWithUsers {
		groupIDs = append(groupIDs, groupID)
	}

	// Load group information from API
	groupInfos := w.groupFetcher.FetchGroupInfos(context.Background(), groupIDs)
	if len(groupInfos) == 0 {
		return
	}

	// Check which groups exceed the percentage threshold
	flaggedGroups := w.groupChecker.CheckGroupPercentages(groupInfos, groupsWithUsers)
	if len(flaggedGroups) == 0 {
		return
	}

	// Add thumbnails to flagged groups
	flaggedGroups = w.thumbnailFetcher.AddGroupImageURLs(context.Background(), flaggedGroups)

	// Save flagged groups to database
	if err := w.db.Service().Group().SaveGroups(context.Background(), flaggedGroups); err != nil {
		w.logger.Error("Failed to save flagged groups", zap.Error(err))
		return
	}

	// Extract group IDs that were flagged
	flaggedGroupIDs := make([]uint64, 0, len(flaggedGroups))
	for _, group := range flaggedGroups {
		flaggedGroupIDs = append(flaggedGroupIDs, group.ID)
	}

	// Update tracking entries to mark them as flagged
	if err := w.db.Model().Tracking().UpdateFlaggedGroups(context.Background(), flaggedGroupIDs); err != nil {
		w.logger.Error("Failed to update tracking entries", zap.Error(err))
		return
	}

	w.logger.Info("Processed group trackings",
		zap.Int("checkedGroups", len(groupInfos)),
		zap.Int("flaggedGroups", len(flaggedGroups)))
}

// processUserThumbnails updates user thumbnails.
func (w *Worker) processUserThumbnails() {
	w.bar.SetStepMessage("Processing user thumbnails", 72)
	w.reporter.UpdateStatus("Processing user thumbnails", 72)

	// Get users that need thumbnail updates
	users, err := w.db.Model().User().GetUsersForThumbnailUpdate(context.Background(), w.thumbnailUserBatchSize)
	if err != nil {
		w.logger.Error("Error getting users for thumbnail update", zap.Error(err))
		w.reporter.SetHealthy(false)
		return
	}

	if len(users) == 0 {
		w.logger.Info("No users need thumbnail updates")
		return
	}

	// Update thumbnails
	thumbnailMap := w.thumbnailFetcher.GetImageURLs(context.Background(), users)

	// Update last thumbnail update time
	now := time.Now()
	for id, thumbnail := range thumbnailMap {
		if user, ok := users[id]; ok {
			user.ThumbnailURL = thumbnail
			user.LastThumbnailUpdate = now
		}
	}

	// Save updated users
	if err := w.db.Service().User().SaveUsers(context.Background(), users); err != nil {
		w.logger.Error("Error saving updated user thumbnails", zap.Error(err))
		w.reporter.SetHealthy(false)
		return
	}

	w.logger.Info("Updated user thumbnails",
		zap.Int("processedCount", len(users)),
		zap.Int("updatedCount", len(thumbnailMap)))
}

// processGroupThumbnails updates group thumbnails.
func (w *Worker) processGroupThumbnails() {
	w.bar.SetStepMessage("Processing group thumbnails", 84)
	w.reporter.UpdateStatus("Processing group thumbnails", 84)

	// Get groups that need thumbnail updates
	groups, err := w.db.Model().Group().GetGroupsForThumbnailUpdate(context.Background(), w.thumbnailGroupBatchSize)
	if err != nil {
		w.logger.Error("Error getting groups for thumbnail update", zap.Error(err))
		w.reporter.SetHealthy(false)
		return
	}

	if len(groups) == 0 {
		w.logger.Info("No groups need thumbnail updates")
		return
	}

	// Update thumbnails
	updatedGroups := w.thumbnailFetcher.AddGroupImageURLs(context.Background(), groups)

	// Save updated groups
	if err := w.db.Service().Group().SaveGroups(context.Background(), groups); err != nil {
		w.logger.Error("Error saving updated group thumbnails", zap.Error(err))
		w.reporter.SetHealthy(false)
		return
	}

	w.logger.Info("Updated group thumbnails",
		zap.Int("processedCount", len(groups)),
		zap.Int("updatedCount", len(updatedGroups)))
}

// processOldServerMembers removes Discord server member records older than 7 days.
func (w *Worker) processOldServerMembers() {
	w.bar.SetStepMessage("Processing old Discord server members", 96)
	w.reporter.UpdateStatus("Processing old Discord server members", 96)

	cutoffDate := time.Now().AddDate(0, 0, -7) // 7 days ago
	affected, err := w.db.Model().Sync().PurgeOldServerMembers(context.Background(), cutoffDate)
	if err != nil {
		w.logger.Error("Error purging old Discord server members", zap.Error(err))
		w.reporter.SetHealthy(false)
		return
	}

	if affected > 0 {
		w.logger.Info("Purged old Discord server members",
			zap.Int("affected", affected),
			zap.Time("cutoffDate", cutoffDate))
	}
}
