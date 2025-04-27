package controller

import (
	"errors"
	"slices"

	"nerdface-ai/browser-use-go/browser-use/browser"

	"github.com/playwright-community/playwright-go"
	"github.com/tmc/langchaingo/llms"
)

// TODO: Registry should be rechecked
// The main service class that manages action registration and execution
type Registry struct {
	Registry       *ActionRegistry
	ExcludeActions []string
}

func NewRegistry() *Registry {
	return &Registry{
		Registry:       NewActionRegistry(),
		ExcludeActions: []string{},
	}
}

// Action registers a new action into the registry.
// should be called after registry initialization
// registry.Action("click_element", ClickElementFunc, "click action", paramModel, domains, pageFilter)
func (r *Registry) RegisterAction(name string, function interface{}, description string, paramModel string, domains []string, pageFilter func(*playwright.Page) bool) {
	// if ExcludeActions contains name, return
	if slices.Contains(r.ExcludeActions, name) {
		return
	}

	action := RegisteredAction{
		Name:        name,
		Description: description,
		Function:    function,
		ParamModel:  paramModel,
		Domains:     domains,
		PageFilter:  pageFilter,
	}

	r.Registry.actions[name] = &action
}

// Execute a registered action
// TODO: support Context
func (r *Registry) ExecuteAction(
	actionName string,
	params map[string]interface{},
	browser *browser.BrowserContext,
	pageExtractionLlm llms.Model,
	sensitiveData map[string]string,
	availableFilePaths []string,
	/*context Context*/) (interface{}, error) {

	action, ok := r.Registry.actions[actionName]
	if !ok {
		return nil, errors.New("action not found")
	}

	// Create thervalidated Pydantic model
	// validatedParams = action.paramModel(params)

	// Check if the first parameter is a Pydantic model
	// sig = signature(action.function)
	// parameters = list(sig.parameters.values())
	// is_pydantic = parameters && issubclass(parameters[0].annotation, BaseModel)
	// parameter_names = [param.name for param in parameters]
	parameterNames := []string{}
	for name := range params {
		parameterNames = append(parameterNames, name)
	}

	// TODO: replace sensitive data
	// if sensitive_data {
	// 	validated_params = self._replace_sensitive_data(validated_params, sensitive_data)
	// }
	// Check if the action requires browser
	if !slices.Contains(parameterNames, "browser") && browser == nil {
		return nil, errors.New("action requires browser but none provided")
	}
	if !slices.Contains(parameterNames, "page_extraction_llm") && pageExtractionLlm == nil {
		return nil, errors.New("action requires page_extraction_llm but none provided")
	}
	if !slices.Contains(parameterNames, "available_file_paths") && availableFilePaths == nil {
		return nil, errors.New("action requires available_file_paths but none provided")
	}
	// if !slices.Contains(parameterNames, "context") && context == nil {
	// 	return nil, errors.New("action requires context but none provided")
	// }

	// Prepare arguments based on parameter type
	extraArgs := make(map[string]interface{})
	// if slices.Contains(parameterNames, "context") {
	// 	extraArgs["context"] = context
	// }
	if slices.Contains(parameterNames, "browser") {
		extraArgs["browser"] = browser
	}
	if slices.Contains(parameterNames, "page_extraction_llm") {
		extraArgs["page_extraction_llm"] = pageExtractionLlm
	}
	if slices.Contains(parameterNames, "available_file_paths") {
		extraArgs["available_file_paths"] = availableFilePaths
	}
	if actionName == "input_text" && sensitiveData != nil {
		extraArgs["has_sensitive_data"] = true
	}
	// if isPydantic {
	// 	return action.Function.(func(map[string]interface{}, map[string]interface{}) (interface{}, error))(params, extraArgs)
	// }
	return action.Function.(func(map[string]interface{}) interface{})(extraArgs), nil
}

func (r *Registry) CreateActionModel(includeActions []string, page *playwright.Page) *ActionModel {
	// Create model from registered actions, used by LLM APIs that support tool calling

	// Filter actions based on page if provided:
	//   if page is None, only include actions with no filters
	//   if page is provided, only include actions that match the page

	availableActions := make(map[string]*RegisteredAction)
	for name, action := range r.Registry.actions {
		if includeActions != nil && !slices.Contains(includeActions, name) {
			continue
		}

		// If no page provided, only include actions with no filters
		if page == nil {
			if action.PageFilter == nil && action.Domains == nil {
				availableActions[name] = action
				continue
			}
		}

		// Check page_filter if present
		domainIsAllowed := r.Registry.matchDomains(action.Domains, (*page).URL())
		pageIsAllowed := r.Registry.matchPageFilter(action.PageFilter, page)

		// Include action if both filters match (or if either is not present)
		if domainIsAllowed && pageIsAllowed {
			availableActions[name] = action
		}
	}

	actionModel := &ActionModel{
		Actions: make(map[string]interface{}),
	}

	for name, action := range availableActions {
		actionModel.Actions[name] = action.ParamModel
	}

	return actionModel
}

func (r *Registry) GetPromptDescription(page playwright.Page) string {
	return r.Registry.GetPromptDescription(page)
}
