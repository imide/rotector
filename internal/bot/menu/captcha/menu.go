package captcha

import (
	"bytes"

	"github.com/disgoorg/disgo/discord"
	builder "github.com/robalyx/rotector/internal/bot/builder/captcha"
	"github.com/robalyx/rotector/internal/bot/constants"
	"github.com/robalyx/rotector/internal/bot/core/captcha"
	"github.com/robalyx/rotector/internal/bot/core/interaction"
	"github.com/robalyx/rotector/internal/bot/core/session"
	"go.uber.org/zap"
)

// Menu handles the CAPTCHA verification interface.
type Menu struct {
	layout  *Layout
	page    *interaction.Page
	captcha *captcha.Manager
}

// NewMenu creates a new CAPTCHA menu.
func NewMenu(layout *Layout) *Menu {
	m := &Menu{layout: layout}
	m.page = &interaction.Page{
		Name: constants.CaptchaPageName,
		Message: func(s *session.Session) *discord.MessageUpdateBuilder {
			return builder.NewBuilder(s).Build()
		},
		CleanupHandlerFunc: m.Cleanup,
		ShowHandlerFunc:    m.Show,
		ButtonHandlerFunc:  m.handleButton,
		ModalHandlerFunc:   m.handleModal,
	}
	return m
}

// Show displays the CAPTCHA verification interface.
func (m *Menu) Show(ctx *interaction.Context, s *session.Session) {
	// Generate CAPTCHA image
	digits, imgBuffer, err := m.captcha.GenerateImage()
	if err != nil {
		m.layout.logger.Error("Failed to generate CAPTCHA image", zap.Error(err))
		ctx.Error("Failed to generate CAPTCHA. Please try again.")
		return
	}

	// Store data in session
	session.CaptchaAnswer.Set(s, string(digits))
	session.ImageBuffer.Set(s, imgBuffer)

	ctx.Show(constants.CaptchaPageName, "Generated new CAPTCHA.")
}

// Cleanup handles the cleanup of the CAPTCHA menu.
func (m *Menu) Cleanup(s *session.Session) {
	session.CaptchaAnswer.Delete(s)
	session.ImageBuffer.Delete(s)
}

// handleButton processes button interactions.
func (m *Menu) handleButton(ctx *interaction.Context, _ *session.Session, customID string) {
	switch customID {
	case constants.BackButtonCustomID:
		ctx.NavigateBack("")
	case constants.CaptchaRefreshButtonCustomID:
		ctx.Reload("Generated new CAPTCHA.")
	case constants.CaptchaAnswerButtonCustomID:
		m.handleCaptchaAnswer(ctx)
	}
}

// handleModal processes modal submissions.
func (m *Menu) handleModal(ctx *interaction.Context, s *session.Session) {
	if ctx.Event().CustomID() != constants.CaptchaAnswerModalCustomID {
		return
	}

	// Convert user's answer to digits
	answer := ctx.Event().ModalData().Text(constants.CaptchaAnswerInputCustomID)
	if len(answer) != 6 {
		ctx.Error("❌ Invalid answer length. Please enter exactly 6 digits.")
		return
	}

	userDigits := make([]byte, 6)
	for i, rn := range answer {
		if rn < '0' || rn > '9' {
			ctx.Error("❌ Invalid answer. Please enter only digits.")
			return
		}
		userDigits[i] = byte(rn - '0')
	}

	// Compare answers
	correctDigits := session.CaptchaAnswer.Get(s)

	if !bytes.Equal(userDigits, []byte(correctDigits)) {
		ctx.Error("❌ Incorrect CAPTCHA answer. Please try again.")
		return
	}

	// Reset reviews counter
	session.UserCaptchaUsageCaptchaReviewCount.Set(s, 0)

	// Return to previous page
	ctx.NavigateBack("✅ CAPTCHA verified successfully!")
	session.CaptchaAnswer.Delete(s)
	session.ImageBuffer.Delete(s)
}

// handleCaptchaAnswer handles the modal for the CAPTCHA answer.
func (m *Menu) handleCaptchaAnswer(ctx *interaction.Context) {
	modal := discord.NewModalCreateBuilder().
		SetCustomID(constants.CaptchaAnswerModalCustomID).
		SetTitle("Enter CAPTCHA Answer").
		AddActionRow(
			discord.NewTextInput(constants.CaptchaAnswerInputCustomID, discord.TextInputStyleShort, "Answer").
				WithRequired(true).
				WithPlaceholder("Enter the 6 digits you see..."),
		)

	ctx.Modal(modal)
}
