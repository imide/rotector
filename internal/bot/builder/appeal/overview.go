package appeal

import (
	"fmt"
	"strconv"

	"github.com/disgoorg/disgo/discord"
	"github.com/robalyx/rotector/internal/bot/constants"
	"github.com/robalyx/rotector/internal/bot/core/session"
	"github.com/robalyx/rotector/internal/bot/utils"
	"github.com/robalyx/rotector/internal/common/storage/database/types"
	"github.com/robalyx/rotector/internal/common/storage/database/types/enum"
)

// OverviewBuilder creates the visual layout for the appeal overview interface.
type OverviewBuilder struct {
	appeals      []*types.FullAppeal
	sortBy       enum.AppealSortBy
	statusFilter enum.AppealStatus
	hasNextPage  bool
	hasPrevPage  bool
	isReviewer   bool
	streamerMode bool
}

// NewOverviewBuilder creates a new overview builder.
func NewOverviewBuilder(s *session.Session) *OverviewBuilder {
	return &OverviewBuilder{
		appeals:      session.AppealList.Get(s),
		sortBy:       session.UserAppealDefaultSort.Get(s),
		statusFilter: session.UserAppealStatusFilter.Get(s),
		hasNextPage:  session.PaginationHasNextPage.Get(s),
		hasPrevPage:  session.PaginationHasPrevPage.Get(s),
		isReviewer:   s.BotSettings().IsReviewer(session.UserID.Get(s)),
		streamerMode: session.UserStreamerMode.Get(s),
	}
}

// Build creates a Discord message showing the appeals list and controls.
func (b *OverviewBuilder) Build() *discord.MessageUpdateBuilder {
	embed := b.buildEmbed()
	components := b.buildComponents()

	return discord.NewMessageUpdateBuilder().
		SetEmbeds(embed.Build()).
		AddContainerComponents(components...)
}

// buildEmbed creates the main embed showing appeal information.
func (b *OverviewBuilder) buildEmbed() *discord.EmbedBuilder {
	embed := discord.NewEmbedBuilder().
		SetTitle("Appeal Tickets").
		SetColor(utils.GetMessageEmbedColor(b.streamerMode))

	if len(b.appeals) == 0 {
		embed.SetDescription("No appeals found.")
		return embed
	}

	// Add appeal entries
	for _, appeal := range b.appeals {
		fieldName, fieldValue := b.formatAppealField(appeal)
		embed.AddField(fieldName, fieldValue, false)
	}

	// Add sequence count to footer
	if len(b.appeals) > 0 {
		firstAppeal := b.appeals[0]
		lastAppeal := b.appeals[len(b.appeals)-1]
		embed.SetFooter(fmt.Sprintf("Sequence %d-%d | %d appeals shown",
			firstAppeal.ID,
			lastAppeal.ID,
			len(b.appeals)),
			"")
	}

	return embed
}

// formatAppealField formats a single appeal entry for the embed.
func (b *OverviewBuilder) formatAppealField(appeal *types.FullAppeal) (string, string) {
	// Format status with emoji
	var statusEmoji string
	switch appeal.Status {
	case enum.AppealStatusPending:
		statusEmoji = "⏳"
	case enum.AppealStatusAccepted:
		statusEmoji = "✅"
	case enum.AppealStatusRejected:
		statusEmoji = "❌"
	}

	// Format claimed status
	claimedInfo := ""
	if appeal.ClaimedBy != 0 {
		claimedInfo = fmt.Sprintf("\nClaimed by: <@%d>", appeal.ClaimedBy)
	}

	// Format timestamps
	submitted := "N/A"
	if !appeal.Timestamp.IsZero() {
		submitted = fmt.Sprintf("<t:%d:R>", appeal.Timestamp.Unix())
	}

	lastViewed := "N/A"
	if !appeal.LastViewed.IsZero() {
		lastViewed = fmt.Sprintf("<t:%d:R>", appeal.LastViewed.Unix())
	}

	lastActivity := "N/A"
	if !appeal.LastActivity.IsZero() {
		lastActivity = fmt.Sprintf("<t:%d:R>", appeal.LastActivity.Unix())
	}

	fieldName := fmt.Sprintf("%s Appeal `#%d`", statusEmoji, appeal.ID)
	fieldValue := fmt.Sprintf(
		"User: [%s](https://www.roblox.com/users/%d/profile)\n"+
			"Requester: <@%d>%s\n"+
			"Submitted: %s\n"+
			"Last Viewed: %s\n"+
			"Last Activity: %s",
		utils.CensorString(strconv.FormatUint(appeal.UserID, 10), b.streamerMode),
		appeal.UserID,
		appeal.RequesterID,
		claimedInfo,
		submitted,
		lastViewed,
		lastActivity,
	)

	return fieldName, fieldValue
}

