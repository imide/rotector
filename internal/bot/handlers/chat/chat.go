package chat

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/robalyx/rotector/internal/ai"
	"github.com/robalyx/rotector/internal/bot/constants"
	"github.com/robalyx/rotector/internal/bot/core/interaction"
	"github.com/robalyx/rotector/internal/bot/core/session"
	view "github.com/robalyx/rotector/internal/bot/views/chat"
	"github.com/robalyx/rotector/internal/database/types/enum"
	"go.uber.org/zap"
)

var ErrResponseTimedOut = errors.New("response timed out")

// Menu handles the display and interaction logic for AI chat.
type Menu struct {
	layout *Layout
	page   *interaction.Page
}

// NewMenu creates a new chat menu.
func NewMenu(layout *Layout) *Menu {
	m := &Menu{layout: layout}
	m.page = &interaction.Page{
		Name: constants.ChatPageName,
		Message: func(s *session.Session) *discord.MessageUpdateBuilder {
			return view.NewBuilder(s).Build()
		},
		ShowHandlerFunc:   m.Show,
		SelectHandlerFunc: m.handleSelectMenu,
		ButtonHandlerFunc: m.handleButton,
		ModalHandlerFunc:  m.handleModal,
	}

	return m
}

// Show prepares and displays the chat interface.
func (m *Menu) Show(_ *interaction.Context, s *session.Session) {
	// Check if credits should be reset
	now := time.Now()

	firstMessageTime := session.UserChatMessageUsageFirstMessageTime.Get(s)
	if !firstMessageTime.IsZero() && now.Sub(firstMessageTime) > constants.ChatMessageResetLimit {
		session.UserChatMessageUsageFirstMessageTime.Set(s, time.Unix(0, 0))
		session.UserChatMessageUsageMessageCount.Set(s, 0)
	}
}

// handleButton processes button interactions.
func (m *Menu) handleButton(ctx *interaction.Context, s *session.Session, customID string) {
	action := session.ViewerAction(customID)
	switch action {
	case session.ViewerFirstPage, session.ViewerPrevPage, session.ViewerNextPage, session.ViewerLastPage:
		chatContext := session.ChatContext.Get(s)
		groupedContext := chatContext.GroupByType()

		// Calculate total pairs from human messages since they initiate each pair
		totalPairs := len(groupedContext[ai.ContextTypeHuman])
		maxPage := (totalPairs - 1) / constants.ChatMessagesPerPage
		page := action.ParsePageAction(s, maxPage)

		session.PaginationPage.Set(s, page)
		ctx.Reload("")

		return
	}

	switch customID {
	case constants.ChatSendButtonID:
		m.handleChatSend(ctx)
	case constants.BackButtonCustomID:
		ctx.NavigateBack("")
	case constants.ChatClearHistoryButtonID:
		// Clear entire chat history
		session.ChatContext.Set(s, make(ai.ChatContext, 0))
		session.PaginationPage.Set(s, 0)
		ctx.Reload("Chat history cleared.")
	case constants.ChatClearContextButtonID:
		// Only clear review contexts that haven't been used in messages yet
		chatContext := session.ChatContext.Get(s)
		groupedContext := chatContext.GroupByType()

		// Get all chat messages
		chatMessages := make([]ai.Context, 0)
		chatMessages = append(chatMessages, groupedContext[ai.ContextTypeHuman]...)
		chatMessages = append(chatMessages, groupedContext[ai.ContextTypeAI]...)

		// Find the last chat message's position in the full context
		lastMessageIndex := -1

		if len(chatMessages) > 0 {
			lastMessage := chatMessages[len(chatMessages)-1]
			for i, ctx := range chatContext {
				if ctx == lastMessage {
					lastMessageIndex = i
					break
				}
			}
		}

		// Create new context keeping everything up to and including the last message
		newContext := make(ai.ChatContext, 0)
		if lastMessageIndex >= 0 {
			newContext = append(newContext, chatContext[:lastMessageIndex+1]...)
		}

		session.ChatContext.Set(s, newContext)
		ctx.Reload("Unused review contexts cleared.")
	}
}

// handleSelectMenu processes select menu interactions.
func (m *Menu) handleSelectMenu(ctx *interaction.Context, s *session.Session, customID, option string) {
	switch customID {
	case constants.ChatModelSelectID:
		// Parse option to chat model
		chatModel, err := enum.ChatModelString(option)
		if err != nil {
			m.layout.logger.Error("Failed to parse chat model", zap.Error(err))
			ctx.Error("Failed to parse chat model. Please try again.")

			return
		}

		// Update user settings with new chat model
		session.UserChatModel.Set(s, chatModel)

		// Refresh the menu
		ctx.Reload(fmt.Sprintf("Switched to %s model", chatModel.String()))
	}
}

