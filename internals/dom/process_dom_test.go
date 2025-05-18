package dom

import (
	"encoding/json"
	"log"
	"os"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
)

func TestProcessDOM(t *testing.T) {
	pw, err := playwright.Run()
	if err != nil {
		t.Fatalf("could not start playwright: %v", err)
	}
	browser, err := pw.Chromium.Launch()
	if err != nil {
		t.Fatalf("could not launch browser: %v", err)
	}
	page, err := browser.NewPage()
	if err != nil {
		t.Fatalf("could not create page: %v", err)
	}
	if _, err = page.Goto("https://kayak.com/flights"); err != nil {
		t.Fatalf("could not goto: %v", err)
	}

	jsCode, err := os.ReadFile("./buildDomTree.js")
	if err != nil {
		t.Fatalf("failed to read JS file: %v", err)
	}

	start := time.Now()
	domTree, err := page.Evaluate(string(jsCode))
	if err != nil {
		log.Fatalf("failed to evaluate JS: %v", err)
	}
	elapsed := time.Since(start)
	t.Logf("Time: %.2fs\n", elapsed.Seconds())

	if err := os.MkdirAll("./tmp", 0755); err != nil {
		log.Fatalf("failed to create tmp dir: %v", err)
	}
	domTreeJSON, err := json.MarshalIndent(domTree, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal domTree: %v", err)
	}
	if err := os.WriteFile("./tmp/dom.json", domTreeJSON, 0644); err != nil {
		log.Fatalf("failed to write dom.json: %v", err)
	}
}
