package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/robalyx/rotector/internal/database/dbretry"
	"github.com/robalyx/rotector/internal/database/models"
	"github.com/robalyx/rotector/internal/database/types"
	"github.com/robalyx/rotector/internal/database/types/enum"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// GroupService handles group-related business logic.
type GroupService struct {
	db         *bun.DB
	model      *models.GroupModel
	activity   *models.ActivityModel
	reputation *models.ReputationModel
	votes      *models.VoteModel
	logger     *zap.Logger
}

// NewGroup creates a new group service.
func NewGroup(
	db *bun.DB,
	model *models.GroupModel,
	activity *models.ActivityModel,
	reputation *models.ReputationModel,
	votes *models.VoteModel,
	logger *zap.Logger,
) *GroupService {
	return &GroupService{
		db:         db,
		model:      model,
		activity:   activity,
		reputation: reputation,
		votes:      votes,
		logger:     logger.Named("group_service"),
	}
}

// ConfirmGroup moves a group to confirmed status and creates a verification record.
func (s *GroupService) ConfirmGroup(ctx context.Context, group *types.ReviewGroup, reviewerID uint64) error {
	return dbretry.Transaction(ctx, s.db, func(ctx context.Context, tx bun.Tx) error {
		return s.ConfirmGroupWithTx(ctx, tx, group, reviewerID)
	})
}

// ConfirmGroupWithTx moves a group to confirmed status and creates a verification record using the provided transaction.
func (s *GroupService) ConfirmGroupWithTx(ctx context.Context, tx bun.Tx, group *types.ReviewGroup, reviewerID uint64) error {
	// Set reviewer ID
	group.ReviewerID = reviewerID
	group.Status = enum.GroupTypeConfirmed

	// Update group status and create verification record
	if err := s.model.ConfirmGroupWithTx(ctx, tx, group); err != nil {
		return err
	}

	// Verify votes for the group
	if err := s.votes.VerifyVotesWithTx(ctx, tx, group.ID, true, enum.VoteTypeGroup); err != nil {
		s.logger.Error("Failed to verify votes", zap.Error(err))
		return err
	}

	return nil
}

// ClearGroup moves a group to cleared status and creates a clearance record.
func (s *GroupService) ClearGroup(ctx context.Context, group *types.ReviewGroup, reviewerID uint64) error {
	return dbretry.Transaction(ctx, s.db, func(ctx context.Context, tx bun.Tx) error {
		return s.ClearGroupWithTx(ctx, tx, group, reviewerID)
	})
}

// ClearGroupWithTx moves a group to cleared status and creates a clearance record using the provided transaction.
func (s *GroupService) ClearGroupWithTx(ctx context.Context, tx bun.Tx, group *types.ReviewGroup, reviewerID uint64) error {
	// Set reviewer ID
	group.ReviewerID = reviewerID
	group.Status = enum.GroupTypeCleared

	// Update group status and create clearance record
	if err := s.model.ClearGroupWithTx(ctx, tx, group); err != nil {
		return err
	}

	// Verify votes for the group
	if err := s.votes.VerifyVotesWithTx(ctx, tx, group.ID, false, enum.VoteTypeGroup); err != nil {
		s.logger.Error("Failed to verify votes", zap.Error(err))
		return err
	}

	return nil
}

// GetGroupByID retrieves a group by ID with reputation information.
func (s *GroupService) GetGroupByID(
	ctx context.Context, groupID string, fields types.GroupField,
) (*types.ReviewGroup, error) {
	// Get the group from the model layer
	group, err := s.model.GetGroupByID(ctx, groupID, fields)
	if err != nil {
		return nil, err
	}

	// Get reputation if requested
	if fields.HasReputation() {
		reputation, err := s.reputation.GetGroupReputation(ctx, group.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get group reputation: %w", err)
		}
		group.Reputation = reputation
	}

	return group, nil
}

