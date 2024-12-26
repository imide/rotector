package group

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	apiTypes "github.com/jaxron/roapi.go/pkg/api/types"
	"github.com/rotector/rotector/assets"
	"github.com/rotector/rotector/internal/bot/constants"
	"github.com/rotector/rotector/internal/bot/core/session"
	"github.com/rotector/rotector/internal/bot/utils"
	"github.com/rotector/rotector/internal/common/client/fetcher"
	"github.com/rotector/rotector/internal/common/storage/database"
	"github.com/rotector/rotector/internal/common/storage/database/types"
)

// ReviewBuilder creates the visual layout for reviewing a group.
type ReviewBuilder struct {
	db          *database.Client
	settings    *types.UserSetting
	botSettings *types.BotSetting
	userID      uint64
	group       *types.ReviewGroup
	groupInfo   *apiTypes.GroupResponse
	memberIDs   []uint64
}

// NewReviewBuilder creates a new review builder.
func NewReviewBuilder(s *session.Session, db *database.Client) *ReviewBuilder {
	var settings *types.UserSetting
	s.GetInterface(constants.SessionKeyUserSettings, &settings)
	var botSettings *types.BotSetting
	s.GetInterface(constants.SessionKeyBotSettings, &botSettings)
	var group *types.ReviewGroup
	s.GetInterface(constants.SessionKeyGroupTarget, &group)
	var groupInfo *apiTypes.GroupResponse
	s.GetInterface(constants.SessionKeyGroupInfo, &groupInfo)
	var memberIDs []uint64
	s.GetInterface(constants.SessionKeyGroupMemberIDs, &memberIDs)

	return &ReviewBuilder{
		db:          db,
		settings:    settings,
		botSettings: botSettings,
		userID:      s.GetUint64(constants.SessionKeyUserID),
		group:       group,
		groupInfo:   groupInfo,
		memberIDs:   memberIDs,
	}
}

// Build creates a Discord message with group information in an embed and adds
// interactive components for reviewing the group.
func (b *ReviewBuilder) Build() *discord.MessageUpdateBuilder {
	// Create embeds
	modeEmbed := b.buildModeEmbed()
	reviewEmbed := b.buildReviewEmbed()

	// Create components
	components := b.buildComponents()

	// Create builder and handle thumbnail
	builder := discord.NewMessageUpdateBuilder()
	if b.group.ThumbnailURL != "" && b.group.ThumbnailURL != fetcher.ThumbnailPlaceholder {
		reviewEmbed.SetThumbnail(b.group.ThumbnailURL)
	} else {
		// Load and attach placeholder image
		placeholderImage, err := assets.Images.Open("images/content_deleted.png")
		if err == nil {
			builder.SetFiles(discord.NewFile("content_deleted.png", "", placeholderImage))
			_ = placeholderImage.Close()
		}
		reviewEmbed.SetThumbnail("attachment://content_deleted.png")
	}

	return builder.
		SetEmbeds(modeEmbed.Build(), reviewEmbed.Build()).
		AddContainerComponents(components...)
}

// buildModeEmbed creates the review mode info embed.
func (b *ReviewBuilder) buildModeEmbed() *discord.EmbedBuilder {
	var mode string
	var description string

	// Format review mode
	switch b.settings.ReviewMode {
	case types.TrainingReviewMode:
		mode = "🎓 Training Mode"
		description += `
		**You are not an official reviewer.**
		You may help moderators by using upvotes/downvotes to indicate suspicious activity. Information is censored and external links are disabled.
		`
	case types.StandardReviewMode:
		mode = "⚠️ Standard Mode"
		description += `
		Your actions are recorded and affect the database. Please review carefully before taking action.
		`
	default:
		mode = "❌ Unknown Mode"
		description = "Error encountered. Please check your settings."
	}

	return discord.NewEmbedBuilder().
		SetTitle(mode).
		SetDescription(description).
		SetColor(utils.GetMessageEmbedColor(b.settings.StreamerMode))
}

