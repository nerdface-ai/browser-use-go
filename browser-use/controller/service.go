package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"nerdface-ai/browser-use-go/browser-use/browser"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/playwright-community/playwright-go"

	"github.com/adrg/xdg"
)

type ActionResult struct {
	IsDone           *bool   `json:"is_done,omitempty"`
	Success          *bool   `json:"success,omitempty"`
	ExtractedContent *string `json:"extracted_content,omitempty"`
	Error            *string `json:"error,omitempty"`
	IncludeInMemory  bool    `json:"include_in_memory"`
}

func NewActionResult() *ActionResult {
	return &ActionResult{
		IsDone:           playwright.Bool(false),
		Success:          playwright.Bool(false),
		ExtractedContent: nil,
		Error:            nil,
		IncludeInMemory:  false,
	}
}

func getActionParams[T any](params interface{}) (*T, error) {
	actionParams, ok := params.(*T)
	if !ok {
		return nil, errors.New("failed to cast params to action params")
	}
	return actionParams, nil
}

func getBrowserContext(extraArgs map[string]interface{}) (*browser.BrowserContext, error) {
	if bc, ok := extraArgs["browser"].(*browser.BrowserContext); ok {
		return bc, nil
	}
	return nil, errors.New("browserContext is not found")
}

func getActionParamsAndBrowserContext[T any](params interface{}, extraArgs map[string]interface{}) (*T, *browser.BrowserContext, error) {
	actionParams, err := getActionParams[T](params)
	if err != nil {
		return nil, nil, err
	}
	bc, err := getBrowserContext(extraArgs)
	if err != nil {
		return nil, nil, err
	}
	return actionParams, bc, nil
}

type Controller struct {
	Registry *Registry
}

func NewController() *Controller {
	c := &Controller{
		Registry: NewRegistry(),
	}
	c.RegisterAction("Done", "Complete task - with return text and if the task is finished (success=True) or not yet  completely finished (success=False), because last step is reached", DoneAction{}, c.Done, []string{}, nil)
	c.RegisterAction("ClickElementByIndex", "Click element by index", ClickElementAction{}, c.ClickElementByIndex, []string{}, nil)
	c.RegisterAction("InputText", "Input text into a input interactive element", InputTextAction{}, c.InputText, []string{}, nil)
	c.RegisterAction("SearchGoogle", "Search the query in Google in the current tab, the query should be a search query like humans search in Google, concrete and not vague or super long. More the single most important items.", SearchGoogleAction{}, c.SearchGoogle, []string{}, nil)
	c.RegisterAction("GoToUrl", "Navigate to URL in the current tab", GoToUrlAction{}, c.GoToUrl, []string{}, nil)
	c.RegisterAction("GoBack", "Go back to the previous page", GoBackAction{}, c.GoBack, []string{}, nil)
	c.RegisterAction("Wait", "Wait for x seconds default 3", WaitAction{}, c.Wait, []string{}, nil)
	c.RegisterAction("SavePdf", "Save the current page as a PDF file", SavePdfAction{}, c.SavePdf, []string{}, nil)
	c.RegisterAction("SwitchTab", "Switch tab", SwitchTabAction{}, c.SwitchTab, []string{}, nil)
	c.RegisterAction("OpenTab", "Open url in new tab", OpenTabAction{}, c.OpenTab, []string{}, nil)
	c.RegisterAction("CloseTab", "Close an existing tab", CloseTabAction{}, c.CloseTab, []string{}, nil)
	c.RegisterAction("ExtractContent", "Extract page content to retrieve specific information from the page, e.g. all company names, a specific description, all information about, links with companies in structured format or simply links", ExtractContentAction{}, c.ExtractContent, []string{}, nil)
	c.RegisterAction("ScrollDown", "Scroll down the page by pixel amount - if no amount is specified, scroll down one page", ScrollDownAction{}, c.ScrollDown, []string{}, nil)
	c.RegisterAction("ScrollUp", "Scroll up the page by pixel amount - if no amount is specified, scroll up one page", ScrollUpAction{}, c.ScrollUp, []string{}, nil)
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
	pageExtractionLlm model.ToolCallingChatModel,
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
			actionResult.ExtractedContent = &result
			return actionResult, nil
		}
		if result, ok := result.(*ActionResult); ok {
			return result, nil
		}
	}
	return NewActionResult(), nil
}

