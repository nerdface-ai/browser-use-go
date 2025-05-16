package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/nerdface-ai/browser-use-go/internals/utils"
	"github.com/nerdface-ai/browser-use-go/pkg/agent"
)

func TestAgentRun(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") == "1" {
		t.Skip("skip test")
	}
	utils.LoadEnv("../../.env")

	ctx := context.Background()
	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:   "gpt-4o-mini",
		Timeout: 30 * time.Second,
		APIKey:  os.Getenv("OPENAI_API_KEY"),
	})
	if err != nil {
		log.Fatal(err)
	}

	task := "do google search and find who is Elon Musk's wife"
	ag := agent.NewAgent(task, model)
	ag.Run(10, nil, nil)

	log.Info("agent output: %v", ag.AgentOutput)

}