// buildComponents creates all the interactive components.
func (b *OverviewBuilder) buildComponents() []discord.ContainerComponent {
	var components []discord.ContainerComponent

	// Add appeal selector
	if len(b.appeals) > 0 {
		options := make([]discord.StringSelectMenuOption, 0, len(b.appeals))
		for _, appeal := range b.appeals {
			// Format status emoji
			var statusEmoji string
			switch appeal.Status {
			case enum.AppealStatusPending:
				statusEmoji = "⏳"
			case enum.AppealStatusAccepted:
				statusEmoji = "✅"
			case enum.AppealStatusRejected:
				statusEmoji = "❌"
			}

			// Create option for each appeal
			option := discord.NewStringSelectMenuOption(
				fmt.Sprintf("%s Appeal #%d", statusEmoji, appeal.ID),
				strconv.FormatInt(appeal.ID, 10),
			).WithDescription(
				"View appeal for User ID: " +
					utils.CensorString(strconv.FormatUint(appeal.UserID, 10), b.streamerMode),
			)

			options = append(options, option)
		}

		components = append(components, discord.NewActionRow(
			discord.NewStringSelectMenu(constants.AppealSelectID, "Select Appeal", options...),
		))
	}

	// Add status filter dropdown
	components = append(components, discord.NewActionRow(
		discord.NewStringSelectMenu(constants.AppealStatusSelectID, "Filter by Status",
			discord.NewStringSelectMenuOption("Pending Appeals", enum.AppealStatusPending.String()).
				WithDescription("Show only pending appeals").
				WithDefault(b.statusFilter == enum.AppealStatusPending),
			discord.NewStringSelectMenuOption("Accepted Appeals", enum.AppealStatusAccepted.String()).
				WithDescription("Show only accepted appeals").
				WithDefault(b.statusFilter == enum.AppealStatusAccepted),
			discord.NewStringSelectMenuOption("Rejected Appeals", enum.AppealStatusRejected.String()).
				WithDescription("Show only rejected appeals").
				WithDefault(b.statusFilter == enum.AppealStatusRejected)),
	))

	if b.isReviewer {
		// Add sorting options for reviewers
		components = append(components, discord.NewActionRow(
			discord.NewStringSelectMenu(constants.AppealSortSelectID, "Sort by",
				discord.NewStringSelectMenuOption("Oldest First", enum.AppealSortByOldest.String()).
					WithDescription("Show oldest appeals first").
					WithDefault(b.sortBy == enum.AppealSortByOldest),
				discord.NewStringSelectMenuOption("My Claims", enum.AppealSortByClaimed.String()).
					WithDescription("Show appeals claimed by you").
					WithDefault(b.sortBy == enum.AppealSortByClaimed),
				discord.NewStringSelectMenuOption("Newest First", enum.AppealSortByNewest.String()).
					WithDescription("Show newest appeals first").
					WithDefault(b.sortBy == enum.AppealSortByNewest),
			),
		))
	}

	// Add action buttons row
	var actionButtons []discord.InteractiveComponent

	// Add refresh button for everyone
	actionButtons = append(actionButtons,
		discord.NewSecondaryButton("🔄 Refresh", constants.RefreshButtonCustomID))

	// Add new appeal button only for non-reviewers
	if !b.isReviewer {
		actionButtons = append(actionButtons,
			discord.NewPrimaryButton("New Appeal", constants.AppealCreateButtonCustomID))
	}

	components = append(components, discord.NewActionRow(actionButtons...))

	// Add navigation buttons
	components = append(components, discord.NewActionRow(
		discord.NewSecondaryButton("◀️", constants.BackButtonCustomID),
		discord.NewSecondaryButton("⏮️", string(session.ViewerFirstPage)).
			WithDisabled(!b.hasPrevPage),
		discord.NewSecondaryButton("◀️", string(session.ViewerPrevPage)).
			WithDisabled(!b.hasPrevPage),
		discord.NewSecondaryButton("▶️", string(session.ViewerNextPage)).
			WithDisabled(!b.hasNextPage),
		discord.NewSecondaryButton("⏭️", string(session.ViewerLastPage)).
			WithDisabled(true),
	))

	return components
}
