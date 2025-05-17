package dom

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/nerdface-ai/browser-use-go/internals/utils"

	"github.com/charmbracelet/log"

	"github.com/playwright-community/playwright-go"
)

type ViewportShortInfo struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type DomService struct {
	Page       playwright.Page `json:"page"`
	XpathCache map[string]any  `json:"xpathCache"`
	JsCode     string          `json:"jsCode"`
}

func NewDomService(page playwright.Page) *DomService {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get pwd")
	}
	dirname := filepath.Dir(filename)
	jsPath := filepath.Join(dirname, "buildDomTree.js")
	jsCode, err := os.ReadFile(jsPath)
	if err != nil {
		panic(err)
	}
	return &DomService{
		Page:       page,
		XpathCache: make(map[string]any),
		JsCode:     string(jsCode),
	}
}

func (s *DomService) GetClickableElements(highlightElements bool, focusElement int, viewportExpansion int) (*DOMState, error) {
	elementTree, selectorMap, err := s.buildDomTree(highlightElements, focusElement, viewportExpansion)
	if err != nil {
		return nil, err
	}

	return &DOMState{
		ElementTree: elementTree,
		SelectorMap: selectorMap,
	}, nil
}

func (s *DomService) GetCrossOriginIframes() []string {
	// invisible cross-origin iframes are used for ads and tracking, dont open those
	hiddenFrameUrls, _ := s.Page.Locator("iframe").Filter(playwright.LocatorFilterOptions{Visible: playwright.Bool(false)}).EvaluateAll("e => e.map(e => e.src)")

	var adDomains = []string{"doubleclick.net", "adroll.com", "googletagmanager.com"}
	isAdUrl := func(url string) bool {
		for _, domain := range adDomains {
			if strings.Contains(url, domain) {
				return true
			}
		}
		return false
	}

	// Get all frames
	frames := s.Page.Frames()
	pageUrl := s.Page.URL()
	pageHost, err := url.Parse(pageUrl)
	if err != nil {
		return []string{}
	}

	var crossOriginIframes []string
	for _, frame := range frames {
		frameUrl := frame.URL()
		parsed, err := url.Parse(frameUrl)
		if err != nil {
			continue
		}
		frameHost := parsed.Host

		// Exclude data:urls and about:blank, same-origin iframes
		if frameHost == "" || frameHost == pageHost.Host {
			continue
		}
		// Exclude hidden frames
		if _, exists := hiddenFrameUrls.(map[string]any)[frameUrl]; exists {
			continue
		}
		// Exclude ad network tracker frame URLs
		if isAdUrl(frameUrl) {
			continue
		}
		crossOriginIframes = append(crossOriginIframes, frameUrl)
	}

	return crossOriginIframes
}

func (s *DomService) buildDomTree(highlightElements bool, focusElement int, viewportExpansion int) (*DOMElementNode, *SelectorMap, error) {
	result, err := s.Page.Evaluate("1+1")
	errorOccured := false
	if err != nil {
		errorOccured = true
	} else if resultValue, ok := result.(float64); ok {
		if resultValue != 2 {
			errorOccured = true
		}
	}
	if errorOccured {
		return nil, nil, errors.New("failed to evaluate JS")
	}

	if s.Page.URL() == "about:blank" {
		return &DOMElementNode{
			TagName:    "body",
			Xpath:      "",
			Attributes: map[string]string{},
			Children:   []DOMBaseNode{},
			IsVisible:  false,
			Parent:     nil,
		}, &SelectorMap{}, nil
	}

	debugMode := log.GetLevel() == log.DebugLevel
	args := map[string]interface{}{
		"doHighlightElements": highlightElements,
		"focusHighlightIndex": focusElement,
		"viewportExpansion":   viewportExpansion,
		"debugMode":           debugMode,
	}
	evalPage, err := s.Page.Evaluate(s.JsCode, args)
	if err != nil {
		return nil, nil, err
	}

	evalPageMap, ok := evalPage.(map[string]any)
	if !ok {
		return nil, nil, errors.New("failed to cast evalPage to map[string]any")
	}

	if debugMode && evalPageMap["perfMetrics"] != nil {
		metrics, err := json.MarshalIndent(evalPageMap["perfMetrics"], "", "  ")
		if err != nil {
			return nil, nil, err
		}
		log.Debug("DOM Tree Building Performance Metrics for: %s\n%s", s.Page.URL(), string(metrics))
	}

	return s.constructDomTree(evalPageMap)
}

