package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/nerdface-ai/browser-use-go/pkg/browser"

	"github.com/charmbracelet/log"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/cloudwego/eino/components/model"
	einoUtils "github.com/cloudwego/eino/components/tool/utils"
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

func getBrowserContext(ctx context.Context) (*browser.BrowserContext, error) {
	if bc, ok := ctx.Value(browserKey).(*browser.BrowserContext); ok {
		return bc, nil
	}
	return nil, errors.New("browserContext is not found")
}

type Controller struct {
	Registry *Registry
}

func NewController() *Controller {
	c := &Controller{
		Registry: NewRegistry(),
	}
	RegisterAction(c, "done", "Complete task - with return text and if the task is finished (success=True) or not yet  completely finished (success=False), because last step is reached", c.Done, []string{}, nil)
	RegisterAction(c, "click_element_by_index", "Click element by index", c.ClickElementByIndex, []string{}, nil)
	RegisterAction(c, "input_text", "Input text into a input interactive element", c.InputText, []string{}, nil)
	RegisterAction(c, "search_google", "Search the query in Google in the current tab, the query should be a search query like humans search in Google, concrete and not vague or super long. More the single most important items.", c.SearchGoogle, []string{}, nil)
	RegisterAction(c, "go_to_url", "Navigate to URL in the current tab", c.GoToUrl, []string{}, nil)
	RegisterAction(c, "go_back", "Go back to the previous page", c.GoBack, []string{}, nil)
	RegisterAction(c, "wait", "Wait for x seconds default 3", c.Wait, []string{}, nil)
	RegisterAction(c, "save_pdf", "Save the current page as a PDF file", c.SavePdf, []string{}, nil)
	RegisterAction(c, "switch_tab", "Switch tab", c.SwitchTab, []string{}, nil)
	RegisterAction(c, "open_tab", "Open url in new tab", c.OpenTab, []string{}, nil)
	RegisterAction(c, "close_tab", "Close an existing tab", c.CloseTab, []string{}, nil)
	RegisterAction(c, "extract_content", "Extract page content to retrieve specific information from the page, e.g. all company names, a specific description, all information about, links with companies in structured format or simply links", c.ExtractContent, []string{}, nil)
	RegisterAction(c, "scroll_down", "Scroll down the page by pixel amount - if no amount is specified, scroll down one page", c.ScrollDown, []string{}, nil)
	RegisterAction(c, "scroll_up", "Scroll up the page by pixel amount - if no amount is specified, scroll up one page", c.ScrollUp, []string{}, nil)
	RegisterAction(c, "send_keys", "Send strings of special keys like Escape,Backspace, Insert, PageDown, Delete, Enter, Shortcuts such as `Control+o`, `Control+Shift+T` are supported as well. This gets used in keyboard.press.", c.SendKeys, []string{}, nil)
	RegisterAction(c, "scroll_to_text", "If you dont find something which you want to interact with, scroll to it", c.ScrollToText, []string{}, nil)
	RegisterAction(c, "get_dropdown_options", "Get all options from a native dropdown", c.GetDropdownOptions, []string{}, nil)
	RegisterAction(c, "select_dropdown_option", "Select dropdown option for interactive element index by the text of the option you want to select", c.SelectDropdownOption, []string{}, nil)
	RegisterAction(c, "drag_drop", "Drag and drop elements or between coordinates on the page - useful for canvas drawing, sortable lists, sliders, file uploads, and UI rearrangement", c.DragDrop, []string{}, nil)
	return c
}

func RegisterAction[T, D any](
	c *Controller,
	name string,
	description string,
	function einoUtils.InvokeFunc[T, D],
	domains []string,
	pageFilter func(playwright.Page) bool,
) error {
	if c.Registry == nil {
		return errors.New("registry is nil")
	}
	return registerAction(c.Registry, name, description, function, domains, pageFilter)
}

