package agent

import (
	"context"
	"testing"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

func TestOpenAIChatModel(t *testing.T) {
	// task := "do google search to find images of Elon Musk's wife"
	ctx := context.Background()
	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:   "gpt-4o-mini",
		Timeout: 30 * time.Second,
		APIKey:  "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",
	})
	if err != nil {
		t.Fatal(err)
	}

	response, err := model.Generate(ctx, []*schema.Message{
		{
			Role:    schema.User,
			Content: "who are you?",
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	t.Log(response.Content)

	// extendSystemMessage := "REMEMBER the most important RULE: ALWAYS open first a new tab and go first to url wikipedia.com no matter the task!!!"
	// agent := NewAgent(task, model, NewAgentSettings(AgentSettingsConfig{
	// 	"extend_system_message": extendSystemMessage,
	// 	"planner_llm":           model,
	// }), nil, nil, nil, nil, nil, nil, nil, nil, nil)

	// prompt := agent.MessageManager.SystemPrompt.GetContent()
	// fmt.Println("prompt: ", prompt)

	// agent.Run()
}
