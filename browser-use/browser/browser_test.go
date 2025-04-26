package browser

import (
	"testing"
)

func TestNewBrowser(t *testing.T) {
	browser := NewBrowser(nil)
	bc := browser.NewContext()

	page := bc.GetCurrentPage()
	print(page.URL())

	browser.Close()
}

func TestNavigateTo(t *testing.T) {
	browser := NewBrowser(nil)
	bc := browser.NewContext()

	bc.NavigateTo("https://www.google.com")
	page := bc.GetCurrentPage()
	print(page.URL())

	browser.Close()
}
