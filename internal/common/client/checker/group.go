package checker

import (
	"context"
	"math"
	"sync"
	"time"

	apiTypes "github.com/jaxron/roapi.go/pkg/api/types"
	"github.com/robalyx/rotector/internal/common/client/fetcher"
	"github.com/robalyx/rotector/internal/common/storage/database"
	"github.com/robalyx/rotector/internal/common/storage/database/types"
	"github.com/robalyx/rotector/internal/common/storage/database/types/enum"
	"go.uber.org/zap"
)

// GroupCheckResult contains the result of checking a user's groups.
type GroupCheckResult struct {
	UserID      uint64
	User        *types.User
	AutoFlagged bool
}

// GroupChecker handles the checking of user groups by comparing them against
// a database of known inappropriate groups.
type GroupChecker struct {
	db                   *database.Client
	logger               *zap.Logger
	maxGroupMembersTrack uint64
	minFlaggedOverride   int
	minFlaggedPercentage float64
}

// NewGroupChecker creates a GroupChecker with database access for looking up
// flagged group information.
func NewGroupChecker(db *database.Client, logger *zap.Logger, maxGroupMembersTrack uint64, minFlaggedOverride int, minFlaggedPercentage float64) *GroupChecker {
	return &GroupChecker{
		db:                   db,
		logger:               logger,
		maxGroupMembersTrack: maxGroupMembersTrack,
		minFlaggedOverride:   minFlaggedOverride,
		minFlaggedPercentage: minFlaggedPercentage,
	}
}

// CheckGroupPercentages analyzes groups to find those exceeding the flagged user threshold.
func (c *GroupChecker) CheckGroupPercentages(groupInfos []*apiTypes.GroupResponse, groupToFlaggedUsers map[uint64][]uint64) map[uint64]*types.Group {
	flaggedGroups := make(map[uint64]*types.Group)

	// Identify groups that exceed thresholds
	for _, groupInfo := range groupInfos {
		flaggedUsers := groupToFlaggedUsers[groupInfo.ID]

		var reason string

		// Calculate percentage of flagged users
		percentage := (float64(len(flaggedUsers)) / float64(groupInfo.MemberCount)) * 100

		// Determine if group should be flagged
		switch {
		case len(flaggedUsers) >= c.minFlaggedOverride:
			reason = "Group has large number of flagged users"
		case percentage >= c.minFlaggedPercentage:
			reason = "Group has large percentage of flagged users"
		default:
			continue
		}

		now := time.Now()
		flaggedGroups[groupInfo.ID] = &types.Group{
			ID:             groupInfo.ID,
			Name:           groupInfo.Name,
			Description:    groupInfo.Description,
			Owner:          groupInfo.Owner,
			Shout:          groupInfo.Shout,
			Reason:         reason,
			Confidence:     0, // NOTE: Confidence will be updated
			LastUpdated:    now,
			LastPurgeCheck: now,
		}
	}

	// If no groups were flagged, return empty map
	if len(flaggedGroups) == 0 {
		return flaggedGroups
	}

	// Collect all unique flagged user IDs
	allFlaggedUserIDs := make([]uint64, 0)
	for groupID := range flaggedGroups {
		allFlaggedUserIDs = append(allFlaggedUserIDs, groupToFlaggedUsers[groupID]...)
	}

	// Get user data for confidence calculation
	users, err := c.db.Users().GetUsersByIDs(context.Background(), allFlaggedUserIDs, types.UserFields{
		Basic:      true,
		Confidence: true,
	})
	if err != nil {
		c.logger.Error("Failed to get user confidence data", zap.Error(err))
		return flaggedGroups
	}

	// Calculate average confidence for each flagged group
	for groupID, group := range flaggedGroups {
		group.Confidence = c.calculateGroupConfidence(groupToFlaggedUsers[groupID], users)
	}

	return flaggedGroups
}

// calculateGroupConfidence computes the confidence score for a group based on its flagged users.
func (c *GroupChecker) calculateGroupConfidence(flaggedUsers []uint64, users map[uint64]*types.ReviewUser) float64 {
	var totalConfidence float64
	var validUserCount int

	for _, userID := range flaggedUsers {
		if user, exists := users[userID]; exists && user.Status != enum.UserTypeUnflagged {
			totalConfidence += user.Confidence
			validUserCount++
		}
	}

	if validUserCount == 0 {
		c.logger.Fatal("Unreachable: No valid users found for group")
		return 0.50
	}

	// Calculate average confidence
	avgConfidence := totalConfidence / float64(validUserCount)

	// Apply 20% boost if group exceeds override threshold
	if len(flaggedUsers) >= c.minFlaggedOverride {
		avgConfidence *= 1.2
	}

	// Clamp confidence between 0 and 1
	avgConfidence = math.Min(avgConfidence, 1.0)

	// Round confidence to 2 decimal places
	return math.Round(avgConfidence*100) / 100
}

