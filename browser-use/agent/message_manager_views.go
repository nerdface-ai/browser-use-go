package agent

import (
	"slices"

	"github.com/cloudwego/eino/schema"
)

type MessageMetadata struct {
	Tokens      int     `json:"tokens"`
	MessageType *string `json:"message_type,omitempty"`
}

type ManagedMessage struct {
	Message  *schema.Message  `json:"message"`
	Metadata *MessageMetadata `json:"metadata"`
}

type MessageHistory struct {
	Messages      []ManagedMessage `json:"messages"`
	CurrentTokens int              `json:"current_tokens"`
}

func (m *MessageHistory) AddMessage(message *schema.Message, metadata *MessageMetadata, position *int) {
	// None for last, -1 for second last, etc.
	if position == nil {
		m.Messages = append(m.Messages, ManagedMessage{Message: message, Metadata: metadata})
	} else {
		idx := *position
		if idx < 0 {
			idx = len(m.Messages) - 1 + idx
		}
		m.Messages = slices.Insert(m.Messages, idx, ManagedMessage{Message: message, Metadata: metadata})
	}
	m.CurrentTokens += metadata.Tokens
}

func (m *MessageHistory) AddModelOutput(output *AgentOutput) {
	// Add model output as AI message
	toolCalls := []schema.ToolCall{
		{
			ID:   "1",
			Type: "tool_call",
			Function: schema.FunctionCall{
				Name:      "AgentOutput",
				Arguments: output.ToString(),
			},
		},
	}

	msg := &schema.Message{
		Role:      schema.Assistant,
		Content:   "",
		ToolCalls: toolCalls,
	}
	m.AddMessage(msg, &MessageMetadata{Tokens: 100}, nil) // Estimate tokens for tool calls

	// Empty tool response
	toolMessage := &schema.Message{
		Role:       schema.Tool,
		Content:    "",
		ToolCallID: "1",
	}
	m.AddMessage(toolMessage, &MessageMetadata{Tokens: 10}, nil) // Estimate tokens for tool response
}

func (m *MessageHistory) GetMessages() []*schema.Message {
	var messages []*schema.Message
	for _, msg := range m.Messages {
		messages = append(messages, msg.Message)
	}
	return messages
}

func (m *MessageHistory) GetTotalTokens() int {
	return m.CurrentTokens
}

func (m *MessageHistory) RemoveOldestMessage() {
	for i, msg := range m.Messages {
		if msg.Message.Role != schema.System {
			m.CurrentTokens -= msg.Metadata.Tokens
			m.Messages = slices.Delete(m.Messages, i, i+1)
			break
		}
	}
}

func (m *MessageHistory) RemoveLastStateMessage() {
	lastIdx := len(m.Messages) - 1
	if lastIdx >= 2 && m.Messages[lastIdx].Message.Role == schema.User {
		m.CurrentTokens -= m.Messages[lastIdx].Metadata.Tokens
		m.Messages = slices.Delete(m.Messages, lastIdx, lastIdx+1)
	}
}

type MessageManagerState struct {
	History *MessageHistory
	ToolId  int
}

func NewMessageManagerState() *MessageManagerState {
	return &MessageManagerState{
		History: &MessageHistory{
			Messages:      make([]ManagedMessage, 0),
			CurrentTokens: 0,
		},
		ToolId: 1,
	}
}