// buildReviewEmbed creates the main review information embed.
func (b *ReviewBuilder) buildReviewEmbed() *discord.EmbedBuilder {
	embed := discord.NewEmbedBuilder().
		SetColor(utils.GetMessageEmbedColor(b.settings.StreamerMode))

	// Add status indicator based on group status
	var status string
	switch b.group.Status {
	case types.GroupTypeFlagged:
		status = "⏳ Flagged Group"
	case types.GroupTypeConfirmed:
		status = "⚠️ Confirmed Group"
	case types.GroupTypeCleared:
		status = "✅ Cleared Group"
	case types.GroupTypeLocked:
		status = "🔒 Locked Group"
	case types.GroupTypeUnflagged:
		status = "🔄 Unflagged Group"
	}

	header := fmt.Sprintf("%s • 👍 %d | 👎 %d", status, b.group.Upvotes, b.group.Downvotes)
	lastUpdated := fmt.Sprintf("<t:%d:R>", b.group.LastUpdated.Unix())
	confidence := fmt.Sprintf("%.2f", b.group.Confidence)
	memberCount := strconv.FormatUint(b.groupInfo.MemberCount, 10)
	flaggedMembers := strconv.Itoa(len(b.memberIDs))

	// Censor reason if needed
	reason := utils.CensorStringsInText(
		b.group.Reason,
		b.settings.StreamerMode,
		strconv.FormatUint(b.group.ID, 10),
		b.group.Name,
		strconv.FormatUint(b.group.Owner.UserID, 10),
	)

	if b.settings.ReviewMode == types.TrainingReviewMode {
		// Training mode - show limited information without links
		embed.SetAuthorName(header).
			AddField("ID", utils.CensorString(strconv.FormatUint(b.group.ID, 10), true), true).
			AddField("Name", utils.CensorString(b.group.Name, true), true).
			AddField("Owner", utils.CensorString(strconv.FormatUint(b.group.Owner.UserID, 10), true), true).
			AddField("Members", memberCount, true).
			AddField("Flagged Members", flaggedMembers, true).
			AddField("Confidence", confidence, true).
			AddField("Last Updated", lastUpdated, true).
			AddField("Reason", reason, false).
			AddField("Shout", b.getShout(), false).
			AddField("Description", b.getDescription(), false)
	} else {
		// Standard mode - show all information with links
		embed.SetAuthorName(header).
			AddField("ID", fmt.Sprintf(
				"[%s](https://www.roblox.com/groups/%d)",
				utils.CensorString(strconv.FormatUint(b.group.ID, 10), b.settings.StreamerMode),
				b.group.ID,
			), true).
			AddField("Name", utils.CensorString(b.group.Name, b.settings.StreamerMode), true).
			AddField("Owner", fmt.Sprintf(
				"[%s](https://www.roblox.com/users/%d/profile)",
				utils.CensorString(strconv.FormatUint(b.group.Owner.UserID, 10), b.settings.StreamerMode),
				b.group.Owner.UserID,
			), true).
			AddField("Members", memberCount, true).
			AddField("Flagged Members", flaggedMembers, true).
			AddField("Confidence", confidence, true).
			AddField("Last Updated", lastUpdated, true).
			AddField("Reason", reason, false).
			AddField("Shout", b.getShout(), false).
			AddField("Description", b.getDescription(), false).
			AddField("Review History", b.getReviewHistory(), false)
	}

	// Add status-specific timestamps
	if !b.group.VerifiedAt.IsZero() {
		embed.AddField("Verified At", fmt.Sprintf("<t:%d:R>", b.group.VerifiedAt.Unix()), true)
	}
	if !b.group.ClearedAt.IsZero() {
		embed.AddField("Cleared At", fmt.Sprintf("<t:%d:R>", b.group.ClearedAt.Unix()), true)
	}
	if !b.group.LockedAt.IsZero() {
		embed.AddField("Locked At", fmt.Sprintf("<t:%d:R>", b.group.LockedAt.Unix()), true)
	}

	return embed
}

// buildActionOptions creates the action menu options.
func (b *ReviewBuilder) buildActionOptions() []discord.StringSelectMenuOption {
	options := []discord.StringSelectMenuOption{
		discord.NewStringSelectMenuOption("View Flagged Members", constants.GroupViewMembersButtonCustomID).
			WithDescription("View all flagged members of this group").
			WithEmoji(discord.ComponentEmoji{Name: "👥"}),
		discord.NewStringSelectMenuOption("Change Review Target", constants.ReviewTargetModeOption).
			WithEmoji(discord.ComponentEmoji{Name: "🎯"}).
			WithDescription("Change what type of groups to review"),
	}

	// Add reviewer-only options
	if b.botSettings.IsReviewer(b.userID) {
		reviewerOptions := []discord.StringSelectMenuOption{
			discord.NewStringSelectMenuOption("Ask AI about group", constants.OpenAIChatButtonCustomID).
				WithEmoji(discord.ComponentEmoji{Name: "🤖"}).
				WithDescription("Ask the AI questions about this group"),
			discord.NewStringSelectMenuOption("Confirm with reason", constants.GroupConfirmWithReasonButtonCustomID).
				WithEmoji(discord.ComponentEmoji{Name: "🚫"}).
				WithDescription("Confirm the group with a custom reason"),
			discord.NewStringSelectMenuOption("View group logs", constants.GroupViewLogsButtonCustomID).
				WithEmoji(discord.ComponentEmoji{Name: "📋"}).
				WithDescription("View activity logs for this group"),
			discord.NewStringSelectMenuOption("Change Review Mode", constants.ReviewModeOption).
				WithEmoji(discord.ComponentEmoji{Name: "🎓"}).
				WithDescription("Switch between training and standard modes"),
		}
		options = append(options, reviewerOptions...)
	}

	return options
}

