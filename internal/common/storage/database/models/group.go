package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/robalyx/rotector/internal/common/storage/database/types"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// GroupModel handles database operations for group records.
type GroupModel struct {
	db       *bun.DB
	activity *ActivityModel
	logger   *zap.Logger
}

// NewGroup creates a GroupModel with database access for
// storing and retrieving group information.
func NewGroup(db *bun.DB, activity *ActivityModel, logger *zap.Logger) *GroupModel {
	return &GroupModel{
		db:       db,
		activity: activity,
		logger:   logger,
	}
}

// SaveGroups updates or inserts groups into their appropriate tables based on their current status.
func (r *GroupModel) SaveGroups(ctx context.Context, groups map[uint64]*types.Group) error {
	// Get list of group IDs to check
	groupIDs := make([]uint64, 0, len(groups))
	for id := range groups {
		groupIDs = append(groupIDs, id)
	}

	// Get existing groups with all their data
	existingGroups, err := r.GetGroupsByIDs(ctx, groupIDs, types.GroupFields{
		Timestamps: true,
	})
	if err != nil {
		return fmt.Errorf("failed to get existing groups: %w", err)
	}

	// Initialize slices for each table
	flaggedGroups := make([]*types.FlaggedGroup, 0)
	confirmedGroups := make([]*types.ConfirmedGroup, 0)
	clearedGroups := make([]*types.ClearedGroup, 0)
	lockedGroups := make([]*types.LockedGroup, 0)
	counts := make(map[types.GroupType]int)

	// Group groups by their target tables
	for id, group := range groups {
		// Generate UUID for new groups
		if group.UUID == uuid.Nil {
			group.UUID = uuid.New()
		}

		// Get existing group data if available
		var status types.GroupType
		existingGroup := existingGroups[id]
		if existingGroup.Status != types.GroupTypeUnflagged {
			status = existingGroup.Status
		} else {
			// Default to flagged_groups for new groups
			status = types.GroupTypeFlagged
		}

		switch status {
		case types.GroupTypeConfirmed:
			confirmedGroups = append(confirmedGroups, &types.ConfirmedGroup{
				Group:      *group,
				VerifiedAt: existingGroup.VerifiedAt,
			})
		case types.GroupTypeFlagged:
			flaggedGroups = append(flaggedGroups, &types.FlaggedGroup{
				Group: *group,
			})
		case types.GroupTypeCleared:
			clearedGroups = append(clearedGroups, &types.ClearedGroup{
				Group:     *group,
				ClearedAt: existingGroup.ClearedAt,
			})
		case types.GroupTypeLocked:
			lockedGroups = append(lockedGroups, &types.LockedGroup{
				Group:    *group,
				LockedAt: existingGroup.LockedAt,
			})
		case types.GroupTypeUnflagged:
			continue
		}
		counts[status]++
	}

	// Update each table
	err = r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Helper function to update a table
		updateTable := func(groups interface{}, status types.GroupType) error {
			if counts[status] == 0 {
				return nil
			}

			_, err := tx.NewInsert().
				Model(groups).
				On("CONFLICT (id) DO UPDATE").
				Set("uuid = EXCLUDED.uuid").
				Set("name = EXCLUDED.name").
				Set("description = EXCLUDED.description").
				Set("owner = EXCLUDED.owner").
				Set("shout = EXCLUDED.shout").
				Set("reason = EXCLUDED.reason").
				Set("confidence = EXCLUDED.confidence").
				Set("last_scanned = EXCLUDED.last_scanned").
				Set("last_updated = EXCLUDED.last_updated").
				Set("last_viewed = EXCLUDED.last_viewed").
				Set("last_purge_check = EXCLUDED.last_purge_check").
				Set("thumbnail_url = EXCLUDED.thumbnail_url").
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to update %s groups: %w", status, err)
			}
			return nil
		}

		// Update each table with its corresponding slice
		if err := updateTable(&flaggedGroups, types.GroupTypeFlagged); err != nil {
			return err
		}
		if err := updateTable(&confirmedGroups, types.GroupTypeConfirmed); err != nil {
			return err
		}
		if err := updateTable(&clearedGroups, types.GroupTypeCleared); err != nil {
			return err
		}
		if err := updateTable(&lockedGroups, types.GroupTypeLocked); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to save groups: %w", err)
	}

	r.logger.Debug("Successfully saved groups",
		zap.Int("totalGroups", len(groups)),
		zap.Int("flaggedGroups", counts[types.GroupTypeFlagged]),
		zap.Int("confirmedGroups", counts[types.GroupTypeConfirmed]),
		zap.Int("clearedGroups", counts[types.GroupTypeCleared]),
		zap.Int("lockedGroups", counts[types.GroupTypeLocked]))

	return nil
}

