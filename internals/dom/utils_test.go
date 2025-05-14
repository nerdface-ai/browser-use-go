package dom

import (
	"testing"
)

func TestConvertSimpleXpathToCssSelector(t *testing.T) {
	// Test empty xpath returns empty string
	if got := ConvertSimpleXpathToCssSelector(""); got != "" {
		t.Errorf("Expected empty string, got %q", got)
	}

	// Test a simple xpath without indices
	xpath := "/html/body/div/span"
	expected := "html > body > div > span"
	if got := ConvertSimpleXpathToCssSelector(xpath); got != expected {
		t.Errorf("Expected %q, got %q", expected, got)
	}

	// Test xpath with an index on one element: [2] should translate to :nth-of-type(2)
	xpath = "/html/body/div[2]/span"
	expected = "html > body > div:nth-of-type(2) > span"
	if got := ConvertSimpleXpathToCssSelector(xpath); got != expected {
		t.Errorf("Expected %q, got %q", expected, got)
	}

	// Test xpath with indices on multiple elements
	xpath = "/ul/li[3]/a[1]"
	expected = "ul > li:nth-of-type(3) > a:nth-of-type(1)"
	if got := ConvertSimpleXpathToCssSelector(xpath); got != expected {
		t.Errorf("Expected %q, got %q", expected, got)
	}
}

func TestEnhancedCssSelectorForElement(t *testing.T) {
	dummyElement := &DOMElementNode{
		TagName:   "div",
		IsVisible: true,
		Parent:    nil,
		Xpath:     "/html/body/div[2]",
		Attributes: map[string]string{
			"class":       "foo bar",
			"id":          "my-id",
			"placeholder": `some "quoted" text`,
			"data-testid": "123",
		},
		Children: []DOMBaseNode{},
	}

	actualSelector := EnhancedCssSelectorForElement(dummyElement, true)
	expectedSelector := `html > body > div:nth-of-type(2).foo.bar[data-testid="123"][id="my-id"][placeholder*="some \"quoted\" text"]`

	if actualSelector != expectedSelector {
		t.Errorf("Expected %s, but got %s", expectedSelector, actualSelector)
	}
}
