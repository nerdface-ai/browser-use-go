package dom

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
)

type HashedDomElement struct {
	BranchPathHash string `json:"branchPathHash"`
	AttributesHash string `json:"attributesHash"`
	XpathHash      string `json:"xpathHash"`
	// TextHash string
}

type Coordinates struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func (c *Coordinates) ToDict() map[string]int {
	return map[string]int{
		"x": c.X,
		"y": c.Y,
	}
}

type CoordinateSet struct {
	TopLeft     Coordinates `json:"topLeft"`
	TopRight    Coordinates `json:"topRight"`
	BottomLeft  Coordinates `json:"bottomLeft"`
	BottomRight Coordinates `json:"bottomRight"`
	Center      Coordinates `json:"center"`
	Width       int         `json:"width"`
	Height      int         `json:"height"`
}

func (c *CoordinateSet) ToDict() map[string]any {
	return map[string]any{
		"topLeft":     c.TopLeft.ToDict(),
		"topRight":    c.TopRight.ToDict(),
		"bottomLeft":  c.BottomLeft.ToDict(),
		"bottomRight": c.BottomRight.ToDict(),
		"center":      c.Center.ToDict(),
		"width":       c.Width,
		"height":      c.Height,
	}
}

type ViewportInfo struct {
	ScrollX int `json:"scrollX"`
	ScrollY int `json:"scrollY"`
	Width   int `json:"width"`
	Height  int `json:"height"`
}

func (v *ViewportInfo) ToDict() map[string]int {
	return map[string]int{
		"scrollX": v.ScrollX,
		"scrollY": v.ScrollY,
		"width":   v.Width,
		"height":  v.Height,
	}
}

type DOMHistoryElement struct {
	TagName                string            `json:"tagName"`
	Xpath                  string            `json:"xpath"`
	HighlightIndex         *int              `json:"highlightIndex,omitempty"`
	EntireParentBranchPath []string          `json:"entireParentBranchPath"`
	Attributes             map[string]string `json:"attributes"`
	ShadowRoot             bool              `json:"shadowRoot"`
	CssSelector            *string           `json:"cssSelector,omitempty"`
	PageCoordinates        *CoordinateSet    `json:"pageCoordinates"`
	ViewportCoordinates    *CoordinateSet    `json:"viewportCoordinates"`
	ViewportInfo           *ViewportInfo     `json:"viewportInfo"`
}

func (e *DOMHistoryElement) ToDict() map[string]any {
	var pageCoordinates map[string]any = nil
	var viewportCoordinates map[string]any = nil
	var viewportInfo map[string]int = nil
	if e.PageCoordinates != nil {
		pageCoordinates = e.PageCoordinates.ToDict()
	}
	if e.ViewportCoordinates != nil {
		viewportCoordinates = e.ViewportCoordinates.ToDict()
	}
	if e.ViewportInfo != nil {
		viewportInfo = e.ViewportInfo.ToDict()
	}

	return map[string]any{
		"tagName":                e.TagName,
		"xpath":                  e.Xpath,
		"highlightIndex":         e.HighlightIndex,
		"entireParentBranchPath": e.EntireParentBranchPath,
		"attributes":             e.Attributes,
		"shadowRoot":             e.ShadowRoot,
		"css_selector":           e.CssSelector,
		"page_coordinates":       pageCoordinates,
		"viewport_coordinates":   viewportCoordinates,
		"viewport_info":          viewportInfo,
	}
}

type HistoryTreeProcessor struct {
}

func (h HistoryTreeProcessor) ConvertDomElementToHistoryElement(domElement *DOMElementNode) *DOMHistoryElement {
	parentBranchPath := h.getParentBranchPath(domElement)
	cssSelector := EnhancedCssSelectorForElement(domElement, false)
	return &DOMHistoryElement{
		TagName:                domElement.TagName,
		Xpath:                  domElement.Xpath,
		HighlightIndex:         domElement.HighlightIndex,
		EntireParentBranchPath: parentBranchPath,
		Attributes:             domElement.Attributes,
		ShadowRoot:             domElement.ShadowRoot,
		CssSelector:            &cssSelector,
		PageCoordinates:        domElement.PageCoordinates,
		ViewportCoordinates:    domElement.ViewportCoordinates,
		ViewportInfo:           domElement.ViewportInfo,
	}
}

