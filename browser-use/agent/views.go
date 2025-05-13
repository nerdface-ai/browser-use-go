package agent

import (
	"context"
	"encoding/json"
	"nerdface-ai/browser-use-go/browser-use/browser"
	"nerdface-ai/browser-use-go/browser-use/controller"
	"nerdface-ai/browser-use-go/browser-use/dom"
	"nerdface-ai/browser-use-go/browser-use/utils"

	"github.com/charmbracelet/log"
	"github.com/cloudwego/eino/components/model"
	einoUtils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/uuid"
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
	UseVision             bool                       `json:"use_vision"`
	UseVisionForPlanner   bool                       `json:"use_vision_for_planner"`
	SaveConversationPath  *string                    `json:"save_conversation_path,omitempty"`
	MaxFailures           int                        `json:"max_failures"`
	RetryDelay            int                        `json:"retry_delay"`
	MaxInputTokens        int                        `json:"max_input_tokens"`
	ValidateOutput        bool                       `json:"validate_output"`
	MessageContext        *string                    `json:"message_context,omitempty"`
	GenerateGif           bool                       `json:"generate_gif"`
	AvailableFilePaths    []string                   `json:"available_file_paths"`
	OverrideSystemMessage *string                    `json:"override_system_message,omitempty"`
	ExtendSystemMessage   *string                    `json:"extend_system_message,omitempty"`
	IncludeAttributes     []string                   `json:"include_attributes"`
	MaxActionsPerStep     int                        `json:"max_actions_per_step"`
	ToolCallingMethod     *ToolCallingMethod         `json:"tool_calling_method,omitempty"`
	PageExtractionLLM     model.ToolCallingChatModel `json:"page_extraction_llm"`
	PlannerLLM            model.ToolCallingChatModel `json:"planner_llm"`
	PlannerInterval       int                        `json:"planner_interval"`
	IsPlannerReasoning    bool                       `json:"is_planner_reasoning"`

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
		SaveConversationPath:  utils.GetDefaultValue[*string](config, "save_conversation_path", nil),
		MaxFailures:           utils.GetDefaultValue[int](config, "max_failures", 3),
		RetryDelay:            utils.GetDefaultValue[int](config, "retry_delay", 10),
		MaxInputTokens:        utils.GetDefaultValue[int](config, "max_input_tokens", 128000),
		ValidateOutput:        utils.GetDefaultValue[bool](config, "validate_output", false),
		MessageContext:        utils.GetDefaultValue[*string](config, "message_context", nil),
		GenerateGif:           utils.GetDefaultValue[bool](config, "generate_gif", false),
		AvailableFilePaths:    utils.GetDefaultValue[[]string](config, "available_file_paths", nil),
		OverrideSystemMessage: utils.GetDefaultValue[*string](config, "override_system_message", nil),
		ExtendSystemMessage:   utils.GetDefaultValue[*string](config, "extend_system_message", nil),
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
		ToolCallingMethod:  utils.GetDefaultValue[*ToolCallingMethod](config, "tool_calling_method", nil),
		PageExtractionLLM:  utils.GetDefaultValue[model.ToolCallingChatModel](config, "page_extraction_llm", nil),
		PlannerLLM:         utils.GetDefaultValue[model.ToolCallingChatModel](config, "planner_llm", nil),
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
	LastPlan            *string                    `json:"last_plan,omitempty"`
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
	CurrentState *AgentBrain            `json:"current_state"`
	Action       []*controller.ActModel `json:"action" jsonschema:"minItems=1"` // List of actions to execute
}

func (ao *AgentOutput) ToString() string {
	b, _ := json.Marshal(ao)
	return string(b)
}

func ToolInfoWithCustomActions(customActions *controller.ActionModel) *schema.ToolInfo {
	actionSchemas := map[string]*openapi3.SchemaRef{}
	ctx := context.Background()
	for _, action := range customActions.Actions {
		actionTool := *action.Tool
		actionInfo, err := actionTool.Info(ctx)
		if err != nil {
			log.Printf("Failed to get action info: %v", err)
			continue
		}
		actionSchema, err := actionInfo.ToOpenAPIV3()
		actionSchema.Title = actionInfo.Name
		actionSchema.Description = actionInfo.Desc
		if err != nil {
			log.Printf("Failed to get action schema: %v", err)
			continue
		}
		actionSchemas[actionInfo.Name] = &openapi3.SchemaRef{
			Value: actionSchema,
		}
	}
	agentBrain, err := einoUtils.GoStruct2ParamsOneOf[AgentBrain]()
	if err != nil {
		log.Printf("Failed to get agent brain schema: %v", err)
		return nil
	}
	agentBrainSchema, err := agentBrain.ToOpenAPIV3()
	if err != nil {
		log.Printf("Failed to get agent brain schema: %v", err)
		return nil
	}
	agentBrainSchema.Description = "Current state of the agent"

	// Extend actions with custom actions
	return &schema.ToolInfo{
		Name: "AgentOutput",
		Desc: "AgentOutput model with custom actions",
		ParamsOneOf: schema.NewParamsOneOfByOpenAPIV3(&openapi3.Schema{
			Type: openapi3.TypeObject,
			Properties: map[string]*openapi3.SchemaRef{
				"action": {
					Value: &openapi3.Schema{
						Description: "List of actions to execute",
						Type:        openapi3.TypeArray,
						Items: &openapi3.SchemaRef{
							Value: &openapi3.Schema{
								Properties: actionSchemas,
							},
						},
						MinItems: 1,
					},
				},
				"current_state": {
					Value: agentBrainSchema,
				},
			},
			Required: []string{"action", "current_state"},
		}),
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
			el := (*selectorMap)[*index]
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

	// TODO(HIGH): implement model dump
	return ""
}

type AgentHistoryList struct {
	History []*AgentHistory `json:"history"`
}

func (ahl *AgentHistoryList) IsDone() bool {
	if len(ahl.History) > 0 && len(ahl.History[len(ahl.History)-1].Result) > 0 {
		lastResult := ahl.History[len(ahl.History)-1].Result[len(ahl.History[len(ahl.History)-1].Result)-1]
		if lastResult.IsDone == nil {
			return false
		} else {
			return *lastResult.IsDone
		}
	}
	return false
}

func (ahl *AgentHistoryList) IsSuccessful() *bool {
	if len(ahl.History) > 0 && len(ahl.History[len(ahl.History)-1].Result) > 0 {
		lastResult := ahl.History[len(ahl.History)-1].Result[len(ahl.History[len(ahl.History)-1].Result)-1]
		if lastResult.IsDone != nil && *lastResult.IsDone == true {
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
