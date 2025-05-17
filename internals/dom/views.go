package dom

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Base interface for all DOM nodes
type DOMBaseNode interface {
	ToJson() map[string]any
	SetParent(parent *DOMElementNode)
}

// DOMTextNode
type DOMTextNode struct {
	Text      string          `json:"text"`
	Type      string          `json:"type"` // default: TEXT_NODE
	Parent    *DOMElementNode `json:"parent"`
	IsVisible bool            `json:"isVisible"`
}

func (n *DOMTextNode) SetParent(parent *DOMElementNode) {
	n.Parent = parent
}

func (n *DOMTextNode) HasParentWithHighlightIndex() bool {
	current := n.Parent
	for current != nil {
		if current.HighlightIndex != nil {
			return true
		}
		current = current.Parent
	}
	return false
}

func (n *DOMTextNode) IsParentInViewport() bool {
	if n.Parent == nil {
		return false
	}
	return n.Parent.IsInViewport
}

func (n *DOMTextNode) IsParentTopElement() bool {
	if n.Parent == nil {
		return false
	}
	return n.Parent.IsTopElement
}

func (n *DOMTextNode) ToJson() map[string]any {
	return map[string]any{
		"text": n.Text,
		"type": n.Type,
	}
}

// DOMElementNode
type DOMElementNode struct {
	TagName             string            `json:"tagName"`
	Xpath               string            `json:"xpath"`
	Attributes          map[string]string `json:"attributes"`
	Children            []DOMBaseNode     `json:"children"`
	IsInteractive       bool              `json:"isInteractive"`
	IsTopElement        bool              `json:"isTopElement"`
	IsInViewport        bool              `json:"isInViewport"`
	ShadowRoot          bool              `json:"shadowRoot"`
	HighlightIndex      *int              `json:"highlightIndex,omitempty"`
	ViewportCoordinates *CoordinateSet    `json:"viewportCoordinates"`
	PageCoordinates     *CoordinateSet    `json:"pageCoordinates"`
	ViewportInfo        *ViewportInfo     `json:"viewportInfo"`
	Parent              *DOMElementNode   `json:"parent"`
	IsVisible           bool              `json:"isVisible"`
	IsNew               *bool             `json:"isNew,omitempty"`
}

func (n *DOMElementNode) SetParent(parent *DOMElementNode) {
	n.Parent = parent
}

func (n *DOMElementNode) ToJson() map[string]any {
	var children []map[string]any
	if n.Children != nil {
		for _, child := range n.Children {
			if child, ok := child.(*DOMElementNode); ok {
				children = append(children, child.ToJson())
			}
			if child, ok := child.(*DOMTextNode); ok {
				children = append(children, child.ToJson())
			}
		}
	}
	return map[string]any{
		"tag_name":            n.TagName,
		"xpath":               n.Xpath,
		"attributes":          n.Attributes,
		"isVisible":           n.IsVisible,
		"isInteractive":       n.IsInteractive,
		"isTopElement":        n.IsTopElement,
		"isInViewport":        n.IsInViewport,
		"shadowRoot":          n.ShadowRoot,
		"highlightIndex":      n.HighlightIndex,
		"viewportCoordinates": n.ViewportCoordinates,
		"pageCoordinates":     n.PageCoordinates,
		"children":            children,
		"parent":              n.Parent,
	}
}

func (n *DOMElementNode) ToString() string {
	tagStr := "<" + n.TagName
	for k, v := range n.Attributes {
		tagStr += " " + k + "=\"" + v + "\""
	}
	tagStr += ">"
	extras := []string{}
	if n.IsInteractive {
		extras = append(extras, "interactive")
	}
	if n.IsTopElement {
		extras = append(extras, "top")
	}
	if n.ShadowRoot {
		extras = append(extras, "shadow-root")
	}
	if n.HighlightIndex != nil {
		extras = append(extras, "highlight:"+strconv.Itoa(*n.HighlightIndex))
	}
	if len(extras) > 0 {
		tagStr += " [" + strings.Join(extras, ", ") + "]"
	}
	return tagStr
}

func (n *DOMElementNode) Hash() HashedDomElement {
	return *HistoryTreeProcessor{}.hashDomElement(n)
}

func (n *DOMElementNode) GetAllTextTillNextClickableElement(maxDepth int) string {
	textParts := []string{}
	var collectText func(node DOMBaseNode, currentDepth int)
	collectText = func(node DOMBaseNode, currentDepth int) {
		if maxDepth != -1 && currentDepth > maxDepth {
			return
		}
		// Skip this branch if we hit a highlighted element (except for the current node)
		if el, ok := node.(*DOMElementNode); ok && el != n && el.HighlightIndex != nil {
			return
		}
		switch t := node.(type) {
		case *DOMTextNode:
			textParts = append(textParts, t.Text)
		case *DOMElementNode:
			for _, child := range t.Children {
				collectText(child, currentDepth+1)
			}
		}
	}
	collectText(n, 0)
	return strings.TrimSpace(strings.Join(textParts, "\n"))
}