func (c *Controller) Done(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	actionParams, err := getActionParams[DoneAction](params)
	if err != nil {
		return nil, err
	}
	actionResult := NewActionResult()
	actionResult.IsDone = playwright.Bool(true)
	actionResult.Success = &actionParams.Success
	actionResult.ExtractedContent = &actionParams.Text
	return actionResult, nil
}

// ExecuteAction: action.Function(validatedParams, extraArgs)
func (c *Controller) ClickElementByIndex(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	actionParams, bc, err := getActionParamsAndBrowserContext[ClickElementAction](params, extraArgs)
	if err != nil {
		return nil, err
	}
	session := bc.GetSession()
	initialPages := len(session.Context.Pages())

	elementNode, err := bc.GetDomElementByIndex(actionParams.Index)
	if err != nil {
		return nil, err
	}

	// TODO: if element has file uploader then dont click

	// TODO: error handling
	downloadPath, err := bc.ClickElementNode(elementNode)
	if err != nil {
		return nil, err
	}

	msg := ""
	if downloadPath != nil {
		msg = fmt.Sprintf("💾  Downloaded file to %s", *downloadPath)
	} else {
		msg = fmt.Sprintf("🖱️  Clicked button with index %d: %s", actionParams.Index, elementNode.GetAllTextTillNextClickableElement())
	}

	if len(session.Context.Pages()) > initialPages {
		newTabMsg := "New tab opened - switching to it"
		msg += " - " + newTabMsg
		log.Debug(newTabMsg)
		bc.SwitchToTab(-1)
	}

	actionResult := NewActionResult()
	actionResult.ExtractedContent = &msg
	actionResult.IncludeInMemory = true

	return actionResult, nil
}

func (c *Controller) InputText(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	actionParams, bc, err := getActionParamsAndBrowserContext[InputTextAction](params, extraArgs)
	if err != nil {
		return nil, err
	}
	selectorMap := bc.GetSelectorMap()
	if (*selectorMap)[actionParams.Index] == nil {
		return nil, errors.New("element with index " + strconv.Itoa(actionParams.Index) + " does not exist")
	}

	elementNode, err := bc.GetDomElementByIndex(actionParams.Index)
	if err != nil {
		return nil, err
	}
	bc.InputTextElementNode(elementNode, actionParams.Text)

	msg := fmt.Sprintf("Input %s into index %d", actionParams.Text, actionParams.Index)

	actionResult := NewActionResult()
	actionResult.ExtractedContent = &msg
	actionResult.IncludeInMemory = true

	return actionResult, nil
}

func (c *Controller) SearchGoogle(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	actionParams, bc, err := getActionParamsAndBrowserContext[SearchGoogleAction](params, extraArgs)
	if err != nil {
		return nil, err
	}
	page := bc.GetCurrentPage()
	page.Goto(fmt.Sprintf("https://www.google.com/search?q=%s&udm=14", actionParams.Query))
	page.WaitForLoadState()
	msg := fmt.Sprintf("🔍  Searched for \"%s\" in Google", actionParams.Query)
	log.Debug(msg)
	actionResult := NewActionResult()
	actionResult.ExtractedContent = &msg
	actionResult.IncludeInMemory = true
	return actionResult, nil
}

func (c *Controller) GoToUrl(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	actionParams, bc, err := getActionParamsAndBrowserContext[GoToUrlAction](params, extraArgs)
	if err != nil {
		return nil, err
	}
	page := bc.GetCurrentPage()
	page.Goto(actionParams.Url)
	page.WaitForLoadState()
	msg := fmt.Sprintf("🔗  Navigated to %s", actionParams.Url)
	log.Debug(msg)
	actionResult := NewActionResult()
	actionResult.ExtractedContent = &msg
	actionResult.IncludeInMemory = true
	return actionResult, nil
}

func (c *Controller) GoBack(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	bc, err := getBrowserContext(extraArgs)
	if err != nil {
		return nil, err
	}
	bc.GoBack()
	msg := "🔙  Navigated back"
	log.Debug(msg)
	actionResult := NewActionResult()
	actionResult.ExtractedContent = &msg
	actionResult.IncludeInMemory = true
	return actionResult, nil
}

