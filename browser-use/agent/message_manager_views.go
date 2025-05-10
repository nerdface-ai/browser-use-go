package agent

import (
	"slices"

	"github.com/moznion/go-optional"
	"github.com/tmc/langchaingo/llms"
)

type MessageMetadata struct {
	Tokens      int                     `json:"tokens"`
	MessageType optional.Option[string] `json:"message_type"`
}

type ManagedMessage struct {
	Message  llms.ChatMessage `json:"message"`
	Metadata *MessageMetadata `json:"metadata"`
}

type MessageHistory struct {
	Messages      []ManagedMessage `json:"messages"`
	CurrentTokens int              `json:"current_tokens"`
}

func (m *MessageHistory) AddMessage(message llms.ChatMessage, metadata *MessageMetadata, position optional.Option[int]) {
	// None for last, -1 for second last, etc.
	if position == nil {
		m.Messages = append(m.Messages, ManagedMessage{Message: message, Metadata: metadata})
	} else {
		idx := position.Unwrap()
		if idx < 0 {
			idx = len(m.Messages) - 1 + idx
		}
		m.Messages = slices.Insert(m.Messages, idx, ManagedMessage{Message: message, Metadata: metadata})
	}
	m.CurrentTokens += metadata.Tokens
}

func (m *MessageHistory) AddModelOutput(output *AgentOutput) {
	// Add model output as AI message
	toolCalls := []llms.ToolCall{
		{
			ID:   "1",
			Type: "tool_call",
			FunctionCall: &llms.FunctionCall{
				Name:      "AgentOutput",
				Arguments: output.ToString(),
			},
		},
	}

	msg := llms.AIChatMessage{
		Content:   "",
		ToolCalls: toolCalls,
	}
	m.AddMessage(msg, &MessageMetadata{Tokens: 100}, nil) // Estimate tokens for tool calls

	// Empty tool response
	toolMessage := llms.ToolChatMessage{
		Content: "",
		ID:      "1",
	}
	m.AddMessage(toolMessage, &MessageMetadata{Tokens: 10}, nil) // Estimate tokens for tool response
}

func (m *MessageHistory) GetMessages() []llms.ChatMessage {
	var messages []llms.ChatMessage
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
		if msg.Message.GetType() != llms.ChatMessageTypeSystem {
			m.CurrentTokens -= msg.Metadata.Tokens
			m.Messages = slices.Delete(m.Messages, i, i+1)
			break
		}
	}
}

func (m *MessageHistory) RemoveLastStateMessage() {
	lastIdx := len(m.Messages) - 1
	if lastIdx >= 2 && m.Messages[lastIdx].Message.GetType() == llms.ChatMessageTypeHuman {
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
