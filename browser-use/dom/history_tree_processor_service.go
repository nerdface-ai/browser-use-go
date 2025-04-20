package dom

import (
	"nerdface-ai/browser-use-go/browser-use/utils"

	"github.com/moznion/go-optional"
)

type HistoryTreeProcessor struct {
}

func (h *HistoryTreeProcessor) convertDomElementToHistoryElement(domElement *DOMElementNode) *DOMHistoryElement {
	parentBranchPath := h.getParentBranchPath(domElement)
	cssSelector := EnhancedCssSelectorForElement(domElement, false)
	return &DOMHistoryElement{
		TagName:                domElement.TagName,
		Xpath:                  domElement.Xpath,
		HighlightIndex:         domElement.HighlightIndex,
		EntireParentBranchPath: parentBranchPath,
		Attributes:             domElement.Attributes,
		ShadowRoot:             domElement.ShadowRoot,
		CssSelector:            optional.Some(cssSelector),
		PageCoordinates:        domElement.PageCoordinates,
		ViewportCoordinates:    domElement.ViewportCoordinates,
		ViewportInfo:           domElement.ViewportInfo,
	}
}

func (h *HistoryTreeProcessor) getParentBranchPath(domElement *DOMElementNode) []string {
	parents := []string{}
	currentElement := domElement
	for currentElement.Parent != nil {
		parents = append(parents, currentElement.Parent.TagName)
		currentElement = currentElement.Parent
	}

	parents = utils.Reverse(parents)
	return parents
}
