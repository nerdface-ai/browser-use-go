package dom

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"

	"github.com/moznion/go-optional"
)

type HashedDomElement struct {
	BranchPathHash string
	AttributesHash string
	XpathHash      string
	// TextHash string
}

type Coordinates struct {
	X int
	Y int
}

func (c *Coordinates) ToDict() map[string]int {
	return map[string]int{
		"x": c.X,
		"y": c.Y,
	}
}

type CoordinateSet struct {
	TopLeft     Coordinates
	TopRight    Coordinates
	BottomLeft  Coordinates
	BottomRight Coordinates
	Center      Coordinates
	Width       int
	Height      int
}

func (c *CoordinateSet) ToDict() map[string]any {
	return map[string]any{
		"top_left":     c.TopLeft.ToDict(),
		"top_right":    c.TopRight.ToDict(),
		"bottom_left":  c.BottomLeft.ToDict(),
		"bottom_right": c.BottomRight.ToDict(),
		"center":       c.Center.ToDict(),
		"width":        c.Width,
		"height":       c.Height,
	}
}

type ViewportInfo struct {
	ScrollX int
	ScrollY int
	Width   int
	Height  int
}

func (v *ViewportInfo) ToDict() map[string]int {
	return map[string]int{
		"scroll_x": v.ScrollX,
		"scroll_y": v.ScrollY,
		"width":    v.Width,
		"height":   v.Height,
	}
}

type DOMHistoryElement struct {
	TagName                string
	Xpath                  string
	HighlightIndex         optional.Option[int]
	EntireParentBranchPath []string
	Attributes             map[string]string
	ShadowRoot             bool
	CssSelector            optional.Option[string]
	PageCoordinates        *CoordinateSet
	ViewportCoordinates    *CoordinateSet
	ViewportInfo           *ViewportInfo
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
		"tag_name":                  e.TagName,
		"xpath":                     e.Xpath,
		"highlight_index":           e.HighlightIndex,
		"entire_parent_branch_path": e.EntireParentBranchPath,
		"attributes":                e.Attributes,
		"shadow_root":               e.ShadowRoot,
		"css_selector":              e.CssSelector,
		"page_coordinates":          pageCoordinates,
		"viewport_coordinates":      viewportCoordinates,
		"viewport_info":             viewportInfo,
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
		CssSelector:            optional.Some(cssSelector),
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
	if node.HighlightIndex.IsSome() {
		hashedNode := h.hashDomElement(node)
		if hashedNode.BranchPathHash == hashedDomHistoryElement.BranchPathHash &&
			hashedNode.AttributesHash == hashedDomHistoryElement.AttributesHash &&
			hashedNode.XpathHash == hashedDomHistoryElement.XpathHash {
			return node
		}
	}
	for _, child := range node.Children {
		if child, ok := (*child).(*DOMElementNode); ok {
			result := h.processNode(child, hashedDomHistoryElement)
			if result != nil {
				return result
			}
		}
	}
	return nil
}