func (c *Controller) Wait(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	actionParams, err := getActionParams[WaitAction](params)
	if err != nil {
		return nil, err
	}
	msg := fmt.Sprintf("🕒  Waiting for %d seconds", actionParams.Seconds)
	log.Debug(msg)
	time.Sleep(time.Duration(actionParams.Seconds) * time.Second)
	actionResult := NewActionResult()
	actionResult.ExtractedContent = &msg
	actionResult.IncludeInMemory = true
	return actionResult, nil
}

func (c *Controller) SavePdf(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	bc, err := getBrowserContext(extraArgs)
	if err != nil {
		return nil, err
	}
	page := bc.GetCurrentPage()
	shortUrl := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(page.URL(), "https://", ""), "http://", ""), "www.", ""), "/", "")
	slug := strings.ToLower(strings.ReplaceAll(shortUrl, "[^a-zA-Z0-9]+", "-"))
	sanitizedFilename := fmt.Sprintf("%s.pdf", slug)

	pdfPath := xdg.UserDirs.Download + "/" + sanitizedFilename
	page.EmulateMedia(playwright.PageEmulateMediaOptions{Media: playwright.MediaScreen})
	page.PDF(playwright.PagePdfOptions{Path: &pdfPath, Format: playwright.String("A4"), PrintBackground: playwright.Bool(false)})
	msg := fmt.Sprintf("Saving page with URL %s as PDF to %s", page.URL(), pdfPath)
	log.Debug(msg)
	actionResult := NewActionResult()
	actionResult.ExtractedContent = &msg
	actionResult.IncludeInMemory = true
	return actionResult, nil
}

func (c *Controller) OpenTab(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	actionParams, bc, err := getActionParamsAndBrowserContext[OpenTabAction](params, extraArgs)
	if err != nil {
		return nil, err
	}
	err = bc.CreateNewTab(actionParams.Url)
	if err != nil {
		return nil, err
	}
	msg := fmt.Sprintf("🔗  Opened new tab with %s", actionParams.Url)
	log.Print(msg)
	actionResult := NewActionResult()
	actionResult.ExtractedContent = &msg
	actionResult.IncludeInMemory = true
	return actionResult, nil
}

func (c *Controller) CloseTab(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	actionParams, bc, err := getActionParamsAndBrowserContext[CloseTabAction](params, extraArgs)
	if err != nil {
		return nil, err
	}
	bc.SwitchToTab(actionParams.PageId)
	page := bc.GetCurrentPage()
	page.WaitForLoadState()
	url := page.URL()
	err = page.Close()
	if err != nil {
		return nil, err
	}
	msg := fmt.Sprintf("❌  Closed tab %d with url %s", actionParams.PageId, url)
	log.Debug(msg)
	actionResult := NewActionResult()
	actionResult.ExtractedContent = &msg
	actionResult.IncludeInMemory = true
	return actionResult, nil
}

func (c *Controller) SwitchTab(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	actionParams, bc, err := getActionParamsAndBrowserContext[SwitchTabAction](params, extraArgs)
	if err != nil {
		return nil, err
	}
	bc.SwitchToTab(actionParams.PageId)
	page := bc.GetCurrentPage()
	page.WaitForLoadState()
	msg := fmt.Sprintf("🔄  Switched to tab %d", actionParams.PageId)
	log.Debug(msg)
	actionResult := NewActionResult()
	actionResult.ExtractedContent = &msg
	actionResult.IncludeInMemory = true
	return actionResult, nil
}