// ConfirmGroup moves a group from other group tables to confirmed_groups.
// This happens when a moderator confirms that a group is inappropriate.
func (r *GroupModel) ConfirmGroup(ctx context.Context, group *types.ReviewGroup) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		confirmedGroup := &types.ConfirmedGroup{
			Group:      group.Group,
			VerifiedAt: time.Now(),
		}

		// Move group to confirmed_groups table
		_, err := tx.NewInsert().Model(confirmedGroup).
			On("CONFLICT (id) DO UPDATE").
			Set("name = EXCLUDED.name").
			Set("description = EXCLUDED.description").
			Set("owner = EXCLUDED.owner").
			Set("reason = EXCLUDED.reason").
			Set("confidence = EXCLUDED.confidence").
			Set("last_scanned = EXCLUDED.last_scanned").
			Set("last_updated = EXCLUDED.last_updated").
			Set("last_viewed = EXCLUDED.last_viewed").
			Set("thumbnail_url = EXCLUDED.thumbnail_url").
			Set("verified_at = EXCLUDED.verified_at").
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to insert or update group in confirmed_groups: %w (groupID=%d)", err, group.ID)
		}

		// Delete from flagged_groups table
		_, err = tx.NewDelete().Model((*types.FlaggedGroup)(nil)).Where("id = ?", group.ID).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete group from flagged_groups: %w (groupID=%d)", err, group.ID)
		}

		// Delete from cleared_groups table
		_, err = tx.NewDelete().Model((*types.ClearedGroup)(nil)).Where("id = ?", group.ID).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete group from cleared_groups: %w (groupID=%d)", err, group.ID)
		}

		// Delete from locked_groups table
		_, err = tx.NewDelete().Model((*types.LockedGroup)(nil)).Where("id = ?", group.ID).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete group from locked_groups: %w (groupID=%d)", err, group.ID)
		}

		return nil
	})
}

// ClearGroup moves a group from other group tables to cleared_groups.
// This happens when a moderator determines that a group was incorrectly flagged.
func (r *GroupModel) ClearGroup(ctx context.Context, group *types.ReviewGroup) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		clearedGroup := &types.ClearedGroup{
			Group:     group.Group,
			ClearedAt: time.Now(),
		}

		// Move group to cleared_groups table
		_, err := tx.NewInsert().Model(clearedGroup).
			On("CONFLICT (id) DO UPDATE").
			Set("name = EXCLUDED.name").
			Set("description = EXCLUDED.description").
			Set("owner = EXCLUDED.owner").
			Set("reason = EXCLUDED.reason").
			Set("confidence = EXCLUDED.confidence").
			Set("last_scanned = EXCLUDED.last_scanned").
			Set("last_updated = EXCLUDED.last_updated").
			Set("last_viewed = EXCLUDED.last_viewed").
			Set("thumbnail_url = EXCLUDED.thumbnail_url").
			Set("cleared_at = EXCLUDED.cleared_at").
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to insert or update group in cleared_groups: %w (groupID=%d)", err, group.ID)
		}

		// Delete from flagged_groups table
		_, err = tx.NewDelete().Model((*types.FlaggedGroup)(nil)).Where("id = ?", group.ID).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete group from flagged_groups: %w (groupID=%d)", err, group.ID)
		}

		// Delete from confirmed_groups table
		_, err = tx.NewDelete().Model((*types.ConfirmedGroup)(nil)).Where("id = ?", group.ID).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete group from confirmed_groups: %w (groupID=%d)", err, group.ID)
		}

		// Delete from locked_groups table
		_, err = tx.NewDelete().Model((*types.LockedGroup)(nil)).Where("id = ?", group.ID).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete group from locked_groups: %w (groupID=%d)", err, group.ID)
		}

		r.logger.Debug("Group cleared and moved to cleared_groups", zap.Uint64("groupID", group.ID))

		return nil
	})
}

