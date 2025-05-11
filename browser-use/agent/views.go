package agent

import (
	"encoding/json"
	"nerdface-ai/browser-use-go/browser-use/browser"
	"nerdface-ai/browser-use-go/browser-use/controller"
	"nerdface-ai/browser-use-go/browser-use/dom"
	"nerdface-ai/browser-use-go/browser-use/utils"

	"github.com/google/uuid"
	"github.com/moznion/go-optional"
	"github.com/tmc/langchaingo/llms"
)

type ToolCallingMethod string

const (
	FunctionCalling ToolCallingMethod = "function_calling"
	JSONMode        ToolCallingMethod = "json_mode"
	Raw             ToolCallingMethod = "raw"
	Auto            ToolCallingMethod = "auto"
)

// 2. REQUIRED_LLM_API_ENV_VARS: map[string][]string 으로 선언
var REQUIRED_LLM_API_ENV_VARS = map[string][]string{"ChatOpenAI": {"OPENAI_API_KEY"}, "AzureOpenAI": {"AZURE_ENDPOINT", "AZURE_OPENAI_API_KEY"}, "ChatBedrockConverse": {"ANTHROPIC_API_KEY"}, "ChatAnthropic": {"ANTHROPIC_API_KEY"}, "ChatGoogleGenerativeAI": {"GEMINI_API_KEY"}, "ChatDeepSeek": {"DEEPSEEK_API_KEY"}, "ChatOllama": {}, "ChatGrok": {"GROK_API_KEY"}}

// Options for the agent
type AgentSettings struct {
	UseVision             bool                               `json:"use_vision"`
	UseVisionForPlanner   bool                               `json:"use_vision_for_planner"`
	SaveConversationPath  optional.Option[string]            `json:"save_conversation_path"`
	MaxFailures           int                                `json:"max_failures"`
	RetryDelay            int                                `json:"retry_delay"`
	MaxInputTokens        int                                `json:"max_input_tokens"`
	ValidateOutput        bool                               `json:"validate_output"`
	MessageContext        optional.Option[string]            `json:"message_context"`
	GenerateGif           bool                               `json:"generate_gif"`
	AvailableFilePaths    []string                           `json:"available_file_paths"`
	OverrideSystemMessage optional.Option[string]            `json:"override_system_message"`
	ExtendSystemMessage   optional.Option[string]            `json:"extend_system_message"`
	IncludeAttributes     []string                           `json:"include_attributes"`
	MaxActionsPerStep     int                                `json:"max_actions_per_step"`
	ToolCallingMethod     optional.Option[ToolCallingMethod] `json:"tool_calling_method"`
	PageExtractionLLM     llms.Model                         `json:"page_extraction_llm"`
	PlannerLLM            llms.Model                         `json:"planner_llm"`
	PlannerInterval       int                                `json:"planner_interval"`
	IsPlannerReasoning    bool                               `json:"is_planner_reasoning"`

	// Procedural memory settings
	EnableMemory   bool                   `json:"enable_memory"`
	MemoryInterval int                    `json:"memory_interval"`
	MemoryConfig   map[string]interface{} `json:"memory_config"`
}

type AgentSettingsConfig map[string]interface{}

func NewAgentSettings(config AgentSettingsConfig) *AgentSettings {
	return &AgentSettings{
		UseVision:             utils.GetDefaultValue[bool](config, "use_vision", true),
		UseVisionForPlanner:   utils.GetDefaultValue[bool](config, "use_vision_for_planner", false),
		SaveConversationPath:  utils.GetDefaultValue[optional.Option[string]](config, "save_conversation_path", nil),
		MaxFailures:           utils.GetDefaultValue[int](config, "max_failures", 3),
		RetryDelay:            utils.GetDefaultValue[int](config, "retry_delay", 10),
		MaxInputTokens:        utils.GetDefaultValue[int](config, "max_input_tokens", 128000),
		ValidateOutput:        utils.GetDefaultValue[bool](config, "validate_output", false),
		MessageContext:        utils.GetDefaultValue[optional.Option[string]](config, "message_context", nil),
		GenerateGif:           utils.GetDefaultValue[bool](config, "generate_gif", false),
		AvailableFilePaths:    utils.GetDefaultValue[[]string](config, "available_file_paths", nil),
		OverrideSystemMessage: utils.GetDefaultValue[optional.Option[string]](config, "override_system_message", nil),
		ExtendSystemMessage:   utils.GetDefaultValue[optional.Option[string]](config, "extend_system_message", nil),
		IncludeAttributes: utils.GetDefaultValue[[]string](config, "include_attributes", []string{
			"title",
			"type",
			"name",
			"role",
			"tabindex",
			"aria-label",
			"placeholder",
			"value",
			"alt",
			"aria-expanded",
		}),
		MaxActionsPerStep:  utils.GetDefaultValue[int](config, "max_actions_per_step", 10),
		ToolCallingMethod:  utils.GetDefaultValue[optional.Option[ToolCallingMethod]](config, "tool_calling_method", nil),
		PageExtractionLLM:  utils.GetDefaultValue[llms.Model](config, "page_extraction_llm", nil),
		PlannerLLM:         utils.GetDefaultValue[llms.Model](config, "planner_llm", nil),
		PlannerInterval:    utils.GetDefaultValue[int](config, "planner_interval", 1),
		IsPlannerReasoning: utils.GetDefaultValue[bool](config, "is_planner_reasoning", false),
		EnableMemory:       utils.GetDefaultValue[bool](config, "enable_memory", true),
		MemoryInterval:     utils.GetDefaultValue[int](config, "memory_interval", 10),
		MemoryConfig:       utils.GetDefaultValue[map[string]interface{}](config, "memory_config", nil),
	}
}

