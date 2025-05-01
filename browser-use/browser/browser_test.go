package browser

import (
	"nerdface-ai/browser-use-go/browser-use/dom"
	"nerdface-ai/browser-use-go/browser-use/utils"
	"strings"
	"testing"
)

func TestNewBrowser(t *testing.T) {
	browser := NewBrowser(BrowserConfig{
		"headless": true,
	})
	defer browser.Close()
	bc := browser.NewContext()
	defer bc.Close()

	page := bc.GetCurrentPage()
	t.Log(page.URL())
	if page.URL() != "about:blank" {
		t.Errorf("Expected URL to be about:blank, got %s", page.URL())
	}
}

func TestNavigateTo(t *testing.T) {
	browser := NewBrowser(BrowserConfig{
		"headless": false,
	})
	defer browser.Close()
	bc := browser.NewContext()
	defer bc.Close()

	bc.NavigateTo("https://www.google.com")
	page := bc.GetCurrentPage()
	t.Log(page.URL())
	if !strings.HasPrefix(page.URL(), "https://www.google.com") {
		t.Errorf("Expected URL to be https://www.google.com, got %s", page.URL())
	}
}

func TestClickElementNode(t *testing.T) {
	browser := NewBrowser(BrowserConfig{
		"headless": false,
	})
	defer browser.Close()
	bc := browser.NewContext()
	defer bc.Close()

	bc.NavigateTo("https://example.com")

	// bc.GetState()
	// _get_updated_state()
	page := bc.GetCurrentPage()

	domService := dom.NewDomService(&page)
	focus_element := -1 // default
	content, err := domService.GetClickableElements(
		utils.GetDefaultValue(bc.Config, "highlight_elements", true),
		focus_element,
		utils.GetDefaultValue(bc.Config, "viewport_expansion", 0),
	)
	// time.Sleep(100000 * time.Millisecond)
	t.Log(page.URL())
	t.Log("content", content)
	if err != nil {
		t.Errorf("Failed to get clickable elements: %s", err)
	}

	tabsInfo := bc.GetTabsInfo()

	// TODO
	// screenshot_b64 = await self.take_screenshot()
	// pixels_above, pixels_below = await self.get_scroll_info(page)

	title, _ := page.Title()
	// updated_state
	currentState := BrowserState{
		ElementTree:   content.ElementTree,
		SelectorMap:   content.SelectorMap,
		Url:           page.URL(),
		Title:         title,
		Tabs:          tabsInfo,
		Screenshot:    nil, // TODO
		PixelAbove:    0,   // TODO
		PixelBelow:    0,   // TODO
		BrowserErrors: []string{},
	}

	session := bc.GetSession()
	session.CachedState = &currentState

	processor := &dom.ClickableElementProcessor{}

	clickableElements := processor.GetClickableElements(content.ElementTree)

	t.Log("clickableElements", clickableElements)

	bc.ClickElementNode(clickableElements[0])
}
