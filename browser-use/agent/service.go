package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"nerdface-ai/browser-use-go/browser-use/browser"
	"nerdface-ai/browser-use-go/browser-use/controller"
	"nerdface-ai/browser-use-go/browser-use/dom"
	"reflect"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/playwright-community/playwright-go"
)

type Agent struct {
	Task                   string
	LLM                    model.ToolCallingChatModel
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

	ToolCallingMethod *ToolCallingMethod `json:"tool_calling_method,omitempty"`

	ActionModel     *controller.ActionModel
	AgentOutput     *schema.ToolInfo
	DoneActionModel *controller.ActionModel
	DoneAgentOutput *schema.ToolInfo

	MessageManager *MessageManager

	UnfilteredActions string
	InitialActions    []*controller.ActModel
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
// TODO(HIGH): set agent options gracefully
func NewAgent(
	task string,
	llm model.ToolCallingChatModel,
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
	// TODO(LOW): self._set_browser_use_version_and_source()
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

// TODO(HIGH): implement convertInitialActions
func (ag *Agent) convertInitialActions(actions []interface{}) []*controller.ActModel {
	return []*controller.ActModel{}
}

func (ag *Agent) setMessageContext() *string {
	if ag.ToolCallingMethod != nil && *ag.ToolCallingMethod == "raw" {
		// For raw tool calling, only include actions with no filters initially
		messageContext := ag.Settings.MessageContext
		if messageContext != nil && len(*messageContext) > 0 {
			*messageContext += fmt.Sprintf("\n\nAvailable actions: %s", ag.UnfilteredActions)
		} else {
			*messageContext = fmt.Sprintf("Available actions: %s", ag.UnfilteredActions)
		}
		ag.Settings.MessageContext = messageContext
	}
	return ag.Settings.MessageContext
}

func (ag *Agent) logAgentRun() {
	log.Printf("🚀 Starting task: %s", ag.Task)
	// log.Printf("Version: %s, Source: %s", ag.Version, ag.Source)
}

func (ag *Agent) logAgentInfo() {
	log.Printf("🧠 Starting an agent with main_model=%s", ag.ModelName)

	if ag.ToolCallingMethod != nil && *ag.ToolCallingMethod == "function_calling" {
		log.Printf(" +tools")
	}
	if ag.ToolCallingMethod != nil && *ag.ToolCallingMethod == "raw" {
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

	// TODO(MID): we removed langchaingo, check for eino
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

func (ag *Agent) setToolCallingMethod() *ToolCallingMethod {
	toolCallingMethod := ag.Settings.ToolCallingMethod
	if toolCallingMethod == nil {
		return nil
	}
	if *toolCallingMethod == Auto {
		switch {
		case strings.Contains(ag.ModelName, "openai") ||
			strings.Contains(ag.ModelName, "googleai") ||
			strings.Contains(ag.ModelName, "anthropic"):
			fc := FunctionCalling
			return &fc
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
	ag.AgentOutput = ToolInfoWithCustomActions(ag.ActionModel)

	// used to force the done action when max steps is reached
	ag.DoneActionModel = ag.Controller.Registry.CreateActionModel([]string{"Done"}, nil)
	ag.DoneAgentOutput = ToolInfoWithCustomActions(ag.DoneActionModel)
}

func (ag *Agent) Step(stepInfo *AgentStepInfo) error {
	// Execute one step of the task
	log.Printf("📍 Step %d\n", ag.State.NSteps)
	stepStartTime := time.Now().UnixNano()

	browserState := ag.BrowserContext.GetState(true)
	activePage := ag.BrowserContext.GetCurrentPage()

	// TODO(MID): generate procedural memory if needed
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
		ag.MessageManager.AddMessageWithTokens(&schema.Message{
			Role:    schema.User,
			Content: pageActionMessage,
		}, nil, nil)
	}

	// TODO(MID): should check after support deepseek model
	// If using raw tool calling method, we need to update the message context with new actions
	// if *ag.ToolCallingMethod == "raw" {
	// 	// For raw tool calling, get all non-filtered actions plus the page-filtered ones
	// 	allActions := ag.Controller.Registry.GetPromptDescription(nil)
	// 	if pageFilteredActions != "" {
	// 		allActions += "\n" + pageFilteredActions
	// 	}

	// 	contextLines := strings.Split(*ag.MessageManager.Settings.MessageContext, "\n")
	// 	var nonActionLines []string
	// 	for _, line := range contextLines {
	// 		if !strings.Contains(line, "Available actions:") {
	// 			nonActionLines = append(nonActionLines, line)
	// 		}
	// 	}
	// 	updatedContext := strings.Join(nonActionLines, "\n")
	// 	if updatedContext != "" {
	// 		updatedContext += "\n\nAvailable actions: " + allActions
	// 	} else {
	// 		updatedContext = "Available actions: " + allActions
	// 	}
	// 	ag.MessageManager.Settings.MessageContext = playwright.string(updatedContext)
	// }

	ag.MessageManager.AddStateMessage(browserState, ag.State.LastResult, stepInfo, ag.Settings.UseVision)

	// TODO(MID): support planner
	// Run planner at specified intervals if planner is configured
	// if self.settings.planner_llm and self.state.n_steps % self.settings.planner_interval == 0:
	// 	plan = await self._run_planner()
	// 	# add plan before last state message
	// 	self._message_manager.add_plan(plan, position=-1)

	if stepInfo != nil && stepInfo.IsLastStep() {
		// Add last step warning if needed
		msg := "Now comes your last step. Use only the \"done\" action now. No other actions - so here your action sequence must have length 1."
		msg += "\nIf the task is not yet fully finished as requested by the user, set success in \"done\" to false! E.g. if not all steps are fully completed."
		msg += "\nIf the task is fully finished, set success in \"done\" to true."
		msg += "\nInclude everything you found out for the ultimate task in the done text."
		log.Print("Last step finishing up")
		ag.MessageManager.AddMessageWithTokens(&schema.Message{
			Role:    schema.User,
			Content: msg,
		}, nil, nil)
		ag.AgentOutput = ag.DoneAgentOutput
	}

	inputMessages := ag.MessageManager.GetMessages()
	tokens := ag.MessageManager.State.History.CurrentTokens

	modelOutput, err := ag.GetNextAction(inputMessages)
	if err != nil {
		ag.MessageManager.RemoveLastStateMessage()
		return errors.New("failed to get next action")
	}

	// Check again for paused/stopped state after getting model output
	// This is needed in case Ctrl+C was pressed during the get_next_action call
	if err = ag.raiseIfStoppedOrPaused(); err != nil {
		return err
	}

	ag.State.NSteps++

	if ag.RegisterNewStepCallback != nil {
		ag.RegisterNewStepCallback(browserState, modelOutput, ag.State.NSteps)
	}
	if ag.Settings.SaveConversationPath != nil {
		target := *ag.Settings.SaveConversationPath + fmt.Sprintf("_%d.txt", ag.State.NSteps)
		ag.MessageManager.SaveConversation(inputMessages, modelOutput, target)
	}

	ag.MessageManager.RemoveLastStateMessage() // we dont want the whole state in the chat history

	// check again if Ctrl+C was pressed before we commit the output to history
	if err = ag.raiseIfStoppedOrPaused(); err != nil {
		return err
	}

	ag.MessageManager.AddModelOutput(modelOutput)

	result, err := ag.MultiAct(modelOutput.Action, true)
	if err != nil {
		// TODO(MID): complement error handling
		errStr := err.Error()
		ag.State.LastResult = []*controller.ActionResult{
			{
				Error:           &errStr,
				IncludeInMemory: false,
			},
		}
		return err
	}

	ag.State.LastResult = result

	if len(result) > 0 {
		lastResult := result[len(result)-1]
		if lastResult.IsDone != nil && *lastResult.IsDone && lastResult.ExtractedContent != nil {
			log.Printf("📄 Result: %s", *lastResult.ExtractedContent)
		}
	}

	ag.State.ConsecutiveFailures = 0

	// @@@
	// TODO(MID): finally part
	if len(result) == 0 {
		return nil
	}

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

// TODO(MID): support deepseek
// Convert input messages to the correct format
// func (ag *Agent) convertInputMessages(inputMessages []*schema.Message) []*schema.Message {
// 	return inputMessages
// }

// Get next action from LLM based on current state
func (ag *Agent) GetNextAction(inputMessages []*schema.Message) (*AgentOutput, error) {
	// TODO(MID): support deepseek
	// TODO(MID): support other models like gemini, hugginface

	toolLLM, err := ag.LLM.WithTools([]*schema.ToolInfo{ag.AgentOutput})
	if err != nil {
		return nil, err
	}
	// log.Debug("Using %s for %s", *ag.ToolCallingMethod, ag.ChatModelLibrary)
	response, err := toolLLM.Generate(context.Background(), inputMessages)
	if err != nil {
		return nil, err
	}

	toolCalls := response.ToolCalls
	if len(toolCalls) == 0 {
		return nil, errors.New("no tool calls")
	}
	toolCall := toolCalls[0]

	var parsed AgentOutput
	toolCallName := toolCall.Function.Name
	if toolCallName == "" {
		return nil, errors.New("failed to get tool call name")
	}
	toolCallArgs := toolCall.Function.Arguments
	if toolCallArgs == "" {
		return nil, errors.New("failed to get tool call args")
	}
	log.Printf("Tool call args: %s\n", toolCallArgs)

	err = json.Unmarshal([]byte(toolCallArgs), &parsed)
	if err != nil {
		log.Debug("failed to unmarshal tool call args: %s", toolCallArgs)
		// currentState := map[string]interface{}{
		// 	"page_summary":             "Processing tool call",
		// 	"evaluation_previous_goal": "Executing action",
		// 	"memory":                   "Using tool call",
		// 	"next_goal":                fmt.Sprintf("Execute %s", toolCallName),
		// }

		// // Create action from tool call
		// action := map[string]interface{}{
		// 	toolCallName: toolCallArgs,
		// }
	}

	// TODO(MID): extract_json_from_model_output

	return &parsed, nil
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
	ag.AgentOutput = ToolInfoWithCustomActions(ag.ActionModel)

	// Update done action model too
	ag.DoneActionModel = ag.Controller.Registry.CreateActionModel([]string{"done"}, page)
	ag.DoneAgentOutput = ToolInfoWithCustomActions(ag.DoneActionModel)
}

func (ag *Agent) Run(maxSteps int, onStepStart func(*Agent), onStepEnd func(*Agent)) (*AgentHistoryList, error) {
	defer ag.Close()
	// TODO(LOW): implement signal handler (Set up the Ctrl+C signal handler with callbacks specific to this agent)
	// TODO(LOW): implement verification llm (Wait for verification task to complete if it exists)
	// TODO(LOW): implement generate gif

	ag.logAgentRun()

	// Execute initial actions if provided
	if len(ag.InitialActions) > 0 {
		result, err := ag.MultiAct(ag.InitialActions, false)
		if err != nil {
			return nil, err
		}
		ag.State.LastResult = result
	}

	stepCheck := 0
	for step := range maxSteps {
		if ag.State.Paused {
			// TODO(LOW): implement signal handler
			// signal_handler.wait_for_resume()
			// signal_handler.reset()
		}
		if ag.State.ConsecutiveFailures >= ag.Settings.MaxFailures {
			log.Fatalf("❌ Stopping due to %d consecutive failures", ag.Settings.MaxFailures)
			break
		}

		if ag.State.Stopped {
			log.Printf("Agent stopped")
			break
		}

		for ag.State.Paused {
			time.Sleep(200 * time.Millisecond)
			if ag.State.Stopped {
				break
			}
		}

		if onStepStart != nil {
			onStepStart(ag)
		}

		stepInfo := &AgentStepInfo{
			StepNumber: step,
			MaxSteps:   maxSteps,
		}
		ag.Step(stepInfo)

		if onStepEnd != nil {
			onStepEnd(ag)
		}

		if ag.State.History.IsDone() {
			if ag.Settings.ValidateOutput && step < maxSteps-1 {
				if !ag.validateOutput() {
					continue
				}
			}

			ag.logCompletion()
			break
		}
		stepCheck++
	}
	if stepCheck == maxSteps {
		log.Printf("❌ Failed to complete task in maximum steps")
	}

	return ag.State.History, nil
}

// Close all resources
func (ag *Agent) Close() {
	// First close browser resources
	var err error
	if ag.BrowserContext != nil && !ag.InjectedBrowserContext {
		ag.BrowserContext.Close()
	}
	if ag.Browser != nil && !ag.InjectedBrowser {
		err = ag.Browser.Close()
	}
	if err != nil {
		log.Fatalf("Error during cleanup: %s", err)
	}
}

// Execute multiple actions
func (ag *Agent) MultiAct(
	actions []*controller.ActModel,
	checkForNewElements bool,
) ([]*controller.ActionResult, error) {
	results := []*controller.ActionResult{}

	cachedSelectorMap := ag.BrowserContext.GetSelectorMap()
	cachedPathHashes := mapset.NewSet[string]()
	if cachedSelectorMap != nil {
		for _, e := range *cachedSelectorMap {
			cachedPathHashes.Add(e.Hash().BranchPathHash)
		}
	}

	ag.BrowserContext.RemoveHighlights()

	for i, action := range actions {
		if action.GetIndex() != nil && i != 0 {
			newState := ag.BrowserContext.GetState(false)
			newSelectorMap := newState.SelectorMap

			// Detect index change after previous action
			index := action.GetIndex()
			if index != nil {
				origTarget := (*cachedSelectorMap)[*index]
				var origTargetHash *string = nil
				if origTarget != nil {
					origTargetHash = playwright.String(origTarget.Hash().BranchPathHash)
				}
				newTarget := (*newSelectorMap)[*index]
				var newTargetHash *string = nil
				if newTarget != nil {
					newTargetHash = playwright.String(newTarget.Hash().BranchPathHash)
				}

				if origTargetHash == nil || newTargetHash == nil || *origTargetHash != *newTargetHash {
					msg := fmt.Sprintf("Element index changed after action %d / %d, because page changed.", i, len(actions))
					log.Print(msg)
					results = append(results, &controller.ActionResult{ExtractedContent: &msg, IncludeInMemory: true})
					break
				}

				newPathHashes := mapset.NewSet[string]()
				if newSelectorMap != nil {
					for _, e := range *newSelectorMap {
						newPathHashes.Add(e.Hash().BranchPathHash)
					}
				}

				if checkForNewElements && !newPathHashes.IsSubset(cachedPathHashes) {
					msg := fmt.Sprintf("Something new appeared after action %d / %d", i, len(actions))
					log.Print(msg)
					results = append(results, &controller.ActionResult{ExtractedContent: &msg, IncludeInMemory: true})
					break
				}
			}
		}

		ag.raiseIfStoppedOrPaused()
		result, err := ag.Controller.ExecuteAction(action, ag.BrowserContext, ag.Settings.PageExtractionLLM, ag.SensitiveData, ag.Settings.AvailableFilePaths)
		if err != nil {
			return nil, err
			// TODO(LOW): implement signal handler error
			// log.Printf("Action %d was cancelled due to Ctrl+C", i+1)
			// if len(results) > 0 {
			// 	results = append(results, &controller.ActionResult{Error: playwright.String("The action was cancelled due to Ctrl+C"), IncludeInMemory: true})
			// }
			// return nil, errors.New("Action cancelled by user")
		}
		results = append(results, result)
		log.Debug(fmt.Sprintf("Executed action %d / %d", i+1, len(actions)))
		lastIndex := len(results) - 1
		if (results[lastIndex].IsDone != nil && *results[lastIndex].IsDone) || results[lastIndex].Error != nil || i == len(actions)-1 {
			break
		}

		time.Sleep(500 * time.Millisecond) // ag.BrowserContext.Config.WaitBetweenActions
	}

	return results, nil
}

// Create and store history item
func (ag *Agent) makeHistoryItem(
	modelOutput *AgentOutput,
	browserState *browser.BrowserState,
	result []*controller.ActionResult,
	metaData *StepMetadata,
) {
	var interactedElements []*dom.DOMHistoryElement
	if modelOutput != nil {
		interactedElements = GetInteractedElement(modelOutput, browserState.SelectorMap)
	} else {
		interactedElements = []*dom.DOMHistoryElement{nil}
	}
	stateHistory := &browser.BrowserStateHistory{
		Url:               browserState.Url,
		Title:             browserState.Title,
		Tabs:              browserState.Tabs,
		InteractedElement: interactedElements,
	}

	historyItem := &AgentHistory{
		ModelOutput: modelOutput,
		Result:      result,
		State:       stateHistory,
		Metadata:    metaData,
	}

	ag.State.History.History = append(ag.State.History.History, historyItem)
}

//	type validationOutput struct {
//		IsValid bool
//		Reason  string
//	}
//
// Validate the output of the last action is what the user wanted
func (ag *Agent) validateOutput() bool {
	// TODO(MID): implement output validator
	return true
	// systemMsg := fmt.Sprintf(
	// 	"You are a validator of an agent who interacts with a browser." +
	// 	"Validate if the output of last action is what the user wanted and if the task is completed." +
	// 	"If the task is unclear defined, you can let it pass. But if something is missing or the image does not show what was requested dont let it pass." +
	// 	"Try to understand the page and help the model with suggestions like scroll, do x, ... to get the solution right." +
	// 	"Task to validate: %s. Return a JSON object with 2 keys: is_valid and reason." +
	// 	"is_valid is a boolean that indicates if the output is correct." +
	// 	"reason is a string that explains why it is valid or not." +
	// 	"reason is a string that explains why it is valid or not." +
	// 	" example: {{\"is_valid\": false, \"reason\": \"The user wanted to search for \"cat photos\", but the agent searched for \"dog photos\" instead.\"}}",
	// 	ag.Task)

	// if ag.BrowserContext.Session != nil {
	// 	state :=  ag.BrowserContext.GetState(false)
	// 	content := AgentMessagePrompt{
	// 		State: state,
	// 		Result: ag.State.LastResult,
	// 		IncludeAttributes: ag.Settings.IncludeAttributes,
	// 	}
	// 	msg := []*schema.Message{schema.Message{Role: schema.System, Content: systemMsg}, content.GetUserMessage(ag.Settings.UseVision)}
	// } else {
	// 	return true
	// }

	// validator := ag.LLM.GenerateContent(ValidationResult, true)
	// response := validator.ainvoke(msg)
	// parsed := response.Parsed()
	// is_valid := parsed.IsValid
	// if !is_valid {
	// 	log.Printf("❌ Validator decision: %s", parsed.Reason)
	// 	msg := fmt.Sprintf("The output is not yet correct. %s.", parsed.Reason)
	// 	ag.State.LastResult = []*controller.ActionResult{controller.ActionResult{ExtractedContent: &msg, IncludeInMemory: true}}
	// } else {
	// 	log.Printf("✅ Validator decision: %s", parsed.Reason)
	// }
	// return is_valid
}

// Log the completion of the task
func (ag *Agent) logCompletion() {
	log.Printf("✅ Task completed")
	if success := ag.State.History.IsSuccessful(); success != nil && *success {
		log.Printf("✅ Successfully")
	} else {
		log.Printf("❌ Unfinished")
	}

	totalTokens := ag.State.History.TotalInputTokens()
	log.Printf("📝 Total input tokens used (approximate): %d", totalTokens)

	if ag.RegisterDoneCallback != nil {
		ag.RegisterDoneCallback(ag.State.History)
	}
}
