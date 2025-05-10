package agent

import (
	"errors"
	"fmt"
	"log"
	"nerdface-ai/browser-use-go/browser-use/browser"
	"nerdface-ai/browser-use-go/browser-use/controller"
	"reflect"
	"strings"
	"time"

	"github.com/moznion/go-optional"
	"github.com/playwright-community/playwright-go"
	"github.com/tmc/langchaingo/llms"
)

type Agent struct {
	Task                   string
	LLM                    llms.Model
	Controller             *controller.Controller
	SensitiveData          map[string]string
	Settings               *AgentSettings
	State                  *AgentState
	InjectedBrowser        bool
	InjectedBrowserContext bool
	Browser                *browser.Browser
	BrowserContext         *browser.BrowserContext

	// model
	ChatModelLibrary        string
	ModelName               string // e.g., openai, googleai, anthropic, huggingface
	PlannerModelName        string
	PageExtractionModelName string

	RegisterNewStepCallback                       func(state *browser.BrowserState, output *AgentOutput, n int)
	RegisterDoneCallback                          func(history *AgentHistoryList)
	RegisterExternalAgentStatusRaiseErrorCallback func() bool

	ToolCallingMethod optional.Option[ToolCallingMethod]

	ActionModel     *controller.ActionModel
	AgentOutput     *AgentOutput
	DoneActionModel *controller.ActionModel
	DoneAgentOutput *AgentOutput

	MessageManager *MessageManager

	UnfilteredActions string
	InitialActions    []*controller.ActionModel
}

/*
if you want to specify config, fill in field AgentSettings to NewAgent
To provide custom configuration, pass an AgentSettings instance to NewAgent fuction.
e.g.,

	NewAgentSettings(AgentSettingsConfig{
		"use_vision": true,
		"use_vision_for_planner": true,
		"save_conversation_path": "./conversation.json",
		...
	})
*/
func NewAgent(
	task string,
	llm llms.Model,
	// AgentSettings
	settings *AgentSettings,

	// Optional parameters
	browserInst *browser.Browser,
	browserContext *browser.BrowserContext,
	controller *controller.Controller,

	// Initial agent run parameters
	sensitiveData map[string]string,
	initialActions []interface{},

	// Cloud Callbacks
	registerNewStepCallback func(state *browser.BrowserState, output *AgentOutput, n int),
	registerDoneCallback func(history *AgentHistoryList),
	registerExternalAgentStatusRaiseErrorCallback func() bool,

	// Inject sate
	injectedAgentState *AgentState,

	// Memory settings
) *Agent {
	if settings.PageExtractionLLM == nil {
		settings.PageExtractionLLM = llm
	}

	// Core components
	agent := &Agent{
		Task:          task,
		LLM:           llm,
		Controller:    controller,
		SensitiveData: sensitiveData,
	}

	agent.Settings = settings

	// Initial state
	state := injectedAgentState
	if state == nil {
		state = NewAgentState()
	}
	agent.State = state

	// Action setup
	agent.setupActionModels()
	// TODO
	// self._set_browser_use_version_and_source()
	agent.InitialActions = agent.convertInitialActions(initialActions)

	// Model setup
	agent.setModelNames()
	agent.ToolCallingMethod = agent.setToolCallingMethod()

	// Handle users trying to use use_vision=True with DeepSeek models

	agent.logAgentInfo()

	// Initialize available actions for system prompt (only non-filtered actions)
	// These will be used for the system prompt to maintain caching
	agent.UnfilteredActions = agent.Controller.Registry.GetPromptDescription(nil)
	log.Printf("[DEBUG] Agent.__init__ - Unfiltered actions: %s", agent.UnfilteredActions)

	agent.Settings.MessageContext = agent.setMessageContext()

	// Initialize message manager with state
	// Initial system prompt with all actions - will be updated during each step
	systemPrompt := NewSystemPrompt(
		agent.UnfilteredActions,
		agent.Settings.MaxActionsPerStep,
		agent.Settings.OverrideSystemMessage,
		agent.Settings.ExtendSystemMessage,
	)

	agent.MessageManager = NewMessageManager(
		task,
		systemPrompt.SystemMessage,
		NewMessageManagerSettings(MessageManagerConfig{
			"max_input_tokens":     agent.Settings.MaxInputTokens,
			"include_attributes":   agent.Settings.IncludeAttributes,
			"message_context":      agent.Settings.MessageContext,
			"sensitive_data":       agent.SensitiveData,
			"available_file_paths": agent.Settings.AvailableFilePaths,
		}),
		agent.State.MessageManagerState,
	)

	// Browser setup
	agent.InjectedBrowser = browserInst != nil
	agent.InjectedBrowserContext = browserContext != nil
	if browserInst == nil {
		browserInst = browser.NewBrowser(browser.BrowserConfig{})
	}
	agent.Browser = browserInst
	if browserContext == nil {
		browserContext = browserInst.NewContext()
	}
	agent.BrowserContext = browserContext

	// Callbacks
	agent.RegisterNewStepCallback = registerNewStepCallback
	agent.RegisterDoneCallback = registerDoneCallback
	agent.RegisterExternalAgentStatusRaiseErrorCallback = registerExternalAgentStatusRaiseErrorCallback

	return agent
}