// GetGroupByID retrieves a group by either their numeric ID or UUID.
func (r *GroupModel) GetGroupByID(ctx context.Context, groupID string, fields types.GroupFields) (*types.ReviewGroup, error) {
	var result types.ReviewGroup

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Try each model in order until we find a group
		models := []interface{}{
			&types.FlaggedGroup{},
			&types.ConfirmedGroup{},
			&types.ClearedGroup{},
			&types.LockedGroup{},
		}

		for _, model := range models {
			query := tx.NewSelect().
				Model(model).
				Column(fields.Columns()...).
				For("UPDATE")

			// Check if input is numeric (ID) or string (UUID)
			if id, err := strconv.ParseUint(groupID, 10, 64); err == nil {
				query.Where("id = ?", id)
			} else {
				// Parse UUID string
				uid, err := uuid.Parse(groupID)
				if err != nil {
					return fmt.Errorf("invalid UUID format: %w", err)
				}
				query.Where("uuid = ?", uid)
			}

			err := query.Scan(ctx)
			if err == nil {
				// Set result based on model type
				switch m := model.(type) {
				case *types.FlaggedGroup:
					result.Group = m.Group
					result.Status = types.GroupTypeFlagged
				case *types.ConfirmedGroup:
					result.Group = m.Group
					result.VerifiedAt = m.VerifiedAt
					result.Status = types.GroupTypeConfirmed
				case *types.ClearedGroup:
					result.Group = m.Group
					result.ClearedAt = m.ClearedAt
					result.Status = types.GroupTypeCleared
				case *types.LockedGroup:
					result.Group = m.Group
					result.LockedAt = m.LockedAt
					result.Status = types.GroupTypeLocked
				}

				// Get reputation
				var reputation types.GroupReputation
				err = tx.NewSelect().
					Model(&reputation).
					Where("id = ?", result.ID).
					Scan(ctx)
				if err != nil && !errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("failed to get group reputation: %w", err)
				}
				result.Reputation = reputation.Reputation

				// Update last_viewed if requested
				_, err = tx.NewUpdate().
					Model(model).
					Set("last_viewed = ?", time.Now()).
					Where("id = ?", result.ID).
					Exec(ctx)
				if err != nil {
					return err
				}

				return nil
			}
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
		}

		return types.ErrGroupNotFound
	})
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetGroupsByIDs retrieves specified group information for a list of group IDs.
// Returns a map of group IDs to review groups.
func (r *GroupModel) GetGroupsByIDs(ctx context.Context, groupIDs []uint64, fields types.GroupFields) (map[uint64]*types.ReviewGroup, error) {
	groups := make(map[uint64]*types.ReviewGroup)

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Build query with selected fields
		columns := fields.Columns()

		// Query confirmed groups
		var confirmedGroups []types.ConfirmedGroup
		err := tx.NewSelect().
			Model(&confirmedGroups).
			Column(columns...).
			Where("id IN (?)", bun.In(groupIDs)).
			Scan(ctx)
		if err != nil {
			return fmt.Errorf("failed to get confirmed groups: %w", err)
		}
		for _, group := range confirmedGroups {
			groups[group.ID] = &types.ReviewGroup{
				Group:      group.Group,
				VerifiedAt: group.VerifiedAt,
				Status:     types.GroupTypeConfirmed,
			}
		}

		// Query flagged groups
		var flaggedGroups []types.FlaggedGroup
		err = tx.NewSelect().
			Model(&flaggedGroups).
			Column(columns...).
			Where("id IN (?)", bun.In(groupIDs)).
			Scan(ctx)
		if err != nil {
			return fmt.Errorf("failed to get flagged groups: %w", err)
		}
		for _, group := range flaggedGroups {
			groups[group.ID] = &types.ReviewGroup{
				Group:  group.Group,
				Status: types.GroupTypeFlagged,
			}
		}

		// Query cleared groups
		var clearedGroups []types.ClearedGroup
		err = tx.NewSelect().
			Model(&clearedGroups).
			Column(columns...).
			Where("id IN (?)", bun.In(groupIDs)).
			Scan(ctx)
		if err != nil {
			return fmt.Errorf("failed to get cleared groups: %w", err)
		}
		for _, group := range clearedGroups {
			groups[group.ID] = &types.ReviewGroup{
				Group:     group.Group,
				ClearedAt: group.ClearedAt,
				Status:    types.GroupTypeCleared,
			}
		}

		// Query locked groups
		var lockedGroups []types.LockedGroup
		err = tx.NewSelect().
			Model(&lockedGroups).
			Column(columns...).
			Where("id IN (?)", bun.In(groupIDs)).
			Scan(ctx)
		if err != nil {
			return fmt.Errorf("failed to get locked groups: %w", err)
		}
		for _, group := range lockedGroups {
			groups[group.ID] = &types.ReviewGroup{
				Group:    group.Group,
				LockedAt: group.LockedAt,
				Status:   types.GroupTypeLocked,
			}
		}

		// Mark remaining IDs as unflagged
		for _, id := range groupIDs {
			if _, ok := groups[id]; !ok {
				groups[id] = &types.ReviewGroup{
					Group:  types.Group{ID: id},
					Status: types.GroupTypeUnflagged,
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get groups by IDs: %w (groupCount=%d)", err, len(groupIDs))
	}

	r.logger.Debug("Retrieved groups by IDs",
		zap.Int("requestedCount", len(groupIDs)),
		zap.Int("foundCount", len(groups)))

	return groups, nil
}

func (r *GroupModel) UpdateTrainingVotes(ctx context.Context, groupID uint64, isUpvote bool) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var reputation types.GroupReputation
		err := tx.NewSelect().
			Model(&reputation).
			Where("id = ?", groupID).
			For("UPDATE").
			Scan(ctx)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		if isUpvote {
			reputation.Upvotes++
		} else {
			reputation.Downvotes++
		}
		reputation.ID = groupID
		reputation.Score = reputation.Upvotes - reputation.Downvotes
		reputation.UpdatedAt = time.Now()

		_, err = tx.NewInsert().
			Model(&reputation).
			On("CONFLICT (id) DO UPDATE").
			Set("upvotes = EXCLUDED.upvotes").
			Set("downvotes = EXCLUDED.downvotes").
			Set("score = EXCLUDED.score").
			Set("updated_at = EXCLUDED.updated_at").
			Exec(ctx)
		return err
	})
}

