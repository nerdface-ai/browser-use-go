package dom

import (
	"fmt"

	"github.com/moznion/go-optional"
)

type HashedDomElement struct {
	BranchPathHash string
	AttributesHash string
}

// Base interface for all DOM nodes
type DOMBaseNode interface {
	SetParent(parent *DOMElementNode)
}

// DOMTextNode
type DOMTextNode struct {
	IsVisible bool
	Text      string
	Parent    *DOMElementNode
}

func (n *DOMTextNode) SetParent(p *DOMElementNode) { n.Parent = p }
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

// DOMElementNode
type DOMElementNode struct {
	IsVisible      bool
	TagName        string
	Xpath          string
	Attributes     map[string]string
	Children       []*DOMBaseNode
	IsInteractive  bool
	IsTopElement   bool
	ShadowRoot     bool
	HighlightIndex optional.Option[int]
	Parent         *DOMElementNode
}

func (n *DOMElementNode) SetParent(p *DOMElementNode) { n.Parent = p }

func (n *DOMElementNode) ToJson() map[string]any {
	return map[string]any{
		"isVisible":      n.IsVisible,
		"tagName":        n.TagName,
		"xpath":          n.Xpath,
		"attributes":     n.Attributes,
		"children":       n.Children,
		"isInteractive":  n.IsInteractive,
		"isTopElement":   n.IsTopElement,
		"shadowRoot":     n.ShadowRoot,
		"highlightIndex": n.HighlightIndex,
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
	if n.HighlightIndex.IsSome() {
		extras = append(extras, "highlight:"+itoa(n.HighlightIndex.Unwrap()))
	}
	if len(extras) > 0 {
		tagStr += " [" + join(extras, ", ") + "]"
	}
	return tagStr
}

// Helper functions for string join and int to string
func join(arr []string, sep string) string {
	result := ""
	for i, s := range arr {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}

func (n *DOMElementNode) GetAllTextTillNextClickableElement() string {
	var textParts []string
	var collectText func(node DOMBaseNode)
	collectText = func(node DOMBaseNode) {
		if el, ok := node.(*DOMElementNode); ok && el != n && el.HighlightIndex.IsSome() {
			return
		}
		switch t := node.(type) {
		case *DOMTextNode:
			textParts = append(textParts, t.Text)
		case *DOMElementNode:
			for _, child := range t.Children {
				collectText(*child)
			}
		}
	}
	collectText(n)
	return join(textParts, "\n")
}

func (n *DOMElementNode) ClickableElementsToString(includeAttributes []string) string {
	var formattedText []string
	var processNode func(node DOMBaseNode, depth int)
	processNode = func(node DOMBaseNode, depth int) {
		switch el := node.(type) {
		case *DOMElementNode:
			if el.HighlightIndex.IsNone() {
				attributesStr := ""
				if len(includeAttributes) > 0 {
					for _, key := range includeAttributes {
						if val, ok := el.Attributes[key]; ok {
							attributesStr += " " + key + "=\"" + val + "\""
						}
					}
				}
				formattedText = append(formattedText,
					fmt.Sprintf("%d[:]<%s%s>%s</%s>",
						el.HighlightIndex.Unwrap(), el.TagName, attributesStr, el.GetAllTextTillNextClickableElement(), el.TagName))
			}
			for _, child := range el.Children {
				processNode(*child, depth+1)
			}
		case *DOMTextNode:
			if !el.HasParentWithHighlightIndex() {
				formattedText = append(formattedText, fmt.Sprintf("_[:]{%s}", el.Text))
			}
		}
	}
	processNode(n, 0)
	return join(formattedText, "\n")
}

func (n *DOMElementNode) GetFileUploadElement(checkSiblings bool) *DOMElementNode {
	if n.TagName == "input" && n.Attributes["type"] == "file" {
		return n
	}
	for _, child := range n.Children {
		if el, ok := (*child).(*DOMElementNode); ok {
			if result := el.GetFileUploadElement(false); result != nil {
				return result
			}
		}
	}
	if checkSiblings && n.Parent != nil {
		for _, sibling := range n.Parent.Children {
			if el, ok := (*sibling).(*DOMElementNode); ok && el != n {
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
				children = append(children, nodeToDict(*child))
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
	ElementTree *DOMElementNode
	SelectorMap SelectorMap
}