// GetGroupToReview finds a group to review based on the sort method and target mode.
func (s *GroupService) GetGroupToReview(
	ctx context.Context,
	sortBy enum.ReviewSortBy,
	targetMode enum.ReviewTargetMode,
	reviewerID uint64,
) (*types.ReviewGroup, error) {
	// Get recently reviewed group IDs
	recentIDs, err := s.activity.GetRecentlyReviewedIDs(ctx, reviewerID, true, 50)
	if err != nil {
		s.logger.Error("Failed to get recently reviewed group IDs", zap.Error(err))
		recentIDs = []uint64{} // Continue without filtering if there's an error
	}

	// Determine target status based on mode
	var targetStatus enum.GroupType
	switch targetMode {
	case enum.ReviewTargetModeFlagged:
		targetStatus = enum.GroupTypeFlagged
	case enum.ReviewTargetModeConfirmed:
		targetStatus = enum.GroupTypeConfirmed
	case enum.ReviewTargetModeCleared:
		targetStatus = enum.GroupTypeCleared
	}

	// Get next group to review
	result, err := s.model.GetNextToReview(ctx, targetStatus, sortBy, recentIDs)
	if err != nil {
		if errors.Is(err, types.ErrNoGroupsToReview) {
			// If no groups found with primary status, try other statuses in order
			var fallbackStatuses []enum.GroupType
			switch targetMode {
			case enum.ReviewTargetModeFlagged:
				fallbackStatuses = []enum.GroupType{enum.GroupTypeConfirmed, enum.GroupTypeCleared}
			case enum.ReviewTargetModeConfirmed:
				fallbackStatuses = []enum.GroupType{enum.GroupTypeFlagged, enum.GroupTypeCleared}
			case enum.ReviewTargetModeCleared:
				fallbackStatuses = []enum.GroupType{enum.GroupTypeFlagged, enum.GroupTypeConfirmed}
			}

			for _, status := range fallbackStatuses {
				result, err = s.model.GetNextToReview(ctx, status, sortBy, recentIDs)
				if err == nil {
					break
				}
				if !errors.Is(err, types.ErrNoGroupsToReview) {
					return nil, err
				}
			}

			if err != nil {
				return nil, types.ErrNoGroupsToReview
			}
		} else {
			return nil, err
		}
	}

	// Get reputation
	reputation, err := s.reputation.GetGroupReputation(ctx, result.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group reputation: %w", err)
	}
	result.Reputation = reputation

	return result, nil
}

// SaveGroups handles the business logic for saving groups.
func (s *GroupService) SaveGroups(ctx context.Context, groups map[uint64]*types.ReviewGroup) error {
	// Get list of group IDs to check
	groupIDs := make([]uint64, 0, len(groups))
	for id := range groups {
		groupIDs = append(groupIDs, id)
	}

	// Get existing groups with all their data
	existingGroups, err := s.model.GetGroupsByIDs(
		ctx,
		groupIDs,
		types.GroupFieldBasic|types.GroupFieldTimestamps|types.GroupFieldReasons,
	)
	if err != nil {
		return fmt.Errorf("failed to get existing groups: %w", err)
	}

	// Prepare groups for saving
	groupsToSave := make([]*types.ReviewGroup, 0, len(groups))
	for id, group := range groups {
		// Generate UUID for new groups
		if group.UUID == uuid.Nil {
			group.UUID = uuid.New()
		}

		// Handle reasons merging and determine status
		existingGroup, ok := existingGroups[id]
		if ok {
			group.Status = existingGroup.Status

			// Create new reasons map if it doesn't exist
			if group.Reasons == nil {
				group.Reasons = make(types.Reasons[enum.GroupReasonType])
			}

			// Copy over existing reasons, only adding new ones
			for reasonType, reason := range existingGroup.Reasons {
				if _, exists := group.Reasons[reasonType]; !exists {
					group.Reasons[reasonType] = reason
				}
			}
		} else {
			group.Status = enum.GroupTypeFlagged
		}

		groupsToSave = append(groupsToSave, group)
	}

	// Save the groups
	err = dbretry.Transaction(ctx, s.db, func(ctx context.Context, tx bun.Tx) error {
		// Save groups with their reasons
		if err := s.model.SaveGroups(ctx, tx, groupsToSave); err != nil {
			return err
		}

		// NOTE: any additional logic can be added here

		return nil
	})
	if err != nil {
		return err
	}

	s.logger.Debug("Successfully saved groups",
		zap.Int("totalGroups", len(groups)))

	return nil
}