func (n *DOMElementNode) ClickableElementsToString(includeAttributes []string) string {
	var formattedText []string
	var processNode func(node DOMBaseNode, depth int)
	processNode = func(node DOMBaseNode, depth int) {
		nextDepth := depth
		depthStr := strings.Repeat("\t", depth)

		switch el := node.(type) {
		case *DOMElementNode:
			if el.HighlightIndex != nil {
				nextDepth += 1

				text := el.GetAllTextTillNextClickableElement(-1)
				attributesHTMLStr := ""

				if len(includeAttributes) > 0 {
					attributesToInclude := map[string]string{}
					for key, value := range el.Attributes {
						if slices.Contains(includeAttributes, key) {
							attributesToInclude[key] = value
						}
					}

					// Easy LLM optimizations
					// if tag == role attribute, don't include it
					if el.TagName == attributesToInclude["role"] {
						delete(attributesToInclude, "role")
					}
					// if aria-label == text of the node, don't include it
					if ariaLabel, ok := attributesToInclude["aria-label"]; ok && strings.TrimSpace(ariaLabel) == strings.TrimSpace(text) {
						delete(attributesToInclude, "aria-label")
					}
					// if placeholder == text of the node, don't include it
					if placeholder, ok := attributesToInclude["placeholder"]; ok && strings.TrimSpace(placeholder) == strings.TrimSpace(text) {
						delete(attributesToInclude, "placeholder")
					}

					if len(attributesToInclude) > 0 {
						// Format as key1='value1' key2='value2'
						var attributeStrs []string
						for k, v := range attributesToInclude {
							attributeStrs = append(attributeStrs, fmt.Sprintf("%s='%s'", k, v))
						}
						attributesHTMLStr = strings.Join(attributeStrs, " ")
					}
				}

				// Build the line
				var highlightIndicator string
				if el.IsNew != nil && *el.IsNew {
					highlightIndicator = fmt.Sprintf("*[%d]", *el.HighlightIndex)
				} else {
					highlightIndicator = fmt.Sprintf("[%d]", *el.HighlightIndex)
				}

				line := fmt.Sprintf("%s%s<%s", depthStr, highlightIndicator, el.TagName)

				if len(attributesHTMLStr) > 0 {
					line += " " + attributesHTMLStr
				}
				if len(text) > 0 {
					// Add space before >text only if there were NO attributes added before
					if attributesHTMLStr == "" {
						line += " "
					}
					line += fmt.Sprintf(">%s", text)
				} else if attributesHTMLStr == "" {
					// Add space before /> only if neither attributes NOR text were added
					line += " "
				}

				line += " />" // 1 token
				formattedText = append(formattedText, line)
			}

			// Process children regardless
			for _, child := range el.Children {
				processNode(child, nextDepth)
			}
		case *DOMTextNode:
			if !el.HasParentWithHighlightIndex() && el.Parent != nil && el.Parent.IsVisible && el.Parent.IsTopElement {
				formattedText = append(formattedText, fmt.Sprintf("%s%s", depthStr, el.Text))
			}
		}
	}

	processNode(n, 0)
	return strings.Join(formattedText, "\n")
}

func (n *DOMElementNode) GetFileUploadElement(checkSiblings bool) *DOMElementNode {
	if n.TagName == "input" && n.Attributes["type"] == "file" {
		return n
	}
	for _, child := range n.Children {
		if el, ok := child.(*DOMElementNode); ok {
			if result := el.GetFileUploadElement(false); result != nil {
				return result
			}
		}
	}
	if checkSiblings && n.Parent != nil {
		for _, sibling := range n.Parent.Children {
			if el, ok := sibling.(*DOMElementNode); ok && el != n {
				if result := el.GetFileUploadElement(false); result != nil {
					return result
				}
			}
		}
	}
	return nil
}

// Serialization helpers
type ElementTreeSerializer struct{}

func (ElementTreeSerializer) SerializeClickableElements(elementTree *DOMElementNode) string {
	return elementTree.ClickableElementsToString(nil)
}

func (ElementTreeSerializer) DomElementNodeToJson(elementTree *DOMElementNode) map[string]interface{} {
	var nodeToDict func(node DOMBaseNode) map[string]interface{}
	nodeToDict = func(node DOMBaseNode) map[string]interface{} {
		switch t := node.(type) {
		case *DOMTextNode:
			return map[string]interface{}{"type": "text", "text": t.Text}
		case *DOMElementNode:
			children := []map[string]interface{}{}
			for _, child := range t.Children {
				children = append(children, nodeToDict(child))
			}
			m := map[string]interface{}{
				"type":           "element",
				"tagName":        t.TagName,
				"attributes":     t.Attributes,
				"highlightIndex": t.HighlightIndex,
				"children":       children,
			}
			return m
		default:
			return map[string]interface{}{}
		}
	}
	return nodeToDict(elementTree)
}

// SelectorMap and DOMState
type SelectorMap map[int]*DOMElementNode

type DOMState struct {
	ElementTree *DOMElementNode `json:"elementTree"`
	SelectorMap *SelectorMap    `json:"selectorMap"`
}
