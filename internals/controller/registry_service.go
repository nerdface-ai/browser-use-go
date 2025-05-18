package controller

import (
	"context"
	"errors"
	"slices"

	"github.com/nerdface-ai/browser-use-go/pkg/browser"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	einoUtils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/playwright-community/playwright-go"
)

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
// registry.Action("click_element_by_index", ClickElementFunc, "click action", paramModel, domains, pageFilter)
func registerAction[T, D any](
	r *Registry,
	name string,
	description string,
	function einoUtils.InvokeFunc[T, D],
	domains []string,
	pageFilter func(playwright.Page) bool,
) error {
	// if ExcludeActions contains name, return
	if slices.Contains(r.ExcludeActions, name) {
		return errors.New("action " + name + " is already registered")
	}

	action, err := NewRegisteredAction(name, description, function, domains, pageFilter)
	if err != nil {
		return err
	}
	r.Registry.Actions[name] = action
	return nil
}

type contextKey string

const (
	browserKey            contextKey = "browser"
	pageExtractionLlmKey  contextKey = "page_extraction_llm"
	availableFilePathsKey contextKey = "available_file_paths"
)

// Execute a registered action
// TODO(LOW): support Context
func (r *Registry) ExecuteAction(
	actionName string,
	argumentsInJson string,
	browser *browser.BrowserContext,
	pageExtractionLlm model.ToolCallingChatModel,
	sensitiveData map[string]string,
	availableFilePaths []string,
	/*context Context*/) (string, error) {

	// ex) actionName: "ClickElementAction"
	action, ok := r.Registry.Actions[actionName]
	if !ok {
		return "", errors.New("action not found")
	}

	ctx := context.Background()
	if browser != nil {
		ctx = context.WithValue(ctx, browserKey, browser)
	}
	if pageExtractionLlm != nil {
		ctx = context.WithValue(ctx, pageExtractionLlmKey, pageExtractionLlm)
	}
	if availableFilePaths != nil {
		ctx = context.WithValue(ctx, availableFilePathsKey, availableFilePaths)
	}

	// TODO(HIGH): replace sensitive data
	// if sensitive_data {
	// 	validated_params = self._replace_sensitive_data(validated_params, sensitive_data)
	// }
	// Check if the action requires browser
	// if !slices.Contains(parameterNames, "context") && context == nil {
	// 	return nil, errors.New("action requires context but none provided")
	// }

	result, err := (*action.Tool).InvokableRun(ctx, argumentsInJson, tool.Option{})
	if err != nil {
		return "", err
	}

	return result, nil
}

func (r *Registry) CreateActionModel(includeActions []string, page playwright.Page) *ActionModel {
	// Create model from registered actions, used by LLM APIs that support tool calling

	// Filter actions based on page if provided:
	//   if page is None, only include actions with no filters
	//   if page is provided, only include actions that match the page

	availableActions := make(map[string]*RegisteredAction)
	for name, action := range r.Registry.Actions {
		if includeActions != nil && !slices.Contains(includeActions, name) {
			continue
		}

		// If no page provided, only include actions with no filters
		if page == nil {
			if action.PageFilter == nil && len(action.Domains) == 0 {
				availableActions[name] = action
			}
			continue
		}

		// Check page_filter if present
		domainIsAllowed := r.Registry.matchDomains(action.Domains, page.URL())
		pageIsAllowed := r.Registry.matchPageFilter(action.PageFilter, page)

		// Include action if both filters match (or if either is not present)
		if domainIsAllowed && pageIsAllowed {
			availableActions[name] = action
		}
	}

	return &ActionModel{
		Actions: availableActions,
	}
}

func (r *Registry) GetPromptDescription(page playwright.Page) string {
	return r.Registry.GetPromptDescription(page)
}
