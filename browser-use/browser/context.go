package browser

import (
	"fmt"
	"log"
	"nerdface-ai/browser-use-go/browser-use/dom"
	"nerdface-ai/browser-use-go/browser-use/utils"
	"slices"
	"strconv"
	"strings"

	"github.com/moznion/go-optional"
	"github.com/playwright-community/playwright-go"
)

type CachedStateClickableElementsHashes struct {
	Url    string
	Hashes []string
}

type BrowserSession struct {
	ActiveTab                          playwright.Page
	Context                            playwright.BrowserContext
	CachedState                        *BrowserState
	CachedStateClickableElementsHashes *CachedStateClickableElementsHashes
}

func NewSession(context playwright.BrowserContext, cachedState *BrowserState) *BrowserSession {

	browserSession := BrowserSession{
		ActiveTab:                          nil,
		Context:                            context,
		CachedState:                        cachedState,
		CachedStateClickableElementsHashes: nil,
	}

	browserSession.Context.OnPage(func(page playwright.Page) {
		initScript := `
			(() => {
				if (!window.getEventListeners) {
					window.getEventListeners = function (node) {
						return node.__listeners || {};
					};

					// Save the original addEventListener
					const originalAddEventListener = Element.prototype.addEventListener;

					const eventProxy = {
						addEventListener: function (type, listener, options = {}) {
							// Initialize __listeners if not exists
							const defaultOptions = { once: false, passive: false, capture: false };
							if(typeof options === 'boolean') {
								options = { capture: options };
							}
							options = { ...defaultOptions, ...options };
							if (!this.__listeners) {
								this.__listeners = {};
							}

							// Initialize array for this event type if not exists
							if (!this.__listeners[type]) {
								this.__listeners[type] = [];
							}
							

							// Add the listener to __listeners
							this.__listeners[type].push({
								listener: listener,
								type: type,
								...options
							});

							// Call original addEventListener using the saved reference
							return originalAddEventListener.call(this, type, listener, options);
						}
					};

					Element.prototype.addEventListener = eventProxy.addEventListener;
				}
			})()`
		page.AddInitScript(playwright.Script{Content: &initScript})
	})

	return &browserSession
}

// State of the browser context
type BrowserContextState struct {
	TargetId optional.Option[string]
}

type BrowserContext struct {
	ContextId        string
	Config           BrowserConfig
	Browser          *Browser
	Session          *BrowserSession
	State            *BrowserContextState
	ActiveTab        playwright.Page
	pageEventHandler func(page playwright.Page)
}

func (bc *BrowserContext) ConvertSimpleXpathToCssSelector(xpath string) string {
	return dom.ConvertSimpleXpathToCssSelector(xpath)
}

func (bc *BrowserContext) EnhancedCssSelectorForElement(element *dom.DOMElementNode, includeDynamicAttributes bool) string {
	return dom.EnhancedCssSelectorForElement(element, includeDynamicAttributes)
}

func (bc *BrowserContext) GetState(cacheClickableElementsHashes bool) *BrowserState {
	/* Get the current state of the browser
	cache_clickable_elements_hashes: bool
		If True, cache the clickable elements hashes for the current state. This is used to calculate which elements are new to the llm (from last message) -> reduces token usage.
	*/

	// TODO
	// await self._wait_for_page_and_frames_load()
	page := bc.GetCurrentPage()

	updatedState := bc.getUpdatedState(page)

	return updatedState
}

func (bc *BrowserContext) getUpdatedState(page playwright.Page) *BrowserState {
	domService := dom.NewDomService(page)
	focus_element := -1 // default
	content, err := domService.GetClickableElements(
		utils.GetDefaultValue(bc.Config, "highlight_elements", true),
		focus_element,
		utils.GetDefaultValue(bc.Config, "viewport_expansion", 0),
	)
	if err != nil {
		log.Printf("Failed to get clickable elements: %s", err)
	}

	tabsInfo := bc.GetTabsInfo()

	// TODO @@@
	// screenshot_b64 = await self.take_screenshot()
	// pixels_above, pixels_below = await self.get_scroll_info(page)

	title, _ := page.Title()
	// updated_state
	currentState := BrowserState{
		ElementTree:   content.ElementTree,
		SelectorMap:   content.SelectorMap,
		Url:           page.URL(),
		Title:         title,
		Tabs:          tabsInfo,
		Screenshot:    nil, // TODO
		PixelAbove:    0,   // TODO
		PixelBelow:    0,   // TODO
		BrowserErrors: []string{},
	}
	return &currentState
}

