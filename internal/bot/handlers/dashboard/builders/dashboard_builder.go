package builders

import (
	"strconv"

	"github.com/disgoorg/disgo/discord"
	"github.com/rotector/rotector/internal/bot/constants"
)

// DashboardBuilder is the builder for the dashboard.
type DashboardBuilder struct {
	flaggedCount   int
	confirmedCount int
}

// NewDashboardBuilder creates a new DashboardBuilder.
func NewDashboardBuilder(flaggedCount, confirmedCount int) *DashboardBuilder {
	return &DashboardBuilder{
		flaggedCount:   flaggedCount,
		confirmedCount: confirmedCount,
	}
}

// Build builds the dashboard.
func (b *DashboardBuilder) Build() *discord.MessageUpdateBuilder {
	embed := discord.NewEmbedBuilder().
		AddField("Flagged Users", strconv.Itoa(b.flaggedCount), true).
		AddField("Confirmed Users", strconv.Itoa(b.confirmedCount), true).
		SetColor(constants.DefaultEmbedColor).
		Build()

	components := []discord.ContainerComponent{
		discord.NewActionRow(
			discord.NewStringSelectMenu(constants.ActionSelectMenuCustomID, "Select an action",
				discord.NewStringSelectMenuOption("Review Flagged Users", constants.StartReviewCustomID),
				discord.NewStringSelectMenuOption("User Settings", constants.UserSettingsCustomID),
				discord.NewStringSelectMenuOption("Guild Settings", constants.GuildSettingsCustomID),
			),
		),
	}

	return discord.NewMessageUpdateBuilder().
		SetEmbeds(embed).
		AddContainerComponents(components...)
}
