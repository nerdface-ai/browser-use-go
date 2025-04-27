package controller

import (
	"strings"

	"github.com/playwright-community/playwright-go"
)

/*
----- ExecuteAction -----
action_name: "open_tab",
params: {'url': 'https://techcrunch.com'}
parameter names: ['params', 'browser']

*/

type RegisteredAction struct {
	Name        string
	Description string
	Function    interface{}
	ParamModel  map[string]interface{} // needed params for click, search, etc.

	// filters: provide specific domains or a function to determine whether the action should be available on the given page or not
	Domains    []string // # e.g. ['*.google.com', 'www.bing.com', 'yahoo.*]
	PageFilter func(*playwright.Page) bool
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
	var sb strings.Builder
	// sb.WriteString(ra.Description + ":\n")
	// sb.WriteString("{" + ra.Name + ": ")

	// if ra.ParamModel != nil && ra.ParamModel.Kind() == reflect.Struct {
	// 	sb.WriteString("{")
	// 	for i := 0; i < ra.ParamModel.NumField(); i++ {
	// 		field := ra.ParamModel.Field(i)
	// 		if field.Name == "title" {
	// 			continue
	// 		}
	// 		sb.WriteString(field.Name + ": " + field.Type.String())
	// 		if i < ra.ParamModel.NumField()-1 {
	// 			sb.WriteString(", ")
	// 		}
	// 	}
	// 	sb.WriteString("}")
	// }
	// sb.WriteString("}")
	return sb.String()
}

// Base model for dynamically created action models
type ActionModel struct {
	/*
	* this will have all the registered actions, e.g.
	* click_element = param_model = ClickElementParams
	* done = param_model = nil
	 */
	Actions map[string]interface{} `json:"actions"` // use as model.Actions["clicked_element"]
}

// Get the index of the action
func (am *ActionModel) GetIndex() int {
	// {'clicked_element': {'index':5}}

	for _, param := range am.Actions {
		if param == nil {
			continue
		}
		if paramMap, ok := param.(map[string]interface{}); ok {
			if index, ok := paramMap["index"]; ok {
				if indexInt, ok := index.(int); ok {
					return indexInt
				}
			}
		}
	}
	return -1
}

// Overwrite the index of the action
func (am *ActionModel) SetIndex(index int) error {
	// Get the action name and params (first field)
	// v := reflect.ValueOf(am).Elem()
	// var actionData = make(map[string]interface{})
	// for i := 0; i < v.NumField(); i++ {
	// 	field := v.Field(i)
	// 	if !field.IsNil() {
	// 		fieldName := v.Type().Field(i).Name
	// 		actionData[fieldName] = field.Interface()
	// 	}
	// }

	// // actionName: first field name
	// var actionName string
	// var actionParams reflect.Value
	// for name, param := range actionData {
	// 	actionName = name
	// 	actionParams = reflect.ValueOf(param).Elem()
	// 	break
	// }

	// // Update the index directly on the model
	// indexField := actionParams.FieldByName("index")
	// if indexField.IsValid() && indexField.CanSet() && indexField.Kind() == reflect.Int {
	// 	indexField.SetInt(int64(index))
	// }

	return nil
}

type ActionRegistry struct {
	actions map[string]RegisteredAction
}

func NewActionRegistry() ActionRegistry {
	return ActionRegistry{
		actions: make(map[string]RegisteredAction),
	}
}

func (ar *ActionRegistry) MatchDomains(domains []string, url string) bool {
	return true
}

func (ar *ActionRegistry) matchPageFilter(pageFilter func(*playwright.Page) bool, page *playwright.Page) bool {
	// match a page filter against a page
	if pageFilter == nil {
		return true
	}
	return pageFilter(page)
}

func (ar *ActionRegistry) GetPromptDescription(page *playwright.Page) string {
	/*
		Get a description of all actions for the prompt

		Args:
			page: If provided, filter actions by page using page_filter and domains.

		Returns:
			A string description of available actions.
			- If page is None: return only actions with no page_filter and no domains (for system prompt)
			- If page is provided: return only filtered actions that match the current page (excluding unfiltered actions)
	*/

	// domain_is_allowed = self._match_domains(action.domains, page.url)
	// page_is_allowed = self._match_page_filter(action.page_filter, page)

	return ""
}