func (bc *BrowserContext) GetSession() *BrowserSession {
	if bc.Session == nil {
		session, err := bc.initializeSession()
		if err != nil {
			panic(err)
		}
		return session
	}
	return bc.Session
}

// Get the current page
func (bc *BrowserContext) GetCurrentPage() playwright.Page {
	session := bc.GetSession()
	return bc.getCurrentPage(session)
}

func (bc *BrowserContext) Close() {
	if bc.Session == nil {
		return
	}
	if bc.pageEventHandler != nil && bc.Session.Context != nil {
		bc.Session.Context.RemoveListener("page", bc.pageEventHandler)
		bc.pageEventHandler = nil
	}

	// TODO: bc.SaveCookies()

	if keepAlive, ok := bc.Config["keep_alive"].(bool); (ok && !keepAlive) || !ok {
		err := bc.Session.Context.Close()
		if err != nil {
			log.Printf("🪨  Failed to close browser context: %s", err)
		}
	}

	// Dereference everything
	bc.Session = nil
	bc.ActiveTab = nil
	bc.pageEventHandler = nil
}

func (bc *BrowserContext) GetSelectorMap() *dom.SelectorMap {
	session := bc.GetSession()
	if session.CachedState == nil {
		return nil
	}
	return session.CachedState.SelectorMap
}

func (bc *BrowserContext) GetDomElementByIndex(index int) (*dom.DOMElementNode, error) {
	selectorMap := bc.GetSelectorMap()
	if selectorMap == nil || (*selectorMap)[index] == nil {
		return nil, fmt.Errorf("element with index %d does not exist - retry or use alternative actions", index)
	}
	return (*selectorMap)[index], nil
}

// sync DOMElementNode with Playwright
func (bc *BrowserContext) GetLocateElement(element *dom.DOMElementNode) playwright.Locator {
	currentPage := bc.GetCurrentPage()
	var currentFrame playwright.FrameLocator = nil

	// Start with the target element and collect all parents
	parents := []*dom.DOMElementNode{}
	current := element
	for current.Parent != nil {
		parent := current.Parent
		parents = append(parents, parent)
		current = parent
	}

	// Reverse the parents list to process from top to bottom
	slices.Reverse(parents)

	// Process all iframe parents in sequence
	iframes := []*dom.DOMElementNode{}
	for _, item := range parents {
		if item.TagName == "iframe" {
			iframes = append(iframes, item)
		}
	}
	includeDynamicAttributes := utils.GetDefaultValue(bc.Config, "include_dynamic_attributes", true)
	for _, parent := range iframes {
		cssSelector := bc.EnhancedCssSelectorForElement(parent, includeDynamicAttributes)
		if currentFrame != nil {
			currentFrame = currentFrame.FrameLocator(cssSelector)
		} else {
			currentFrame = currentPage.FrameLocator(cssSelector)
		}
	}
	cssSelector := bc.EnhancedCssSelectorForElement(element, includeDynamicAttributes)
	if currentFrame != nil {
		return currentFrame.Locator(cssSelector)
	} else {
		return currentPage.Locator(cssSelector)
	}
}

func (bc *BrowserContext) NavigateTo(url string) error {
	if !bc.isUrlAllowed(url) {
		return &BrowserError{Message: "Navigation to non-allowed URL: " + url}
	}

	page := bc.GetCurrentPage()
	page.Goto(url)
	page.WaitForLoadState()
	return nil
}

// TODO: error handling
func (bc *BrowserContext) PerformClick(clickFunc func(), page playwright.Page) optional.Option[string] {
	// Performs the actual click, handling both download and navigation scenarios.

	// TODO
	// if self.config.save_downloads_path: return downloadPath
	//
	// }

	// Wait for new page to open. If not, just close it
	newPage, _ := bc.GetSession().Context.ExpectPage(func() error {
		clickFunc()
		return nil
	}, playwright.BrowserContextExpectPageOptions{Timeout: playwright.Float(1500)})

	if newPage != nil {
		newPage.WaitForLoadState()
	}
	page.WaitForLoadState()
	return nil
}

