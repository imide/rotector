package builders

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/rotector/rotector/assets"
	"github.com/rotector/rotector/internal/bot/handlers/reviewer/constants"
	"github.com/rotector/rotector/internal/common/database"
	"github.com/rotector/rotector/internal/common/translator"
)

var multipleNewlinesRegex = regexp.MustCompile(`\n{4,}`)

// ReviewEmbed builds the embed for the review message.
type ReviewEmbed struct {
	user           *database.PendingUser
	translator     *translator.Translator
	flaggedFriends map[uint64]string
	sortBy         string
}

// NewReviewEmbed creates a new ReviewEmbed.
func NewReviewEmbed(user *database.PendingUser, translator *translator.Translator, flaggedFriends map[uint64]string, sortBy string) *ReviewEmbed {
	return &ReviewEmbed{
		user:           user,
		translator:     translator,
		flaggedFriends: flaggedFriends,
		sortBy:         sortBy,
	}
}

// Build constructs and returns the discord.Embed.
func (b *ReviewEmbed) Build() *discord.MessageUpdateBuilder {
	embedBuilder := discord.NewEmbedBuilder().
		AddField("ID", fmt.Sprintf("[%d](https://www.roblox.com/users/%d/profile)", b.user.ID, b.user.ID), true).
		AddField("Name", b.user.Name, true).
		AddField("Display Name", b.user.DisplayName, true).
		AddField("Created At", fmt.Sprintf("<t:%d:R>", b.user.CreatedAt.Unix()), true).
		AddField("Confidence", fmt.Sprintf("%.2f", b.user.Confidence), true).
		AddField("Reason", b.user.Reason, false).
		AddField("Description", b.getDescription(), false).
		AddField("Groups", b.getGroups(), false).
		AddField(b.getFriendsField(), b.getFriends(), false).
		AddField("Outfits", b.getOutfits(), false).
		AddField(b.getFlaggedType(), b.getFlaggedContent(), false).
		AddField("Last Updated", fmt.Sprintf("<t:%d:R>", b.user.LastUpdated.Unix()), true).
		AddField("Last Reviewed", b.getLastReviewed(), true).
		SetColor(0x312D2B)

	components := []discord.ContainerComponent{
		discord.NewActionRow(
			discord.NewStringSelectMenu(constants.SortOrderSelectMenuCustomID, "Sorting",
				discord.NewStringSelectMenuOption("Selected by random", database.SortByRandom).
					WithDefault(b.sortBy == database.SortByRandom).
					WithEmoji(discord.ComponentEmoji{Name: "🔀"}),
				discord.NewStringSelectMenuOption("Selected by confidence", database.SortByConfidence).
					WithDefault(b.sortBy == database.SortByConfidence).
					WithEmoji(discord.ComponentEmoji{Name: "🔮"}),
				discord.NewStringSelectMenuOption("Selected by last updated time", database.SortByLastUpdated).
					WithDefault(b.sortBy == database.SortByLastUpdated).
					WithEmoji(discord.ComponentEmoji{Name: "📅"}),
			),
		),
		discord.NewActionRow(
			discord.NewStringSelectMenu(constants.ActionSelectMenuCustomID, "Actions",
				discord.NewStringSelectMenuOption("Ban with reason", constants.BanWithReasonButtonCustomID),
				discord.NewStringSelectMenuOption("Open outfit viewer", constants.OpenOutfitsMenuButtonCustomID),
				discord.NewStringSelectMenuOption("Open friends viewer", constants.OpenFriendsMenuButtonCustomID),
				discord.NewStringSelectMenuOption("Open group viewer", constants.OpenGroupViewerButtonCustomID),
			),
		),
		discord.NewActionRow(
			discord.NewSecondaryButton("◀️", constants.BackButtonCustomID),
			discord.NewDangerButton("Ban", constants.BanButtonCustomID),
			discord.NewSuccessButton("Clear", constants.ClearButtonCustomID),
			discord.NewSecondaryButton("Skip", constants.SkipButtonCustomID),
		),
	}

	// Set thumbnail URL or use placeholder image
	if b.user.ThumbnailURL != "" {
		embedBuilder.SetThumbnail(b.user.ThumbnailURL)
	} else {
		embedBuilder.SetThumbnail("attachment://content_deleted.png")
	}

	// Create the message update builder
	builder := discord.NewMessageUpdateBuilder().
		SetEmbeds(embedBuilder.Build()).
		AddContainerComponents(components...)

	// Add placeholder image if thumbnail URL is empty
	if b.user.ThumbnailURL == "" {
		placeholderImage, err := assets.Images.ReadFile("images/content_deleted.png")
		if err == nil {
			builder.SetFiles(discord.NewFile("content_deleted.png", "", bytes.NewReader(placeholderImage)))
		}
	}

	return builder
}