// GetGroupsToCheck finds groups that haven't been checked for locked status recently.
func (r *GroupModel) GetGroupsToCheck(ctx context.Context, limit int) ([]uint64, error) {
	var groupIDs []uint64
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Get and update confirmed groups
		err := tx.NewRaw(`
			WITH updated AS (
				UPDATE confirmed_groups
				SET last_purge_check = NOW()
				WHERE id IN (
					SELECT id FROM confirmed_groups
					WHERE last_purge_check < NOW() - INTERVAL '1 day'
					ORDER BY last_purge_check ASC
					LIMIT ?
					FOR UPDATE SKIP LOCKED
				)
				RETURNING id
			)
			SELECT * FROM updated
		`, limit/2).Scan(ctx, &groupIDs)
		if err != nil {
			return fmt.Errorf("failed to get and update confirmed groups: %w", err)
		}

		// Get and update flagged groups
		var flaggedIDs []uint64
		err = tx.NewRaw(`
			WITH updated AS (
				UPDATE flagged_groups
				SET last_purge_check = NOW()
				WHERE id IN (
					SELECT id FROM flagged_groups
					WHERE last_purge_check < NOW() - INTERVAL '1 day'
					ORDER BY last_purge_check ASC
					LIMIT ?
					FOR UPDATE SKIP LOCKED
				)
				RETURNING id
			)
			SELECT * FROM updated
		`, limit/2).Scan(ctx, &flaggedIDs)
		if err != nil {
			return fmt.Errorf("failed to get and update flagged groups: %w", err)
		}
		groupIDs = append(groupIDs, flaggedIDs...)

		return nil
	})

	return groupIDs, err
}