func (s *DomService) constructDomTree(evalPage map[string]any) (*DOMElementNode, *SelectorMap, error) {
	jsNodeMap, ok := evalPage["map"].(map[string]any)
	if !ok {
		return nil, nil, errors.New("failed to cast map[string]any to map[string]any")
	}
	jsRootId, err := strconv.Atoi(evalPage["rootId"].(string))
	if err != nil {
		return nil, nil, err
	}

	selectorMap := &SelectorMap{}
	nodeMap := map[int]DOMBaseNode{}

	for id, nodeData := range jsNodeMap {
		node, childrenIds := s.parseNode(nodeData.(map[string]any))
		if node == nil {
			continue
		}
		idInt, err := strconv.Atoi(id)
		if err != nil {
			return nil, nil, err
		}
		nodeMap[idInt] = node

		if node, ok := node.(*DOMElementNode); ok && node.HighlightIndex != nil {
			(*selectorMap)[*node.HighlightIndex] = node
		}

		// NOTE: We know that we are building the tree bottom up
		//       and all children are already processed.
		if node, ok := node.(*DOMElementNode); ok {
			for _, childId := range childrenIds {
				childNode, ok := nodeMap[childId]
				if !ok {
					continue
				}

				// childNode.Parent = node
				childNode.SetParent(node)
				node.Children = append(node.Children, childNode)
			}
		}
	}
	htmlToDict := nodeMap[jsRootId]

	// del node_map
	// del js_node_map
	// del js_root_id

	elem, ok := htmlToDict.(*DOMElementNode)
	if !ok || elem == nil {
		return nil, nil, errors.New("failed to parse HTML to dictionary")
	}
	return elem, selectorMap, nil
}

// Set node type as TextNode or ElementNode with some default values
func (s *DomService) parseNode(nodeData map[string]any) (DOMBaseNode, []int) {
	if nodeData == nil {
		return nil, []int{}
	}

	// Process text nodes immediately
	if nodeData["type"] == "TEXT_NODE" {
		textNode := &DOMTextNode{
			Text:      nodeData["text"].(string),
			IsVisible: nodeData["isVisible"].(bool),
			Parent:    nil,
		}
		return textNode, []int{}
	}

	// Process coordinates if they exist for element nodes

	var viewportInfo *ViewportInfo

	if nodeData["viewport"] != nil {
		viewportInfo = &ViewportInfo{
			Width:  nodeData["viewport"].(map[string]int)["width"],
			Height: nodeData["viewport"].(map[string]int)["height"],
		}
	}

	elementNode := &DOMElementNode{
		TagName:        nodeData["tagName"].(string),
		Xpath:          nodeData["xpath"].(string),
		Attributes:     utils.ConvertToStringMap(nodeData["attributes"].(map[string]any)),
		Children:       []DOMBaseNode{},
		IsVisible:      utils.GetDefaultValue(nodeData, "isVisible", false),
		IsInteractive:  utils.GetDefaultValue(nodeData, "isInteractive", false),
		IsTopElement:   utils.GetDefaultValue(nodeData, "isTopElement", false),
		IsInViewport:   utils.GetDefaultValue(nodeData, "isInViewport", false),
		HighlightIndex: utils.ConvertToOptional[int](nodeData["highlightIndex"]),
		ShadowRoot:     utils.GetDefaultValue(nodeData, "shadowRoot", false),
		Parent:         nil,
		ViewportInfo:   viewportInfo,
	}

	childrenIds, err := utils.ConvertToSliceOfInt(nodeData["children"])
	if err != nil {
		return elementNode, []int{}
	}

	return elementNode, childrenIds
}