func (h HistoryTreeProcessor) FindHistoryElementInTree(domHistoryElement *DOMHistoryElement, tree *DOMElementNode) *DOMElementNode {
	hashedDomHistoryElement := h.hashDomHistoryElement(domHistoryElement)
	return h.processNode(tree, hashedDomHistoryElement)
}

func (h HistoryTreeProcessor) CompareHistoryElementAndDomeElement(domHistoryElement *DOMHistoryElement, domElement *DOMElementNode) bool {
	hashedDomHistoryElement := h.hashDomHistoryElement(domHistoryElement)
	hashedDomElement := h.hashDomElement(domElement)
	return hashedDomHistoryElement.BranchPathHash == hashedDomElement.BranchPathHash &&
		hashedDomHistoryElement.AttributesHash == hashedDomElement.AttributesHash &&
		hashedDomHistoryElement.XpathHash == hashedDomElement.XpathHash
}

func (h HistoryTreeProcessor) getParentBranchPath(domElement *DOMElementNode) []string {
	parents := []string{}
	currentElement := domElement
	for currentElement.Parent != nil {
		parents = append(parents, currentElement.Parent.TagName)
		currentElement = currentElement.Parent
	}

	slices.Reverse(parents)
	return parents
}

func (h HistoryTreeProcessor) parentBranchPathHash(parentBranchPath []string) string {
	parentBranchPathString := strings.Join(parentBranchPath, "/")
	return fmt.Sprintf("%x", sha256.Sum256([]byte(parentBranchPathString)))
}

func (h HistoryTreeProcessor) attributesHash(attributes map[string]string) string {
	attributesString := ""
	for key, value := range attributes {
		attributesString += fmt.Sprintf("%s=%s", key, value)
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(attributesString)))
}

func (h HistoryTreeProcessor) xpathHash(xpath string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(xpath)))
}

func (h HistoryTreeProcessor) hashDomHistoryElement(domHistoryElement *DOMHistoryElement) *HashedDomElement {
	branchPathHash := h.parentBranchPathHash(domHistoryElement.EntireParentBranchPath)
	attributesHash := h.attributesHash(domHistoryElement.Attributes)
	xpathHash := h.xpathHash(domHistoryElement.Xpath)

	return &HashedDomElement{
		BranchPathHash: branchPathHash,
		AttributesHash: attributesHash,
		XpathHash:      xpathHash,
	}
}

func (h HistoryTreeProcessor) hashDomElement(domElement *DOMElementNode) *HashedDomElement {
	parentBranchPath := h.getParentBranchPath(domElement)
	branchPathHash := h.parentBranchPathHash(parentBranchPath)
	attributesHash := h.attributesHash(domElement.Attributes)
	xpathHash := h.xpathHash(domElement.Xpath)

	return &HashedDomElement{
		BranchPathHash: branchPathHash,
		AttributesHash: attributesHash,
		XpathHash:      xpathHash,
	}
}

func (h HistoryTreeProcessor) processNode(node *DOMElementNode, hashedDomHistoryElement *HashedDomElement) *DOMElementNode {
	if node.HighlightIndex != nil {
		hashedNode := h.hashDomElement(node)
		if hashedNode.BranchPathHash == hashedDomHistoryElement.BranchPathHash &&
			hashedNode.AttributesHash == hashedDomHistoryElement.AttributesHash &&
			hashedNode.XpathHash == hashedDomHistoryElement.XpathHash {
			return node
		}
	}
	for _, child := range node.Children {
		if child, ok := child.(*DOMElementNode); ok {
			result := h.processNode(child, hashedDomHistoryElement)
			if result != nil {
				return result
			}
		}
	}
	return nil
}