// ProcessUsers checks multiple users' groups concurrently and returns flagged users.
func (c *GroupChecker) ProcessUsers(userInfos []*fetcher.Info) map[uint64]*types.User {
	// Collect all unique group IDs across all users
	uniqueGroupIDs := make(map[uint64]struct{})
	for _, userInfo := range userInfos {
		for _, group := range userInfo.Groups.Data {
			uniqueGroupIDs[group.Group.ID] = struct{}{}
		}
	}

	// Convert unique IDs to slice
	groupIDs := make([]uint64, 0, len(uniqueGroupIDs))
	for groupID := range uniqueGroupIDs {
		groupIDs = append(groupIDs, groupID)
	}

	// Fetch all existing groups
	existingGroups, err := c.db.Groups().GetGroupsByIDs(context.Background(), groupIDs, types.GroupFields{
		Basic:  true,
		Reason: true,
	})
	if err != nil {
		c.logger.Error("Failed to fetch existing groups", zap.Error(err))
		return nil
	}

	// Process each user concurrently
	var wg sync.WaitGroup
	resultsChan := make(chan GroupCheckResult, len(userInfos))

	// Spawn a goroutine for each user
	for _, userInfo := range userInfos {
		wg.Add(1)
		go func(info *fetcher.Info) {
			defer wg.Done()

			// Process user groups
			user, autoFlagged := c.processUserGroups(info, existingGroups)
			resultsChan <- GroupCheckResult{
				UserID:      info.ID,
				User:        user,
				AutoFlagged: autoFlagged,
			}
		}(userInfo)
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect flagged users
	flaggedUsers := make(map[uint64]*types.User)
	for result := range resultsChan {
		if result.AutoFlagged {
			flaggedUsers[result.UserID] = result.User
		}
	}

	return flaggedUsers
}

// processUserGroups checks if a user should be flagged based on their groups.
func (c *GroupChecker) processUserGroups(userInfo *fetcher.Info, existingGroups map[uint64]*types.ReviewGroup) (*types.User, bool) {
	// Skip users with very few groups to avoid false positives
	if len(userInfo.Groups.Data) < 2 {
		return nil, false
	}

	// Count confirmed and flagged groups
	confirmedCount := 0
	flaggedCount := 0

	for _, group := range userInfo.Groups.Data {
		if reviewGroup, exists := existingGroups[group.Group.ID]; exists {
			switch reviewGroup.Status {
			case enum.GroupTypeConfirmed:
				confirmedCount++
			case enum.GroupTypeFlagged:
				flaggedCount++
			} //exhaustive:ignore
		}
	}

	// Calculate confidence score
	confidence := c.calculateConfidence(confirmedCount, flaggedCount, len(userInfo.Groups.Data))

	// Flag user if confidence exceeds threshold
	if confidence >= 0.4 {
		c.logger.Info("User automatically flagged",
			zap.Uint64("userID", userInfo.ID),
			zap.Int("confirmedGroups", confirmedCount),
			zap.Int("flaggedGroups", flaggedCount),
			zap.Float64("confidence", confidence))

		return &types.User{
			ID:             userInfo.ID,
			Name:           userInfo.Name,
			DisplayName:    userInfo.DisplayName,
			Description:    userInfo.Description,
			CreatedAt:      userInfo.CreatedAt,
			Reason:         "Group Analysis: Member of multiple inappropriate groups.",
			Groups:         userInfo.Groups.Data,
			Friends:        userInfo.Friends.Data,
			Games:          userInfo.Games.Data,
			FollowerCount:  userInfo.FollowerCount,
			FollowingCount: userInfo.FollowingCount,
			Confidence:     math.Round(confidence*100) / 100, // Round to 2 decimal places
			LastUpdated:    userInfo.LastUpdated,
			LastPurgeCheck: userInfo.LastPurgeCheck,
		}, true
	}

	return nil, false
}

// calculateConfidence computes a weighted confidence score based on group memberships.
func (c *GroupChecker) calculateConfidence(confirmedCount, flaggedCount, totalGroups int) float64 {
	var confidence float64

	// Factor 1: Absolute number of inappropriate groups - 60% weight
	inappropriateWeight := c.calculateInappropriateWeight(confirmedCount, flaggedCount)
	confidence += inappropriateWeight * 0.60

	// Factor 2: Ratio of inappropriate groups - 40% weight
	if totalGroups > 0 {
		totalInappropriate := float64(confirmedCount) + (float64(flaggedCount) * 0.5)
		ratioWeight := math.Min(totalInappropriate/float64(totalGroups), 1.0)
		confidence += ratioWeight * 0.40
	}

	return confidence
}

// calculateInappropriateWeight returns a weight based on the total number of inappropriate groups.
func (c *GroupChecker) calculateInappropriateWeight(confirmedCount, flaggedCount int) float64 {
	totalWeight := float64(confirmedCount) + (float64(flaggedCount) * 0.5)

	switch {
	case confirmedCount >= 4 || totalWeight >= 6:
		return 1.0
	case confirmedCount >= 3 || totalWeight >= 4:
		return 0.8
	case confirmedCount >= 2 || totalWeight >= 3:
		return 0.6
	case confirmedCount >= 1 || totalWeight >= 1:
		return 0.4
	default:
		return 0.0
	}
}
