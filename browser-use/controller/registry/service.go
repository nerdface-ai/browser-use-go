package controller

import (
	"slices"

	"github.com/playwright-community/playwright-go"
)

type Registry struct {
	Registry       ActionRegistry
	ExcludeActions []string
}

func NewRegistry() *Registry {
	return &Registry{
		Registry:       NewActionRegistry(),
		ExcludeActions: []string{},
	}
}

func (r *Registry) CreateActionModel(includeActions []string, page *playwright.Page) *ActionModel {
	// Create model from registered actions, used by LLM APIs that support tool calling

	// Filter actions based on page if provided:
	//   if page is None, only include actions with no filters
	//   if page is provided, only include actions that match the page

	availableActions := make(map[string]RegisteredAction)
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
		domainIsAllowed := r.Registry.MatchDomains(action.Domains, (*page).URL())
		pageIsAllowed := r.Registry.matchPageFilter(action.PageFilter, page)

		// Include action if both filters match (or if either is not present)
		if domainIsAllowed && pageIsAllowed {
			availableActions[name] = action
		}
	}

	fields := make(map[string]interface{})
	for name, action := range availableActions {
		fields[name] = action.ParamModel
	}

	return &ActionModel{
		Actions: fields,
	}
}
