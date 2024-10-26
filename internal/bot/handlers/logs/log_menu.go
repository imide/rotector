package logs

import (
	"fmt"
	"strconv"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/rotector/rotector/internal/bot/constants"
	"github.com/rotector/rotector/internal/bot/handlers/logs/builders"
	"github.com/rotector/rotector/internal/bot/interfaces"
	"github.com/rotector/rotector/internal/bot/pagination"
	"github.com/rotector/rotector/internal/bot/session"
	"github.com/rotector/rotector/internal/bot/utils"
	"github.com/rotector/rotector/internal/common/database"
	"go.uber.org/zap"
)

// LogMenu handles the log querying functionality.
type LogMenu struct {
	handler *Handler
	page    *pagination.Page
}

// NewLogMenu creates a new LogMenu instance.
func NewLogMenu(h *Handler) *LogMenu {
	m := &LogMenu{handler: h}
	m.page = &pagination.Page{
		Name: "Log Menu",
		Message: func(s *session.Session) *discord.MessageUpdateBuilder {
			return builders.NewLogEmbed(s).Build()
		},
		SelectHandlerFunc: m.handleSelectMenu,
		ButtonHandlerFunc: m.handleButton,
		ModalHandlerFunc:  m.handleModal,
	}
	return m
}

// ShowLogMenu displays the log querying menu.
func (m *LogMenu) ShowLogMenu(event interfaces.CommonEvent, s *session.Session) {
	// Initialize or reset session data for the log menu
	s.Set(constants.SessionKeyLogs, []*database.UserActivityLog{})
	s.Set(constants.SessionKeyUserID, uint64(0))
	s.Set(constants.SessionKeyReviewerID, uint64(0))
	s.Set(constants.SessionKeyActivityTypeFilter, "")
	s.Set(constants.SessionKeyDateRangeStart, time.Time{})
	s.Set(constants.SessionKeyDateRangeEnd, time.Time{})
	s.Set(constants.SessionKeyTotalItems, 0)
	s.Set(constants.SessionKeyStart, 0)
	s.Set(constants.SessionKeyPaginationPage, 0)

	m.handler.paginationManager.NavigateTo(event, s, m.page, "")
}

// handleSelectMenu handles the select menu interactions for the log menu.
func (m *LogMenu) handleSelectMenu(event *events.ComponentInteractionCreate, s *session.Session, customID string, option string) {
	switch customID {
	case constants.ActionSelectMenuCustomID:
		switch option {
		case constants.LogsQueryUserIDOption:
			m.showQueryModal(event, option, "User ID", "ID", "Enter the User ID to query logs")
		case constants.LogsQueryReviewerIDOption:
			m.showQueryModal(event, option, "Reviewer ID", "ID", "Enter the Reviewer ID to query logs")
		case constants.LogsQueryDateRangeOption:
			m.showQueryModal(event, constants.LogsQueryDateRangeOption, "Date Range", "Date Range", "YYYY-MM-DD to YYYY-MM-DD")
		}
	case constants.LogsQueryActivityTypeFilterCustomID:
		s.Set(constants.SessionKeyActivityTypeFilter, option)
		s.Set(constants.SessionKeyStart, 0)
		s.Set(constants.SessionKeyPaginationPage, 0)
		m.updateLogData(event, s, 0)
	}
}

// handleButton handles the button interactions for the log menu.
func (m *LogMenu) handleButton(event *events.ComponentInteractionCreate, s *session.Session, customID string) {
	switch customID {
	case string(constants.BackButtonCustomID):
		m.handler.dashboardHandler.ShowDashboard(event)
	case string(utils.ViewerFirstPage), string(utils.ViewerPrevPage), string(utils.ViewerNextPage), string(utils.ViewerLastPage):
		m.handlePagination(event, s, utils.ViewerAction(customID))
	}
}

// handleModal handles the modal submit interactions for the log menu.
func (m *LogMenu) handleModal(event *events.ModalSubmitInteractionCreate, s *session.Session) {
	customID := event.Data.CustomID
	switch customID {
	case constants.LogsQueryUserIDOption, constants.LogsQueryReviewerIDOption:
		m.handleIDModalSubmit(event, s, customID)
	case constants.LogsQueryDateRangeOption:
		m.handleDateRangeModalSubmit(event, s)
	}
}