// getDescription returns the description field for the embed.
func (b *ReviewEmbed) getDescription() string {
	description := b.user.Description

	// Check if description is empty
	if description == "" {
		return constants.NotApplicable
	}

	// Trim leading and trailing whitespace
	description = strings.TrimSpace(description)
	// Replace multiple newlines with a single newline
	description = multipleNewlinesRegex.ReplaceAllString(description, "\n")
	// Remove all backticks
	description = strings.ReplaceAll(description, "`", "")
	// Enclose in markdown
	description = fmt.Sprintf("```\n%s\n```", description)

	// Translate the description
	translatedDescription, err := b.translator.Translate(context.Background(), description, "auto", "en")
	if err == nil && translatedDescription != description {
		description = fmt.Sprintf("%s\n%s", description, translatedDescription)
	}

	return description
}

// getGroups returns the groups field for the embed.
func (b *ReviewEmbed) getGroups() string {
	// Get the first 10 groups
	groups := []string{}
	for i, group := range b.user.Groups {
		if i >= 10 {
			groups = append(groups, fmt.Sprintf("... and %d more", len(b.user.Groups)-10))
			break
		}
		groups = append(groups, fmt.Sprintf("[%s](https://www.roblox.com/groups/%d)", group.Group.Name, group.Group.ID))
	}

	// If no groups are found, return NotApplicable
	if len(groups) == 0 {
		return constants.NotApplicable
	}

	return strings.Join(groups, ", ")
}

// getFriendsField returns the friends field name for the embed.
func (b *ReviewEmbed) getFriendsField() string {
	if len(b.flaggedFriends) > 0 {
		return "Friends ⚠️"
	}
	return "Friends"
}

// getFriends returns the friends field for the embed.
func (b *ReviewEmbed) getFriends() string {
	// Get the first 10 friends
	friends := []string{}
	for i, friend := range b.user.Friends {
		if i >= 10 {
			friends = append(friends, fmt.Sprintf("... and %d more", len(b.user.Friends)-10))
			break
		}
		friends = append(friends, fmt.Sprintf("[%s](https://www.roblox.com/users/%d/profile)", friend.Name, friend.ID))
	}

	// Add flagged or pending status if needed
	if len(b.flaggedFriends) > 0 {
		flaggedCount := 0
		pendingCount := 0
		for _, friend := range b.flaggedFriends {
			if friend == "flagged" {
				flaggedCount++
			} else if friend == "pending" {
				pendingCount++
			}
		}

		friends = append(friends, fmt.Sprintf(" (%d flagged, %d pending)", flaggedCount, pendingCount))
	}

	// If no friends are found, return NotApplicable
	if len(friends) == 0 {
		return constants.NotApplicable
	}

	return strings.Join(friends, ", ")
}

// getOutfits returns the outfits field for the embed.
func (b *ReviewEmbed) getOutfits() string {
	// Get the first 10 outfits
	outfits := []string{}
	for i, outfit := range b.user.Outfits {
		if i >= 10 {
			outfits = append(outfits, fmt.Sprintf("... and %d more", len(b.user.Outfits)-10))
			break
		}
		outfits = append(outfits, outfit.Name)
	}
	// If no outfits are found, return NotApplicable
	if len(outfits) == 0 {
		return constants.NotApplicable
	}

	return strings.Join(outfits, ", ")
}

// getFlaggedType returns the flagged type field for the embed.
func (b *ReviewEmbed) getFlaggedType() string {
	if len(b.user.FlaggedGroups) > 0 {
		return "Flagged Groups"
	}
	return "Flagged Content"
}

// getFlaggedContent returns the flagged content field for the embed.
func (b *ReviewEmbed) getFlaggedContent() string {
	flaggedGroups := b.user.FlaggedGroups
	if len(flaggedGroups) > 0 {
		var content strings.Builder
		for _, flaggedGroupID := range flaggedGroups {
			for _, group := range b.user.Groups {
				if group.Group.ID == flaggedGroupID {
					content.WriteString(fmt.Sprintf("- [%s](https://www.roblox.com/groups/%d) (%s)\n",
						group.Group.Name, group.Group.ID, group.Role.Name))
					break
				}
			}
		}
		return content.String()
	}

	flaggedContent := b.user.FlaggedContent
	if len(flaggedContent) > 0 {
		for i := range flaggedContent {
			// Remove all newlines and backticks
			flaggedContent[i] = strings.ReplaceAll(flaggedContent[i], "\n", " ")
			flaggedContent[i] = strings.ReplaceAll(flaggedContent[i], "`", "")
		}
		return fmt.Sprintf("- `%s`", strings.Join(flaggedContent, "`\n- `"))
	}

	return constants.NotApplicable
}

// getLastReviewed returns the last reviewed field for the embed.
func (b *ReviewEmbed) getLastReviewed() string {
	if b.user.LastReviewed.IsZero() {
		return "Never Reviewed"
	}
	return fmt.Sprintf("<t:%d:R>", b.user.LastReviewed.Unix())
}
