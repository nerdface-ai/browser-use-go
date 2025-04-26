package browser

import (
	"log"
	"nerdface-ai/browser-use-go/browser-use/dom"
	"slices"
	"strings"

	"github.com/moznion/go-optional"
	"github.com/playwright-community/playwright-go"
)

type CachedStateClickableElementsHashes struct {
	Url    string
	Hashes map[string]int
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

func (self *BrowserContext) ConvertSimpleXpathToCssSelector(xpath string) string {
	return dom.ConvertSimpleXpathToCssSelector(xpath)
}

func (self *BrowserContext) EnhancedCssSelectorForElement(element *dom.DOMElementNode, includeDynamicAttributes bool) string {
	return dom.EnhancedCssSelectorForElement(element, includeDynamicAttributes)
}

func (self *BrowserContext) GetState(cacheClickableElementsHashes bool) *BrowserState {
	return nil
}

func (self *BrowserContext) NavigateTo(url string) error {
	if !self.isUrlAllowed(url) {
		return &BrowserError{Message: "Navigation to non-allowed URL: " + url}
	}

	page := self.GetCurrentPage()
	page.Goto(url)
	page.WaitForLoadState()
	return nil
}

func (self *BrowserContext) GetSession() *BrowserSession {
	if self.Session == nil {
		session, err := self.initializeSession()
		if err != nil {
			panic(err)
		}
		return session
	}
	return self.Session
}

// Get the current page
func (self *BrowserContext) GetCurrentPage() playwright.Page {
	session := self.GetSession()
	return self.getCurrentPage(session)
}

func (self *BrowserContext) Close() {
	if self.Session == nil {
		return
	}
	if self.pageEventHandler != nil && self.Session.Context != nil {
		self.Session.Context.RemoveListener("page", self.pageEventHandler)
		self.pageEventHandler = nil
	}

	// TODO: self.SaveCookies()

	if keepAlive, ok := self.Config["keep_alive"].(bool); (ok && !keepAlive) || !ok {
		err := self.Session.Context.Close()
		if err != nil {
			log.Printf("🪨  Failed to close browser context: %s", err)
		}
	}

	// Dereference everything
	self.Session = nil
	self.ActiveTab = nil
	self.pageEventHandler = nil
}

func (self *BrowserContext) initializeSession() (*BrowserSession, error) {
	log.Printf("🌎  Initializing new browser context with id: %s", self.ContextId)
	pwBrowser := self.Browser.GetPlaywrightBrowser()

	context, err := self.createContext(pwBrowser)
	if err != nil {
		return nil, err
	}
	self.pageEventHandler = nil

	pages := context.Pages()
	self.Session = &BrowserSession{
		Context:     context,
		CachedState: nil,
	}

	var activePage playwright.Page = nil
	if self.Browser.Config["cdp_url"] != nil {
		// If we have a saved target ID, try to find and activate it
		if self.State.TargetId != nil {
			targets := self.getCdpTargets()
			for _, target := range targets {
				if target["targetId"] == self.State.TargetId.Unwrap() {
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
		if self.Browser.Config["cdp_url"] != nil {
			targets := self.getCdpTargets()
			for _, target := range targets {
				if target["url"] == activePage.URL() {
					self.State.TargetId = optional.Some(activePage.URL())
					break
				}
			}
		}
	}
	log.Printf("🫨  Bringing tab to front: %s", activePage.URL())
	activePage.BringToFront()
	activePage.WaitForLoadState() // 'load'

	self.ActiveTab = activePage

	return self.Session, nil
}

func (self *BrowserContext) onPage(page playwright.Page) {
	if self.Browser.Config["cdp_url"] != nil {
		page.Reload()
	}
	page.WaitForLoadState()
	log.Printf("📑  New page opened: %s", page.URL())

	if !strings.HasPrefix(page.URL(), "chrome-extension://") && !strings.HasPrefix(page.URL(), "chrome://") {
		self.ActiveTab = page
	}

	if self.Session != nil {
		self.State.TargetId = nil
	}
}

func (self *BrowserContext) getCdpTargets() []map[string]interface{} {
	if self.Browser.Config["cdp_url"] == nil || self.Session == nil {
		return []map[string]interface{}{}
	}
	pages := self.Session.Context.Pages()
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

func (self *BrowserContext) addNewPageListener(context playwright.BrowserContext) {
	self.pageEventHandler = self.onPage
	context.OnPage(self.pageEventHandler)
}

func (self *BrowserContext) isUrlAllowed(url string) bool {
	return true
}

// Creates a new browser context with anti-detection measures and loads cookies if available.
func (self *BrowserContext) createContext(browser playwright.Browser) (playwright.BrowserContext, error) {
	var context playwright.BrowserContext
	var err error
	if self.Browser.Config["cdp_url"] != nil && len(browser.Contexts()) > 0 {
		context = browser.Contexts()[0]
	} else if self.Browser.Config["browser_binary_path"] != nil && len(browser.Contexts()) > 0 {
		context = browser.Contexts()[0]
	} else {
		context, err = browser.NewContext(
			playwright.BrowserNewContextOptions{
				NoViewport:        playwright.Bool(true),
				UserAgent:         playwright.String(self.Browser.Config["user_agent"].(string)),
				JavaScriptEnabled: playwright.Bool(true),
				BypassCSP:         playwright.Bool(self.Browser.Config["disable_security"].(bool)),
				IgnoreHttpsErrors: playwright.Bool(self.Browser.Config["disable_security"].(bool)),
				RecordVideo: &playwright.RecordVideo{
					Dir: self.Browser.Config["save_recording_path"].(string),
					Size: &playwright.Size{
						Width:  self.Browser.Config["browser_window_size"].(map[string]interface{})["width"].(int),
						Height: self.Browser.Config["browser_window_size"].(map[string]interface{})["height"].(int),
					},
				},
				RecordHarPath:   playwright.String(self.Browser.Config["save_har_path"].(string)),
				Locale:          playwright.String(self.Browser.Config["locale"].(string)),
				HttpCredentials: self.Browser.Config["http_credentials"].(*playwright.HttpCredentials),
				IsMobile:        playwright.Bool(self.Browser.Config["is_mobile"].(bool)),
				HasTouch:        playwright.Bool(self.Browser.Config["has_touch"].(bool)),
				Geolocation:     self.Browser.Config["geolocation"].(*playwright.Geolocation),
				Permissions:     self.Browser.Config["permissions"].([]string),
				TimezoneId:      playwright.String(self.Browser.Config["timezone_id"].(string)),
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

func (self *BrowserContext) getCurrentPage(session *BrowserSession) playwright.Page {
	pages := session.Context.Pages()
	if self.Browser.Config["cdp_url"] != nil && self.State.TargetId != nil {
		targets := self.getCdpTargets()
		for _, target := range targets {
			if target["targetId"] == self.State.TargetId.Unwrap() {
				for _, page := range pages {
					if page.URL() == target["url"] {
						return page
					}
				}
			}
		}
	}
	if self.ActiveTab != nil && !self.ActiveTab.IsClosed() && slices.Contains(session.Context.Pages(), self.ActiveTab) {
		return self.ActiveTab
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
	session, err = self.initializeSession()
	if err != nil {
		panic(err)
	}
	page, err = session.Context.NewPage()
	if err != nil {
		panic(err)
	}
	self.ActiveTab = page
	return page
}