// PurgeOldClearedGroups removes cleared groups older than the cutoff date.
// This helps maintain database size by removing groups that were cleared long ago.
func (r *GroupModel) PurgeOldClearedGroups(ctx context.Context, cutoffDate time.Time) (int, error) {
	result, err := r.db.NewDelete().
		Model((*types.ClearedGroup)(nil)).
		Where("cleared_at < ?", cutoffDate).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf(
			"failed to purge old cleared groups: %w (cutoffDate=%s)",
			err, cutoffDate.Format(time.RFC3339),
		)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	r.logger.Debug("Purged old cleared groups",
		zap.Int64("rowsAffected", affected),
		zap.Time("cutoffDate", cutoffDate))

	return int(affected), nil
}

// RemoveLockedGroups moves groups from confirmed_groups and flagged_groups to locked_groups.
// This happens when groups are found to be locked by Roblox.
func (r *GroupModel) RemoveLockedGroups(ctx context.Context, groupIDs []uint64) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Move confirmed groups to locked_groups
		var confirmedGroups []types.ConfirmedGroup
		err := tx.NewSelect().Model(&confirmedGroups).
			Where("id IN (?)", bun.In(groupIDs)).
			Scan(ctx)
		if err != nil {
			return fmt.Errorf("failed to select confirmed groups for locking: %w", err)
		}

		for _, group := range confirmedGroups {
			lockedGroup := &types.LockedGroup{
				Group:    group.Group,
				LockedAt: time.Now(),
			}
			_, err = tx.NewInsert().Model(lockedGroup).
				On("CONFLICT (id) DO UPDATE").
				Exec(ctx)
			if err != nil {
				return fmt.Errorf(
					"failed to insert locked group from confirmed_groups: %w (groupID=%d)",
					err, group.ID,
				)
			}
		}

		// Move flagged groups to locked_groups
		var flaggedGroups []types.FlaggedGroup
		err = tx.NewSelect().Model(&flaggedGroups).
			Where("id IN (?)", bun.In(groupIDs)).
			Scan(ctx)
		if err != nil {
			return fmt.Errorf("failed to select flagged groups for locking: %w", err)
		}

		for _, group := range flaggedGroups {
			lockedGroup := &types.LockedGroup{
				Group:    group.Group,
				LockedAt: time.Now(),
			}
			_, err = tx.NewInsert().Model(lockedGroup).
				On("CONFLICT (id) DO UPDATE").
				Exec(ctx)
			if err != nil {
				return fmt.Errorf(
					"failed to insert locked group from flagged_groups: %w (groupID=%d)",
					err, group.ID,
				)
			}
		}

		// Remove groups from confirmed_groups
		_, err = tx.NewDelete().Model((*types.ConfirmedGroup)(nil)).
			Where("id IN (?)", bun.In(groupIDs)).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf(
				"failed to remove locked groups from confirmed_groups: %w (groupCount=%d)",
				err, len(groupIDs),
			)
		}

		// Remove groups from flagged_groups
		_, err = tx.NewDelete().Model((*types.FlaggedGroup)(nil)).
			Where("id IN (?)", bun.In(groupIDs)).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf(
				"failed to remove locked groups from flagged_groups: %w (groupCount=%d)",
				err, len(groupIDs),
			)
		}

		r.logger.Debug("Moved locked groups to locked_groups", zap.Int("count", len(groupIDs)))
		return nil
	})
}