// buildComponents creates all interactive components for the review menu.
func (b *ReviewBuilder) buildComponents() []discord.ContainerComponent {
	return []discord.ContainerComponent{
		// Sorting options menu
		discord.NewActionRow(
			discord.NewStringSelectMenu(constants.SortOrderSelectMenuCustomID, "Sorting",
				discord.NewStringSelectMenuOption("Selected by random", string(types.ReviewSortByRandom)).
					WithDefault(b.settings.GroupDefaultSort == types.ReviewSortByRandom).
					WithEmoji(discord.ComponentEmoji{Name: "🔀"}),
				discord.NewStringSelectMenuOption("Selected by confidence", string(types.ReviewSortByConfidence)).
					WithDefault(b.settings.GroupDefaultSort == types.ReviewSortByConfidence).
					WithEmoji(discord.ComponentEmoji{Name: "🔍"}),
				discord.NewStringSelectMenuOption("Selected by last updated time", string(types.ReviewSortByLastUpdated)).
					WithDefault(b.settings.GroupDefaultSort == types.ReviewSortByLastUpdated).
					WithEmoji(discord.ComponentEmoji{Name: "📅"}),
				discord.NewStringSelectMenuOption("Selected by bad reputation", string(types.ReviewSortByReputation)).
					WithDefault(b.settings.GroupDefaultSort == types.ReviewSortByReputation).
					WithEmoji(discord.ComponentEmoji{Name: "👎"}),
			),
		),
		// Action options menu
		discord.NewActionRow(
			discord.NewStringSelectMenu(constants.ActionSelectMenuCustomID, "Actions", b.buildActionOptions()...),
		),
		// Quick action buttons
		discord.NewActionRow(
			discord.NewSecondaryButton("◀️", constants.BackButtonCustomID),
			discord.NewDangerButton(b.getConfirmButtonLabel(), constants.GroupConfirmButtonCustomID),
			discord.NewSuccessButton(b.getClearButtonLabel(), constants.GroupClearButtonCustomID),
			discord.NewSecondaryButton("Skip", constants.GroupSkipButtonCustomID),
		),
	}
}

// getDescription returns the description field for the embed.
func (b *ReviewBuilder) getDescription() string {
	description := b.group.Description

	// Check if description is empty
	if description == "" {
		return constants.NotApplicable
	}

	// Prepare description
	description = utils.TruncateString(description, 400)
	description = utils.FormatString(description)
	description = utils.CensorStringsInText(
		description,
		b.settings.StreamerMode,
		strconv.FormatUint(b.group.ID, 10),
		b.group.Name,
		strconv.FormatUint(b.group.Owner.UserID, 10),
	)

	return description
}

// getShout returns the shout field for the embed.
func (b *ReviewBuilder) getShout() string {
	// Skip if shout is not available
	if b.group.Shout == nil {
		return constants.NotApplicable
	}

	// Prepare shout
	shout := utils.TruncateString(b.group.Shout.Body, 400)
	shout = utils.FormatString(shout)

	return shout
}

// getReviewHistory returns the review history field for the embed.
func (b *ReviewBuilder) getReviewHistory() string {
	logs, nextCursor, err := b.db.UserActivity().GetLogs(
		context.Background(),
		types.ActivityFilter{
			GroupID:      b.group.ID,
			ReviewerID:   0,
			ActivityType: types.ActivityTypeAll,
			StartDate:    time.Time{},
			EndDate:      time.Time{},
		},
		nil,
		constants.ReviewHistoryLimit,
	)
	if err != nil {
		return "Failed to fetch review history"
	}

	if len(logs) == 0 {
		return constants.NotApplicable
	}

	history := make([]string, 0, len(logs))
	for _, log := range logs {
		history = append(history, fmt.Sprintf("- <@%d> (%s) - <t:%d:R>",
			log.ReviewerID, log.ActivityType.String(), log.ActivityTimestamp.Unix()))
	}

	if nextCursor != nil {
		history = append(history, "... and more")
	}

	return strings.Join(history, "\n")
}

// getConfirmButtonLabel returns the appropriate label for the confirm button based on review mode.
func (b *ReviewBuilder) getConfirmButtonLabel() string {
	if b.settings.ReviewMode == types.TrainingReviewMode {
		return "Downvote"
	}
	return "Confirm"
}

// getClearButtonLabel returns the appropriate label for the clear button based on review mode.
func (b *ReviewBuilder) getClearButtonLabel() string {
	if b.settings.ReviewMode == types.TrainingReviewMode {
		return "Upvote"
	}
	return "Clear"
}