func (bc *BrowserContext) ClickElementNode(elementNode *dom.DOMElementNode) (optional.Option[string], error) {
	// Optimized method to click an element using xpath.
	page := bc.GetCurrentPage()

	elementLocator := bc.GetLocateElement(elementNode)
	if elementLocator == nil {
		return optional.None[string](), &BrowserError{Message: "Element: " + elementNode.Xpath + " not found"}
	}

	return bc.PerformClick(func() {
		elementLocator.Click(playwright.LocatorClickOptions{Timeout: playwright.Float(1500)})
	}, page), nil
}

func (bc *BrowserContext) InputTextElementNode(elementNode *dom.DOMElementNode, text string) error {
	/*
		Input text into an element with proper error handling and state management.
		Handles different types of input fields and ensures proper element state before input.
	*/
	locator := bc.GetLocateElement(elementNode)

	if locator == nil {
		return &BrowserError{Message: "Element: " + elementNode.Xpath + " not found"}
	}

	// Ensure element is ready for input
	selectorState := playwright.WaitForSelectorState("visible")
	locator.WaitFor(playwright.LocatorWaitForOptions{State: &selectorState, Timeout: playwright.Float(1000)})
	isHidden, err := locator.IsHidden()
	if err != nil {
		return &BrowserError{Message: "Failed to check if element is hidden: " + elementNode.Xpath}
	}
	if !isHidden {
		locator.ScrollIntoViewIfNeeded(playwright.LocatorScrollIntoViewIfNeededOptions{Timeout: playwright.Float(1000)})
	}

	// Get element properties to determine input method
	tagNameAny, _ := locator.Evaluate("el => el.tagName.toLowerCase()", nil)
	tagName := tagNameAny.(string)

	if tagName == "input" || tagName == "textarea" {
		locator.Evaluate("el => { el.textContent = ''; el.value = ''; }", nil)
		err := locator.Fill(text)

		if err != nil {
			return &BrowserError{Message: "Failed to fill element: " + elementNode.Xpath}
		}

		value, err := locator.InputValue()
		if err != nil {
			return &BrowserError{Message: "Failed to get input value: " + elementNode.Xpath}
		}
		if value != text {
			return &BrowserError{Message: "Input value does not match: " + elementNode.Xpath}
		}
	} else {
		log.Printf("Element: %s is not editable.", elementNode.Xpath)
		locator.Fill(text)
	}

	return nil
}

func (bc *BrowserContext) initializeSession() (*BrowserSession, error) {
	log.Printf("🌎  Initializing new browser context with id: %s", bc.ContextId)
	pwBrowser := bc.Browser.GetPlaywrightBrowser()

	context, err := bc.createContext(pwBrowser)
	if err != nil {
		return nil, err
	}
	bc.pageEventHandler = nil

	pages := context.Pages()
	bc.Session = &BrowserSession{
		Context:     context,
		CachedState: nil,
	}

	var activePage playwright.Page = nil
	if bc.Browser.Config["cdp_url"] != nil {
		// If we have a saved target ID, try to find and activate it
		if bc.State.TargetId != nil {
			targets := bc.getCdpTargets()
			for _, target := range targets {
				if target["targetId"] == bc.State.TargetId.Unwrap() {
					// Find matching page by URL
					for _, page := range pages {
						if page.URL() == target["url"] {
							activePage = page
							break
						}
					}
					break
				}
			}
		}
	}

	if activePage == nil {
		if len(pages) > 0 && !strings.HasPrefix(pages[0].URL(), "chrome://") && !strings.HasPrefix(pages[0].URL(), "chrome-extension://") {
			activePage = pages[0]
			log.Printf("🔍  Using existing page: %s", activePage.URL())
		} else {
			activePage, err = context.NewPage()
			if err != nil {
				return nil, err
			}
			activePage.Goto("about:blank")
			log.Printf("🆕  Created new page: %s", activePage.URL())
		}

		// Get target ID for the active page
		if bc.Browser.Config["cdp_url"] != nil {
			targets := bc.getCdpTargets()
			for _, target := range targets {
				if target["url"] == activePage.URL() {
					bc.State.TargetId = optional.Some(activePage.URL())
					break
				}
			}
		}
	}
	log.Printf("🫨  Bringing tab to front: %s", activePage.URL())
	activePage.BringToFront()
	activePage.WaitForLoadState() // 'load'

	bc.ActiveTab = activePage

	return bc.Session, nil
}

