package browser

import (
	"strings"
	"testing"
	"time"

	"github.com/nerdface-ai/browser-use-go/internals/dom"
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
		"headless": true,
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
		"headless": true,
	})
	defer browser.Close()
	bc := browser.NewContext()
	defer bc.Close()

	bc.NavigateTo("https://example.com")

	currentState := bc.GetState(false)
	time.Sleep(1 * time.Second)

	session := bc.GetSession()
	session.CachedState = currentState

	processor := &dom.ClickableElementProcessor{}

	clickableElements := processor.GetClickableElements(currentState.ElementTree)

	t.Log("clickableElements", clickableElements)

	if len(clickableElements) == 0 {
		t.Log("No clickable elements found")
		return
	}
	bc.ClickElementNode(clickableElements[0])
	time.Sleep(1 * time.Second)
}

func TestInputTextElementNode(t *testing.T) {
	browser := NewBrowser(BrowserConfig{
		"headless": true,
	})
	defer browser.Close()
	bc := browser.NewContext()
	defer bc.Close()

	bc.NavigateTo("https://www.google.com")

	// ------- test -------
	currentState := bc.GetState(false)
	time.Sleep(1 * time.Second)

	session := bc.GetSession()
	session.CachedState = currentState

	// ------- test -------
	selectorMap := bc.GetSelectorMap()

	inputElement := (*selectorMap)[6]
	bc.InputTextElementNode(inputElement, "Golang")
}
