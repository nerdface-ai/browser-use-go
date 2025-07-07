package agent

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/cloudwego/eino/schema"
)

// MessageMetadata represents metadata for a message
type MessageMetadata struct {
	Tokens      int     `json:"tokens"`
	MessageType *string `json:"message_type,omitempty"`
}

// ManagedMessage represents a message with metadata
type ManagedMessage struct {
	Message  *schema.Message  `json:"message"`
	Metadata *MessageMetadata `json:"metadata"`
}

// HistoryItem represents a single agent history item with its data and string representation
type HistoryItem struct {
	StepNumber              *int    `json:"step_number,omitempty"`
	EvaluationPreviousGoal  *string `json:"evaluation_previous_goal,omitempty"`
	Memory                  *string `json:"memory,omitempty"`
	NextGoal                *string `json:"next_goal,omitempty"`
	ActionResults           *string `json:"action_results,omitempty"`
	Error                   *string `json:"error,omitempty"`
	SystemMessage           *string `json:"system_message,omitempty"`
}

// NewHistoryItem creates a new HistoryItem and validates it
func NewHistoryItem(stepNumber *int, evaluationPreviousGoal *string, memory *string, nextGoal *string, actionResults *string, errorMsg *string, systemMessage *string) (*HistoryItem, error) {
	item := &HistoryItem{
		StepNumber:              stepNumber,
		EvaluationPreviousGoal:  evaluationPreviousGoal,
		Memory:                  memory,
		NextGoal:                nextGoal,
		ActionResults:           actionResults,
		Error:                   errorMsg,
		SystemMessage:           systemMessage,
	}

	if err := item.Validate(); err != nil {
		return nil, err
	}

	return item, nil
}

// Validate checks that error and system_message are not both provided
func (h *HistoryItem) Validate() error {
	if h.Error != nil && h.SystemMessage != nil {
		return errors.New("cannot have both error and system_message at the same time")
	}
	return nil
}

// ToString gets string representation of the history item
func (h *HistoryItem) ToString() string {
	var stepStr string
	if h.StepNumber != nil {
		stepStr = fmt.Sprintf("step_%d", *h.StepNumber)
	} else {
		stepStr = "step_unknown"
	}

	if h.Error != nil {
		return fmt.Sprintf("<%s>\n%s\n</%s>", stepStr, *h.Error, stepStr)
	} else if h.SystemMessage != nil {
		return fmt.Sprintf("<sys>\n%s\n</sys>", *h.SystemMessage)
	} else {
		var contentParts []string

		if h.EvaluationPreviousGoal != nil {
			contentParts = append(contentParts, fmt.Sprintf("Evaluation of Previous Step: %s", *h.EvaluationPreviousGoal))
		}
		if h.Memory != nil {
			contentParts = append(contentParts, fmt.Sprintf("Memory: %s", *h.Memory))
		}
		if h.NextGoal != nil {
			contentParts = append(contentParts, fmt.Sprintf("Next Goal: %s", *h.NextGoal))
		}
		if h.ActionResults != nil {
			contentParts = append(contentParts, *h.ActionResults)
		}

		content := strings.Join(contentParts, "\n")
		return fmt.Sprintf("<%s>\n%s\n</%s>", stepStr, content, stepStr)
	}
}

// MessageHistory represents history of messages with token tracking
type MessageHistory struct {
	Messages      []ManagedMessage `json:"messages"`
	CurrentTokens int              `json:"current_tokens"`
}

// NewMessageHistory creates a new MessageHistory
func NewMessageHistory() *MessageHistory {
	return &MessageHistory{
		Messages:      make([]ManagedMessage, 0),
		CurrentTokens: 0,
	}
}

// AddMessage adds message to history
func (m *MessageHistory) AddMessage(message *schema.Message, metadata *MessageMetadata, position *int) {
	managedMsg := ManagedMessage{Message: message, Metadata: metadata}
	if position == nil {
		m.Messages = append(m.Messages, managedMsg)
	} else {
		pos := *position
		if pos < 0 {
			pos = len(m.Messages) + pos + 1
		}
		if pos < 0 || pos > len(m.Messages) {
			pos = len(m.Messages)
		}
		m.Messages = slices.Insert(m.Messages, pos, managedMsg)
	}
	m.CurrentTokens += metadata.Tokens
}

// GetMessages returns all schema messages
func (m *MessageHistory) GetMessages() []*schema.Message {
	var messages []*schema.Message
	for _, msg := range m.Messages {
		messages = append(messages, msg.Message)
	}
	return messages
}

// GetTotalTokens returns current token count
func (m *MessageHistory) GetTotalTokens() int {
	return m.CurrentTokens
}

// RemoveOldestMessage removes the oldest non-system message
func (m *MessageHistory) RemoveOldestMessage() {
	for i, msg := range m.Messages {
		if msg.Message.Role != schema.System {
			m.CurrentTokens -= msg.Metadata.Tokens
			m.Messages = slices.Delete(m.Messages, i, i+1)
			break
		}
	}
}

// RemoveLastStateMessage removes last state message from history
func (m *MessageHistory) RemoveLastStateMessage() {
	lastIdx := len(m.Messages) - 1
	if lastIdx >= 2 && m.Messages[lastIdx].Message.Role == schema.User {
		m.CurrentTokens -= m.Messages[lastIdx].Metadata.Tokens
		m.Messages = slices.Delete(m.Messages, lastIdx, lastIdx+1)
	}
}

// MessageManagerState holds the state for MessageManager
type MessageManagerState struct {
	History               *MessageHistory `json:"history"`
	ToolId                int             `json:"tool_id"`
	AgentHistoryItems     []*HistoryItem  `json:"agent_history_items"`
	ReadStateDescription  string          `json:"read_state_description"`
}

// NewMessageManagerState creates a new MessageManagerState with default values
func NewMessageManagerState() *MessageManagerState {
	initialItem, _ := NewHistoryItem(
		intPtr(0),
		nil,
		nil,
		nil,
		nil,
		nil,
		stringPtr("Agent initialized"),
	)

	return &MessageManagerState{
		History:               NewMessageHistory(),
		ToolId:                1,
		AgentHistoryItems:     []*HistoryItem{initialItem},
		ReadStateDescription:  "",
	}
}

// Helper functions for pointer creation
func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}
