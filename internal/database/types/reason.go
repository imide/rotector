package types

import (
	"fmt"

	"github.com/robalyx/rotector/internal/database/types/enum"
)

// Reason represents a structured reason for flagging a user or group.
type Reason struct {
	Message    string   `json:"message"`    // The actual reason message
	Confidence float64  `json:"confidence"` // Confidence score for this specific reason
	Evidence   []string `json:"evidence"`   // Any evidence (like flagged content) specific to this reason
}

// ReasonType represents a type that can be used as a reason identifier.
type ReasonType interface {
	enum.UserReasonType | enum.GroupReasonType
	fmt.Stringer
}

// Reasons maps reason types to their corresponding reason details.
type Reasons[T ReasonType] map[T]*Reason

// Add adds or updates a reason in the reasons map.
// If the reason type already exists, it updates the existing entry.
func (r Reasons[T]) Add(reasonType T, reason *Reason) {
	r[reasonType] = reason
}

// Messages returns an array of all reason messages.
func (r Reasons[T]) Messages() []string {
	messages := make([]string, 0, len(r))
	for _, reason := range r {
		messages = append(messages, reason.Message)
	}

	return messages
}

// Types returns an array of all reason types.
func (r Reasons[T]) Types() []string {
	types := make([]string, 0, len(r))
	for reasonType := range r {
		types = append(types, reasonType.String())
	}

	return types
}

// ReasonInfos returns an array of ReasonInfo structs containing both type and message.
// This is used for AI analysis where both the type and detailed message are needed.
func (r Reasons[T]) ReasonInfos() []ReasonInfo {
	infos := make([]ReasonInfo, 0, len(r))
	for reasonType, reason := range r {
		infos = append(infos, ReasonInfo{
			Type:    reasonType.String(),
			Message: reason.Message,
		})
	}

	return infos
}

// ReasonInfo represents a reason with both type and message for AI analysis.
type ReasonInfo struct {
	Type    string `json:"type"`    // The type of reason (e.g., Profile, Friend, Group, etc.)
	Message string `json:"message"` // The detailed reason message explaining why this was flagged
}
