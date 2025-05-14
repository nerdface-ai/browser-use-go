package dom

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

func ConvertSimpleXpathToCssSelector(xpath string) string {
	if xpath == "" {
		return ""
	}

	// Remove leading slash if present
	xpath = strings.TrimPrefix(xpath, "/")

	// Split into parts
	parts := strings.Split(xpath, "/")
	cssParts := make([]string, 0, len(parts))

	for _, part := range parts {
		if part == "" {
			continue
		}

		// Handle custom elements with colons by escaping them
		if strings.Contains(part, ":") && !strings.Contains(part, "[") {
			basePart := strings.Replace(part, ":", `\:`, -1)
			cssParts = append(cssParts, basePart)
			continue
		}

		// Handle index notation [n]
		if strings.Contains(part, "[") {
			basePart := part[:strings.Index(part, "[")]
			// Handle custom elements with colons in the base part
			basePart = strings.Replace(basePart, ":", `\:`, -1)
			indexPart := part[strings.Index(part, "["):]

			// Handle multiple indices
			indices := strings.Split(indexPart, "]")
			indices = indices[:len(indices)-1]

			for _, idx := range indices {
				idx = strings.Trim(idx, "[]")
				// Handle numeric indices
				if idx, err := strconv.Atoi(idx); err == nil {
					index := int(idx) - 1
					basePart += fmt.Sprintf(":nth-of-type(%d)", index+1)
				}
				// Handle last() function
				if idx == "last()" {
					basePart += ":last-of-type"
					// Handle position() functions
					if strings.Contains(idx, "position()") {
						if strings.Contains(idx, ">1") {
							basePart += ":nth-of-type(n+2)"
						}
					}
				}
			}

			cssParts = append(cssParts, basePart)
		} else {
			cssParts = append(cssParts, part)
		}
	}

	baseSelector := strings.Join(cssParts, " > ")
	return baseSelector
}

func EnhancedCssSelectorForElement(element *DOMElementNode, includeDynamicAttributes bool) string {
	/*
		Creates a CSS selector for a DOM element, handling various edge cases and special characters.

		Args:
		        element: The DOM element to create a selector for

		Returns:
		        A valid CSS selector string
	*/
	// return ""
	// Get base selector from XPath
	css_selector := ConvertSimpleXpathToCssSelector(element.Xpath)

	// Handle class attributes
	if _, ok := element.Attributes["class"]; ok && element.Attributes["class"] != "" && includeDynamicAttributes {
		// Define a regex pattern for valid class names in CSS
		valid_class_name_pattern := regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_-]*)$`)

		// Iterate through the class attribute values
		classes := strings.Split(element.Attributes["class"], " ")
		for _, className := range classes {
			// Skip empty class names
			if strings.TrimSpace(className) == "" {
				continue
			}

			// Check if the class name is valid
			if valid_class_name_pattern.MatchString(className) {
				// Append the valid class name to the CSS selector
				css_selector += fmt.Sprintf(".%s", className)
			}
			// Skip invalid class names
			continue
		}

		// Expanded set of safe attributes that are stable and useful for selection
		SAFE_ATTRIBUTES := []string{
			// Data attributes (if they're stable in your application)
			"id",
			// Standard HTML attributes
			"name",
			"type",
			"placeholder",
			// Accessibility attributes
			"aria-label",
			"aria-labelledby",
			"aria-describedby",
			"role",
			// Common form attributes
			"for",
			"autocomplete",
			"required",
			"readonly",
			// Media attributes
			"alt",
			"title",
			"src",
			// Custom stable attributes (add any application-specific ones)
			"href",
			"target",
		}

		if includeDynamicAttributes {
			dynamic_attributes := []string{
				"data-id",
				"data-qa",
				"data-cy",
				"data-testid",
			}
			SAFE_ATTRIBUTES = append(SAFE_ATTRIBUTES, dynamic_attributes...)
		}

		var keys []string
		for k := range element.Attributes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		// Handle other attributes
		for _, attribute := range keys {
			value := element.Attributes[attribute]
			if attribute == "class" {
				continue
			}

			// Skip invalid attribute names
			if strings.TrimSpace(attribute) == "" {
				continue
			}

			if !slices.Contains(SAFE_ATTRIBUTES, attribute) {
				continue
			}

			// Escape special characters in attribute names
			safe_attribute := strings.Replace(attribute, ":", "\\:", -1)

			// Handle different value cases
			if value == "" {
				css_selector += fmt.Sprintf("[%s]", safe_attribute)
			} else if strings.ContainsAny(value, "\"'<>`\\n\\r\\t") {
				// Use contains for values with special characters
				// For newline-containing text, only use the part before the newline
				if strings.Contains(value, "\n") {
					value = strings.Split(value, "\n")[0]
				}
				// Regex-substitute *any* whitespace with a single space, then strip.
				re := regexp.MustCompile("\\s+")
				collapsed_value := re.ReplaceAllString(value, " ")
				// Escape embedded double-quotes.
				safe_value := strings.Replace(collapsed_value, "\"", "\\\"", -1)
				css_selector += fmt.Sprintf("[%s*=\"%s\"]", safe_attribute, safe_value)
			} else {
				css_selector += fmt.Sprintf("[%s=\"%s\"]", safe_attribute, value)
			}
		}
	}

	return css_selector
}
