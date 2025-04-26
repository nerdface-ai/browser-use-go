package browser

import (
	"fmt"
	"net"
	"os"

	"github.com/google/uuid"
	"github.com/playwright-community/playwright-go"
)

var IN_DOCKER = os.Getenv("IN_DOCKER") == "true"

type BrowserConfig = map[string]interface{}

func NewBrowserConfig() BrowserConfig {
	return BrowserConfig{
		"headless":         false,
		"disable_security": false,
		"browser_class":    "chromium",
		"is_mobile":        false,
		"has_touch":        false,
	}
}

// Example: GetBrowserConfig(config, "headless", false)
func GetBrowserConfig[T any](config BrowserConfig, key string, defaultValue T) T {
	if value, ok := config[key]; ok {
		if value, ok := value.(T); ok {
			return value
		}
	}
	return defaultValue
}

type Browser struct {
	Config            BrowserConfig
	Playwright        *playwright.Playwright
	PlaywrightBrowser playwright.Browser
}

func NewBrowser(customConfig BrowserConfig) *Browser {
	config := NewBrowserConfig()
	for key, value := range customConfig {
		config[key] = value
	}
	return &Browser{
		Config:            config,
		Playwright:        nil,
		PlaywrightBrowser: nil,
	}
}

func (b *Browser) NewContext() *BrowserContext {
	return &BrowserContext{
		ContextId: uuid.New().String(),
		Config:    b.Config,
		Browser:   b,
		Session:   nil,
		State:     &BrowserContextState{},
	}
}

// Get a browser context
func (b *Browser) GetPlaywrightBrowser() playwright.Browser {
	if b.PlaywrightBrowser == nil {
		return b.init()
	}
	return b.PlaywrightBrowser
}

func (b *Browser) Close(options ...playwright.BrowserCloseOptions) error {
	return b.PlaywrightBrowser.Close(options...)
}

func (b *Browser) init() playwright.Browser {
	playwright, err := playwright.Run()
	if err != nil {
		panic(err)
	}
	b.Playwright = playwright

	b.PlaywrightBrowser = b.setupBrowser(playwright)
	return b.PlaywrightBrowser
}

func (b *Browser) setupBrowser(pw *playwright.Playwright) playwright.Browser {
	// TODO: implement remote browser setup
	// if b.Config["cdp_url"] != nil {
	// 	return self.setupRemoteCdpBrowser(playwright)
	// }
	// if self.Config["wss_url"] != nil {
	// 	return self.setupRemoteWssBrowser(playwright)
	// }

	// if self.Config["headless"] != nil {
	// 	log.Println("⚠️ Headless mode is not recommended. Many sites will detect and block all headless browsers.")
	// }

	// if b.Config["browser_binary_path"] != nil {
	// 	return b.setupUserProvidedBrowser(playwright)
	// }
	return b.setupBuiltinBrowser(pw)
}

// func (self *Browser) setupRemoteCdpBrowser(playwright playwright.Playwright) playwright.Browser {
// }

// func (self *Browser) setupRemoteWssBrowser(playwright playwright.Playwright) playwright.Browser {
// }

// func (b *Browser) setupUserProvidedBrowser(playwright playwright.Playwright) playwright.Browser {
// }

// Sets up and returns a Playwright Browser instance with anti-detection measures.
func (b *Browser) setupBuiltinBrowser(pw *playwright.Playwright) playwright.Browser {
	if b.Config["browser_binary_path"] != nil {
		panic("browser_binary_path should be None if trying to use the builtin browsers")
	}
	var screenSize map[string]int
	var offsetX, offsetY int
	if headless, ok := b.Config["headless"].(bool); ok && headless {
		screenSize = map[string]int{"width": 1920, "height": 1080}
		offsetX, offsetY = 0, 0
	} else {
		screenSize = getScreenResolution()
		offsetX, offsetY = getWindowAdjustments()
	}

	chromeArgs := []string{}
	chromeArgs = append(chromeArgs, CHROME_ARGS...) // default args

	if IN_DOCKER {
		chromeArgs = append(chromeArgs, CHROME_DOCKER_ARGS...)
	}
	if b.Config["headless"] != nil && b.Config["headless"].(bool) {
		chromeArgs = append(chromeArgs, CHROME_HEADLESS_ARGS...)
	}
	if b.Config["disable_security"] != nil && b.Config["disable_security"].(bool) {
		chromeArgs = append(chromeArgs, CHROME_DISABLE_SECURITY_ARGS...)
	}
	if b.Config["deterministic_rendering"] != nil && b.Config["deterministic_rendering"].(bool) {
		chromeArgs = append(chromeArgs, CHROME_DETERMINISTIC_RENDERING_ARGS...)
	}

	// window position and size
	chromeArgs = append(chromeArgs,
		fmt.Sprintf("--window-position=%d,%d", offsetX, offsetY),
		fmt.Sprintf("--window-size=%d,%d", screenSize["width"], screenSize["height"]),
	)

	// additional user specified args
	if extraArgs, ok := b.Config["extra_browser_args"].([]string); ok {
		chromeArgs = append(chromeArgs, extraArgs...)
	}

	// check if port 9222 is already taken, if so remove the remote-debugging-port arg to prevent conflicts
	ln, err := net.Listen("tcp", "127.0.0.1:9222")
	if err != nil {
		for i, arg := range chromeArgs {
			if arg == "--remote-debugging-port=9222" {
				chromeArgs = append(chromeArgs[:i], chromeArgs[i+1:]...)
				break
			}
		}
	} else {
		ln.Close()
	}

	browserType := pw.Chromium
	// TODO: support firefox and webkit
	// switch self.Config["browser_class"] {
	// case "chromium":
	// 	browserType = playwright.Chromium
	// case "firefox":
	// 	browserType = playwright.Firefox
	// case "webkit":
	// 	browserType = playwright.WebKit
	// default:
	// 	browserType = playwright.Chromium
	// }
	// args := map[string]interface{}{
	// 	"chromium": chromeArgs,
	// 	"firefox": []interface{}{
	// 		"-no-remote",
	// 		self.Config["extra_browser_args"],
	// 	},
	// 	"webkit": []interface{}{
	// 		"--no-startup-window",
	// 		self.Config["extra_browser_args"],
	// 	},
	// }

	browser, err := browserType.Launch(
		playwright.BrowserTypeLaunchOptions{
			Headless: playwright.Bool(b.Config["headless"].(bool)),
			Args:     chromeArgs,
			Proxy:    nil,
			// TODO: implement proxy
			// &playwright.Proxy{
			// 	Server:   b.Config["proxy"].(map[string]interface{})["server"].(string),
			// 	Bypass:   playwright.String(b.Config["proxy"].(map[string]interface{})["bypass"].(string)),
			// 	Username: playwright.String(b.Config["proxy"].(map[string]interface{})["username"].(string)),
			// 	Password: playwright.String(b.Config["proxy"].(map[string]interface{})["password"].(string)),
			// },
			HandleSIGTERM: playwright.Bool(false),
			HandleSIGINT:  playwright.Bool(false),
		},
	)
	if err != nil {
		panic(err)
	}
	return browser
}