func (ag *Agent) convertInitialActions(actions []interface{}) []*controller.ActionModel {
	return []*controller.ActionModel{}
}

func (ag *Agent) setMessageContext() optional.Option[string] {
	if ag.ToolCallingMethod.Unwrap() == "raw" {
		// For raw tool calling, only include actions with no filters initially
		messageContext := ag.Settings.MessageContext.Unwrap()
		if messageContext != "" {
			messageContext += fmt.Sprintf("\n\nAvailable actions: %s", ag.UnfilteredActions)
		} else {
			messageContext = fmt.Sprintf("Available actions: %s", ag.UnfilteredActions)
		}
		ag.Settings.MessageContext = optional.Some(messageContext)
	}
	return ag.Settings.MessageContext
}

func (ag *Agent) logAgentInfo() {
	log.Printf("🧠 Starting an agent with main_model=%s", ag.ModelName)
	if ag.ToolCallingMethod.Unwrap() == "function_calling" {
		log.Printf(" +tools")
	}
	if ag.ToolCallingMethod.Unwrap() == "raw" {
		log.Printf(" +rawtools")
	}
	if ag.Settings.UseVision {
		log.Printf(" +vision")
	}
	if ag.Settings.EnableMemory {
		log.Printf(" +memory")
	}
	log.Printf("planner_model=%s", ag.PlannerModelName)
	if ag.Settings.IsPlannerReasoning {
		log.Printf(" +reasoning")
	}
	if ag.Settings.UseVisionForPlanner {
		log.Printf(" +vision")
	}
	log.Printf("extraction_model=%s", ag.PageExtractionModelName)
}

func (ag *Agent) setModelNames() {
	ag.ChatModelLibrary = reflect.TypeOf(ag.LLM).Elem().Name()

	// LangchainGo does not support model name method
	typePkg := reflect.TypeOf(ag.LLM).Elem().PkgPath()
	pkgName := strings.Split(typePkg, "/")[len(strings.Split(typePkg, "/"))-1]
	ag.ModelName = pkgName

	if ag.Settings.PlannerLLM != nil {
		typePkg = reflect.TypeOf(ag.Settings.PlannerLLM).Elem().PkgPath()
		pkgName = strings.Split(typePkg, "/")[len(strings.Split(typePkg, "/"))-1]
		ag.PlannerModelName = pkgName
	}

	if ag.Settings.PageExtractionLLM != nil {
		typePkg = reflect.TypeOf(ag.Settings.PageExtractionLLM).Elem().PkgPath()
		pkgName = strings.Split(typePkg, "/")[len(strings.Split(typePkg, "/"))-1]
		ag.PageExtractionModelName = pkgName
	}
}

