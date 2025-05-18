package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	einoUtils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/playwright-community/playwright-go"
)

type RegisteredAction struct {
	Tool *tool.InvokableTool
	// filters: provide specific domains or a function to determine whether the action should be available on the given page or not
	Domains    []string // # e.g. ['*.google.com', 'www.bing.com', 'yahoo.*]
	PageFilter func(playwright.Page) bool
}

func NewRegisteredAction[T, D any](
	name string,
	description string,
	actionFunc einoUtils.InvokeFunc[T, D],
	domains []string,
	pageFilter func(playwright.Page) bool,
) (*RegisteredAction, error) {
	customTool, err := einoUtils.InferTool(name, description, actionFunc)

	if err != nil {
		return nil, err
	}
	return &RegisteredAction{
		Tool:       &customTool,
		Domains:    domains,
		PageFilter: pageFilter,
	}, nil
}

/*
	example

----------------- INPUT ------------------------------
description: "Search for text"
name: "search"
param_model:

	class SearchParams(BaseModel):
		query: str
		case_sensitive: bool

	{
	    "query": {"type": "string", "title": "검색어"},
	    "case_sensitive": {"type": "boolean", "title": "대소문자 구분"}
	}

----------------- OUTPUT ------------------------------
Search for text:
{search: {'query': {'type': 'string'}, 'case_sensitive': {'type': 'boolean'}}}
*/
func (ra *RegisteredAction) PromptDescription() string {
	// Get a description of the action for the prompt
	toolInfo, err := (*ra.Tool).Info(context.Background())
	if err != nil {
		panic(err)
	}
	name := toolInfo.Name
	desc := toolInfo.Desc

	s := fmt.Sprintf("%s: \n", desc)
	fmtObj := make(map[string]interface{})
	schema, err := toolInfo.ToOpenAPIV3()
	if err != nil {
		panic(err)
	}
	properties := schema.Properties
	fmtObj[name] = properties
	json, err := json.Marshal(fmtObj)
	if err != nil {
		panic(err)
	}
	s += string(json)
	return s
}

// Base model for dynamically created action models
type ActionModel struct {
	/*
	* this will have all the registered actions, e.g.
	* click_element_by_index = param_model = ClickElementParams
	* done = param_model = nil
	 */
	Actions map[string]*RegisteredAction `json:"actions"`
}

type ActModel map[string]interface{}

// Get the index of the action
func (am *ActModel) GetIndex() *int {
	for _, params := range *am {
		paramJson, ok := params.(map[string]interface{})
		if !ok {
			continue
		}
		if index, ok := paramJson["index"]; ok {
			if indexInt, ok := index.(int); ok {
				return &indexInt
			}
		}
	}
	return nil
}

// Overwrite the index of the action
func (am *ActModel) SetIndex(index int) {
	for key, params := range *am {
		paramJson, ok := params.(map[string]interface{})
		if !ok {
			continue
		}
		if paramJson["index"] != nil {
			paramJson["index"] = index
		}
		(*am)[key] = paramJson
	}
}

// Model representing the action registry
type ActionRegistry struct {
	Actions map[string]*RegisteredAction
}

func NewActionRegistry() *ActionRegistry {
	return &ActionRegistry{
		Actions: make(map[string]*RegisteredAction),
	}
}

func (ar *ActionRegistry) matchDomains(domains []string, urlStr string) bool {
	if len(domains) == 0 || urlStr == "" {
		return true
	}

	// Parse the URL to get the domain
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	if parsedURL.Host == "" {
		return false
	}

	domain := parsedURL.Host
	// Remove port if present
	if colonIndex := strings.Index(domain, ":"); colonIndex >= 0 {
		domain = domain[:colonIndex]
	}

	// Match domain against patterns
	for _, domainPattern := range domains {
		matched, err := filepath.Match(domainPattern, domain)
		if err == nil && matched {
			return true
		}
	}

	return false
}

func (ar *ActionRegistry) matchPageFilter(pageFilter func(playwright.Page) bool, page playwright.Page) bool {
	// match a page filter against a page
	if pageFilter == nil {
		return true
	}
	return pageFilter(page)
}

// Get a description of all actions for the prompt
func (ar *ActionRegistry) GetPromptDescription(page playwright.Page) string {
	/*
		Args:
			page: If provided, filter actions by page using page_filter and domains.

		Returns:
			A string description of available actions.
			- If page is None: return only actions with no page_filter and no domains (for system prompt)
			- If page is provided: return only filtered actions that match the current page (excluding unfiltered actions)
	*/
	if page == nil {
		var descriptions []string
		for _, action := range ar.Actions {
			if action.PageFilter == nil && len(action.Domains) == 0 {
				descriptions = append(descriptions, action.PromptDescription())
			}
		}
		return strings.Join(descriptions, "\n")
	}

	// only include filtered actions for the current page
	var filteredActions []*RegisteredAction
	for _, action := range ar.Actions {
		if action.PageFilter == nil && len(action.Domains) == 0 {
			continue
		}

		domainIsAllowed := ar.matchDomains(action.Domains, page.URL())
		pageIsAllowed := ar.matchPageFilter(action.PageFilter, page)

		if domainIsAllowed && pageIsAllowed {
			filteredActions = append(filteredActions, action)
		}
	}

	var descriptions []string
	for _, action := range filteredActions {
		descriptions = append(descriptions, action.PromptDescription())
	}
	return strings.Join(descriptions, "\n")
}
