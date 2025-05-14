package browser

import (
	"fmt"
	"strings"

	"github.com/nerdface-ai/browser-use-go/internals/dom"
)

type TabInfo struct {
	PageId       int
	Url          string
	Title        string
	ParentPageId *int
}

func (ti *TabInfo) String() string {
	if ti.ParentPageId == nil {
		return fmt.Sprintf("Tab(page_id=%d, url=%s, title=%s, parent_page_id=null)", ti.PageId, ti.Url, ti.Title)
	}
	return fmt.Sprintf("Tab(page_id=%d, url=%s, title=%s, parent_page_id=%d)", ti.PageId, ti.Url, ti.Title, *ti.ParentPageId)
}

func TabsToString(tabs []*TabInfo) string {
	var tabStrings []string
	for _, tab := range tabs {
		tabStrings = append(tabStrings, tab.String())
	}
	return strings.Join(tabStrings, ", ")
}

type GroupTabsAction struct {
	TabIds []int   `json:"tab_ids"`
	Title  string  `json:"title"`
	Color  *string `json:"color,omitempty"`
}

type UngroupTabsAction struct {
	TabIds []int `json:"tab_ids"`
}

type BrowserState struct {
	Url           string              `json:"url"`
	Title         string              `json:"title"`
	Tabs          []*TabInfo          `json:"tabs"`
	Screenshot    *string             `json:"screenshot,omitempty"`
	PixelAbove    int                 `json:"pixel_above"`
	PixelBelow    int                 `json:"pixel_below"`
	BrowserErrors []string            `json:"browser_errors"`
	ElementTree   *dom.DOMElementNode `json:"element_tree"`
	SelectorMap   *dom.SelectorMap    `json:"selector_map"`
}

type BrowserStateHistory struct {
	Url               string                   `json:"url"`
	Title             string                   `json:"title"`
	Tabs              []*TabInfo               `json:"tabs"`
	InteractedElement []*dom.DOMHistoryElement `json:"interacted_element"`
}

// BrowserError is the base error type for all browser errors.
type BrowserError struct {
	Message string `json:"message"`
}

func (e *BrowserError) Error() string {
	return e.Message
}

// URLNotAllowedError is returned when a URL is not allowed.
type URLNotAllowedError struct {
	BrowserError
}

func NewURLNotAllowedError(url string) error {
	return &URLNotAllowedError{
		BrowserError: BrowserError{
			Message: "URL not allowed: " + url,
		},
	}
}