// Holds all state information for an Agent
type AgentState struct {
	AgentId             string                     `json:"agent_id"`
	NSteps              int                        `json:"n_steps"`
	ConsecutiveFailures int                        `json:"consecutive_failures"`
	LastResult          []*controller.ActionResult `json:"last_result"`
	History             *AgentHistoryList          `json:"history"`
	LastPlan            optional.Option[string]    `json:"last_plan"`
	Paused              bool                       `json:"paused"`
	Stopped             bool                       `json:"stopped"`
	MessageManagerState *MessageManagerState       `json:"message_manager_state"`
}

func NewAgentState() *AgentState {
	return &AgentState{
		AgentId:             uuid.New().String(),
		NSteps:              1,
		ConsecutiveFailures: 0,
		LastResult:          nil,
		History:             &AgentHistoryList{History: []*AgentHistory{}},
		LastPlan:            nil,
		Paused:              false,
		Stopped:             false,
		MessageManagerState: NewMessageManagerState(),
	}
}

// Current state of the agent
type AgentBrain struct {
	EvaluationPreviousGoal string `json:"evaluation_previous_goal"`
	Memory                 string `json:"memory"`
	NextGoal               string `json:"next_goal"`
}

// Output model for agent
// @dev note: this model is extended with custom actions in AgentService.
// You can also use some fields that are not in this model as provided by the linter, as long as they are registered in the DynamicActions model.
type AgentOutput struct {
	CurrentState *AgentBrain               `json:"current_state"`
	Action       []*controller.ActionModel `json:"action" jsonschema:"minItems=1"` // List of actions to execute
}

func (ao *AgentOutput) ToString() string {
	b, _ := json.Marshal(ao)
	return string(b)
}

func TypeWithCustomActions(customActions *controller.ActionModel) *AgentOutput {
	// Extend actions with custom actions
	return &AgentOutput{
		CurrentState: nil,
		Action:       []*controller.ActionModel{customActions},
	}
}

// Metadata for a single step including timing and token information
type StepMetadata struct {
	StepStartTime float64
	StepEndTime   float64
	InputTokens   int
	StepNumber    int
}

// Calculate step duration in seconds
func (sm *StepMetadata) DurationSeconds() float64 {
	return sm.StepEndTime - sm.StepStartTime
}

// History item for agent actions
type AgentHistory struct {
	ModelOutput *AgentOutput                 `json:"model_output"`
	Result      []*controller.ActionResult   `json:"result"`
	State       *browser.BrowserStateHistory `json:"state"`
	Metadata    *StepMetadata                `json:"metadata"`
}

func GetInteractedElement(modelOutput *AgentOutput, selectorMap *dom.SelectorMap) []*dom.DOMHistoryElement {
	elements := []*dom.DOMHistoryElement{}
	for _, action := range modelOutput.Action {
		index := action.GetIndex()
		if index != nil {
			el := (*selectorMap)[index.Unwrap()]
			if el != nil {
				elements = append(elements, dom.HistoryTreeProcessor{}.ConvertDomElementToHistoryElement(el))
			}
		} else {
			elements = append(elements, nil)
		}
	}
	return elements
}

func (ah *AgentHistory) ModelDump() string {
	// Custom serialization handling circular references

	// TODO
	return ""
}

type AgentHistoryList struct {
	History []*AgentHistory `json:"history"`
}

func (ahl *AgentHistoryList) IsDone() bool {
	if len(ahl.History) > 0 && len(ahl.History[len(ahl.History)-1].Result) > 0 {
		lastResult := ahl.History[len(ahl.History)-1].Result[len(ahl.History[len(ahl.History)-1].Result)-1]
		return lastResult.IsDone.Unwrap()
	}
	return false
}

func (ahl *AgentHistoryList) IsSuccessful() optional.Option[bool] {
	if len(ahl.History) > 0 && len(ahl.History[len(ahl.History)-1].Result) > 0 {
		lastResult := ahl.History[len(ahl.History)-1].Result[len(ahl.History[len(ahl.History)-1].Result)-1]
		if lastResult.IsDone.Unwrap() {
			return lastResult.Success
		}
	}
	return nil
}

func (ahl *AgentHistoryList) TotalInputTokens() int {
	totalTokens := 0
	for _, history := range ahl.History {
		if history.Metadata != nil {
			totalTokens += history.Metadata.InputTokens
		}
	}
	return totalTokens
}

type AgentStepInfo struct {
	StepNumber int
	MaxSteps   int
}

func (asi *AgentStepInfo) IsLastStep() bool {
	return asi.StepNumber >= asi.MaxSteps-1
}
