package models

import (
	"context"
	"fmt"

	"github.com/robalyx/rotector/internal/database/types"
	"github.com/robalyx/rotector/internal/database/types/enum"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

// ReviewerModel handles database operations for reviewer statistics.
type ReviewerModel struct {
	db     *bun.DB
	logger *zap.Logger
}

// NewReviewer creates a new ReviewerModel instance.
func NewReviewer(db *bun.DB, logger *zap.Logger) *ReviewerModel {
	return &ReviewerModel{
		db:     db,
		logger: logger.Named("db_reviewer"),
	}
}

// GetReviewerStats retrieves paginated reviewer statistics for a specific time period.
//
// Deprecated: Use Service().Reviewer().GetReviewerStats() instead.
func (r *ReviewerModel) GetReviewerStats(
	ctx context.Context, period enum.ReviewerStatsPeriod, cursor *types.ReviewerStatsCursor, limit int,
) (map[uint64]*types.ReviewerStats, *types.ReviewerStatsCursor, error) {
	// Initialize result map
	results := make(map[uint64]*types.ReviewerStats)
	var nextCursor *types.ReviewerStatsCursor

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Get bot settings to filter for valid reviewer IDs
		var settings types.BotSetting
		err := tx.NewSelect().
			Model(&settings).
			Where("id = 1").
			Scan(ctx)
		if err != nil {
			return fmt.Errorf("failed to get bot settings: %w", err)
		}

		// Return empty results if no reviewers configured
		if len(settings.ReviewerIDs) == 0 {
			return nil
		}

		query := tx.NewSelect().
			TableExpr("reviewer_stats_"+period.String()).
			Where("reviewer_id = ANY(?)", pgdialect.Array(settings.ReviewerIDs)).
			Order("last_activity DESC", "reviewer_id").
			Limit(limit + 1)

		// Add cursor condition if provided
		if cursor != nil {
			query = query.Where("(last_activity, reviewer_id) < (?, ?)",
				cursor.LastActivity, cursor.ReviewerID)
		}

		var stats []*types.ReviewerStats
		err = query.Scan(ctx, &stats)
		if err != nil {
			return fmt.Errorf("failed to scan reviewer stats: %w", err)
		}

		// Check if there are more results
		if len(stats) > limit {
			last := stats[limit-1]
			nextCursor = &types.ReviewerStatsCursor{
				LastActivity: last.LastActivity,
				ReviewerID:   last.ReviewerID,
			}
			stats = stats[:limit] // Remove the extra item
		}

		// Store stats in results map
		for _, stat := range stats {
			results[stat.ReviewerID] = stat
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return results, nextCursor, nil
}
