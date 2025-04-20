package browser

import "nerdface-ai/browser-use-go/browser-use/dom"

type BrowserContext struct {
	ContextId string
}

func (b *BrowserContext) ConvertSimpleXpathToCssSelector(xpath string) string {
	return dom.ConvertSimpleXpathToCssSelector(xpath)
}

func (b *BrowserContext) EnhancedCssSelectorForElement(element *dom.DOMElementNode, includeDynamicAttributes bool) string {
	return dom.EnhancedCssSelectorForElement(element, includeDynamicAttributes)
}
