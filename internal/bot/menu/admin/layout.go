package admin

import (
	"github.com/robalyx/rotector/internal/bot/core/pagination"
	"github.com/robalyx/rotector/internal/common/setup"
	"github.com/robalyx/rotector/internal/common/storage/database"
	"go.uber.org/zap"
)

// Layout handles the admin menu and its submenus.
type Layout struct {
	db          database.Client
	logger      *zap.Logger
	mainMenu    *MainMenu
	confirmMenu *ConfirmMenu
}

// New creates a Layout by initializing all admin menus and registering their
// pages with the pagination manager.
func New(app *setup.App) *Layout {
	// Initialize layout
	l := &Layout{
		db:     app.DB,
		logger: app.Logger,
	}

	// Initialize menus with reference to this layout
	l.mainMenu = NewMainMenu(l)
	l.confirmMenu = NewConfirmMenu(l)

	return l
}

// Pages returns all the pages in the layout.
func (l *Layout) Pages() []*pagination.Page {
	return []*pagination.Page{
		l.mainMenu.page,
		l.confirmMenu.page,
	}
}
