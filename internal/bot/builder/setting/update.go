package setting

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/robalyx/rotector/internal/bot/constants"
	"github.com/robalyx/rotector/internal/bot/core/session"
	"github.com/robalyx/rotector/internal/common/storage/database/types/enum"
)

// UpdateBuilder creates a generic settings change menu.
type UpdateBuilder struct {
	session      *session.Session
	setting      *session.Setting
	settingName  string
	settingType  string
	currentValue string
	customID     string
	page         int
	totalItems   int
	totalPages   int
}

// NewUpdateBuilder creates a new update builder.
func NewUpdateBuilder(s *session.Session) *UpdateBuilder {
	return &UpdateBuilder{
		session:      s,
		setting:      session.SettingValue.Get(s),
		settingName:  session.SettingName.Get(s),
		settingType:  session.SettingType.Get(s),
		currentValue: session.SettingDisplay.Get(s),
		customID:     session.SettingCustomID.Get(s),
		page:         session.PaginationPage.Get(s),
		totalItems:   session.PaginationTotalItems.Get(s),
		totalPages:   session.PaginationTotalPages.Get(s),
	}
}

// Build creates a Discord message showing the current setting value and
// providing appropriate input controls based on the setting type.
func (b *UpdateBuilder) Build() *discord.MessageUpdateBuilder {
	embed := discord.NewEmbedBuilder().
		SetTitle("Change " + b.setting.Name).
		SetDescription(b.setting.Description).
		SetColor(constants.DefaultEmbedColor)

	// Add fields based on setting type
	switch b.setting.Type {
	case enum.SettingTypeID:
		b.addIDFields(embed)
	case enum.SettingTypeAPIKey:
		b.addAPIKeyFields(embed)
	case enum.SettingTypeBool, enum.SettingTypeEnum, enum.SettingTypeNumber, enum.SettingTypeText:
		embed.AddField("Current Value", b.currentValue, false)
	}

	components := b.buildComponents()

	return discord.NewMessageUpdateBuilder().
		SetEmbeds(embed.Build()).
		AddContainerComponents(components...)
}

// addIDFields adds the ID fields to the embed.
func (b *UpdateBuilder) addIDFields(embed *discord.EmbedBuilder) {
	// Get the appropriate ID list based on setting key
	var ids []uint64
	switch b.setting.Key {
	case constants.ReviewerIDsOption:
		ids = session.BotReviewerIDs.Get(b.session)
	case constants.AdminIDsOption:
		ids = session.BotAdminIDs.Get(b.session)
	default:
		embed.AddField("Error", "Unknown ID setting type", false)
		return
	}

	if len(ids) == 0 {
		embed.AddField("No IDs Set", "Use the button below to add IDs", false)
		return
	}

	// Use stored pagination state
	start := b.page * constants.SettingsIDsPerPage
	end := start + constants.SettingsIDsPerPage
	if end > len(ids) {
		end = len(ids)
	}

	// Add fields for this page
	for _, id := range ids[start:end] {
		embed.AddField(
			fmt.Sprintf("ID: %d", id),
			fmt.Sprintf("<@%d>", id),
			false,
		)
	}
}

// addAPIKeyFields adds the API key fields to the embed.
func (b *UpdateBuilder) addAPIKeyFields(embed *discord.EmbedBuilder) {
	apiKeys := session.BotAPIKeys.Get(b.session)
	if len(apiKeys) == 0 {
		embed.AddField("No API Keys", "Use the button below to add keys", false)
		return
	}

	// Calculate pagination
	start := b.page * constants.SettingsKeysPerPage
	end := start + constants.SettingsKeysPerPage
	if end > len(apiKeys) {
		end = len(apiKeys)
	}

	// Add fields for this page
	for _, key := range apiKeys[start:end] {
		maskedKey := fmt.Sprintf("||%s||", key.Key)
		description := fmt.Sprintf("Created: %s\nNotes: %s",
			key.CreatedAt.Format("2006-01-02"),
			key.Description,
		)
		embed.AddField(maskedKey, description, false)
	}
}