func (ag *Agent) setToolCallingMethod() optional.Option[ToolCallingMethod] {
	toolCallingMethod := ag.Settings.ToolCallingMethod
	if toolCallingMethod.Unwrap() == "auto" {
		switch {
		case strings.Contains(ag.ModelName, "openai") ||
			strings.Contains(ag.ModelName, "googleai") ||
			strings.Contains(ag.ModelName, "anthropic"):
			return optional.Some(FunctionCalling)
		default:
			return nil
		}
	} else {
		return toolCallingMethod
	}
}

func (ag *Agent) setupActionModels() {
	// Setup dynamic action models from controller's registry

	// Initially only include actions with no filters
	ag.ActionModel = ag.Controller.Registry.CreateActionModel(nil, nil)
	// Create output model with the dynamic actions
	ag.AgentOutput = AgentOutput{}.TypeWithCustomActions(ag.ActionModel)

	// used to force the done action when max steps is reached
	ag.DoneActionModel = ag.Controller.Registry.CreateActionModel([]string{"done"}, nil)
	ag.DoneAgentOutput = AgentOutput{}.TypeWithCustomActions(ag.DoneActionModel)
}

func (ag *Agent) Step(stepInfo *AgentStepInfo) error {
	// Execute one step of the task
	log.Printf("📍 Step %d\n", ag.State.NSteps)
	stepStartTime := time.Now().UnixNano()

	browserState := ag.BrowserContext.GetState(true)
	activePage := ag.BrowserContext.GetCurrentPage()

	// TODO:
	// generate procedural memory if needed
	// if self.settings.enable_memory and self.memory and self.state.n_steps % self.settings.memory_interval == 0:
	// 	self.memory.create_procedural_memory(self.state.n_steps)

	ag.raiseIfStoppedOrPaused()

	// Update action models with page-specific actions
	ag.updateActionModelsForPage(activePage)

	// Get page-specific filtered actions
	pageFilteredActions := ag.Controller.Registry.GetPromptDescription(activePage)

	// If there are page-specific actions, add them as a special message for this step only
	if pageFilteredActions != "" {
		pageActionMessage := fmt.Sprintf("For this page, these additional actions are available:\n%s", pageFilteredActions)
		ag.MessageManager.addMessageWithTokens(llms.HumanChatMessage{
			Content: pageActionMessage,
		}, nil, nil)
	}

	// If using raw tool calling method, we need to update the message context with new actions
	if ag.ToolCallingMethod.Unwrap() == "raw" {
		// For raw tool calling, get all non-filtered actions plus the page-filtered ones
		allActions := ag.Controller.Registry.GetPromptDescription(nil)
		if pageFilteredActions != "" {
			allActions += "\n" + pageFilteredActions
		}

		contextLines := strings.Split(ag.MessageManager.Settings.MessageContext.Unwrap(), "\n")
		var nonActionLines []string
		for _, line := range contextLines {
			if !strings.Contains(line, "Available actions:") {
				nonActionLines = append(nonActionLines, line)
			}
		}
		updatedContext := strings.Join(nonActionLines, "\n")
		if updatedContext != "" {
			updatedContext += "\n\nAvailable actions: " + allActions
		} else {
			updatedContext = "Available actions: " + allActions
		}
		ag.MessageManager.Settings.MessageContext = optional.Some(updatedContext)
	}

	ag.MessageManager.AddStateMessage(browserState, []*controller.ActionResult{}, stepInfo, ag.Settings.UseVision)

	// TODO: Run planner at specified intervals if planner is configured

	if stepInfo != nil && stepInfo.IsLastStep() {
		// Add last step warning if needed
		msg := "Now comes your last step. Use only the \"done\" action now. No other actions - so here your action sequence must have length 1."
		msg += "\nIf the task is not yet fully finished as requested by the user, set success in \"done\" to false! E.g. if not all steps are fully completed."
		msg += "\nIf the task is fully finished, set success in \"done\" to true."
		msg += "\nInclude everything you found out for the ultimate task in the done text."
		log.Println("Last step finishing up")
		ag.MessageManager.addMessageWithTokens(llms.HumanChatMessage{
			Content: msg,
		}, nil, nil)
		ag.AgentOutput = ag.DoneAgentOutput
	}

	inputMessages := ag.MessageManager.GetMessages()
	tokens := ag.MessageManager.State.History.CurrentTokens

	modelOutput, err := ag.GetNextAction(inputMessages)
	if err != nil {
		return errors.New("Failed to get next action")
	}

	// Check again for paused/stopped state after getting model output
	// This is needed in case Ctrl+C was pressed during the get_next_action call
	if ag.raiseIfStoppedOrPaused() != nil {
		return errors.New("interrupted")
	}

	ag.State.NSteps++

	if ag.RegisterNewStepCallback != nil {
		ag.RegisterNewStepCallback(browserState, modelOutput, ag.State.NSteps)
	}
	if ag.Settings.SaveConversationPath.Unwrap() != "" {
		target := ag.Settings.SaveConversationPath.Unwrap() + fmt.Sprintf("_%d.txt", ag.State.NSteps)
		ag.MessageManager.SaveConversation(inputMessages, modelOutput, target)
	}

	// @@@
	ag.MessageManager.RemoveLastStateMessage() // we dont want the whole state in the chat history

	// check again if Ctrl+C was pressed before we commit the output to history
	if ag.raiseIfStoppedOrPaused() != nil {
		return errors.New("interrupted")
	}

	// @@@
	ag.MessageManager.AddModelOutput(modelOutput)

	// @@@
	result := ag.MultiAct(modelOutput.Action)
	ag.State.LastResult = result

	if len(result) > 0 {
		result := result[len(result)-1]
		if result.IsDone.Unwrap() {
			log.Printf("📄 Result: %s", result.ExtractedContent.Unwrap())
		}
	}

	ag.State.ConsecutiveFailures = 0

	// @@@
	// TODO: finally part
	if browserState != nil {
		metaData := &StepMetadata{
			StepNumber:    ag.State.NSteps,
			StepStartTime: float64(stepStartTime),
			StepEndTime:   float64(time.Now().UnixNano()),
			InputTokens:   tokens,
		}
		ag.makeHistoryItem(modelOutput, browserState, result, metaData)
	}

	return nil
}