// DeleteGroup removes a group and all associated data from the database.
func (r *GroupModel) DeleteGroup(ctx context.Context, groupID uint64) (bool, error) {
	var totalAffected int64
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Delete from flagged_groups
		result, err := tx.NewDelete().
			Model((*types.FlaggedGroup)(nil)).
			Where("id = ?", groupID).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete from flagged_groups: %w", err)
		}
		affected, _ := result.RowsAffected()
		totalAffected += affected

		// Delete from confirmed_groups
		result, err = tx.NewDelete().
			Model((*types.ConfirmedGroup)(nil)).
			Where("id = ?", groupID).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete from confirmed_groups: %w", err)
		}
		affected, _ = result.RowsAffected()
		totalAffected += affected

		// Delete from cleared_groups
		result, err = tx.NewDelete().
			Model((*types.ClearedGroup)(nil)).
			Where("id = ?", groupID).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete from cleared_groups: %w", err)
		}
		affected, _ = result.RowsAffected()
		totalAffected += affected

		// Delete from locked_groups
		result, err = tx.NewDelete().
			Model((*types.LockedGroup)(nil)).
			Where("id = ?", groupID).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete from locked_groups: %w", err)
		}
		affected, _ = result.RowsAffected()
		totalAffected += affected

		return nil
	})

	return totalAffected > 0, err
}

// GetGroupToScan finds the next group to scan from confirmed_groups, falling back to flagged_groups
// if no confirmed groups are available.
func (r *GroupModel) GetGroupToScan(ctx context.Context) (*types.Group, error) {
	var group *types.Group
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// First try confirmed groups
		var confirmedGroup types.ConfirmedGroup
		err := tx.NewSelect().Model(&confirmedGroup).
			Where("last_scanned < NOW() - INTERVAL '1 day'").
			Order("last_scanned ASC").
			Limit(1).
			For("UPDATE SKIP LOCKED").
			Scan(ctx)
		if err == nil {
			// Update last_scanned
			_, err = tx.NewUpdate().Model(&confirmedGroup).
				Set("last_scanned = ?", time.Now()).
				Where("id = ?", confirmedGroup.ID).
				Exec(ctx)
			if err != nil {
				return fmt.Errorf(
					"failed to update last_scanned for confirmed group: %w (groupID=%d)",
					err, confirmedGroup.ID,
				)
			}
			group = &confirmedGroup.Group
			return nil
		}

		// If no confirmed groups, try flagged groups
		var flaggedGroup types.FlaggedGroup
		err = tx.NewSelect().Model(&flaggedGroup).
			Where("last_scanned < NOW() - INTERVAL '1 day'").
			Order("last_scanned ASC").
			Limit(1).
			For("UPDATE SKIP LOCKED").
			Scan(ctx)
		if err != nil {
			return fmt.Errorf("failed to get group to scan: %w", err)
		}

		// Update last_scanned
		_, err = tx.NewUpdate().Model(&flaggedGroup).
			Set("last_scanned = ?", time.Now()).
			Where("id = ?", flaggedGroup.ID).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf(
				"failed to update last_scanned for flagged group: %w (groupID=%d)",
				err, flaggedGroup.ID,
			)
		}
		group = &flaggedGroup.Group
		return nil
	})
	if err != nil {
		return nil, err
	}

	return group, nil
}

// CheckConfirmedGroups checks which groups from a list of IDs exist in any group table.
// Returns a map of group IDs to their status (confirmed, flagged, cleared, locked).
func (r *GroupModel) CheckConfirmedGroups(ctx context.Context, groupIDs []uint64) ([]uint64, error) {
	var confirmedGroupIDs []uint64

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Query confirmed groups
		err := tx.NewSelect().
			Model((*types.ConfirmedGroup)(nil)).
			Column("id").
			Where("id IN (?)", bun.In(groupIDs)).
			Scan(ctx, &confirmedGroupIDs)
		if err != nil {
			return fmt.Errorf("failed to query confirmed groups: %w", err)
		}

		return nil
	})

	return confirmedGroupIDs, err
}