func (c *Controller) ExtractContent(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	actionParams, bc, err := getActionParamsAndBrowserContext[ExtractContentAction](params, extraArgs)
	if err != nil {
		return nil, err
	}
	var llm model.ToolCallingChatModel
	if model, ok := extraArgs["page_extraction_llm"].(model.ToolCallingChatModel); ok {
		llm = model
	} else {
		return nil, errors.New("page_extraction_llm is not found")
	}
	page := bc.GetCurrentPage()

	strip := []string{}
	if actionParams.ShouldStripLinkUrls {
		strip = []string{"a", "img"}
	}

	pageContent, err := page.Content()
	if err != nil {
		return nil, err
	}

	conv := converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(),
		),
	)
	for _, tag := range strip {
		conv.Register.TagType(tag, converter.TagTypeRemove, converter.PriorityStandard)
	}

	content, err := conv.ConvertString(pageContent) // TODO: check strip option ?
	if err != nil {
		return nil, err
	}

	// manually append iframe text into the content so it's readable by the LLM (includes cross-origin iframes)
	for _, iframe := range page.Frames() {
		if iframe.URL() != page.URL() && !strings.HasPrefix(iframe.URL(), "data:") {
			iframeContent, err := iframe.Content()
			if err != nil {
				continue
			}
			ifContent, err := conv.ConvertString(iframeContent)
			if err != nil {
				continue
			}
			content += fmt.Sprintf("\n\nIFRAME %s:\n", iframe.URL())
			content += ifContent
		}
	}

	prompt := fmt.Sprintf("Your task is to extract the content of the page. You will be given a page and a goal and you should extract all relevant information around this goal from the page. If the goal is vague, summarize the page. Respond in json format. Extraction goal: %s, Page: %s", actionParams.Goal, content)
	output, err := llm.Generate(context.Background(), []*schema.Message{{Role: schema.User, Content: prompt}})
	if err != nil {
		log.Debug("Error extracting content: %s", err)
		msg := fmt.Sprintf("📄  Extracted from page\n: %s\n", content)
		log.Debug(msg)
		actionResult := NewActionResult()
		actionResult.ExtractedContent = &msg
		return actionResult, nil
	}
	msg := fmt.Sprintf("📄  Extracted from page\n: %s\n", output)
	log.Debug(msg)
	actionResult := NewActionResult()
	actionResult.ExtractedContent = &msg
	actionResult.IncludeInMemory = true
	return actionResult, nil
}

func (c *Controller) ScrollDown(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	actionParams, bc, err := getActionParamsAndBrowserContext[ScrollDownAction](params, extraArgs)
	if err != nil {
		return nil, err
	}

	page := bc.GetCurrentPage()
	amount := "one page"
	if actionParams.Amount != nil {
		page.Evaluate(fmt.Sprintf("window.scrollBy(0, %d);", *actionParams.Amount))
		amount = fmt.Sprintf("%d pixels", *actionParams.Amount)
	} else {
		page.Evaluate("window.scrollBy(0, window.innerHeight);")
	}
	msg := fmt.Sprintf("🔍  Scrolled down the page by %s", amount)
	log.Debug(msg)
	actionResult := NewActionResult()
	actionResult.ExtractedContent = &msg
	actionResult.IncludeInMemory = true
	return actionResult, nil

}

func (c *Controller) ScrollUp(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	actionParams, bc, err := getActionParamsAndBrowserContext[ScrollUpAction](params, extraArgs)
	if err != nil {
		return nil, err
	}
	page := bc.GetCurrentPage()
	var amount string
	if actionParams.Amount != nil {
		page.Evaluate(fmt.Sprintf("window.scrollBy(0, -%d);", *actionParams.Amount))
		amount = fmt.Sprintf("%d pixels", *actionParams.Amount)
	} else {
		page.Evaluate("window.scrollBy(0, -window.innerHeight);")
		amount = "one page"
	}
	msg := fmt.Sprintf("🔍  Scrolled up the page by %s", amount)
	log.Debug(msg)
	actionResult := NewActionResult()
	actionResult.ExtractedContent = &msg
	actionResult.IncludeInMemory = true
	return actionResult, nil
}

func (c *Controller) SendKeys(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	actionParams, bc, err := getActionParamsAndBrowserContext[SendKeysAction](params, extraArgs)
	if err != nil {
		return nil, err
	}

	page := bc.GetCurrentPage()
	err = page.Keyboard().InsertText(actionParams.Keys)
	if err != nil {
		if strings.Contains(err.Error(), "Unknown key") {
			for _, key := range actionParams.Keys {
				err = page.Keyboard().Press(string(key))
				if err != nil {
					return nil, err
				}
			}
		} else {
			return nil, err
		}
	}
	msg := fmt.Sprintf("⌨️  Sent keys: %s", actionParams.Keys)
	log.Debug(msg)
	actionResult := NewActionResult()
	actionResult.ExtractedContent = &msg
	actionResult.IncludeInMemory = true
	return actionResult, nil
}

