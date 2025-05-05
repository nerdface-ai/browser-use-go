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
	c := &Controller{
		Registry: NewRegistry(),
	}
	c.RegisterAction("DoneAction", "Complete task - with return text and if the task is finished (success=True) or not yet  completely finished (success=False), because last step is reached", DoneAction{}, c.Done, []string{}, nil)
	c.RegisterAction("ClickElementByIndex", "Click element by index", ClickElementAction{}, c.ClickElementByIndex, []string{}, nil)
	c.RegisterAction("InputText", "Input text into a input interactive element", InputTextAction{}, c.InputText, []string{}, nil)
	c.RegisterAction("SearchGoogle", "Search the query in Google in the current tab, the query should be a search query like humans search in Google, concrete and not vague or super long. More the single most important items.", SearchGoogleAction{}, c.SearchGoogle, []string{}, nil)
	c.RegisterAction("GoToUrl", "Navigate to URL in the current tab", GoToUrlAction{}, c.GoToUrl, []string{}, nil)
	c.RegisterAction("GoBack", "Go back to the previous page", NoParamsAction{}, c.GoBack, []string{}, nil)
	c.RegisterAction("Wait", "Wait for x seconds default 3", WaitAction{}, c.Wait, []string{}, nil)
	c.RegisterAction("SavePdf", "Save the current page as a PDF file", SavePdfAction{}, c.SavePdf, []string{}, nil)
	c.RegisterAction("SwitchTab", "Switch tab", SwitchTabAction{}, c.SwitchTab, []string{}, nil)
	c.RegisterAction("OpenTab", "Open url in new tab", OpenTabAction{}, c.OpenTab, []string{}, nil)
	c.RegisterAction("CloseTab", "Close an existing tab", CloseTabAction{}, c.CloseTab, []string{}, nil)
	c.RegisterAction("ExtractContent", "Extract page content to retrieve specific information from the page, e.g. all company names, a specific description, all information about, links with companies in structured format or simply links", ExtractContentAction{}, c.ExtractContent, []string{}, nil)
	c.RegisterAction("ScrollDown", "Scroll down the page by pixel amount - if no amount is specified, scroll down one page", ScrollAction{}, c.ScrollDown, []string{}, nil)
	c.RegisterAction("ScrollUp", "Scroll up the page by pixel amount - if no amount is specified, scroll up one page", ScrollAction{}, c.ScrollUp, []string{}, nil)
	c.RegisterAction("SendKeys", "Send strings of special keys like Escape,Backspace, Insert, PageDown, Delete, Enter, Shortcuts such as `Control+o`, `Control+Shift+T` are supported as well. This gets used in keyboard.press.", SendKeysAction{}, c.SendKeys, []string{}, nil)
	c.RegisterAction("ScrollToText", "If you dont find something which you want to interact with, scroll to it", ScrollToTextAction{}, c.ScrollToText, []string{}, nil)
	c.RegisterAction("GetDropdownOptions", "Get all options from a native dropdown", GetDropdownOptionsAction{}, c.GetDropdownOptions, []string{}, nil)
	c.RegisterAction("SelectDropdownOption", "Select dropdown option for interactive element index by the text of the option you want to select", SelectDropdownOptionAction{}, c.SelectDropdownOption, []string{}, nil)
	c.RegisterAction("DragDrop", "Drag and drop elements or between coordinates on the page - useful for canvas drawing, sortable lists, sliders, file uploads, and UI rearrangement", DragDropAction{}, c.DragDrop, []string{}, nil)
	return c
}

// register
func (c *Controller) RegisterAction(
	name string,
	description string,
	paramModel interface{},
	function func(interface{}, map[string]interface{}) (*ActionResult, error),
	domains []string,
	pageFilter func(*playwright.Page) bool,
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

func (c *Controller) Done(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	actionParams := params.(*DoneAction)
	actionResult := NewActionResult()
	actionResult.IsDone = optional.Some(true)
	actionResult.Success = optional.Some(actionParams.Success)
	actionResult.ExtractedContent = optional.Some(actionParams.Text)
	return actionResult, nil
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

func (c *Controller) SearchGoogle(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	return NewActionResult(), nil
}

func (c *Controller) GoToUrl(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	return NewActionResult(), nil
}

func (c *Controller) GoBack(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	return NewActionResult(), nil
}

func (c *Controller) Wait(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	return NewActionResult(), nil
}

func (c *Controller) SavePdf(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	return NewActionResult(), nil
}

func (c *Controller) SwitchTab(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	return NewActionResult(), nil
}

func (c *Controller) OpenTab(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	return NewActionResult(), nil
}

func (c *Controller) CloseTab(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	return NewActionResult(), nil
}

func (c *Controller) ExtractContent(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	return NewActionResult(), nil
}

func (c *Controller) ScrollDown(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	return NewActionResult(), nil
}

func (c *Controller) ScrollUp(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	return NewActionResult(), nil
}

func (c *Controller) SendKeys(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	return NewActionResult(), nil
}

func (c *Controller) ScrollToText(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	return NewActionResult(), nil
}

func (c *Controller) GetDropdownOptions(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	return NewActionResult(), nil
}

func (c *Controller) SelectDropdownOption(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	return NewActionResult(), nil
}

func (c *Controller) DragDrop(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	return NewActionResult(), nil
}