// handleModal processes modal submissions for chat input.
func (m *Menu) handleModal(ctx *interaction.Context, s *session.Session) {
	switch ctx.Event().CustomID() {
	case constants.ChatInputModalID:
		message := strings.TrimSpace(ctx.Event().ModalData().Text(constants.ChatInputCustomID))
		if message == "" {
			ctx.Cancel("Message cannot be empty")
			return
		}

		// Check message limits
		if allowed, errMsg := m.checkMessageLimits(s); !allowed {
			ctx.Error(errMsg)
			return
		}

		// Add user message to context
		chatContext := session.ChatContext.Get(s)
		chatContext = append(chatContext, ai.Context{
			Type:    ai.ContextTypeHuman,
			Content: message,
		})
		session.ChatContext.Set(s, chatContext)

		// Set streaming state and show initial status
		session.PaginationIsStreaming.Set(s, true)
		session.ChatStreamingMessage.Set(s, "AI is typing...")
		ctx.Reload("")

		// Stream AI response
		if err := m.streamResponse(ctx, s, message); err != nil {
			// Clean up on error
			chatContext = chatContext[:len(chatContext)-1] // Remove the user message
			session.ChatContext.Set(s, chatContext)
			ctx.Error(fmt.Sprintf("Failed to get response: %v", err))

			return
		}

		// Update final state
		session.PaginationPage.Set(s, 0)
		session.PaginationIsStreaming.Set(s, false)
		ctx.Reload("Response completed.")
	}
}

// streamResponse handles the AI response streaming with buffered updates.
func (m *Menu) streamResponse(ctx *interaction.Context, s *session.Session, message string) error {
	chatContext := session.ChatContext.Get(s)
	currentModel := session.UserChatModel.Get(s)

	// Stream response
	responseChan := m.layout.chatHandler.StreamResponse(
		ctx.Context(),
		chatContext,
		currentModel,
		message,
	)

	// Buffer for collecting response chunks
	var (
		aiResponse   strings.Builder
		streamBuffer strings.Builder
	)

	lastUpdate := time.Now()

	// Stream and buffer the response
	for {
		select {
		case response, ok := <-responseChan:
			if !ok {
				// Channel closed, streaming complete
				chatContext = append(chatContext, ai.Context{
					Type:    ai.ContextTypeAI,
					Content: aiResponse.String(),
					Model:   currentModel.String(),
				})
				session.ChatContext.Set(s, chatContext)
				session.ChatStreamingMessage.Set(s, "")
				session.PaginationIsStreaming.Set(s, false)

				return nil
			}

			aiResponse.WriteString(response)

			// Process and append new chunk to stream buffer
			if response != "" {
				streamBuffer.WriteString(response)

				// If buffer exceeds max length, truncate from the start
				content := streamBuffer.String()
				if len(content) > 1024 {
					// Find the first newline after truncation point to maintain message integrity
					truncatePoint := len(content) - 1024
					if newlineIdx := strings.Index(content[truncatePoint:], "\n"); newlineIdx != -1 {
						truncatePoint += newlineIdx + 1
					}

					content = content[truncatePoint:]

					// Set truncated content with indicator
					session.ChatStreamingMessage.Set(s, "*Earlier content truncated for length*\n\n"+content)
				} else {
					session.ChatStreamingMessage.Set(s, content)
				}
			}

			// Update UI if enough time has passed
			if time.Since(lastUpdate) > time.Second {
				ctx.Reload("")

				lastUpdate = time.Now()
			}

		case <-ctx.Context().Done():
			// Clean up streaming state on error
			session.PaginationIsStreaming.Set(s, false)
			session.ChatStreamingMessage.Set(s, "")

			return ErrResponseTimedOut
		}
	}
}

// handleChatSend handles the chat send button.
func (m *Menu) handleChatSend(ctx *interaction.Context) {
	modal := discord.NewModalCreateBuilder().
		SetCustomID(constants.ChatInputModalID).
		SetTitle("Chat with AI").
		AddActionRow(
			discord.NewTextInput(constants.ChatInputCustomID, discord.TextInputStyleParagraph, "Message").
				WithRequired(true).
				WithMaxLength(512).
				WithPlaceholder("Type your message here..."),
		)

	ctx.Modal(modal)
}

// checkMessageLimits checks if the user has exceeded their daily message limit.
// Returns true if the message should be allowed, false if it should be blocked.
func (m *Menu) checkMessageLimits(s *session.Session) (bool, string) {
	now := time.Now()
	firstMessageTime := session.UserChatMessageUsageFirstMessageTime.Get(s)
	messageCount := session.UserChatMessageUsageMessageCount.Get(s)

	if firstMessageTime.IsZero() || now.Sub(firstMessageTime) > constants.ChatMessageResetLimit {
		// First message or past time limit - reset both time and count
		session.UserChatMessageUsageFirstMessageTime.Set(s, now)
		session.UserChatMessageUsageMessageCount.Set(s, 1)
	} else {
		// Within time limit - check and increment message count
		if messageCount >= constants.MaxChatMessagesPerDay {
			timeLeft := firstMessageTime.Add(constants.ChatMessageResetLimit).Sub(now)

			return false, fmt.Sprintf("You have reached the limit of %d messages per day. Please try again in %s.",
				constants.MaxChatMessagesPerDay,
				timeLeft.String())
		}

		session.UserChatMessageUsageMessageCount.Set(s, messageCount+1)
	}

	return true, ""
}
