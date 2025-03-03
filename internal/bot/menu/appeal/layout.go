package appeal

import (
	"github.com/jaxron/roapi.go/pkg/api"
	"github.com/robalyx/rotector/internal/bot/core/pagination"
	"github.com/robalyx/rotector/internal/bot/core/session"
	"github.com/robalyx/rotector/internal/common/setup"
	"github.com/robalyx/rotector/internal/common/storage/database"
	"go.uber.org/zap"
)

// Layout handles the appeal menu and its dependencies.
type Layout struct {
	db           database.Client
	roAPI        *api.API
	logger       *zap.Logger
	overviewMenu *OverviewMenu
	ticketMenu   *TicketMenu
	verifyMenu   *VerifyMenu
}

// New creates a Layout by initializing the appeal menu.
func New(app *setup.App) *Layout {
	// Initialize layout
	l := &Layout{
		db:     app.DB,
		roAPI:  app.RoAPI,
		logger: app.Logger,
	}

	// Initialize menus with reference to this layout
	l.overviewMenu = NewOverviewMenu(l)
	l.ticketMenu = NewTicketMenu(l)
	l.verifyMenu = NewVerifyMenu(l)

	return l
}

// Pages returns all the pages in the layout.
func (l *Layout) Pages() []*pagination.Page {
	return []*pagination.Page{
		l.overviewMenu.page,
		l.ticketMenu.page,
		l.verifyMenu.page,
	}
}

// ResetAppealData resets the appeal data in the session.
func ResetAppealData(s *session.Session) {
	session.AppealList.Delete(s)
	session.AppealCursor.Delete(s)
	session.AppealNextCursor.Delete(s)
	session.AppealPrevCursors.Delete(s)
	session.AppealSelected.Delete(s)
}
