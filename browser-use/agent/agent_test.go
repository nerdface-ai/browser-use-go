package agent

import (
	"fmt"
	"testing"

	"github.com/tmc/langchaingo/llms/openai"
)

func TestOpenAIAgent(t *testing.T) {
	task := "do google search to find images of Elon Musk's wife"
	model, err := openai.New(openai.WithModel("gpt-4o-mini"))
	if err != nil {
		t.Fatal(err)
	}

	extendSystemMessage := "REMEMBER the most important RULE: ALWAYS open first a new tab and go first to url wikipedia.com no matter the task!!!"
	agent := NewAgent(task, model, NewAgentSettings(AgentSettingsConfig{
		"extend_system_message": extendSystemMessage,
		"planner_llm":           model,
	}), nil, nil, nil, nil, nil, nil, nil, nil, nil)

	prompt := agent.MessageManager.SystemPrompt.GetContent()
	fmt.Println("prompt: ", prompt)

	// agent.Run()
}
