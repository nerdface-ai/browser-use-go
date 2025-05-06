package controller_test

import (
	"encoding/json"
	"nerdface-ai/browser-use-go/browser-use/browser"
	"nerdface-ai/browser-use-go/browser-use/controller"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/moznion/go-optional"
	"github.com/playwright-community/playwright-go"
)

func tempFunction(arg1 interface{}, arg2 map[string]interface{}) (*controller.ActionResult, error) {
	b, _ := json.Marshal(arg1)
	return &controller.ActionResult{
		IsDone:           optional.Some(true),
		ExtractedContent: optional.Some(string(b)),
		IncludeInMemory:  true,
		Success:          optional.Some(true),
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
	c.RegisterAction("InputTextAction", "input text", controller.InputTextAction{}, tempFunction, []string{}, nil)
	if len(c.Registry.Registry.Actions) != 1 {
		t.Error("expected 1 action, got", len(c.Registry.Registry.Actions))
	}
	c.RegisterAction("DoneAction", "done action", controller.DoneAction{}, tempFunction, []string{}, nil)
	if len(c.Registry.Registry.Actions) != 2 {
		t.Error("expected 2 actions, got", len(c.Registry.Registry.Actions))
	}
}

func TestExecuteActionInvalidSchema(t *testing.T) {
	c := controller.NewController()
	b := browser.NewBrowser(browser.BrowserConfig{
		"headless": true,
	})
	defer b.Close()
	bc := b.NewContext()
	defer bc.Close()
	_, err := c.ExecuteAction(&controller.ActionModel{
		Actions: map[string]interface{}{
			"InputTextAction": map[string]interface{}{
				"text": "test",
			},
		},
	}, bc, nil, nil, nil)
	if err == nil || err.Error() != "invalid schema" {
		t.Error("this should be error with 'invalid schema', but get ", err)
	}
}

func TestDone(t *testing.T) {
	c := controller.NewController()
	actionResult, err := c.ExecuteAction(&controller.ActionModel{
		Actions: map[string]interface{}{
			"DoneAction": map[string]interface{}{
				"success": true,
				"text":    "test",
			},
		},
	}, nil, nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	if actionResult.IsDone.Unwrap() != true {
		t.Error("expected is_done to be true, got", actionResult.IsDone.Unwrap())
	}
	if actionResult.Success.Unwrap() != true {
		t.Error("expected success to be true, got", actionResult.Success.Unwrap())
	}
	if actionResult.ExtractedContent.Unwrap() != "test" {
		t.Error("expected extracted_content to be 'test', got", actionResult.ExtractedContent.Unwrap())
	}
}

func TestExecuteClickElement(t *testing.T) {
	c := controller.NewController()
	b := browser.NewBrowser(browser.BrowserConfig{
		"headless": true,
	})
	defer b.Close()
	bc := b.NewContext()
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

	actionModel := &controller.ActionModel{
		Actions: map[string]interface{}{
			"ClickElementAction": map[string]interface{}{
				"index": 8, // 0
			},
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
	c := controller.NewController()
	b := browser.NewBrowser(browser.BrowserConfig{
		"headless": true,
	})
	defer b.Close()
	bc := b.NewContext()
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

	actionModel := &controller.ActionModel{
		Actions: map[string]interface{}{
			"InputTextAction": map[string]interface{}{
				"index": key,
				"text":  "Seoul weather",
			},
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
	c := controller.NewController()
	b := browser.NewBrowser(browser.BrowserConfig{
		"headless": true,
	})
	defer b.Close()
	bc := b.NewContext()
	defer bc.Close()
	_, err := c.ExecuteAction(&controller.ActionModel{
		Actions: map[string]interface{}{
			"SearchGoogleAction": map[string]interface{}{
				"query": "Seoul weather",
			},
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
	c := controller.NewController()
	b := browser.NewBrowser(browser.BrowserConfig{
		"headless": true,
	})
	defer b.Close()
	bc := b.NewContext()
	defer bc.Close()
	_, err := c.ExecuteAction(&controller.ActionModel{
		Actions: map[string]interface{}{
			"GoToUrlAction": map[string]interface{}{
				"url": "https://www.duckduckgo.com",
			},
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
	c := controller.NewController()
	b := browser.NewBrowser(browser.BrowserConfig{
		"headless": true,
	})
	defer b.Close()
	bc := b.NewContext()
	defer bc.Close()
	bc.NavigateTo("https://www.duckduckgo.com")
	time.Sleep(1 * time.Second)
	bc.NavigateTo("https://www.google.com")
	time.Sleep(1 * time.Second)
	_, err := c.ExecuteAction(&controller.ActionModel{
		Actions: map[string]interface{}{
			"GoBackAction": map[string]interface{}{},
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

func TestWait(t *testing.T) {
	c := controller.NewController()
	startTime := time.Now()
	_, err := c.ExecuteAction(&controller.ActionModel{
		Actions: map[string]interface{}{
			"WaitAction": map[string]interface{}{
				"seconds": 2,
			},
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
	c := controller.NewController()
	b := browser.NewBrowser(browser.BrowserConfig{
		"headless": true,
	})
	defer b.Close()
	bc := b.NewContext()
	defer bc.Close()
	page := bc.GetCurrentPage()
	page.Goto("https://deepwiki.com/browser-use/browser-use")
	page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{State: playwright.LoadStateDomcontentloaded})
	actionResult, err := c.ExecuteAction(&controller.ActionModel{
		Actions: map[string]interface{}{
			"SavePdfAction": map[string]interface{}{},
		},
	}, bc, nil, nil, nil)
	if err != nil {
		t.Error(err)
		return
	}
	msg := actionResult.ExtractedContent.Unwrap()
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
	c := controller.NewController()
	b := browser.NewBrowser(browser.BrowserConfig{
		"headless": true,
	})
	defer b.Close()
	bc := b.NewContext()
	defer bc.Close()
	_, err := c.ExecuteAction(&controller.ActionModel{
		Actions: map[string]interface{}{
			"OpenTabAction": map[string]interface{}{
				"url": "https://duckduckgo.com",
			},
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
	c := controller.NewController()
	b := browser.NewBrowser(browser.BrowserConfig{
		"headless": true,
	})
	defer b.Close()
	bc := b.NewContext()
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
	_, err = c.ExecuteAction(&controller.ActionModel{
		Actions: map[string]interface{}{
			"CloseTabAction": map[string]interface{}{
				"page_id": 1,
			},
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
	_, err = c.ExecuteAction(&controller.ActionModel{
		Actions: map[string]interface{}{
			"CloseTabAction": map[string]interface{}{
				"page_id": -1,
			},
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
	c := controller.NewController()
	b := browser.NewBrowser(browser.BrowserConfig{
		"headless": true,
	})
	defer b.Close()
	bc := b.NewContext()
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
	_, err = c.ExecuteAction(&controller.ActionModel{
		Actions: map[string]interface{}{
			"SwitchTabAction": map[string]interface{}{
				"page_id": 1,
			},
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
	_, err = c.ExecuteAction(&controller.ActionModel{
		Actions: map[string]interface{}{
			"SwitchTabAction": map[string]interface{}{
				"page_id": 0,
			},
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

}

func TestScrollDown(t *testing.T) {

}

func TestScrollUp(t *testing.T) {

}

func TestSendKeys(t *testing.T) {

}

func TestScrollToText(t *testing.T) {

}

func TestGetDropdownOptions(t *testing.T) {

}

func TestSelectDropdownOption(t *testing.T) {

}

// TODO: implement dragdrop
func TestDragDrop(t *testing.T) {

}