// showQueryModal displays a modal for querying user or reviewer ID.
func (m *LogMenu) showQueryModal(event *events.ComponentInteractionCreate, option, title, label, placeholder string) {
	modal := discord.NewModalCreateBuilder().
		SetCustomID(option).
		SetTitle(title).
		AddActionRow(
			discord.NewTextInput(constants.LogsQueryInputCustomID, discord.TextInputStyleShort, label).
				WithPlaceholder(placeholder).
				WithRequired(true),
		).
		Build()

	if err := event.Modal(modal); err != nil {
		m.handler.logger.Error("Failed to show query modal", zap.Error(err))
	}
}

// handleIDModalSubmit processes the query modal submission.
func (m *LogMenu) handleIDModalSubmit(event *events.ModalSubmitInteractionCreate, s *session.Session, queryType string) {
	idStr := event.Data.Text(constants.LogsQueryInputCustomID)
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		m.handler.paginationManager.NavigateTo(event, s, m.page, "Invalid ID provided. Please enter a valid numeric ID.")
		return
	}

	if queryType == constants.LogsQueryUserIDOption {
		s.Set(constants.SessionKeyUserID, id)
	} else if queryType == constants.LogsQueryReviewerIDOption {
		s.Set(constants.SessionKeyReviewerID, id)
	}
	s.Set(constants.SessionKeyPaginationPage, 0)

	m.updateLogData(event, s, 0)
}

// handleDateRangeModalSubmit processes the date range modal submission.
func (m *LogMenu) handleDateRangeModalSubmit(event *events.ModalSubmitInteractionCreate, s *session.Session) {
	dateRangeStr := event.Data.Text(constants.LogsQueryInputCustomID)
	startDate, endDate, err := utils.ParseDateRange(dateRangeStr)
	if err != nil {
		m.handler.paginationManager.NavigateTo(event, s, m.page, fmt.Sprintf("Invalid date range: %v", err))
		return
	}

	s.Set(constants.SessionKeyDateRangeStart, startDate)
	s.Set(constants.SessionKeyDateRangeEnd, endDate)
	s.Set(constants.SessionKeyPaginationPage, 0)

	m.updateLogData(event, s, 0)
}

// handlePagination handles the pagination for the log menu.
func (m *LogMenu) handlePagination(event *events.ComponentInteractionCreate, s *session.Session, action utils.ViewerAction) {
	totalItems := s.GetInt(constants.SessionKeyTotalItems)
	maxPage := (totalItems - 1) / constants.LogsPerPage

	newPage, ok := action.ParsePageAction(s, action, maxPage)
	if !ok {
		m.handler.logger.Warn("Invalid pagination action", zap.String("action", string(action)))
		m.handler.paginationManager.RespondWithError(event, "Invalid interaction.")
		return
	}

	m.updateLogData(event, s, newPage)
}

// updateLogData fetches and updates the log data based on the current query parameters.
func (m *LogMenu) updateLogData(event interfaces.CommonEvent, s *session.Session, page int) {
	userID := s.GetUint64(constants.SessionKeyUserID)
	reviewerID := s.GetUint64(constants.SessionKeyReviewerID)
	activityTypeFilter := s.GetString(constants.SessionKeyActivityTypeFilter)
	startDate := s.Get(constants.SessionKeyDateRangeStart).(time.Time)
	endDate := s.Get(constants.SessionKeyDateRangeEnd).(time.Time)

	logs, totalLogs, err := m.handler.db.UserActivity().GetLogs(userID, reviewerID, activityTypeFilter, startDate, endDate, page, constants.LogsPerPage)
	if err != nil {
		m.handler.logger.Error("Failed to get logs", zap.Error(err))
		m.handler.paginationManager.NavigateTo(event, s, m.page, "Failed to retrieve log data. Please try again.")
		return
	}

	s.Set(constants.SessionKeyLogs, logs)
	s.Set(constants.SessionKeyTotalItems, totalLogs)
	s.Set(constants.SessionKeyStart, page*constants.LogsPerPage)

	m.handler.paginationManager.NavigateTo(event, s, m.page, "")
}
