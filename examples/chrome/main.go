package main

import (
	"context"
	"os"

	"github.com/charmbracelet/log"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/nerdface-ai/browser-use-go/pkg/agent"
	"github.com/nerdface-ai/browser-use-go/pkg/browser"
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
	task := "do google search and find who is Elon Musk's wife"
	ag := agent.NewAgent(task, model, agent.WithBrowserConfig(browser.BrowserConfig{
		"headless":            false,
		"browser_binary_path": "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
	}))
	historyResult, err := ag.Run()

	if err != nil {
		log.Fatal(err.Error())
	}
	log.Infof("agent output: %s", *historyResult.LastResult().ExtractedContent)
}
