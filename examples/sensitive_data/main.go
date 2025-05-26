package main

import (
	"context"
	"os"

	"github.com/charmbracelet/log"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/nerdface-ai/browser-use-go/pkg/agent"
	"github.com/nerdface-ai/browser-use-go/pkg/dotenv"
)

func main() {
	log.SetLevel(log.DebugLevel)
	dotenv.LoadEnv(".env")

	log.Debug(os.Getenv("OPENAI_API_KEY"))

	ctx := context.Background()
	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:  "gpt-4.1-mini",
		APIKey: os.Getenv("OPENAI_API_KEY"),
	})
	if err != nil {
		log.Fatal(err)
	}

	task := "go to x.com login page and insert x_name and x_password."
	ag := agent.NewAgent(task, model, agent.WithSensitiveData(map[string]string{
		"x_name":     "currybab_",
		"x_password": "testtest",
	}))
	historyResult, err := ag.Run()

	if err != nil {
		log.Fatal(err.Error())
	}
	log.Infof("agent output: %s", *historyResult.LastResult().ExtractedContent)
}