// buildComponents creates the interactive components based on setting type.
func (b *UpdateBuilder) buildComponents() []discord.ContainerComponent {
	var components []discord.ContainerComponent

	// Add type-specific components
	switch b.setting.Type {
	case enum.SettingTypeBool:
		components = append(components, b.buildBooleanComponents())

	case enum.SettingTypeEnum:
		components = append(components, b.buildEnumComponents())

	case enum.SettingTypeID, enum.SettingTypeNumber, enum.SettingTypeText:
		components = append(components, b.buildModalComponents()...)

	case enum.SettingTypeAPIKey:
		components = append(components, b.buildAPIKeyComponents()...)
	}

	// Add back button
	components = append(components, discord.NewActionRow(
		discord.NewSecondaryButton("Back", fmt.Sprintf("%s_%s", b.settingType, constants.BackButtonCustomID)),
	))

	return components
}

// buildAPIKeyComponents creates the components for API key settings.
func (b *UpdateBuilder) buildAPIKeyComponents() []discord.ContainerComponent {
	var components []discord.ContainerComponent

	// Add API key action select menu
	options := make([]discord.StringSelectMenuOption, 0, len(b.setting.Options))
	for _, opt := range b.setting.Options {
		option := discord.NewStringSelectMenuOption(opt.Label, opt.Value).
			WithDescription(opt.Description)
		if opt.Emoji != "" {
			option = option.WithEmoji(discord.ComponentEmoji{Name: opt.Emoji})
		}
		options = append(options, option)
	}
	components = append(components, discord.NewActionRow(
		discord.NewStringSelectMenu(b.customID, "Select action", options...),
	))

	// Add pagination buttons
	components = append(components, b.buildPaginationButtons())

	return components
}

// buildPaginationButtons creates the standard pagination buttons.
func (b *UpdateBuilder) buildPaginationButtons() discord.ContainerComponent {
	return discord.NewActionRow(
		discord.NewSecondaryButton("⏮️", string(session.ViewerFirstPage)).WithDisabled(b.page == 0),
		discord.NewSecondaryButton("◀️", string(session.ViewerPrevPage)).WithDisabled(b.page == 0),
		discord.NewSecondaryButton("▶️", string(session.ViewerNextPage)).WithDisabled(b.page >= b.totalPages-1),
		discord.NewSecondaryButton("⏭️", string(session.ViewerLastPage)).WithDisabled(b.page >= b.totalPages-1),
	)
}

// buildBooleanComponents creates the components for boolean settings.
func (b *UpdateBuilder) buildBooleanComponents() discord.ContainerComponent {
	return discord.NewActionRow(
		discord.NewStringSelectMenu(b.customID, "Select new value",
			discord.NewStringSelectMenuOption("Enable", "true"),
			discord.NewStringSelectMenuOption("Disable", "false"),
		),
	)
}

// buildEnumComponents creates the components for enum settings.
func (b *UpdateBuilder) buildEnumComponents() discord.ContainerComponent {
	options := make([]discord.StringSelectMenuOption, 0, len(b.setting.Options))
	for _, opt := range b.setting.Options {
		option := discord.NewStringSelectMenuOption(opt.Label, opt.Value).
			WithDescription(opt.Description)
		if opt.Emoji != "" {
			option = option.WithEmoji(discord.ComponentEmoji{Name: opt.Emoji})
		}
		options = append(options, option)
	}

	return discord.NewActionRow(
		discord.NewStringSelectMenu(b.customID, "Select new value", options...),
	)
}

// buildModalComponents creates the components for modal settings.
func (b *UpdateBuilder) buildModalComponents() []discord.ContainerComponent {
	var components []discord.ContainerComponent

	// Add modal button
	var buttonText string
	switch b.setting.Type {
	case enum.SettingTypeID:
		buttonText = "Add/Remove ID"
	case enum.SettingTypeNumber:
		buttonText = "Set Value"
	case enum.SettingTypeText:
		buttonText = "Set Description"
	} //exhaustive:ignore

	components = append(components, discord.NewActionRow(
		discord.NewPrimaryButton(buttonText, b.customID+constants.ModalOpenSuffix),
	))

	// Add pagination buttons for ID type settings
	if b.setting.Type == enum.SettingTypeID {
		components = append(components, b.buildPaginationButtons())
	}

	return components
}
