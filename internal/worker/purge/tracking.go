package purge

import (
	"fmt"
	"time"

	"github.com/rotector/rotector/internal/common/database"
	"github.com/rotector/rotector/internal/common/progress"
	"go.uber.org/zap"
)

const (
	DefaultPurgeCutoffDays = 3
	BatchSize              = 1000
	PurgeInterval          = 1 * time.Hour
)

// TrackingWorker represents a purge worker that removes old tracking entries.
type TrackingWorker struct {
	db     *database.Database
	bar    *progress.Bar
	logger *zap.Logger
}

// NewTrackingWorker creates a new purge worker instance.
func NewTrackingWorker(db *database.Database, bar *progress.Bar, logger *zap.Logger) *TrackingWorker {
	return &TrackingWorker{
		db:     db,
		bar:    bar,
		logger: logger,
	}
}

// Start begins the purge worker's main loop.
func (p *TrackingWorker) Start() {
	p.logger.Info("Tracking Purge Worker started")

	for {
		nextRun := time.Now().Add(PurgeInterval)

		// Perform the purge operations
		p.performPurge()

		// Update progress bar until next run
		p.updateProgressUntilNextRun(nextRun)
	}
}

// performPurge executes the purge operations for group member and user affiliate trackings.
func (p *TrackingWorker) performPurge() {
	p.bar.SetTotal(100)
	p.bar.Reset()

	// Step 1: Purge old group member trackings (50%)
	p.bar.SetStepMessage("Purging old group member trackings")
	if err := p.purgeGroupMemberTrackings(); err != nil {
		p.logger.Error("Failed to purge group member trackings", zap.Error(err))
	}
	p.bar.Increment(50)

	// Step 2: Purge old user affiliate trackings (50%)
	p.bar.SetStepMessage("Purging old user affiliate trackings")
	if err := p.purgeUserAffiliateTrackings(); err != nil {
		p.logger.Error("Failed to purge user affiliate trackings", zap.Error(err))
	}
	p.bar.Increment(50)
}

// updateProgressUntilNextRun updates the progress bar until the next run time.
func (p *TrackingWorker) updateProgressUntilNextRun(nextRun time.Time) {
	p.bar.Reset()
	totalDuration := PurgeInterval
	p.bar.SetTotal(int64(totalDuration.Seconds()))

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for remaining := time.Until(nextRun); remaining > 0; remaining = time.Until(nextRun) {
		<-ticker.C
		elapsed := totalDuration - remaining
		p.bar.SetCurrent(int64(elapsed.Seconds()))
		p.bar.SetStepMessage(fmt.Sprintf("Next purge in %s", remaining.Round(time.Second)))
	}
}

// purgeGroupMemberTrackings removes old entries from group_member_trackings.
func (p *TrackingWorker) purgeGroupMemberTrackings() error {
	// Calculate the cutoff date
	cutoffDate := time.Now().AddDate(0, 0, -DefaultPurgeCutoffDays)

	for {
		// Purge old group member trackings in batches
		affected, err := p.db.Tracking().PurgeOldGroupMemberTrackings(cutoffDate, BatchSize)
		if err != nil {
			return err
		}

		p.logger.Info("Purged group member trackings batch",
			zap.Int("count", affected),
			zap.Time("cutoff_date", cutoffDate))

		// If less than BatchSize rows were affected, we're done
		if affected < BatchSize {
			break
		}

		// Add a small delay between batches to reduce database load
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// purgeUserAffiliateTrackings removes old entries from user_affiliate_trackings.
func (p *TrackingWorker) purgeUserAffiliateTrackings() error {
	// Calculate the cutoff date
	cutoffDate := time.Now().AddDate(0, 0, -DefaultPurgeCutoffDays)

	for {
		// Purge old user affiliate trackings in batches
		affected, err := p.db.Tracking().PurgeOldUserAffiliateTrackings(cutoffDate, BatchSize)
		if err != nil {
			return err
		}

		p.logger.Info("Purged user affiliate trackings batch",
			zap.Int("count", affected),
			zap.Time("cutoff_date", cutoffDate))

		// If less than BatchSize rows were affected, we're done
		if affected < BatchSize {
			break
		}

		// Add a small delay between batches to reduce database load
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}
