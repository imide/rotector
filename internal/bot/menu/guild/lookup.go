package guild

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	builder "github.com/robalyx/rotector/internal/bot/builder/guild"
	"github.com/robalyx/rotector/internal/bot/constants"
	"github.com/robalyx/rotector/internal/bot/core/pagination"
	"github.com/robalyx/rotector/internal/bot/core/session"
	"github.com/robalyx/rotector/internal/bot/interfaces"
	"github.com/robalyx/rotector/internal/common/storage/database/types"
	"go.uber.org/zap"
)

// LookupMenu handles the display of Discord user information and their flagged servers.
type LookupMenu struct {
	layout *Layout
	page   *pagination.Page
}

// NewLookupMenu creates a new Discord user lookup menu.
func NewLookupMenu(layout *Layout) *LookupMenu {
	m := &LookupMenu{layout: layout}
	m.page = &pagination.Page{
		Name: constants.GuildLookupPageName,
		Message: func(s *session.Session) *discord.MessageUpdateBuilder {
			return builder.NewLookupBuilder(s).Build()
		},
		ShowHandlerFunc:   m.Show,
		ResetHandlerFunc:  m.Reset,
		ButtonHandlerFunc: m.handleButton,
		SelectHandlerFunc: m.handleSelectMenu,
	}
	return m
}

// Show prepares and displays the Discord user information interface.
func (m *LookupMenu) Show(event interfaces.CommonEvent, s *session.Session, r *pagination.Respond) {
	// Get Discord user ID from session
	discordUserID := session.DiscordUserLookupID.Get(s)

	// Check if user has requested data deletion
	isRedacted, err := m.layout.db.Models().Sync().IsUserDataRedacted(context.Background(), discordUserID)
	if err != nil {
		m.layout.logger.Error("Failed to check data redaction status",
			zap.Error(err),
			zap.Uint64("discord_user_id", discordUserID))
		isRedacted = false // Default to false if there's an error
	}
	session.DiscordUserDataRedacted.Set(s, isRedacted)

	// Attempt to get Discord username if possible
	var username string
	if user, err := event.Client().Rest().GetUser(snowflake.ID(discordUserID)); err == nil {
		username = user.Username
		session.DiscordUserLookupName.Set(s, username)
	}

	// Get total guild count
	totalGuilds, err := m.layout.db.Models().Sync().GetDiscordUserGuildCount(
		context.Background(),
		discordUserID,
	)
	if err != nil {
		m.layout.logger.Error("Failed to get Discord user guild count",
			zap.Error(err),
			zap.Uint64("discord_user_id", discordUserID))
		totalGuilds = 0 // Default to 0 if there's an error
	}
	session.DiscordUserTotalGuilds.Set(s, totalGuilds)

	// Get guilds where the user has inappropriate messages
	messageGuildIDs, err := m.layout.db.Models().Message().GetUserMessageGuilds(
		context.Background(),
		discordUserID,
	)
	if err != nil {
		m.layout.logger.Error("Failed to get user message guilds",
			zap.Error(err),
			zap.Uint64("discord_user_id", discordUserID))
		messageGuildIDs = []uint64{} // Default to empty if there's an error
	}

	// Convert slice to map for O(1) lookups
	messageGuilds := make(map[uint64]struct{})
	for _, guildID := range messageGuildIDs {
		messageGuilds[guildID] = struct{}{}
	}
	session.DiscordUserMessageGuilds.Set(s, messageGuilds)

	// Get cursor from session if it exists
	cursor := session.GuildLookupCursor.Get(s)

	// Fetch the user's guild memberships from database
	userGuilds, nextCursor, err := m.layout.db.Models().Sync().GetDiscordUserGuildsByCursor(
		context.Background(),
		discordUserID,
		cursor,
		constants.GuildMembershipsPerPage,
	)
	if err != nil {
		m.layout.logger.Error("Failed to get Discord user guilds",
			zap.Error(err),
			zap.Uint64("discord_user_id", discordUserID))
		r.Error(event, "Failed to retrieve guild membership data. Please try again.")
		return
	}

	// If we found guilds, get guild names and message summaries
	guildIDs := make([]uint64, len(userGuilds))
	for i, guild := range userGuilds {
		guildIDs[i] = guild.ServerID
	}

	guildNames := make(map[uint64]string)
	var messageSummary *types.InappropriateUserSummary

	if len(guildIDs) > 0 {
		// Get guild names
		guildInfos, err := m.layout.db.Models().Sync().GetServerInfo(
			context.Background(),
			guildIDs,
		)
		if err != nil {
			m.layout.logger.Error("Failed to get guild names",
				zap.Error(err),
				zap.Uint64s("guild_ids", guildIDs))
		} else {
			for _, info := range guildInfos {
				guildNames[info.ServerID] = info.Name
			}
		}

		// Only get message summary if data isn't redacted
		if !isRedacted {
			messageSummary, err = m.layout.db.Models().Message().GetUserInappropriateMessageSummary(
				context.Background(),
				discordUserID,
			)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				m.layout.logger.Error("Failed to get message summary",
					zap.Error(err),
					zap.Uint64("discord_user_id", discordUserID))
			}
		}
	}

	// Get previous cursors array
	prevCursors := session.GuildLookupPrevCursors.Get(s)

	// Store results in session
	session.DiscordUserGuilds.Set(s, userGuilds)
	session.DiscordUserGuildNames.Set(s, guildNames)
	session.DiscordUserMessageSummary.Set(s, messageSummary)
	session.GuildLookupCursor.Set(s, cursor)
	session.GuildLookupNextCursor.Set(s, nextCursor)
	session.PaginationHasNextPage.Set(s, nextCursor != nil)
	session.PaginationHasPrevPage.Set(s, len(prevCursors) > 0)
}

