package browser

import (
	"nerdface-ai/browser-use-go/browser-use/dom"

	"github.com/moznion/go-optional"
)

type TabInfo struct {
	PageId       int
	Url          string
	Title        string
	ParentPageId optional.Option[int]
}

type GroupTabsAction struct {
	TabIds []int
	Title  string
	Color  optional.Option[string]
}

type UngroupTabsAction struct {
	TabIds []int
}

type BrowserState struct {
	Url           string
	Title         string
	Tabs          []*TabInfo
	Screenshot    optional.Option[string]
	PixelAbove    int
	PixelBelow    int
	BrowserErrors []string
	ElementTree   *dom.DOMElementNode
	SelectorMap   *dom.SelectorMap
}

type BrowserStateHistory struct {
	Url               string
	Title             string
	Tabs              []*TabInfo
	InteractedElement []*dom.DOMHistoryElement
}

// BrowserError is the base error type for all browser errors.
type BrowserError struct {
	Message string
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