// Act
func (c *Controller) ExecuteAction(
	action *ActModel,
	browserContext *browser.BrowserContext,
	pageExtractionLlm model.ToolCallingChatModel,
	sensitiveData map[string]string,
	availableFilePaths []string,
	// context: Context | None,
) (*ActionResult, error) {
	for actionName, actionParams := range *action {
		ab, err := json.Marshal(actionParams)
		if err != nil {
			return nil, err
		}
		result, err := c.Registry.ExecuteAction(actionName, string(ab), browserContext, pageExtractionLlm, sensitiveData, availableFilePaths)
		if err != nil {
			return nil, err
		}
		var actionResult ActionResult
		err = json.Unmarshal([]byte(result), &actionResult)
		if err != nil {
			return nil, err
		}
		return &actionResult, nil
	}
	return NewActionResult(), nil
}

func (c *Controller) Done(_ context.Context, params DoneAction) (*ActionResult, error) {
	log.Debug("Done Action called")
	actionResult := NewActionResult()
	actionResult.IsDone = playwright.Bool(true)
	actionResult.Success = &params.Success
	actionResult.ExtractedContent = &params.Text
	return actionResult, nil
}

// ExecuteAction: action.Function(validatedParams, extraArgs)
func (c *Controller) ClickElementByIndex(ctx context.Context, params ClickElementAction) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
	if err != nil {
		return nil, err
	}
	session := bc.GetSession()
	initialPages := len(session.Context.Pages())

	elementNode, err := bc.GetDomElementByIndex(params.Index)
	if err != nil {
		return nil, err
	}

	// if element has file uploader then dont click
	if bc.IsFileUploader(elementNode, 3, 0) {
		msg := fmt.Sprintf("Index %d - has an element which opens file upload dialog. To upload files please use a specific function to upload files", params.Index)
		log.Info(msg)
		actionResult := NewActionResult()
		actionResult.ExtractedContent = &msg
		actionResult.IncludeInMemory = true
		return actionResult, nil
	}

	// TODO(HIGH): support download path in ClickElementNode
	downloadPath, err := bc.ClickElementNode(elementNode)
	if err != nil {
		return nil, err
	}

	msg := ""
	if downloadPath != nil {
		msg = fmt.Sprintf("💾  Downloaded file to %s", *downloadPath)
	} else {
		msg = fmt.Sprintf("🖱️  Clicked button with index %d: %s", params.Index, elementNode.GetAllTextTillNextClickableElement(-1))
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

func (c *Controller) InputText(ctx context.Context, params InputTextAction) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
	if err != nil {
		return nil, err
	}
	selectorMap := bc.GetSelectorMap()
	if (*selectorMap)[params.Index] == nil {
		return nil, errors.New("element with index " + strconv.Itoa(params.Index) + " does not exist")
	}

	elementNode, err := bc.GetDomElementByIndex(params.Index)
	if err != nil {
		return nil, err
	}
	bc.InputTextElementNode(elementNode, params.Text)

	msg := fmt.Sprintf("Input %s into index %d", params.Text, params.Index)

	actionResult := NewActionResult()
	actionResult.ExtractedContent = &msg
	actionResult.IncludeInMemory = true

	return actionResult, nil
}

func (c *Controller) SearchGoogle(ctx context.Context, params SearchGoogleAction) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
	if err != nil {
		return nil, err
	}
	page := bc.GetCurrentPage()
	page.Goto(fmt.Sprintf("https://www.google.com/search?q=%s&udm=14", params.Query))
	page.WaitForLoadState()
	msg := fmt.Sprintf("🔍  Searched for \"%s\" in Google", params.Query)
	log.Debug(msg)
	actionResult := NewActionResult()
	actionResult.ExtractedContent = &msg
	actionResult.IncludeInMemory = true
	return actionResult, nil
}

