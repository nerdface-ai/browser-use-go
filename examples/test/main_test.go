package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/nerdface-ai/browser-use-go/pkg/agent"
	"github.com/nerdface-ai/browser-use-go/pkg/dotenv"
)

func TestAgentRun(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	if os.Getenv("GITHUB_ACTIONS") == "1" {
		t.Skip("skip test")
	}
	dotenv.LoadEnv("../../.env")

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
	historyResult, err := ag.Run(agent.WithMaxSteps(20))

	if err != nil {
		log.Fatal(err.Error())
	}
	log.Infof("agent output: %s", *historyResult.LastResult().ExtractedContent)
}
