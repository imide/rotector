package setting

import (
	"github.com/robalyx/rotector/internal/bot/core/pagination"
	"github.com/robalyx/rotector/internal/bot/core/session"
	"github.com/robalyx/rotector/internal/common/setup"
	"github.com/robalyx/rotector/internal/common/storage/database"
	"go.uber.org/zap"
)

// Layout handles all setting-related menus and their interactions.
type Layout struct {
	db         database.Client
	updateMenu *UpdateMenu
	userMenu   *UserMenu
	botMenu    *BotMenu
	registry   *session.SettingRegistry
	logger     *zap.Logger
}

// New creates a Layout by initializing all setting menus.
func New(app *setup.App) *Layout {
	// Initialize layout
	l := &Layout{
		db:       app.DB,
		logger:   app.Logger,
		registry: session.NewSettingRegistry(),
	}

	// Initialize all menus with references to this layout
	l.updateMenu = NewUpdateMenu(l)
	l.userMenu = NewUserMenu(l)
	l.botMenu = NewBotMenu(l)

	return l
}

// Pages returns all the pages in the layout.
func (l *Layout) Pages() []*pagination.Page {
	return []*pagination.Page{
		l.updateMenu.page,
		l.userMenu.page,
		l.botMenu.page,
	}
}