func (c *Controller) GoToUrl(ctx context.Context, params GoToUrlAction) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
	if err != nil {
		return nil, err
	}
	page := bc.GetCurrentPage()
	page.Goto(params.Url)
	page.WaitForLoadState()
	msg := fmt.Sprintf("🔗  Navigated to %s", params.Url)
	log.Debug(msg)
	actionResult := NewActionResult()
	actionResult.ExtractedContent = &msg
	actionResult.IncludeInMemory = true
	return actionResult, nil
}

func (c *Controller) GoBack(ctx context.Context, params GoBackAction) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
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

func (c *Controller) Wait(ctx context.Context, params WaitAction) (*ActionResult, error) {
	msg := fmt.Sprintf("🕒  Waiting for %d seconds", params.Seconds)
	log.Debug(msg)
	time.Sleep(time.Duration(params.Seconds) * time.Second)
	actionResult := NewActionResult()
	actionResult.ExtractedContent = &msg
	actionResult.IncludeInMemory = true
	return actionResult, nil
}

func (c *Controller) SavePdf(ctx context.Context, params SavePdfAction) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
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

func (c *Controller) OpenTab(ctx context.Context, params OpenTabAction) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
	if err != nil {
		return nil, err
	}
	err = bc.CreateNewTab(params.Url)
	if err != nil {
		return nil, err
	}
	msg := fmt.Sprintf("🔗  Opened new tab with %s", params.Url)
	log.Debug(msg)
	actionResult := NewActionResult()
	actionResult.ExtractedContent = &msg
	actionResult.IncludeInMemory = true
	return actionResult, nil
}

func (c *Controller) CloseTab(ctx context.Context, params CloseTabAction) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
	if err != nil {
		return nil, err
	}
	bc.SwitchToTab(params.PageId)
	page := bc.GetCurrentPage()
	page.WaitForLoadState()
	url := page.URL()
	err = page.Close()
	if err != nil {
		return nil, err
	}
	msg := fmt.Sprintf("❌  Closed tab %d with url %s", params.PageId, url)
	log.Debug(msg)
	actionResult := NewActionResult()
	actionResult.ExtractedContent = &msg
	actionResult.IncludeInMemory = true
	return actionResult, nil
}

func (c *Controller) SwitchTab(ctx context.Context, params SwitchTabAction) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
	if err != nil {
		return nil, err
	}
	bc.SwitchToTab(params.PageId)
	page := bc.GetCurrentPage()
	page.WaitForLoadState()
	msg := fmt.Sprintf("🔄  Switched to tab %d", params.PageId)
	log.Debug(msg)
	actionResult := NewActionResult()
	actionResult.ExtractedContent = &msg
	actionResult.IncludeInMemory = true
	return actionResult, nil
}

