package database

import (
	"context"
	"time"

	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// FlaggedGroup stores information about a group that needs review.
// The confidence score helps prioritize which groups to review first.
type FlaggedGroup struct {
	ID           uint64    `bun:",pk"`
	Name         string    `bun:",notnull"`
	Description  string    `bun:",notnull"`
	Owner        uint64    `bun:",notnull"`
	Reason       string    `bun:",notnull"`
	Confidence   float64   `bun:",notnull"`
	LastUpdated  time.Time `bun:",notnull"`
	ThumbnailURL string
}

// ConfirmedGroup stores information about a group that has been reviewed and confirmed.
// The last_scanned field helps track when to re-check the group's members.
type ConfirmedGroup struct {
	ID          uint64 `bun:",pk"`
	Name        string `bun:",notnull"`
	Description string `bun:",notnull"`
	Owner       uint64 `bun:",notnull"`
	LastScanned time.Time
}

// GroupRepository handles database operations for group records.
type GroupRepository struct {
	db     *bun.DB
	logger *zap.Logger
}

// NewGroupRepository creates a GroupRepository with database access for
// storing and retrieving group information.
func NewGroupRepository(db *bun.DB, logger *zap.Logger) *GroupRepository {
	return &GroupRepository{
		db:     db,
		logger: logger,
	}
}

// GetNextConfirmedGroup retrieves the next confirmed group to be processed.
func (r *GroupRepository) GetNextConfirmedGroup(ctx context.Context) (*ConfirmedGroup, error) {
	var group ConfirmedGroup

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		err := tx.NewSelect().Model(&group).
			Where("last_scanned IS NULL OR last_scanned < NOW() - INTERVAL '1 day'").
			Order("last_scanned ASC NULLS FIRST").
			Limit(1).
			For("UPDATE SKIP LOCKED").
			Scan(ctx)
		if err != nil {
			r.logger.Error("Failed to get next confirmed group", zap.Error(err))
			return err
		}

		_, err = tx.NewUpdate().Model(&group).
			Set("last_scanned = ?", time.Now()).
			Where("id = ?", group.ID).
			Exec(ctx)
		if err != nil {
			r.logger.Error("Failed to update last_scanned", zap.Error(err))
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	r.logger.Info("Retrieved and updated next confirmed group",
		zap.Uint64("groupID", group.ID),
		zap.Time("lastScanned", group.LastScanned))

	return &group, nil
}

// CheckConfirmedGroups finds which groups from a list of IDs exist in confirmed_groups.
// Returns a slice of confirmed group IDs.
func (r *GroupRepository) CheckConfirmedGroups(ctx context.Context, groupIDs []uint64) ([]uint64, error) {
	var confirmedGroupIDs []uint64
	err := r.db.NewSelect().
		Model((*ConfirmedGroup)(nil)).
		Column("id").
		Where("id IN (?)", bun.In(groupIDs)).
		Scan(ctx, &confirmedGroupIDs)
	if err != nil {
		r.logger.Error("Failed to check confirmed groups", zap.Error(err))
		return nil, err
	}

	r.logger.Debug("Checked confirmed groups",
		zap.Int("total", len(groupIDs)),
		zap.Int("confirmed", len(confirmedGroupIDs)))

	return confirmedGroupIDs, nil
}

// SaveFlaggedGroups adds or updates groups in the flagged_groups table.
// For each group, it updates all fields if the group already exists,
// or inserts a new record if they don't.
func (r *GroupRepository) SaveFlaggedGroups(ctx context.Context, flaggedGroups []*FlaggedGroup) {
	r.logger.Info("Saving flagged groups", zap.Int("count", len(flaggedGroups)))

	for _, flaggedGroup := range flaggedGroups {
		_, err := r.db.NewInsert().Model(flaggedGroup).
			On("CONFLICT (id) DO UPDATE").
			Set("name = EXCLUDED.name").
			Set("description = EXCLUDED.description").
			Set("owner = EXCLUDED.owner").
			Set("reason = EXCLUDED.reason").
			Set("confidence = EXCLUDED.confidence").
			Set("last_updated = EXCLUDED.last_updated").
			Set("thumbnail_url = EXCLUDED.thumbnail_url").
			Exec(ctx)
		if err != nil {
			r.logger.Error("Error saving flagged group",
				zap.Uint64("groupID", flaggedGroup.ID),
				zap.String("name", flaggedGroup.Name),
				zap.String("reason", flaggedGroup.Reason),
				zap.Float64("confidence", flaggedGroup.Confidence),
				zap.Error(err))
			continue
		}

		r.logger.Info("Saved flagged group",
			zap.Uint64("groupID", flaggedGroup.ID),
			zap.String("name", flaggedGroup.Name),
			zap.String("reason", flaggedGroup.Reason),
			zap.Float64("confidence", flaggedGroup.Confidence),
			zap.Time("last_updated", time.Now()),
			zap.String("thumbnail_url", flaggedGroup.ThumbnailURL))
	}

	r.logger.Info("Finished saving flagged groups")
}

// ConfirmGroup moves a group from flagged_groups to confirmed_groups.
// This happens when a moderator confirms that a group is inappropriate.
func (r *GroupRepository) ConfirmGroup(ctx context.Context, group *FlaggedGroup) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		confirmedGroup := &ConfirmedGroup{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			Owner:       group.Owner,
		}

		_, err := tx.NewInsert().Model(confirmedGroup).
			On("CONFLICT (id) DO UPDATE").
			Set("name = EXCLUDED.name").
			Set("description = EXCLUDED.description").
			Set("owner = EXCLUDED.owner").
			Exec(ctx)
		if err != nil {
			r.logger.Error("Failed to insert or update group in confirmed_groups",
				zap.Error(err),
				zap.Uint64("groupID", group.ID))
			return err
		}

		_, err = tx.NewDelete().Model((*FlaggedGroup)(nil)).
			Where("id = ?", group.ID).
			Exec(ctx)
		if err != nil {
			r.logger.Error("Failed to delete group from flagged_groups",
				zap.Error(err),
				zap.Uint64("groupID", group.ID))
			return err
		}

		return nil
	})
}
