package controller

import (
	"errors"
	"fmt"
	"log"
	"nerdface-ai/browser-use-go/browser-use/browser"
	"strconv"

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
func (c *Controller) RegisterAction(
	name string,
	description string,
	paramModel interface{},
	function func(interface{}, map[string]interface{}) (*ActionResult, error),
	domains []string,
	pageFilter func(playwright.Page) bool,
) {
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

// ExecuteAction: action.Function(validatedParams, extraArgs)
func (c *Controller) ClickElementByIndex(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	actionParams := params.(*ClickElementAction)
	var browserContext *browser.BrowserContext
	if bc, ok := extraArgs["browser"].(*browser.BrowserContext); ok {
		browserContext = bc
	} else {
		return nil, errors.New("browserContext is not found")
	}
	session := browserContext.GetSession()
	initialPages := len(session.Context.Pages())

	elementNode, err := browserContext.GetDomElementByIndex(actionParams.Index)
	if err != nil {
		return nil, err
	}

	// TODO: if element has file uploader then dont click

	// TODO: error handling
	downloadPath, err := browserContext.ClickElementNode(elementNode)
	if err != nil {
		return nil, err
	}

	msg := ""
	if downloadPath != nil {
		msg = fmt.Sprintf("💾  Downloaded file to %s", downloadPath)
	} else {
		msg = fmt.Sprintf("🖱️  Clicked button with index %d: %s", actionParams.Index, elementNode.GetAllTextTillNextClickableElement())
	}

	if len(session.Context.Pages()) > initialPages {
		newTabMsg := "New tab opened - switching to it"
		msg += " - " + newTabMsg
		log.Println(newTabMsg)
		browserContext.SwitchToTab(-1)
	}

	actionResult := NewActionResult()
	actionResult.ExtractedContent = optional.Some(msg)
	actionResult.IncludeInMemory = true

	return actionResult, nil
}

func (c *Controller) InputText(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	actionParams := params.(*InputTextAction)
	var browserContext *browser.BrowserContext
	if bc, ok := extraArgs["browser"].(*browser.BrowserContext); ok {
		browserContext = bc
	} else {
		return nil, errors.New("browserContext is not found")
	}
	selectorMap := browserContext.GetSelectorMap()
	if (*selectorMap)[actionParams.Index] == nil {
		return nil, errors.New("element with index " + strconv.Itoa(actionParams.Index) + " does not exist")
	}

	elementNode, err := browserContext.GetDomElementByIndex(actionParams.Index)
	if err != nil {
		return nil, err
	}
	browserContext.InputTextElementNode(elementNode, actionParams.Text)

	msg := fmt.Sprintf("Input %s into index %d", actionParams.Text, actionParams.Index)

	actionResult := NewActionResult()
	actionResult.ExtractedContent = optional.Some(msg)
	actionResult.IncludeInMemory = true

	return actionResult, nil
}
