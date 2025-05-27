package main

import (
	"context"
	"os"

	"github.com/charmbracelet/log"
	"github.com/cloudwego/eino-ext/components/model/openai"
	openaiLib "github.com/cloudwego/eino-ext/libs/acl/openai"
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

	validateModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:  "gpt-4.1-mini",
		APIKey: os.Getenv("OPENAI_API_KEY"),
		ResponseFormat: &openaiLib.ChatCompletionResponseFormat{
			Type: openaiLib.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openaiLib.ChatCompletionResponseFormatJSONSchema{
				Name:        "ValidationOutput",
				Description: "Validation output",
				Schema:      agent.ValidationOutputSchema(),
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	task := "do google search and find who is Elon Musk's wife"
	ag := agent.NewAgent(task, model, agent.WithValidateLLM(validateModel))
	historyResult, err := ag.Run()

	if err != nil {
		log.Fatal(err.Error())
	}
	log.Infof("agent output: %s", *historyResult.LastResult().ExtractedContent)
}
