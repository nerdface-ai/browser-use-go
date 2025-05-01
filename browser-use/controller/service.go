package controller

import (
	"nerdface-ai/browser-use-go/browser-use/browser"

	"github.com/moznion/go-optional"
	"github.com/playwright-community/playwright-go"
	"github.com/tmc/langchaingo/llms"
)

type ActionResult struct {
	IsDone           optional.Option[bool]   `json:"is_done"`
	Success          optional.Option[bool]   `json:"success"`
	ExtractedContent optional.Option[string] `json:"extracted_content"`
	Error            optional.Option[string] `json:"error"`
	IncludeInMemory  bool                    `json:"include_in_memory"`
}

func NewActionResult() *ActionResult {
	return &ActionResult{
		IsDone:           optional.Some(false),
		Success:          optional.None[bool](),
		ExtractedContent: optional.None[string](),
		Error:            optional.None[string](),
		IncludeInMemory:  false,
	}
}

type Controller struct {
	Registry *Registry
}

func NewController() *Controller {
	return &Controller{
		Registry: NewRegistry(),
	}
}

// register
func (c *Controller) RegisterAction(name string, description string, paramModel interface{}, function interface{}, domains []string, pageFilter func(*playwright.Page) bool) {
	if c.Registry == nil {
		return
	}
	c.Registry.RegisterAction(name, description, paramModel, function, domains, pageFilter)
}

// Act
func (c *Controller) ExecuteAction(
	action *ActionModel,
	browserContext *browser.BrowserContext,
	pageExtractionLlm llms.Model,
	sensitiveData map[string]string,
	availableFilePaths []string,
	// context: Context | None,
) (*ActionResult, error) {
	for actionName, actionParams := range action.Actions {
		result, err := c.Registry.ExecuteAction(actionName, actionParams.(map[string]interface{}), browserContext, pageExtractionLlm, sensitiveData, availableFilePaths)
		if err != nil {
			return nil, err
		}
		if result, ok := result.(string); ok {
			actionResult := NewActionResult()
			actionResult.ExtractedContent = optional.Some(result)
			return actionResult, nil
		}
		if result, ok := result.(*ActionResult); ok {
			return result, nil
		}
	}
	return NewActionResult(), nil
}
