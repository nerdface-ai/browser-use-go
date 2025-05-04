package agent

import "nerdface-ai/browser-use-go/browser-use/controller"

type AgentBrain struct {
	EvaluationPreviousGoal string `json:"evaluation_previous_goal"`
	Memory                 string `json:"memory"`
	NextGoal               string `json:"next_goal"`
}

type AgentOutput struct {
	CurrentState AgentBrain               `json:"current_state"`
	Action       []controller.ActionModel `json:"action"`
}

func (a *AgentOutput) ToString() string {
	return ""
}

// static
// func TypeWithCustomActions(custom_actions []controller.ActionModel) AgentOutput {

// }
