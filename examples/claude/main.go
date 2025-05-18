package main

import (
	"context"
	"os"

	"github.com/charmbracelet/log"
	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/nerdface-ai/browser-use-go/pkg/agent"
	"github.com/nerdface-ai/browser-use-go/pkg/dotenv"
)

func main() {
	log.SetLevel(log.DebugLevel)
	dotenv.LoadEnv(".env")

	ctx := context.Background()
	model, err := claude.NewChatModel(ctx, &claude.Config{
		Model:     "claude-3-7-sonnet-20250219",
		APIKey:    os.Getenv("CLAUDE_API_KEY"),
		MaxTokens: 64000,
	})
	if err != nil {
		log.Fatal(err)
	}

	task := "do google search and find who is Elon Musk's wife"
	ag := agent.NewAgent(task, model)
	historyResult, err := ag.Run()

	if err != nil {
		log.Fatal(err.Error())
	}
	log.Infof("agent output: %s", *historyResult.LastResult().ExtractedContent)
}
