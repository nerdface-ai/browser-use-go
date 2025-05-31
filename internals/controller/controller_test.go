package controller_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/nerdface-ai/browser-use-go/internals/controller"
	"github.com/nerdface-ai/browser-use-go/pkg/browser"
	"github.com/nerdface-ai/browser-use-go/pkg/dotenv"
	"github.com/stretchr/testify/assert"

	"github.com/charmbracelet/log"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/playwright-community/playwright-go"
)

func initTest(t *testing.T, headless bool) (*controller.Controller, *browser.Browser, *browser.BrowserContext, playwright.Page) {
	if !headless && os.Getenv("GITHUB_ACTIONS") == "1" {
		t.Skip("skip test")
	}
	c := controller.NewController()
	b := browser.NewBrowser(browser.BrowserConfig{
		"headless": headless,
	})
	bc := b.NewContext()
	page := bc.GetCurrentPage()
	return c, b, bc, page
}

func tempFunction(_ context.Context, arg1 controller.DoneAction) (*controller.ActionResult, error) {
	b, _ := json.Marshal(arg1)
	return &controller.ActionResult{
		IsDone:           playwright.Bool(true),
		ExtractedContent: playwright.String(string(b)),
		IncludeInMemory:  true,
		Success:          playwright.Bool(true),
	}, nil
}

func TestNewController(t *testing.T) {
	c := controller.NewController()
	t.Log(c)
	if len(c.Registry.Registry.Actions) != 19 {
		t.Error("expected 19 actions, got", len(c.Registry.Registry.Actions))
	}
}

func TestRegisterAction(t *testing.T) {
	c := &controller.Controller{
		Registry: controller.NewRegistry(),
	}
	t.Log(c)
	if len(c.Registry.Registry.Actions) != 0 {
		t.Error("expected 0 actions, got", len(c.Registry.Registry.Actions))
	}
	controller.RegisterAction(c, "InputTextAction", "input text", tempFunction, []string{}, nil)
	if len(c.Registry.Registry.Actions) != 1 {
		t.Error("expected 1 action, got", len(c.Registry.Registry.Actions))
	}
	controller.RegisterAction(c, "DoneAction", "done action", tempFunction, []string{}, nil)
	if len(c.Registry.Registry.Actions) != 2 {
		t.Error("expected 2 actions, got", len(c.Registry.Registry.Actions))
	}
}

// func TestExecuteActionInvalidSchema(t *testing.T) {
// 	c, b, bc, _ := initTest()
// 	defer b.Close()
// 	defer bc.Close()
// 	_, err := c.ExecuteAction(&controller.ActionModel{
// 		Actions: map[string]interface{}{
// 			"InputTextAction": map[string]interface{}{
// 				"text": "test",
// 			},
// 		},
// 	}, bc, nil, nil, nil)
// 	if err == nil || err.Error() != "invalid schema" {
// 		t.Error("this should be error with 'invalid schema', but get ", err)
// 	}
// }