func (bc *BrowserContext) onPage(page playwright.Page) {
	if bc.Browser.Config["cdp_url"] != nil {
		page.Reload()
	}
	page.WaitForLoadState()
	log.Printf("📑  New page opened: %s", page.URL())

	if !strings.HasPrefix(page.URL(), "chrome-extension://") && !strings.HasPrefix(page.URL(), "chrome://") {
		bc.ActiveTab = page
	}

	if bc.Session != nil {
		bc.State.TargetId = nil
	}
}

func (bc *BrowserContext) getCdpTargets() []map[string]interface{} {
	if bc.Browser.Config["cdp_url"] == nil || bc.Session == nil {
		return []map[string]interface{}{}
	}
	pages := bc.Session.Context.Pages()
	if len(pages) == 0 {
		return []map[string]interface{}{}
	}

	cdpSession, err := pages[0].Context().NewCDPSession(pages[0])
	if err != nil {
		return []map[string]interface{}{}
	}
	result, err := cdpSession.Send("Target.getTargets", map[string]interface{}{})
	if err != nil {
		return []map[string]interface{}{}
	}
	err = cdpSession.Detach()
	if err != nil {
		return []map[string]interface{}{}
	}
	return result.(map[string]interface{})["targetInfos"].([]map[string]interface{})
}

func (bc *BrowserContext) addNewPageListener(context playwright.BrowserContext) {
	bc.pageEventHandler = bc.onPage
	context.OnPage(bc.pageEventHandler)
}

// TODO
func (bc *BrowserContext) isUrlAllowed(url string) bool {
	return true
}

// Creates a new browser context with anti-detection measures and loads cookies if available.
func (bc *BrowserContext) createContext(browser playwright.Browser) (playwright.BrowserContext, error) {
	var context playwright.BrowserContext
	var err error
	if bc.Browser.Config["cdp_url"] != nil && len(browser.Contexts()) > 0 {
		context = browser.Contexts()[0]
	} else if bc.Browser.Config["browser_binary_path"] != nil && len(browser.Contexts()) > 0 {
		context = browser.Contexts()[0]
	} else {
		context, err = browser.NewContext(
			playwright.BrowserNewContextOptions{
				NoViewport:        playwright.Bool(true),
				UserAgent:         playwright.String(utils.GetDefaultValue(bc.Browser.Config, "user_agent", "")),
				JavaScriptEnabled: playwright.Bool(true),
				BypassCSP:         playwright.Bool(bc.Browser.Config["disable_security"].(bool)),
				IgnoreHttpsErrors: playwright.Bool(bc.Browser.Config["disable_security"].(bool)),
				// RecordVideo: &playwright.RecordVideo{
				// 	Dir: bc.Browser.Config["save_recording_path"].(string),
				// 	Size: &playwright.Size{
				// 		Width:  bc.Browser.Config["browser_window_size"].(map[string]interface{})["width"].(int),
				// 		Height: bc.Browser.Config["browser_window_size"].(map[string]interface{})["height"].(int),
				// 	},
				// },
				// RecordHarPath:   playwright.String(bc.Browser.Config["save_har_path"].(string)),
				Locale:          playwright.String(utils.GetDefaultValue(bc.Browser.Config, "locale", "")),
				HttpCredentials: utils.GetDefaultValue[*playwright.HttpCredentials](bc.Browser.Config, "http_credentials", nil),
				IsMobile:        playwright.Bool(utils.GetDefaultValue(bc.Browser.Config, "is_mobile", false)),
				HasTouch:        playwright.Bool(bc.Browser.Config["has_touch"].(bool)),
				// Geolocation: bc.Browser.Config["geolocation"].(*playwright.Geolocation),
				// Permissions:     bc.Browser.Config["permissions"].([]string),
				TimezoneId: playwright.String(utils.GetDefaultValue(bc.Browser.Config, "timezone_id", "")),
			},
		)
		if err != nil {
			return nil, err
		}
	}

	// TODO: provide cookie_path
	initScript := `// Webdriver property
            Object.defineProperty(navigator, 'webdriver', {
                get: () => undefined
            });

            // Languages
            Object.defineProperty(navigator, 'languages', {
                get: () => ['en-US']
            });

            // Plugins
            Object.defineProperty(navigator, 'plugins', {
                get: () => [1, 2, 3, 4, 5]
            });

            // Chrome runtime
            window.chrome = { runtime: {} };

            // Permissions
            const originalQuery = window.navigator.permissions.query;
            window.navigator.permissions.query = (parameters) => (
                parameters.name === 'notifications' ?
                    Promise.resolve({ state: Notification.permission }) :
                    originalQuery(parameters)
            );
            (function () {
                const originalAttachShadow = Element.prototype.attachShadow;
                Element.prototype.attachShadow = function attachShadow(options) {
                    return originalAttachShadow.call(this, { ...options, mode: "open" });
                };
            })();`
	context.AddInitScript(playwright.Script{Content: &initScript})
	return context, nil
}