func (c *Controller) ScrollToText(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	actionParams, bc, err := getActionParamsAndBrowserContext[ScrollToTextAction](params, extraArgs)
	if err != nil {
		return nil, err
	}
	page := bc.GetCurrentPage()
	// Try different locator strategies
	locators := []playwright.Locator{
		page.GetByText(actionParams.Text, playwright.PageGetByTextOptions{Exact: playwright.Bool(false)}),
		page.Locator(fmt.Sprintf("text=%s", actionParams.Text)),
		page.Locator(fmt.Sprintf("//*[contains(text(), '%s')]", actionParams.Text)),
	}

	for _, locator := range locators {
		if visible, err := locator.First().IsVisible(); err == nil && visible {
			err := locator.First().ScrollIntoViewIfNeeded()
			if err != nil {
				log.Debug(fmt.Sprintf("Locator attempt failed: %s", err.Error()))
				continue
			}
			time.Sleep(500 * time.Millisecond)
			msg := fmt.Sprintf("🔍  Scrolled to text: %s", actionParams.Text)
			log.Debug(msg)
			actionResult := NewActionResult()
			actionResult.ExtractedContent = &msg
			actionResult.IncludeInMemory = true
			return actionResult, nil
		}
	}

	msg := fmt.Sprintf("Text '%s' not found or not visible on page", actionParams.Text)
	log.Debug(msg)
	actionResult := NewActionResult()
	actionResult.ExtractedContent = &msg
	actionResult.IncludeInMemory = true
	return actionResult, nil
}

func (c *Controller) GetDropdownOptions(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	actionParams, bc, err := getActionParamsAndBrowserContext[GetDropdownOptionsAction](params, extraArgs)
	if err != nil {
		return nil, err
	}
	page := bc.GetCurrentPage()
	selectorMap := bc.GetSelectorMap()
	domElement := (*selectorMap)[actionParams.Index]

	// Frame-aware approach since we know it works
	allOptions := []string{}
	frameIndex := 0
	for _, frame := range page.Frames() {
		options, err := frame.Evaluate(`
							(xpath) => {
								const select = document.evaluate(xpath, document, null,
									XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue;
								if (!select || select.tagName.toLowerCase() !== 'select') return null;

								return {
									options: Array.from(select.options).map(opt => ({
										text: opt.text, //do not trim, because we are doing exact match in select_dropdown_option
										value: opt.value,
										index: opt.index
									})),
									id: select.id,
									name: select.name
								};
							}`, domElement.Xpath)
		if err != nil {
			log.Debug(fmt.Sprintf("Frame %d evaluation failed: %s", frameIndex, err.Error()))
		}
		if options != nil {
			log.Debug(fmt.Sprintf("Found dropdown in frame %d", frameIndex))
			log.Debug(fmt.Sprintf("Dropdown ID: %s, Name: %s", options.(map[string]interface{})["id"], options.(map[string]interface{})["name"]))

			formattedOptions := []string{}
			for _, opt := range options.(map[string]interface{})["options"].([]interface{}) {
				// encoding ensures AI uses the exact string in select_dropdown_option
				encodedText, _ := json.Marshal(opt.(map[string]interface{})["text"])
				formattedOptions = append(formattedOptions, fmt.Sprintf("%d: text=%s", opt.(map[string]interface{})["index"], encodedText))
			}
			allOptions = append(allOptions, formattedOptions...)
		}
		frameIndex += 1
	}

	if len(allOptions) > 0 {
		msg := strings.Join(allOptions, "\n")
		msg += "\nUse the exact text string in select_dropdown_option"
		log.Debug(msg)
		actionResult := NewActionResult()
		actionResult.ExtractedContent = &msg
		actionResult.IncludeInMemory = true
		return actionResult, nil
	} else {
		msg := "No options found in any frame for dropdown"
		log.Debug(msg)
		actionResult := NewActionResult()
		actionResult.ExtractedContent = &msg
		actionResult.IncludeInMemory = true
		return actionResult, nil
	}
}