// GetGroupToReview finds a group to review based on the sort method and target mode.
func (r *GroupModel) GetGroupToReview(ctx context.Context, sortBy types.ReviewSortBy, targetMode types.ReviewTargetMode, reviewerID uint64) (*types.ReviewGroup, error) {
	// Get recently reviewed group IDs
	recentIDs, err := r.activity.GetRecentlyReviewedIDs(ctx, reviewerID, true, 100)
	if err != nil {
		r.logger.Error("Failed to get recently reviewed group IDs", zap.Error(err))
		// Continue without filtering if there's an error
		recentIDs = []uint64{}
	}

	// Define models in priority order based on target mode
	var models []interface{}
	switch targetMode {
	case types.FlaggedReviewTarget:
		models = []interface{}{
			&types.FlaggedGroup{},   // Primary target
			&types.ConfirmedGroup{}, // First fallback
			&types.ClearedGroup{},   // Second fallback
			&types.LockedGroup{},    // Last fallback
		}
	case types.ConfirmedReviewTarget:
		models = []interface{}{
			&types.ConfirmedGroup{}, // Primary target
			&types.FlaggedGroup{},   // First fallback
			&types.ClearedGroup{},   // Second fallback
			&types.LockedGroup{},    // Last fallback
		}
	case types.ClearedReviewTarget:
		models = []interface{}{
			&types.ClearedGroup{},   // Primary target
			&types.FlaggedGroup{},   // First fallback
			&types.ConfirmedGroup{}, // Second fallback
			&types.LockedGroup{},    // Last fallback
		}
	case types.BannedReviewTarget:
		models = []interface{}{
			&types.LockedGroup{},    // Primary target
			&types.FlaggedGroup{},   // First fallback
			&types.ConfirmedGroup{}, // Second fallback
			&types.ClearedGroup{},   // Last fallback
		}
	}

	// Try each model in order until we find a group
	for _, model := range models {
		result, err := r.getNextToReview(ctx, model, sortBy, recentIDs)
		if err == nil {
			return result, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	return nil, types.ErrNoGroupsToReview
}

// getNextToReview handles the common logic for getting the next item to review.
func (r *GroupModel) getNextToReview(ctx context.Context, model interface{}, sortBy types.ReviewSortBy, recentIDs []uint64) (*types.ReviewGroup, error) {
	var result types.ReviewGroup
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Build subquery to get ID
		subq := tx.NewSelect().
			Model(model).
			Column("id")

		// Exclude recently reviewed IDs if any exist
		if len(recentIDs) > 0 {
			subq.Where("id NOT IN (?)", bun.In(recentIDs))
		}

		// Apply sort order to subquery
		switch sortBy {
		case types.ReviewSortByConfidence:
			subq.Order("confidence DESC")
		case types.ReviewSortByLastUpdated:
			subq.Order("last_updated ASC")
		case types.ReviewSortByReputation:
			subq.Join("LEFT JOIN group_reputations ON group_reputations.id = ?TableAlias.id").
				OrderExpr("COALESCE(group_reputations.score, 0) ASC")
		case types.ReviewSortByRandom:
			subq.OrderExpr("RANDOM()")
		}

		subq.Limit(1)

		// Main query to get the full record with FOR UPDATE
		err := tx.NewSelect().
			Model(model).
			Where("id = (?)", subq).
			For("UPDATE").
			Scan(ctx)
		if err != nil {
			return err
		}

		// Set result based on model type
		switch m := model.(type) {
		case *types.FlaggedGroup:
			result.Group = m.Group
			result.Status = types.GroupTypeFlagged
		case *types.ConfirmedGroup:
			result.Group = m.Group
			result.VerifiedAt = m.VerifiedAt
			result.Status = types.GroupTypeConfirmed
		case *types.ClearedGroup:
			result.Group = m.Group
			result.ClearedAt = m.ClearedAt
			result.Status = types.GroupTypeCleared
		case *types.LockedGroup:
			result.Group = m.Group
			result.LockedAt = m.LockedAt
			result.Status = types.GroupTypeLocked
		default:
			return fmt.Errorf("%w: %T", types.ErrUnsupportedModel, model)
		}

		// Get reputation
		var reputation types.GroupReputation
		err = tx.NewSelect().
			Model(&reputation).
			Where("id = ?", result.ID).
			Scan(ctx)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("failed to get group reputation: %w", err)
		}
		result.Reputation = reputation.Reputation

		// Update last_viewed
		_, err = tx.NewUpdate().
			Model(model).
			Set("last_viewed = ?", time.Now()).
			Where("id = ?", result.ID).
			Exec(ctx)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &result, nil
}
