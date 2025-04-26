package agent

import (
	"log"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

type Agent struct {
	task string
	llm  llms.LLM
}

func NewAgent(task string) *Agent {
	llm, err := openai.New(openai.WithModel("gpt-4.1-mini"))
	if err != nil {
		log.Fatal(err)
	}
	return &Agent{
		task: task,
		llm:  llm,
	}
}
