package browser

import (
	"strings"
	"testing"
)

func TestNewBrowser(t *testing.T) {
	browser := NewBrowser(nil)
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
	browser := NewBrowser(nil)
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
