package models

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/robalyx/rotector/internal/database/dbretry"
	"github.com/robalyx/rotector/internal/database/types"
	"github.com/robalyx/rotector/internal/database/types/enum"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// Add at the top with other constants/types.
var ErrNoLogsFound = errors.New("no logs found")

// ActivityModel handles database operations for moderator action logs.
type ActivityModel struct {
	db     *bun.DB
	logger *zap.Logger
}

// NewActivity creates a repository with database access for
// storing and retrieving moderator action logs.
func NewActivity(db *bun.DB, logger *zap.Logger) *ActivityModel {
	return &ActivityModel{
		db:     db,
		logger: logger.Named("db_activity"),
	}
}

// Log stores a moderator action in the database.
func (r *ActivityModel) Log(ctx context.Context, log *types.ActivityLog) {
	err := dbretry.NoResult(ctx, func(ctx context.Context) error {
		_, err := r.db.NewInsert().Model(log).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to log activity: %w", err)
		}

		return nil
	})
	if err != nil {
		r.logger.Error("Failed to log activity",
			zap.Error(err),
			zap.Uint64("guildID", log.ActivityTarget.GuildID),
			zap.Uint64("discordID", log.ActivityTarget.DiscordID),
			zap.Uint64("userID", log.ActivityTarget.UserID),
			zap.Uint64("groupID", log.ActivityTarget.GroupID),
			zap.Uint64("reviewerID", log.ReviewerID),
			zap.String("activityType", log.ActivityType.String()))

		return
	}

	r.logger.Debug("Logged activity",
		zap.Uint64("guildID", log.ActivityTarget.GuildID),
		zap.Uint64("discordID", log.ActivityTarget.DiscordID),
		zap.Uint64("userID", log.ActivityTarget.UserID),
		zap.Uint64("groupID", log.ActivityTarget.GroupID),
		zap.Uint64("reviewerID", log.ReviewerID),
		zap.String("activityType", log.ActivityType.String()))
}

// LogBatch stores multiple moderator actions in the database.
func (r *ActivityModel) LogBatch(ctx context.Context, logs []*types.ActivityLog) {
	if len(logs) == 0 {
		return
	}

	err := dbretry.NoResult(ctx, func(ctx context.Context) error {
		_, err := r.db.NewInsert().Model(&logs).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to log batch activities: %w", err)
		}

		return nil
	})
	if err != nil {
		r.logger.Error("Failed to log batch activities",
			zap.Error(err),
			zap.Int("count", len(logs)))

		return
	}

	r.logger.Debug("Logged batch activities",
		zap.Int("count", len(logs)))
}

// GetLogs retrieves activity logs based on filter criteria.
func (r *ActivityModel) GetLogs(
	ctx context.Context, filter types.ActivityFilter, cursor *types.LogCursor, limit int,
) ([]*types.ActivityLog, *types.LogCursor, error) {
	var (
		logs       []*types.ActivityLog
		nextCursor *types.LogCursor
	)

	err := dbretry.NoResult(ctx, func(ctx context.Context) error {
		// Build base query conditions
		query := r.db.NewSelect().Model(&logs)

		if filter.GuildID != 0 {
			query = query.Where("guild_id = ?", filter.GuildID)
		}

		if filter.DiscordID != 0 {
			query = query.Where("discord_id = ?", filter.DiscordID)
		}

		if filter.UserID != 0 {
			query = query.Where("user_id = ?", filter.UserID)
		}

		if filter.GroupID != 0 {
			query = query.Where("group_id = ?", filter.GroupID)
		}

		if filter.ReviewerID != 0 {
			query = query.Where("reviewer_id = ?", filter.ReviewerID)
		}

		if filter.ActivityType != enum.ActivityTypeAll {
			query = query.Where("activity_type = ?", filter.ActivityType)
		}

		if !filter.StartDate.IsZero() && !filter.EndDate.IsZero() {
			query = query.Where("activity_timestamp BETWEEN ? AND ?", filter.StartDate, filter.EndDate)
		}

		// Apply cursor conditions if cursor exists
		if cursor != nil {
			query = query.Where("(activity_timestamp, sequence) <= (?, ?)", cursor.Timestamp, cursor.Sequence)
		}

		// Order by timestamp and sequence for stable pagination
		query = query.Order("activity_timestamp DESC", "sequence DESC").
			Limit(limit + 1) // Get one extra to determine if there are more results

		err := query.Scan(ctx)
		if err != nil {
			return fmt.Errorf("failed to get logs: %w", err)
		}

		if len(logs) > limit {
			// Use the extra item as the next cursor
			extraItem := logs[limit]
			nextCursor = &types.LogCursor{
				Timestamp: extraItem.ActivityTimestamp,
				Sequence:  extraItem.Sequence,
			}
			logs = logs[:limit] // Remove the extra item
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return logs, nextCursor, nil
}

// GetRecentlyReviewedIDs returns the IDs of users or groups that were recently reviewed by a specific reviewer.
// Uses a time-based filter (30 minutes) with fallback to no filtering if no items are available.
func (r *ActivityModel) GetRecentlyReviewedIDs(
	ctx context.Context, reviewerID uint64, isGroup bool, limit int,
) ([]uint64, error) {
	return dbretry.Operation(ctx, func(ctx context.Context) ([]uint64, error) {
		var logs []*types.ActivityLog

		// Build query to get recently reviewed IDs
		var (
			itemType     string
			activityType enum.ActivityType
		)

		if isGroup {
			itemType = "group_id"
			activityType = enum.ActivityTypeGroupViewed
		} else {
			itemType = "user_id"
			activityType = enum.ActivityTypeUserViewed
		}

		// Get IDs reviewed by this reviewer in the last 30 minutes
		cutoffTime := time.Now().Add(-30 * time.Minute)

		var ids []uint64

		err := r.db.NewSelect().
			Model(&logs).
			Column(itemType).
			Where(itemType+" > 0").
			Where("reviewer_id = ?", reviewerID).
			Where("activity_type = ?", activityType).
			Where("activity_timestamp >= ?", cutoffTime).
			Order("activity_timestamp DESC").
			Limit(limit).
			Scan(ctx, &ids)
		if err != nil {
			return nil, fmt.Errorf("failed to get recently reviewed IDs: %w", err)
		}

		// If we have reviewed items in the time window, check if filtering would leave enough items
		if len(ids) > 0 {
			var availableCount int
			if isGroup {
				availableCount, err = r.db.NewSelect().
					Model((*types.Group)(nil)).
					Where("status = ?", enum.GroupTypeFlagged).
					Where("id NOT IN (?)", bun.In(ids)).
					Count(ctx)
			} else {
				availableCount, err = r.db.NewSelect().
					Model((*types.User)(nil)).
					Where("status = ?", enum.UserTypeFlagged).
					Where("id NOT IN (?)", bun.In(ids)).
					Count(ctx)
			}

			if err != nil {
				return nil, fmt.Errorf("failed to get available count: %w", err)
			}

			// If filtering would leave no items available, return empty slice (no filtering)
			if availableCount == 0 {
				return []uint64{}, nil
			}
		}

		return ids, nil
	})
}