func (c *Controller) ExtractContent(ctx context.Context, params ExtractContentAction) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
	if err != nil {
		return nil, err
	}
	var llm model.ToolCallingChatModel
	if model, ok := ctx.Value(pageExtractionLlmKey).(model.ToolCallingChatModel); ok {
		llm = model
	} else {
		return nil, errors.New("page_extraction_llm is not found")
	}
	page := bc.GetCurrentPage()

	strip := []string{}
	if params.ShouldStripLinkUrls {
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

	content, err := conv.ConvertString(pageContent)
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

	prompt := fmt.Sprintf("Your task is to extract the content of the page. You will be given a page and a goal and you should extract all relevant information around this goal from the page. If the goal is vague, summarize the page. Respond in json format. Extraction goal: %s, Page: %s", params.Goal, content)
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

func (c *Controller) ScrollDown(ctx context.Context, params ScrollDownAction) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
	if err != nil {
		return nil, err
	}

	page := bc.GetCurrentPage()
	amount := "one page"
	if params.Amount != nil {
		page.Evaluate(fmt.Sprintf("window.scrollBy(0, %d);", *params.Amount))
		amount = fmt.Sprintf("%d pixels", *params.Amount)
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

func (c *Controller) ScrollUp(ctx context.Context, params ScrollUpAction) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
	if err != nil {
		return nil, err
	}
	page := bc.GetCurrentPage()
	var amount string
	if params.Amount != nil {
		page.Evaluate(fmt.Sprintf("window.scrollBy(0, -%d);", *params.Amount))
		amount = fmt.Sprintf("%d pixels", *params.Amount)
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

func (c *Controller) SendKeys(ctx context.Context, params SendKeysAction) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
	if err != nil {
		return nil, err
	}

	page := bc.GetCurrentPage()
	err = page.Keyboard().InsertText(params.Keys)
	if err != nil {
		if strings.Contains(err.Error(), "Unknown key") {
			for _, key := range params.Keys {
				err = page.Keyboard().Press(string(key))
				if err != nil {
					return nil, err
				}
			}
		} else {
			return nil, err
		}
	}
	msg := fmt.Sprintf("⌨️  Sent keys: %s", params.Keys)
	log.Debug(msg)
	actionResult := NewActionResult()
	actionResult.ExtractedContent = &msg
	actionResult.IncludeInMemory = true
	return actionResult, nil
}

func (c *Controller) ScrollToText(ctx context.Context, params ScrollToTextAction) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
	if err != nil {
		return nil, err
	}
	page := bc.GetCurrentPage()
	// Try different locator strategies
	locators := []playwright.Locator{
		page.GetByText(params.Text, playwright.PageGetByTextOptions{Exact: playwright.Bool(false)}),
		page.Locator(fmt.Sprintf("text=%s", params.Text)),
		page.Locator(fmt.Sprintf("//*[contains(text(), '%s')]", params.Text)),
	}

	for _, locator := range locators {
		if visible, err := locator.First().IsVisible(); err == nil && visible {
			err := locator.First().ScrollIntoViewIfNeeded()
			if err != nil {
				log.Debug(fmt.Sprintf("Locator attempt failed: %s", err.Error()))
				continue
			}
			time.Sleep(500 * time.Millisecond)
			msg := fmt.Sprintf("🔍  Scrolled to text: %s", params.Text)
			log.Debug(msg)
			actionResult := NewActionResult()
			actionResult.ExtractedContent = &msg
			actionResult.IncludeInMemory = true
			return actionResult, nil
		}
	}

	msg := fmt.Sprintf("Text '%s' not found or not visible on page", params.Text)
	log.Debug(msg)
	actionResult := NewActionResult()
	actionResult.ExtractedContent = &msg
	actionResult.IncludeInMemory = true
	return actionResult, nil
}

func (c *Controller) GetDropdownOptions(ctx context.Context, params GetDropdownOptionsAction) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
	if err != nil {
		return nil, err
	}
	page := bc.GetCurrentPage()
	selectorMap := bc.GetSelectorMap()
	domElement := (*selectorMap)[params.Index]

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

func (c *Controller) SelectDropdownOption(ctx context.Context, params SelectDropdownOptionAction) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
	if err != nil {
		return nil, err
	}

	page := bc.GetCurrentPage()
	selectorMap := bc.GetSelectorMap()
	domElement := (*selectorMap)[params.Index]
	text := params.Text

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

// Performs a precise drag and drop operation between elements or coordinates.
func (c *Controller) DragDrop(ctx context.Context, params DragDropAction) (*ActionResult, error) {
	bc, err := getBrowserContext(ctx)
	if err != nil {
		return nil, err
	}
	page := bc.GetCurrentPage()

	// Initialize variables
	var sourceX *int = nil
	var sourceY *int = nil
	var targetX *int = nil
	var targetY *int = nil
	steps := 10
	if params.Steps != nil {
		steps = *params.Steps
	}
	delayMs := 5
	if params.DelayMs != nil {
		delayMs = *params.DelayMs
	}
	// Case 1: Element selectors provided
	if params.ElementSource != nil && params.ElementTarget != nil {
		log.Debug("Using element-based approach with selectors")
		sourceElement, targetElement := getDragElements(
			page,
			*params.ElementSource,
			*params.ElementTarget,
		)

		if sourceElement == nil || targetElement == nil {
			errorMsgSub := "target"
			if sourceElement == nil {
				errorMsgSub = "source"
			}
			errorMsg := fmt.Sprintf("Failed to find %s element", errorMsgSub)
			actionResult := NewActionResult()
			actionResult.Error = &errorMsg
			actionResult.IncludeInMemory = true
			return actionResult, nil
		}

		sourceCoords, targetCoords, err := getElementCoordinates(
			sourceElement,
			targetElement,
			params.ElementSourceOffset,
			params.ElementTargetOffset,
		)

		if err != nil {
			errorMsg := fmt.Sprintf("Failed to perform drag and drop: %s", err.Error())
			actionResult := NewActionResult()
			actionResult.Error = &errorMsg
			actionResult.IncludeInMemory = true
			return actionResult, nil
		}

		if sourceCoords == nil || targetCoords == nil {
			errorMsgSub := "source"
			if sourceCoords == nil {
				errorMsgSub = "target"
			}
			errorMsg := fmt.Sprintf("Failed to determine %s coordinates", errorMsgSub)
			actionResult := NewActionResult()
			actionResult.Error = &errorMsg
			actionResult.IncludeInMemory = true
			return actionResult, nil
		}
		sourceX = playwright.Int(sourceCoords.X)
		sourceY = playwright.Int(sourceCoords.Y)
		targetX = playwright.Int(targetCoords.X)
		targetY = playwright.Int(targetCoords.Y)
	} else if params.CoordSourceX != nil && params.CoordSourceY != nil && params.CoordTargetX != nil && params.CoordTargetY != nil {
		// Case 2: Coordinates provided directly
		sourceX = params.CoordSourceX
		sourceY = params.CoordSourceY
		targetX = params.CoordTargetX
		targetY = params.CoordTargetY
	} else {
		errorMsg := "Must provide either source/target selectors or source/target coordinates"
		actionResult := NewActionResult()
		actionResult.Error = &errorMsg
		actionResult.IncludeInMemory = true
		return actionResult, nil
	}

	// Validate coordinates
	if sourceX == nil || sourceY == nil || targetX == nil || targetY == nil {
		errorMsg := "Failed to determine source or target coordinates"
		actionResult := NewActionResult()
		actionResult.Error = &errorMsg
		actionResult.IncludeInMemory = true
		return actionResult, nil
	}

	// Perform the drag operation
	success, message := executeDragOperation(page, *sourceX, *sourceY, *targetX, *targetY, steps, delayMs)
	if !success {
		log.Errorf("Drag operation failed: %s", message)
		actionResult := NewActionResult()
		actionResult.Error = &message
		actionResult.IncludeInMemory = true
		return actionResult, nil
	}

	// Create descriptive message
	var msg string
	if params.ElementSource != nil && params.ElementTarget != nil {
		msg = fmt.Sprintf("🖱️ Dragged element '%s' to '%s'", *params.ElementSource, *params.ElementTarget)
	} else {
		msg = fmt.Sprintf("🖱️ Dragged from (%d, %d) to (%d, %d)", *sourceX, *sourceY, *targetX, *targetY)
	}

	log.Info(msg)
	actionResult := NewActionResult()
	actionResult.ExtractedContent = &msg
	actionResult.IncludeInMemory = true
	return actionResult, nil
}

// Get source and target elements with appropriate error handling.
func getDragElements(
	page playwright.Page,
	sourceSelector string,
	targetSelector string,
) (playwright.Locator, playwright.Locator) {
	var source playwright.Locator = nil
	var target playwright.Locator = nil
	sourceLocator := page.Locator(sourceSelector)
	targetLocator := page.Locator(targetSelector)
	sourceCount, _ := sourceLocator.Count()
	targetCount, _ := targetLocator.Count()
	if sourceCount > 0 {
		source = sourceLocator.First()
		log.Debug(fmt.Sprintf("Found source element with selector: %s", sourceSelector))
	} else {
		log.Warn(fmt.Sprintf("Source element not found: %s", sourceSelector))
	}
	if targetCount > 0 {
		target = targetLocator.First()
		log.Debug(fmt.Sprintf("Found target element with selector: %s", targetSelector))
	} else {
		log.Warn(fmt.Sprintf("Target element not found: %s", targetSelector))
	}
	return source, target
}

// Get coordinates from elements with appropriate error handling.
func getElementCoordinates(
	sourceLocator playwright.Locator,
	targetLocator playwright.Locator,
	sourcePosition *Position,
	targetPosition *Position,
) (*Position, *Position, error) {
	var sourceCoords *Position = nil
	var targetCoords *Position = nil

	// Get source coordinates
	if sourcePosition != nil {
		sourceCoords = sourcePosition
	} else {
		sourceBox, err := sourceLocator.BoundingBox()
		if err != nil {
			return nil, nil, err
		}
		if sourceBox != nil {
			_sourceCoords := Position{
				X: int(sourceBox.X + sourceBox.Width/2),
				Y: int(sourceBox.Y + sourceBox.Height/2),
			}
			sourceCoords = &_sourceCoords
		}
	}
	// Get target coordinates
	if targetPosition != nil {
		targetCoords = targetPosition
	} else {
		targetBox, err := targetLocator.BoundingBox()
		if err != nil {
			return nil, nil, err
		}
		if targetBox != nil {
			_targetCoords := Position{
				X: int(targetBox.X + targetBox.Width/2),
				Y: int(targetBox.Y + targetBox.Height/2),
			}
			targetCoords = &_targetCoords
		}
	}

	return sourceCoords, targetCoords, nil
}

// Execute the drag operation with comprehensive error handling.
func executeDragOperation(
	page playwright.Page,
	sourceX int,
	sourceY int,
	targetX int,
	targetY int,
	steps int,
	delayMs int,
) (bool, string) {
	// Try to move to source position
	err := page.Mouse().Move(float64(sourceX), float64(sourceY))
	if err != nil {
		log.Error(fmt.Sprintf("Failed to move to source position: %s", err.Error()))
		return false, fmt.Sprintf("Failed to move to source position: %s", err.Error())
	}

	// Press mouse button down
	err = page.Mouse().Down()
	if err != nil {
		log.Error(fmt.Sprintf("Failed to press mouse button down: %s", err.Error()))
		return false, fmt.Sprintf("Failed to press mouse button down: %s", err.Error())
	}

	// Move to target position with intermediate steps
	for i := 1; i <= steps; i++ {
		ratio := float64(i) / float64(steps)
		intermediateX := int(float64(sourceX) + float64(targetX-sourceX)*ratio)
		intermediateY := int(float64(sourceY) + float64(targetY-sourceY)*ratio)

		err = page.Mouse().Move(float64(intermediateX), float64(intermediateY))
		if err != nil {
			log.Error(fmt.Sprintf("Failed to move to intermediate position: %s", err.Error()))
			return false, fmt.Sprintf("Failed to move to intermediate position: %s", err.Error())
		}

		if delayMs > 0 {
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
		}
	}

	// Move to final target position
	err = page.Mouse().Move(float64(targetX), float64(targetY))
	if err != nil {
		log.Error(fmt.Sprintf("Failed to move to target position: %s", err.Error()))
		return false, fmt.Sprintf("Failed to move to target position: %s", err.Error())
	}

	// Move again to ensure dragover events are properly triggered
	err = page.Mouse().Move(float64(targetX), float64(targetY))
	if err != nil {
		log.Error(fmt.Sprintf("Failed to move to target position: %s", err.Error()))
		return false, fmt.Sprintf("Failed to move to target position: %s", err.Error())
	}

	// Release mouse button
	err = page.Mouse().Up()
	if err != nil {
		log.Error(fmt.Sprintf("Failed to release mouse button: %s", err.Error()))
		return false, fmt.Sprintf("Failed to release mouse button: %s", err.Error())
	}
	return true, "Drag operation completed successfully"
}