func (c *Controller) SelectDropdownOption(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	actionParams, bc, err := getActionParamsAndBrowserContext[SelectDropdownOptionAction](params, extraArgs)
	if err != nil {
		return nil, err
	}

	page := bc.GetCurrentPage()
	selectorMap := bc.GetSelectorMap()
	domElement := (*selectorMap)[actionParams.Index]
	text := actionParams.Text

	if domElement.TagName != "select" {
		msg := fmt.Sprintf("Element is not a select! Tag: %s, Attributes: %s", domElement.TagName, domElement.Attributes)
		log.Debug(msg)
		actionResult := NewActionResult()
		actionResult.ExtractedContent = &msg
		actionResult.IncludeInMemory = true
		return actionResult, nil
	}

	log.Debug(fmt.Sprintf("Attempting to select '%s' using xpath: %s", text, domElement.Xpath))
	log.Debug(fmt.Sprintf("Element attributes: %s", domElement.Attributes))
	log.Debug(fmt.Sprintf("Element tag: %s", domElement.TagName))

	// xpath := "//" + domElement.Xpath
	frameIndex := 0
	for _, frame := range page.Frames() {
		log.Debug(fmt.Sprintf("Trying frame %d URL: %s", frameIndex, frame.URL()))
		findDropdownJs := `
							(xpath) => {
								try {
									const select = document.evaluate(xpath, document, null,
										XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue;
									if (!select) return null;
									if (select.tagName.toLowerCase() !== 'select') {
										return {
											error: "Found element but it's a " + select.tagName + ", not a SELECT",
											found: false
										};
									}
									return {
										id: select.id,
										name: select.name,
										found: true,
										tagName: select.tagName,
										optionCount: select.options.length,
										currentValue: select.value,
										availableOptions: Array.from(select.options).map(o => o.text.trim())
									};
								} catch (e) {
									return {error: e.toString(), found: false};
								}
							}
						`
		dropdownInfo, err := frame.Evaluate(findDropdownJs, domElement.Xpath)
		if err != nil {
			log.Debug(fmt.Sprintf("Frame %d attempt failed: %s", frameIndex, err.Error()))
			log.Debug(fmt.Sprintf("Frame type: %T", frame))
			log.Debug(fmt.Sprintf("Frame URL: %s", frame.URL()))
		}
		if dropdownInfo, ok := dropdownInfo.(map[string]interface{}); ok {
			found, ok := dropdownInfo["found"].(bool)
			if ok && !found {
				log.Error(fmt.Sprintf("Frame %d error: %s", frameIndex, dropdownInfo["error"]))
				continue
			}
			log.Debug(fmt.Sprintf("Found dropdown in frame %d: %s", frameIndex, dropdownInfo))
			// "label" because we are selecting by text
			// nth(0) to disable error thrown by strict mode
			// timeout=1000 because we are already waiting for all network events, therefore ideally we don't need to wait a lot here (default 30s)
			selectedOptionValues, err := frame.Locator(fmt.Sprintf("//%s", domElement.Xpath)).Nth(0).SelectOption(playwright.SelectOptionValues{Labels: &[]string{text}}, playwright.LocatorSelectOptionOptions{Timeout: playwright.Float(1000.0)})
			if err != nil {
				log.Error(fmt.Sprintf("Frame %d error: %s", frameIndex, err.Error()))
				continue
			}

			msg := fmt.Sprintf("selected option %s with value %s", text, selectedOptionValues)
			log.Debug(msg + fmt.Sprintf(" in frame %d", frameIndex))

			actionResult := NewActionResult()
			actionResult.ExtractedContent = &msg
			actionResult.IncludeInMemory = true
			return actionResult, nil
		}
		frameIndex += 1
	}
	msg := fmt.Sprintf("Could not select option '%s' in any frame", text)
	log.Debug(msg)
	actionResult := NewActionResult()
	actionResult.ExtractedContent = &msg
	actionResult.IncludeInMemory = true
	return actionResult, nil
}

// TODO: implement dragdrop
func (c *Controller) DragDrop(params interface{}, extraArgs map[string]interface{}) (*ActionResult, error) {
	_, _, err := getActionParamsAndBrowserContext[DragDropAction](params, extraArgs)
	if err != nil {
		return nil, err
	}
	return NewActionResult(), nil
}