func (bc *BrowserContext) getCurrentPage(session *BrowserSession) playwright.Page {
	pages := session.Context.Pages()
	if bc.Browser.Config["cdp_url"] != nil && bc.State.TargetId != nil {
		targets := bc.getCdpTargets()
		for _, target := range targets {
			if target["targetId"] == bc.State.TargetId.Unwrap() {
				for _, page := range pages {
					if page.URL() == target["url"] {
						return page
					}
				}
			}
		}
	}
	if bc.ActiveTab != nil && !bc.ActiveTab.IsClosed() && slices.Contains(session.Context.Pages(), bc.ActiveTab) {
		return bc.ActiveTab
	}

	// fall back to most recently opened non-extension page (extensions are almost always invisible background targets)
	nonExtensionPages := []playwright.Page{}
	for _, page := range pages {
		if !strings.HasPrefix(page.URL(), "chrome-extension://") && !strings.HasPrefix(page.URL(), "chrome://") {
			nonExtensionPages = append(nonExtensionPages, page)
		}
	}
	if len(nonExtensionPages) > 0 {
		return nonExtensionPages[len(nonExtensionPages)-1]
	}
	page, err := session.Context.NewPage()
	if err == nil {
		return page
	}
	session, err = bc.initializeSession()
	if err != nil {
		panic(err)
	}
	page, err = session.Context.NewPage()
	if err != nil {
		panic(err)
	}
	bc.ActiveTab = page
	return page
}

func (bc *BrowserContext) GetTabsInfo() []*TabInfo {
	// Get information about all tabs
	session := bc.GetSession()

	tabsInfo := []*TabInfo{}
	for pageId, page := range session.Context.Pages() {
		title, _ := page.Title()
		tabInfo := TabInfo{
			PageId:       pageId,
			Url:          page.URL(),
			Title:        title,
			ParentPageId: nil,
		}
		tabsInfo = append(tabsInfo, &tabInfo)
	}
	return tabsInfo
}

func (bc *BrowserContext) SwitchToTab(pageId int) error {
	// Switch to a specific tab by its PageId
	session := bc.GetSession()
	pages := session.Context.Pages()

	if pageId >= len(pages) {
		message := "No tab found with page_id: " + strconv.Itoa(pageId)
		return &BrowserError{Message: message}
	}

	for pageId < 0 {
		pageId += len(pages)
	}
	page := pages[pageId]

	// TODO: Check if the tab's URL is allowed before switching
	if !bc.isUrlAllowed(page.URL()) {
		return NewURLNotAllowedError(page.URL())
	}

	// Update target ID if using CDP
	if bc.Browser.Config["cdp_url"] != nil {
		targets := bc.getCdpTargets()
		for _, target := range targets {
			if target["url"] == page.URL() {
				bc.State.TargetId = optional.Some(target["targetId"].(string))
				break
			}
		}
	}

	bc.ActiveTab = page
	page.BringToFront()
	page.WaitForLoadState()
	return nil
}
