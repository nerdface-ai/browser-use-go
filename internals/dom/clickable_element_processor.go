package dom

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
)

type ClickableElementProcessor struct {
}

func (c *ClickableElementProcessor) GetClickableElementsHashes(node *DOMElementNode) []string {
	return []string{}
}

func (c *ClickableElementProcessor) GetClickableElements(node *DOMElementNode) []*DOMElementNode {
	clickableElements := []*DOMElementNode{}
	for _, child := range node.Children {
		if child, ok := child.(*DOMElementNode); ok {
			if child.HighlightIndex != nil {
				clickableElements = append(clickableElements, child)
			}
			clickableElements = append(clickableElements, c.GetClickableElements(child)...)
		}
	}

	return clickableElements
}

func (c *ClickableElementProcessor) HashDomElement(domElement *DOMElementNode) string {
	parentBranchPath := c.getParentBranchPath(domElement)
	branchPathHash := c.parentBranchPathHash(parentBranchPath)
	attributesHash := c.attributesHash(domElement.Attributes)
	xpathHash := c.xpathHash(domElement.Xpath)
	// textHash := c.textHash(domElement.GetAllTextTillNextClickableElement())

	return c.hashString(fmt.Sprintf("%s-%s-%s", branchPathHash, attributesHash, xpathHash))
}

func (c *ClickableElementProcessor) getParentBranchPath(domElement *DOMElementNode) []string {
	parents := []string{}
	currentElement := domElement
	for currentElement.Parent != nil {
		parents = append(parents, currentElement.Parent.TagName)
		currentElement = currentElement.Parent
	}

	slices.Reverse(parents)
	return parents
}

func (c *ClickableElementProcessor) parentBranchPathHash(parentBranchPath []string) string {
	parentBranchPathString := strings.Join(parentBranchPath, "/")
	return c.hashString(parentBranchPathString)
}

func (c *ClickableElementProcessor) attributesHash(attributes map[string]string) string {
	attributesString := ""
	for key, value := range attributes {
		attributesString += fmt.Sprintf("%s=%s", key, value)
	}
	return c.hashString(attributesString)
}

func (c *ClickableElementProcessor) xpathHash(xpath string) string {
	return c.hashString(xpath)
}

func (c *ClickableElementProcessor) textHash(domElement *DOMElementNode) string {
	textString := domElement.GetAllTextTillNextClickableElement(-1)
	return c.hashString(textString)
}

func (c *ClickableElementProcessor) hashString(s string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(s)))
}
