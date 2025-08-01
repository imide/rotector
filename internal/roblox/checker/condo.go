package checker

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"sync"

	"github.com/robalyx/rotector/internal/database"
	"github.com/robalyx/rotector/internal/database/types"
	"github.com/robalyx/rotector/internal/database/types/enum"
	"github.com/sourcegraph/conc/pool"
	"go.uber.org/zap"
)

// CondoCheckerParams contains all the parameters needed for condo checker processing.
type CondoCheckerParams struct {
	Users      []*types.ReviewUser                           `json:"users"`
	ReasonsMap map[uint64]types.Reasons[enum.UserReasonType] `json:"reasonsMap"`
}

var (
	ErrPlayerNotFound    = errors.New("player not found")
	ErrPlayerBlacklisted = errors.New("player is blacklisted")
)

// CondoChecker handles the checking of user thumbnails against known condo players.
type CondoChecker struct {
	db     database.Client
	logger *zap.Logger
}

// NewCondoChecker creates a CondoChecker with database access.
func NewCondoChecker(db database.Client, logger *zap.Logger) *CondoChecker {
	return &CondoChecker{
		db:     db,
		logger: logger.Named("condo_checker"),
	}
}

// ProcessUsers checks multiple users' thumbnails concurrently and updates reasonsMap.
func (c *CondoChecker) ProcessUsers(ctx context.Context, params *CondoCheckerParams) {
	existingFlags := len(params.ReasonsMap)

	var (
		p  = pool.New().WithContext(ctx)
		mu sync.Mutex
	)

	// Process each user concurrently
	for _, userInfo := range params.Users {
		p.Go(func(_ context.Context) error {
			// Process user
			reason, err := c.processUser(ctx, userInfo)
			if err != nil {
				if errors.Is(err, ErrPlayerNotFound) || errors.Is(err, ErrPlayerBlacklisted) {
					return nil
				}

				return err
			}

			// Add reason to reasons map
			mu.Lock()

			if _, exists := params.ReasonsMap[userInfo.ID]; !exists {
				params.ReasonsMap[userInfo.ID] = make(types.Reasons[enum.UserReasonType])
			}

			params.ReasonsMap[userInfo.ID].Add(enum.UserReasonTypeCondo, reason)
			mu.Unlock()

			return nil
		})
	}

	// Wait for all goroutines to complete
	if err := p.Wait(); err != nil {
		c.logger.Error("Error during condo processing", zap.Error(err))
	}

	c.logger.Info("Finished processing condo checks",
		zap.Int("totalUsers", len(params.Users)),
		zap.Int("newFlags", len(params.ReasonsMap)-existingFlags))
}

// processUser handles the logic for checking a single user against condo players.
// Returns a reason if the user should be flagged, nil otherwise.
func (c *CondoChecker) processUser(ctx context.Context, user *types.ReviewUser) (*types.Reason, error) {
	// Check if user's thumbnail matches a condo player
	player, err := c.db.Model().Condo().GetPlayerByThumbnail(ctx, user.ThumbnailURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPlayerNotFound
		}

		c.logger.Error("Failed to get player by thumbnail",
			zap.Error(err),
			zap.String("thumbnailURL", user.ThumbnailURL))

		return nil, err
	}

	// Skip if player is blacklisted
	if player.IsBlacklisted {
		return nil, ErrPlayerBlacklisted
	}

	// If player already has a different user ID, blacklist them and delete the existing user
	if player.UserID != nil && *player.UserID != user.ID {
		// Update player to be blacklisted
		if err := c.db.Model().Condo().BlacklistPlayer(ctx, player.ThumbnailURL); err != nil {
			c.logger.Error("Failed to blacklist player",
				zap.Error(err),
				zap.String("thumbnailURL", player.ThumbnailURL))

			return nil, err
		}

		// Delete the existing user if they were only flagged for condo
		existingUser, err := c.db.Service().User().GetUserByID(
			ctx, strconv.FormatUint(*player.UserID, 10), types.UserFieldReasons,
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrPlayerBlacklisted
			}

			c.logger.Error("Failed to get existing user",
				zap.Error(err),
				zap.Uint64("userID", *player.UserID))

			return nil, err
		}

		if existingUser.Status == enum.UserTypeFlagged && len(existingUser.Reasons) == 1 {
			if _, ok := existingUser.Reasons[enum.UserReasonTypeCondo]; ok {
				if _, err := c.db.Service().User().DeleteUser(ctx, *player.UserID); err != nil {
					c.logger.Error("Failed to delete existing user",
						zap.Error(err),
						zap.Uint64("userID", *player.UserID))

					return nil, err
				}
			}
		}

		c.logger.Info("Player blacklisted",
			zap.String("thumbnailURL", player.ThumbnailURL),
			zap.Uint64("userID", *player.UserID))

		return nil, ErrPlayerBlacklisted
	}

	// Update player with user ID if not set
	if player.UserID == nil {
		if err := c.db.Model().Condo().SetPlayerUserID(ctx, player.ThumbnailURL, user.ID); err != nil {
			c.logger.Error("Failed to update condo player user ID",
				zap.Error(err),
				zap.String("thumbnailURL", player.ThumbnailURL),
				zap.Uint64("userID", user.ID))

			return nil, err
		}
	}

	// Return reason for flagging
	return &types.Reason{
		Message:    "User found in condo games.",
		Confidence: 1.0,
	}, nil
}