func (ag *Agent) GetNextAction(inputMessages []llms.ChatMessage) (*AgentOutput, error) {
	// Get next action from LLM based on current state

	// @@@
	return nil, nil
}

func (ag *Agent) raiseIfStoppedOrPaused() error {
	if ag.RegisterExternalAgentStatusRaiseErrorCallback != nil {
		if ag.RegisterExternalAgentStatusRaiseErrorCallback() {
			return errors.New("interrupted")
		}
	}
	if ag.State.Stopped || ag.State.Paused {
		return errors.New("interrupted")
	}
	return nil
}

func (ag *Agent) updateActionModelsForPage(page playwright.Page) {
	// Update action models with page-specific actions

	// Create new action model with current page's filtered actions
	ag.ActionModel = ag.Controller.Registry.CreateActionModel(nil, page)
	// Update output model with the new actions
	ag.AgentOutput = AgentOutput{}.TypeWithCustomActions(ag.ActionModel)

	// Update done action model too
	ag.DoneActionModel = ag.Controller.Registry.CreateActionModel([]string{"done"}, page)
	ag.DoneAgentOutput = AgentOutput{}.TypeWithCustomActions(ag.DoneActionModel)
}

func (ag *Agent) MultiAct(actions []*controller.ActionModel) []*controller.ActionResult {
	return []*controller.ActionResult{}
}

func (ag *Agent) makeHistoryItem(
	modelOutput *AgentOutput,
	browserState *browser.BrowserState,
	result []*controller.ActionResult,
	metaData *StepMetadata,
) error {
	return nil
}