func TestDone(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	c, b, bc, _ := initTest(t, true)
	defer b.Close()
	defer bc.Close()
	actionResult, err := c.ExecuteAction(&controller.ActModel{
		"done": map[string]interface{}{
			"success": true,
			"text":    "test",
		},
	}, nil, nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	t.Logf("actionResult: %v", actionResult)
	if actionResult.IsDone == nil || *actionResult.IsDone != true {
		t.Error("expected is_done to be true, got", actionResult.IsDone)
	}
	if actionResult.Success == nil || *actionResult.Success != true {
		t.Error("expected success to be true, got", actionResult.Success)
	}
	if actionResult.ExtractedContent == nil || *actionResult.ExtractedContent != "test" {
		t.Error("expected extracted_content to be 'test', got", actionResult.ExtractedContent)
	}
}

func TestExecuteClickElement(t *testing.T) {
	c, b, bc, _ := initTest(t, true)
	defer b.Close()
	defer bc.Close()

	bc.NavigateTo("https://www.naver.com")
	time.Sleep(1 * time.Second)

	// ------------------ buildDomTree.js -> set SelectorMap --------------------------
	// this will be done in Agent.Step() later
	currentState := bc.GetState(false)
	time.Sleep(1 * time.Second)

	session := bc.GetSession()
	session.CachedState = currentState
	// ------------------ buildDomTree.js -> set SelectorMap --------------------------

	actionModel := &controller.ActModel{
		"click_element_by_index": map[string]interface{}{
			"index": 8,
		},
	}

	result, err := c.ExecuteAction(actionModel, bc, nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	json, _ := json.Marshal(result)
	t.Log(string(json))
}

func TestExecuteInputText(t *testing.T) {
	c, b, bc, _ := initTest(t, true)
	defer b.Close()
	defer bc.Close()

	bc.NavigateTo("https://www.google.com")
	time.Sleep(1 * time.Second)

	// ------------------ buildDomTree.js -> set SelectorMap --------------------------
	// this will be done in Agent.Step() later
	currentState := bc.GetState(false)
	time.Sleep(1 * time.Second)

	session := bc.GetSession()
	session.CachedState = currentState
	// ------------------ buildDomTree.js -> set SelectorMap --------------------------

	selectorMap := bc.GetSelectorMap()
	key := -1
	for k, v := range *selectorMap {
		if v.TagName == "textarea" {
			key = k
			break
		}
	}
	if key == -1 {
		t.Error("textarea not found")
		return
	}

	actionModel := &controller.ActModel{
		"input_text": map[string]interface{}{
			"index": key,
			"text":  "Seoul weather",
		},
	}

	result, err := c.ExecuteAction(actionModel, bc, nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	json, _ := json.Marshal(result)
	t.Log(string(json))
}

func TestSearchGoogle(t *testing.T) {
	c, b, bc, _ := initTest(t, true)
	defer b.Close()
	defer bc.Close()

	_, err := c.ExecuteAction(&controller.ActModel{
		"search_google": map[string]interface{}{
			"query": "Seoul weather",
		},
	}, bc, nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	url := bc.GetCurrentPage().URL()
	// looks like fails in headless mode
	if !(strings.Contains(url, "https://www.google.com/search") && strings.Contains(url, "Seoul") && strings.Contains(url, "weather")) {
		t.Error("expected google search page, got", url)
	} else {
		t.Log(url)
	}
}

func TestGoToUrl(t *testing.T) {
	c, b, bc, _ := initTest(t, true)
	defer b.Close()
	defer bc.Close()

	_, err := c.ExecuteAction(&controller.ActModel{
		"go_to_url": map[string]interface{}{
			"url": "https://www.duckduckgo.com",
		},
	}, bc, nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	url := bc.GetCurrentPage().URL()
	if !strings.Contains(url, "duckduckgo.com") {
		t.Error("expected duckduckgo.com, got", url)
	}
}

func TestGoBack(t *testing.T) {
	c, b, bc, _ := initTest(t, true)
	defer b.Close()
	defer bc.Close()

	bc.NavigateTo("https://www.duckduckgo.com")
	time.Sleep(1 * time.Second)
	bc.NavigateTo("https://www.google.com")
	time.Sleep(1 * time.Second)
	_, err := c.ExecuteAction(&controller.ActModel{
		"go_back": map[string]interface{}{},
	}, bc, nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	url := bc.GetCurrentPage().URL()
	if !strings.Contains(url, "duckduckgo.com") {
		t.Error("expected duckduckgo.com, got", url)
	}
}

func TestWait(t *testing.T) {
	c, b, bc, _ := initTest(t, true)
	defer b.Close()
	defer bc.Close()

	startTime := time.Now()
	_, err := c.ExecuteAction(&controller.ActModel{
		"wait": map[string]interface{}{
			"seconds": 2,
		},
	}, nil, nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	duration := time.Since(startTime)
	if duration < 2*time.Second || duration > 3*time.Second {
		t.Error("expected duration to be between 2 and 3 seconds, got", duration)
	} else {
		t.Log("wait duration", duration)
	}
}

func TestSavePdf(t *testing.T) {
	c, b, bc, _ := initTest(t, true)
	defer b.Close()
	defer bc.Close()

	page := bc.GetCurrentPage()
	page.Goto("https://deepwiki.com/browser-use/browser-use")
	page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{State: playwright.LoadStateDomcontentloaded})
	actionResult, err := c.ExecuteAction(&controller.ActModel{
		"save_pdf": map[string]interface{}{},
	}, bc, nil, nil, nil)
	if err != nil {
		t.Error(err)
		return
	}
	msg := *actionResult.ExtractedContent
	splits := strings.Split(msg, "as PDF to")
	if len(splits) != 2 {
		t.Error("expected 2 splits, got", len(splits))
		return
	}
	downloadPath := strings.TrimSpace(splits[1])
	fileInfo, err := os.Stat(downloadPath)
	if err != nil {
		t.Error("file not found", err)
		return
	}
	if fileInfo.IsDir() {
		t.Error("expected file, got directory")
		return
	}
	if fileInfo.Size() == 0 {
		t.Error("expected file size to be greater than 0")
		return
	}
	t.Log("download path:", downloadPath)
	t.Log("file size:", fileInfo.Size())
	os.Remove(downloadPath)
}

func TestOpenTab(t *testing.T) {
	c, b, bc, _ := initTest(t, true)
	defer b.Close()
	defer bc.Close()

	_, err := c.ExecuteAction(&controller.ActModel{
		"open_tab": map[string]interface{}{
			"url": "https://duckduckgo.com",
		},
	}, bc, nil, nil, nil)
	if err != nil {
		t.Error(err)
		return
	}
	tabsInfo := bc.GetTabsInfo()
	if len(tabsInfo) != 2 {
		t.Error("expected 2 tabs, got", len(tabsInfo))
		return
	}
	if tabsInfo[0].Url != "about:blank" {
		t.Error("expected about:blank, got", tabsInfo[0].Url)
		return
	}
	if !strings.Contains(tabsInfo[1].Url, "duckduckgo.com") {
		t.Error("expected duckduckgo.com, got", tabsInfo[1].Url)
		return
	}
}

func TestCloseTab(t *testing.T) {
	c, b, bc, _ := initTest(t, true)
	defer b.Close()
	defer bc.Close()

	// Tab 1: bing.com
	bc.NavigateTo("https://bing.com")
	// Tab 2: duckduckgo.com
	err := bc.CreateNewTab("https://duckduckgo.com")
	if err != nil {
		t.Error(err)
		return
	}
	err = bc.CreateNewTab("https://deepwiki.com")
	if err != nil {
		t.Error(err)
		return
	}
	tabsInfo := bc.GetTabsInfo()
	if len(tabsInfo) != 3 {
		t.Error("expected 3 tabs, got", len(tabsInfo))
		return
	}
	// Test 1: index 1
	_, err = c.ExecuteAction(&controller.ActModel{
		"close_tab": map[string]interface{}{
			"page_id": 1,
		},
	}, bc, nil, nil, nil)
	if err != nil {
		t.Error(err)
		return
	}
	tabsInfo = bc.GetTabsInfo()
	if len(tabsInfo) != 2 {
		t.Error("expected 2 tabs, got", len(tabsInfo))
		return
	}
	if !strings.Contains(tabsInfo[0].Url, "bing.com") {
		t.Error("expected bing.com, got", tabsInfo[0].Url)
		return
	}
	if !strings.Contains(tabsInfo[1].Url, "deepwiki.com") {
		t.Error("expected deepwiki.com, got", tabsInfo[1].Url)
		return
	}

	// Test 2: index -1 (last tab)
	_, err = c.ExecuteAction(&controller.ActModel{
		"close_tab": map[string]interface{}{
			"page_id": -1,
		},
	}, bc, nil, nil, nil)
	if err != nil {
		t.Error(err)
		return
	}
	tabsInfo = bc.GetTabsInfo()
	if len(tabsInfo) != 1 {
		t.Error("expected 1 tab, got", len(tabsInfo))
		return
	}
	if !strings.Contains(tabsInfo[0].Url, "bing.com") {
		t.Error("expected bing.com, got", tabsInfo[0].Url)
		return
	}
}

func TestSwitchTab(t *testing.T) {
	c, b, bc, _ := initTest(t, true)
	defer b.Close()
	defer bc.Close()

	// Tab 1: bing.com
	bc.NavigateTo("https://bing.com")
	// Tab 2: duckduckgo.com
	err := bc.CreateNewTab("https://duckduckgo.com")
	if err != nil {
		t.Error(err)
		return
	}
	err = bc.CreateNewTab("https://deepwiki.com")
	if err != nil {
		t.Error(err)
		return
	}
	tabsInfo := bc.GetTabsInfo()
	if len(tabsInfo) != 3 {
		t.Error("expected 3 tabs, got", len(tabsInfo))
		return
	}
	// Test 1: index 1
	_, err = c.ExecuteAction(&controller.ActModel{
		"switch_tab": map[string]interface{}{
			"page_id": 1,
		},
	}, bc, nil, nil, nil)
	if err != nil {
		t.Error(err)
		return
	}
	tabsInfo = bc.GetTabsInfo()
	if len(tabsInfo) != 3 {
		t.Error("expected 3 tabs, got", len(tabsInfo))
		return
	}
	currentPage := bc.GetCurrentPage()
	currentPageURL := currentPage.URL()
	if !strings.Contains(currentPageURL, "duckduckgo.com") {
		t.Error("expected duckduckgo.com, got", currentPageURL)
		return
	}

	// Test 2: index 0 (first tab)
	_, err = c.ExecuteAction(&controller.ActModel{
		"switch_tab": map[string]interface{}{
			"page_id": 0,
		},
	}, bc, nil, nil, nil)
	if err != nil {
		t.Error(err)
		return
	}
	tabsInfo = bc.GetTabsInfo()
	if len(tabsInfo) != 3 {
		t.Error("expected 3 tabs, got", len(tabsInfo))
		return
	}
	currentPage = bc.GetCurrentPage()
	currentPageURL = currentPage.URL()
	if !strings.Contains(currentPageURL, "bing.com") {
		t.Error("expected bing.com, got", currentPageURL)
		return
	}
}

func TestExtractContent(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") == "1" {
		t.Skip("skip test")
	}
	dotenv.LoadEnv("../../.env")

	c, b, bc, page := initTest(t, true)
	defer b.Close()
	defer bc.Close()
	llm, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		Model:   "gpt-4o-mini",
		Timeout: time.Second * 30,
		APIKey:  os.Getenv("OPENAI_API_KEY"),
	})
	if err != nil {
		t.Error(err)
		return
	}
	page.Goto("https://deepwiki.com/browser-use/browser-use")
	page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{State: playwright.LoadStateDomcontentloaded})
	actionResult, err := c.ExecuteAction(&controller.ActModel{
		"extract_content": map[string]interface{}{
			"goal":                   "what is the topic of this page?",
			"should_strip_link_urls": true,
		},
	}, bc, llm, nil, nil)
	if err != nil {
		t.Error(err)
	}
	extractedContent := *actionResult.ExtractedContent
	if !strings.Contains(strings.ToLower(extractedContent), "browser-use") {
		t.Error("expected extracted content to be 'the page provides an overview of the 'browser-use' framework, which enables ai agents to automate web browsing tasks by integrating language models with browser automation technology. it details the system architecture, key components, workflow, supported models, and common use cases for the framework.', but got", extractedContent)
	} else {
		t.Log("extracted content:", extractedContent)
	}
}

func TestScrollDown(t *testing.T) {
	c, b, bc, page := initTest(t, true)
	defer b.Close()
	defer bc.Close()

	page.Goto("https://deepwiki.com/browser-use/browser-use")
	page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{State: playwright.LoadStateDomcontentloaded})

	// Test 1 : no param (should scroll as page height)
	_, err := c.ExecuteAction(&controller.ActModel{
		"scroll_down": map[string]interface{}{},
	}, bc, nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	scrollYRaw, err := page.Evaluate("() => window.scrollY")
	if err != nil {
		t.Error(err)
		return
	}
	scrollY := scrollYRaw.(int)
	height := page.ViewportSize().Height //browser height
	if scrollY-height < 100 && scrollY-height > -100 {
		t.Error("expected scrollY to be height", height, ",but got", scrollY)
	}

	// Test 2 : param
	_, err = c.ExecuteAction(&controller.ActModel{
		"scroll_down": map[string]interface{}{
			"amount": 100,
		},
	}, bc, nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	scrollYRaw, err = page.Evaluate("() => window.scrollY")
	if err != nil {
		t.Error(err)
		return
	}
	scrollY2 := scrollYRaw.(int)
	if scrollY2 != scrollY+100 {
		t.Error("expected scrollY to be height", height, ",but got", scrollY2)
	}
}

func TestScrollUp(t *testing.T) {
	c, b, bc, page := initTest(t, true)
	defer b.Close()
	defer bc.Close()

	page.Goto("https://deepwiki.com/browser-use/browser-use")
	page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{State: playwright.LoadStateDomcontentloaded})

	page.Evaluate("window.scrollBy(0, 2000)")
	_, err := c.ExecuteAction(&controller.ActModel{
		"scroll_up": map[string]interface{}{
			"amount": 200,
		},
	}, bc, nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	scrollYRaw, err := page.Evaluate("() => window.scrollY")
	if err != nil {
		t.Error(err)
		return
	}
	scrollY := scrollYRaw.(int)
	if scrollY != 1800 {
		t.Error("expected scrollY to be 1800, but got", scrollY)
	}
}

func TestScrollToText(t *testing.T) {
	c, b, bc, page := initTest(t, true)
	defer b.Close()
	defer bc.Close()
	page.Goto("https://deepwiki.com/browser-use/browser-use")
	page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{State: playwright.LoadStateDomcontentloaded})
	_, err := c.ExecuteAction(&controller.ActModel{
		"scroll_to_text": map[string]interface{}{
			"text": "the primary supported models:",
		},
	}, bc, nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	scrollYRaw, err := page.Evaluate("() => window.scrollY")
	if err != nil {
		t.Error(err)
		return
	}
	scrollY := scrollYRaw.(int)
	if scrollY < 3000 {
		t.Error("expected scrollY to be greater than 3000, but got", scrollY)
	}
	t.Log("scrollY", scrollY)
}

func TestGetDropdownOptions(t *testing.T) {
	c, b, bc, _ := initTest(t, true)
	defer b.Close()
	defer bc.Close()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot get current file info")
	}
	dir := filepath.Dir(filename)
	htmlPath := filepath.Join(dir, "..", "..", "html_test", "select_page.html")
	url := "file://" + htmlPath

	bc.NavigateTo(url)
	bc.GetState(false)
	time.Sleep(1 * time.Second)

	actionResult, err := c.ExecuteAction(&controller.ActModel{
		"get_dropdown_options": map[string]interface{}{
			"index": 1,
		},
	}, bc, nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	if *actionResult.ExtractedContent != "0: text=\"--Please choose an option--\"\n1: text=\"Dog\"\n2: text=\"Cat\"\n3: text=\"Hamster\"\n4: text=\"Parrot\"\n5: text=\"Spider\"\n6: text=\"Goldfish\"\nUse the exact text string in select_dropdown_option" {
		t.Error("expected some options to be printed, got", *actionResult.ExtractedContent)
	}

	actionResult, err = c.ExecuteAction(&controller.ActModel{
		"get_dropdown_options": map[string]interface{}{
			"index": 0,
		},
	}, bc, nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	if *actionResult.ExtractedContent != "No options found in any frame for dropdown" {
		t.Error("expected 'No options found in any frame for dropdown', got", *actionResult.ExtractedContent)
	}
}

func TestSelectDropdownOption(t *testing.T) {
	c, b, bc, _ := initTest(t, true)
	defer b.Close()
	defer bc.Close()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot get current file info")
	}
	dir := filepath.Dir(filename)
	htmlPath := filepath.Join(dir, "..", "..", "html_test", "select_page.html")
	url := "file://" + htmlPath

	bc.NavigateTo(url)
	_ = bc.GetState(false)
	time.Sleep(1 * time.Second)

	actionResult, err := c.ExecuteAction(&controller.ActModel{
		"select_dropdown_option": map[string]interface{}{
			"index": 1,
			"text":  "Dog",
		},
	}, bc, nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	if *actionResult.ExtractedContent != "selected option Dog with value [dog]" {
		t.Error("expected selected option Dog... , but got", *actionResult.ExtractedContent)
	}
}

func TestSendKeys(t *testing.T) {
	// not working in headless mode
	c, b, bc, _ := initTest(t, false)
	defer b.Close()
	defer bc.Close()

	bc.NavigateTo("https://keycode.info")
	_, err := c.ExecuteAction(&controller.ActModel{
		"send_keys": map[string]interface{}{
			"keys": "t",
		},
	}, bc, nil, nil, nil)
	time.Sleep(1 * time.Second)
	if err != nil {
		t.Error(err)
	}

	page := bc.GetCurrentPage()
	text, err := page.Locator("main > h1").TextContent()
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, "JavaScript Key Code 84", text)
}

// TODO(MID): implement dragdrop test
func TestDragDrop(t *testing.T) {

}
