package browser

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/charmbracelet/log"
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

type Browser struct {
	Config            BrowserConfig
	Playwright        *playwright.Playwright
	PlaywrightBrowser playwright.Browser
	chromeProcess     *os.Process
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
	if b.chromeProcess != nil {
		// b.chromeProcess.Kill()
		log.Debug("Check kill chrome process")
	}
	if b.PlaywrightBrowser == nil {
		return nil
	}
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

// TODO(MID): implement remote browser setup
func (b *Browser) setupBrowser(pw *playwright.Playwright) playwright.Browser {
	// if b.Config["cdp_url"] != nil {
	// 	return self.setupRemoteCdpBrowser(playwright)
	// }
	// if self.Config["wss_url"] != nil {
	// 	return self.setupRemoteWssBrowser(playwright)
	// }

	// if self.Config["headless"] != nil {
	// 	log.Warn("⚠️ Headless mode is not recommended. Many sites will detect and block all headless browsers.")
	// }

	if b.Config["browser_binary_path"] != nil {
		return b.setupUserProvidedBrowser(pw)
	}
	return b.setupBuiltinBrowser(pw)
}

// func (self *Browser) setupRemoteCdpBrowser(playwright playwright.Playwright) playwright.Browser {
// }

// func (self *Browser) setupRemoteWssBrowser(playwright playwright.Playwright) playwright.Browser {
// }

func getChromeUserDataDir() string {
	tempDir := os.TempDir()
	pid := os.Getpid()
	return filepath.Join(tempDir, "chrome-profile-"+strconv.Itoa(pid))
}

// Sets up and returns a Playwright Browser instance with anti-detection measures.
func (b *Browser) setupUserProvidedBrowser(pw *playwright.Playwright) playwright.Browser {
	binaryPath, ok := b.Config["browser_binary_path"].(string)
	if !ok {
		panic("A browser_binary_path is required")
	}

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		log.Errorf("Chrome binary not found: %s", binaryPath)
		panic(err)
	}

	if b.Config["browser_class"] != "chromium" {
		panic("browser_binary_path only supports chromium browsers (make sure browser_class=chromium)")
	}
	browserClass := pw.Chromium

	// check if browser is already running
	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	response, err := client.Get("http://localhost:9222/json/version")
	if err != nil {
		// Check if the error is a connection error
		if _, ok := err.(net.Error); ok {
			log.Debug("🌎  No existing Chrome instance found, starting a new one")
		} else {
			// Not a connection error, panic
			panic(err)
		}
	}
	if err == nil && response != nil && response.StatusCode == 200 {
		log.Info("🔌  Reusing existing browser found running on http://localhost:9222")

		browser, err := browserClass.ConnectOverCDP(
			"http://localhost:9222",
			playwright.BrowserTypeConnectOverCDPOptions{
				Timeout: playwright.Float(20000),
			},
		)
		if err != nil {
			panic(err)
		}
		return browser
	}

	// Start a new Chrome instance
	argsMap := make(map[string]struct{})
	var chromeArgs []string

	addArgs := func(src []string) {
		for _, arg := range src {
			if _, exists := argsMap[arg]; !exists {
				argsMap[arg] = struct{}{}
				chromeArgs = append(chromeArgs, arg)
			}
		}
	}

	if v, ok := b.Config["user_data_dir"].(string); ok && v != "" {
		chromeArgs = append(chromeArgs, "--user-data-dir="+v)
	} else {
		chromeArgs = append(chromeArgs, "--user-data-dir="+getChromeUserDataDir())
	}

	if v, ok := b.Config["profile_directory"].(string); ok && v != "" {
		chromeArgs = append(chromeArgs, "--profile-directory="+v)
	}

	addArgs(CHROME_ARGS)
	if IN_DOCKER {
		addArgs(CHROME_DOCKER_ARGS)
	}
	if v, ok := b.Config["headless"].(bool); ok && v {
		addArgs(CHROME_HEADLESS_ARGS)
	}
	if v, ok := b.Config["disable_security"].(bool); ok && v {
		addArgs(CHROME_DISABLE_SECURITY_ARGS)
	}
	if v, ok := b.Config["deterministic_rendering"].(bool); ok && v {
		addArgs(CHROME_DETERMINISTIC_RENDERING_ARGS)
	}
	if extraArgs, ok := b.Config["extra_browser_args"].([]string); ok {
		addArgs(extraArgs)
	}

	chromeLaunchCmd := append([]string{binaryPath}, chromeArgs...)
	log.Debugf("🚀 Launching Chrome with args: %v", chromeLaunchCmd)

	cmd := exec.Command(chromeLaunchCmd[0], chromeLaunchCmd[1:]...)

	logFile, err := os.Create("/tmp/chrome_launch.log")
	if err != nil {
		log.Errorf("Failed to create chrome log file: %v", err)
		panic(err)
	}
	defer logFile.Close()
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		panic(err)
	}
	// Optionally, store the process if you need to kill or inspect it later
	b.chromeProcess = cmd.Process
	log.Debugf("🚀 Chrome process started with PID: %d", b.chromeProcess.Pid)

	// Attempt to connect again after starting a new instance
	for i := 0; i < 10; i++ {
		response, err := client.Get("http://localhost:9222/json/version")
		if err != nil {
			// Check if the error is a connection error
			if _, ok := err.(net.Error); ok {
				// It's a connection error, retry
			} else {
				// Not a connection error, panic
				panic(err)
			}
		}
		if response != nil && response.StatusCode == 200 {
			break
		}
		time.Sleep(1 * time.Second)
	}

	// Attempt to connect again after starting a new instance
	browser, err := browserClass.ConnectOverCDP(
		"http://localhost:9222",
		playwright.BrowserTypeConnectOverCDPOptions{
			Timeout: playwright.Float(20000),
		},
	)
	if err != nil {
		log.Errorf("❌  Failed to start a new Chrome instance: %s", err)
		panic(err)
	}
	return browser
}

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
	// TODO(LOW): support firefox and webkit
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
	var proxySetting *playwright.Proxy = nil
	if proxy, ok := b.Config["proxy"].(map[string]interface{}); ok {
		server := proxy["server"].(string)
		var bypassPtr *string = nil
		bypass, ok := proxy["bypass"].(string)
		if ok {
			bypassPtr = &bypass
		}
		var usernamePtr *string = nil
		username, ok := proxy["username"].(string)
		if ok {
			usernamePtr = &username
		}
		var passwordPtr *string = nil
		password, ok := proxy["password"].(string)
		if ok {
			passwordPtr = &password
		}

		proxySetting = &playwright.Proxy{
			Server:   server,
			Bypass:   bypassPtr,
			Username: usernamePtr,
			Password: passwordPtr,
		}
	}
	browser, err := browserType.Launch(
		playwright.BrowserTypeLaunchOptions{
			Headless:      playwright.Bool(b.Config["headless"].(bool)),
			Args:          chromeArgs,
			Proxy:         proxySetting,
			HandleSIGTERM: playwright.Bool(false),
			HandleSIGINT:  playwright.Bool(false),
		},
	)
	if err != nil {
		panic(err)
	}
	return browser
}