// Reset resets the lookup menu.
func (m *LookupMenu) Reset(s *session.Session) {
	session.GuildLookupCursor.Delete(s)
	session.GuildLookupNextCursor.Delete(s)
	session.GuildLookupPrevCursors.Delete(s)
	session.PaginationHasNextPage.Delete(s)
	session.PaginationHasPrevPage.Delete(s)
}

// handleButton processes button interactions.
func (m *LookupMenu) handleButton(
	event *events.ComponentInteractionCreate, s *session.Session, r *pagination.Respond, customID string,
) {
	switch customID {
	case constants.BackButtonCustomID:
		r.NavigateBack(event, s, "")
	case constants.RefreshButtonCustomID:
		m.Reset(s)
		r.Reload(event, s, "")
	case string(session.ViewerFirstPage),
		string(session.ViewerPrevPage),
		string(session.ViewerNextPage),
		string(session.ViewerLastPage):
		m.handlePagination(event, s, r, session.ViewerAction(customID))
	}
}

// handleSelectMenu processes select menu interactions.
func (m *LookupMenu) handleSelectMenu(
	event *events.ComponentInteractionCreate, s *session.Session, r *pagination.Respond, customID, option string,
) {
	if customID != constants.GuildMessageSelectMenuCustomID {
		return
	}

	// Parse guild ID from option value
	guildID, err := strconv.ParseUint(option, 10, 64)
	if err != nil {
		r.Error(event, "Failed to parse guild ID.")
		return
	}

	// Store selected guild ID
	session.DiscordUserMessageGuildID.Set(s, guildID)
	r.Show(event, s, constants.GuildMessagesPageName, "")
}

// handlePagination processes page navigation for guild memberships.
func (m *LookupMenu) handlePagination(
	event *events.ComponentInteractionCreate, s *session.Session, r *pagination.Respond, action session.ViewerAction,
) {
	switch action {
	case session.ViewerNextPage:
		if session.PaginationHasNextPage.Get(s) {
			cursor := session.GuildLookupCursor.Get(s)
			nextCursor := session.GuildLookupNextCursor.Get(s)
			prevCursors := session.GuildLookupPrevCursors.Get(s)

			session.GuildLookupCursor.Set(s, nextCursor)
			session.GuildLookupPrevCursors.Set(s, append(prevCursors, cursor))
			r.Reload(event, s, "")
		}
	case session.ViewerPrevPage:
		prevCursors := session.GuildLookupPrevCursors.Get(s)

		if len(prevCursors) > 0 {
			lastIdx := len(prevCursors) - 1
			session.GuildLookupPrevCursors.Set(s, prevCursors[:lastIdx])
			session.GuildLookupCursor.Set(s, prevCursors[lastIdx])
			r.Reload(event, s, "")
		}
	case session.ViewerFirstPage:
		session.GuildLookupCursor.Set(s, nil)
		session.GuildLookupPrevCursors.Set(s, make([]*types.GuildCursor, 0))
		r.Reload(event, s, "")
	case session.ViewerLastPage:
		// Not currently supported
		return
	}
}
