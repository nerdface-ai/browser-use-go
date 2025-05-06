package agent

import (
	"nerdface-ai/browser-use-go/browser-use/browser"
	"nerdface-ai/browser-use-go/browser-use/dom"
	"strings"
	"testing"

	"github.com/tmc/langchaingo/llms"
)

func SampleMessageManager() *MessageManager {
	task := "Test task"
	systemPrompt := llms.SystemChatMessage{
		Content: "Test actions",
	}
	settings := MessageManagerSettings{
		MaxInputTokens:              1000,
		EstimatedCharactersPerToken: 3,
		ImageTokens:                 800,
	}
	state := NewMessageManagerState()
	return NewMessageManager(task, systemPrompt, &settings, state)
}

func TestInitialMessages(t *testing.T) {
	// Test that message manager initializes with system and task messages
	messageManager := SampleMessageManager()

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
	if !strings.Contains(messages[1].GetContent(), messageManager.Task) {
		t.Errorf("Expected task message to include %s, got %s", messageManager.Task, messages[1].GetContent())
	}
}

func TestAddStateMessage(t *testing.T) {
	// Test adding browser state message
	messageManager := SampleMessageManager()
	testUrl := "https://example.com"

	state := browser.BrowserState{
		Url:   testUrl,
		Title: "Test Page",
		ElementTree: &dom.DOMElementNode{
			TagName:    "div",
			Attributes: map[string]string{},
			Children:   []dom.DOMBaseNode{},
			IsVisible:  true,
			Parent:     nil,
			Xpath:      "//div",
		},
		SelectorMap: &dom.SelectorMap{},
		Tabs: []*browser.TabInfo{
			{
				PageId:       1,
				Url:          testUrl,
				Title:        "Test Page",
				ParentPageId: nil,
			},
		},
	}
	messageManager.AddStateMessage(&state, nil, nil, true)

	messages := messageManager.GetMessages()
	if len(messages) != 7 {
		t.Errorf("Expected 7 messages, got %d", len(messages))
	}
	if messages[2].GetType() != llms.ChatMessageTypeHuman {
		t.Errorf("Expected human message, got %T", messages[2].GetType())
	}
	if !strings.Contains(messages[6].GetContent(), testUrl) {
		t.Errorf("Expected state message to include %s, got %s", testUrl, messages[2].GetContent())
	}
}
