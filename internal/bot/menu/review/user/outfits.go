package user

import (
	"bytes"
	"context"
	"strconv"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/jaxron/roapi.go/pkg/api/resources/thumbnails"
	apiTypes "github.com/jaxron/roapi.go/pkg/api/types"
	builder "github.com/robalyx/rotector/internal/bot/builder/review/user"
	"github.com/robalyx/rotector/internal/bot/constants"
	"github.com/robalyx/rotector/internal/bot/core/pagination"
	"github.com/robalyx/rotector/internal/bot/core/session"
	"github.com/robalyx/rotector/internal/bot/interfaces"
	"go.uber.org/zap"
)

// OutfitsMenu handles the display and interaction logic for viewing a user's outfits.
type OutfitsMenu struct {
	layout *Layout
	page   *pagination.Page
}

// NewOutfitsMenu creates a new outfits menu.
func NewOutfitsMenu(layout *Layout) *OutfitsMenu {
	m := &OutfitsMenu{layout: layout}
	m.page = &pagination.Page{
		Name: constants.UserOutfitsPageName,
		Message: func(s *session.Session) *discord.MessageUpdateBuilder {
			return builder.NewOutfitsBuilder(s).Build()
		},
		ShowHandlerFunc:   m.Show,
		ButtonHandlerFunc: m.handlePageNavigation,
	}
	return m
}

// Show prepares and displays the outfits interface for a specific page.
func (m *OutfitsMenu) Show(event interfaces.CommonEvent, s *session.Session, r *pagination.Respond) {
	user := session.UserTarget.Get(s)

	// Return to review menu if user has no outfits
	if len(user.Outfits) == 0 {
		r.Cancel(event, s, "No outfits found for this user.")
		return
	}

	// Calculate page boundaries
	page := session.PaginationPage.Get(s)

	start := page * constants.OutfitsPerPage
	end := min(start+constants.OutfitsPerPage, len(user.Outfits))
	pageOutfits := user.Outfits[start:end]

	// Store data in session for the message builder
	session.UserOutfits.Set(s, pageOutfits)
	session.PaginationOffset.Set(s, start)
	session.PaginationPage.Set(s, page)
	session.PaginationTotalItems.Set(s, len(user.Outfits))

	// Start streaming images
	m.layout.imageStreamer.Stream(pagination.StreamRequest{
		Event:    event,
		Session:  s,
		Page:     m.page,
		URLFunc:  func() []string { return m.fetchOutfitThumbnails(pageOutfits) },
		Columns:  constants.OutfitGridColumns,
		Rows:     constants.OutfitGridRows,
		MaxItems: constants.OutfitsPerPage,
		OnSuccess: func(buf *bytes.Buffer) {
			session.ImageBuffer.Set(s, buf)
		},
	})
}

// handlePageNavigation processes navigation button clicks.
func (m *OutfitsMenu) handlePageNavigation(
	event *events.ComponentInteractionCreate, s *session.Session, r *pagination.Respond, customID string,
) {
	action := session.ViewerAction(customID)
	switch action {
	case session.ViewerFirstPage, session.ViewerPrevPage, session.ViewerNextPage, session.ViewerLastPage:
		user := session.UserTarget.Get(s)

		// Calculate max page and validate navigation action
		maxPage := (len(user.Outfits) - 1) / constants.OutfitsPerPage
		page := action.ParsePageAction(s, action, maxPage)

		session.PaginationPage.Set(s, page)
		r.Reload(event, s, "")
		return
	}

	switch customID {
	case constants.BackButtonCustomID:
		r.NavigateBack(event, s, "")
	default:
		m.layout.logger.Warn("Invalid outfits viewer action", zap.String("action", string(action)))
		r.Error(event, "Invalid interaction.")
	}
}

// fetchOutfitThumbnails gets the thumbnail URLs for a list of outfits.
func (m *OutfitsMenu) fetchOutfitThumbnails(outfits []*apiTypes.Outfit) []string {
	// Create batch request for outfit thumbnails
	requests := thumbnails.NewBatchThumbnailsBuilder()
	for _, outfit := range outfits {
		requests.AddRequest(apiTypes.ThumbnailRequest{
			Type:      apiTypes.OutfitType,
			TargetID:  outfit.ID,
			RequestID: strconv.FormatUint(outfit.ID, 10),
			Size:      apiTypes.Size150x150,
			Format:    apiTypes.WEBP,
		})
	}

	// Process thumbnails
	thumbnailMap := m.layout.thumbnailFetcher.ProcessBatchThumbnails(context.Background(), requests)

	// Convert map to ordered slice of URLs
	thumbnailURLs := make([]string, len(outfits))
	for i, outfit := range outfits {
		if url, ok := thumbnailMap[outfit.ID]; ok {
			thumbnailURLs[i] = url
		}
	}

	return thumbnailURLs
}
