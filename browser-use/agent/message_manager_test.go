package agent

import (
	"strings"
	"testing"

	"github.com/tmc/langchaingo/llms"
)

func TestInitialMessages(t *testing.T) {
	// Test that message manager initializes with system and task messages
	task := "Test task"
	systemPrompt := llms.SystemChatMessage{
		Content: "Test actions",
	}
	settings := MessageManagerSettings{
		MaxInputTokens:              1000,
		EstimatedCharactersPerToken: 3,
		ImageTokens:                 800,
	}

	messageManager := NewMessageManager(task, systemPrompt, &settings, nil)

	messages := messageManager.GetMessages()
	if len(messages) != 6 {
		t.Errorf("Expected 6 messages, got %d", len(messages))
	}
	if messages[0].GetType() != llms.ChatMessageTypeSystem {
		t.Errorf("Expected system message, got %T", messages[0].GetType())
	}
	if messages[1].GetType() != llms.ChatMessageTypeHuman {
		t.Errorf("Expected human message, got %T", messages[1].GetType())
	}
	if !strings.Contains(messages[1].GetContent(), task) {
		t.Errorf("Expected task message to include %s, got %s", task, messages[1].GetContent())
	}
}
