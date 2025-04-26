package agent

import (
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

type Agent struct {
	task string
	llm  llms.LLM
}

func NewAgent(task string) *Agent {
	return &Agent{
		task: task,
		llm:  openai.NewOpenAI(openai.WithModel("gpt-4.1-mini")),
}
