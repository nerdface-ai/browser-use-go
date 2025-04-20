package utils

import (
	"nerdface-ai/browser-use-go/browser-use/dom"
	"strings"
)	

func ConvertSimpleXpathToCssSelector(cls, xpath string) string {
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
			if strings.Contains(part, ":") && !strings.Contains(part, "["):
				base_part = part.replace(':', r'\:')
				css_parts.append(base_part)
				continue

			# Handle index notation [n]
			if '[' in part:
				base_part = part[: part.find('[')]
				# Handle custom elements with colons in the base part
				if ':' in base_part:
					base_part = base_part.replace(':', r'\:')
				index_part = part[part.find('[') :]

				# Handle multiple indices
				indices = [i.strip('[]') for i in index_part.split(']')[:-1]]

				for idx in indices:
					try:
						# Handle numeric indices
						if idx.isdigit():
							index = int(idx) - 1
							base_part += f':nth-of-type({index + 1})'
						# Handle last() function
						elif idx == 'last()':
							base_part += ':last-of-type'
						# Handle position() functions
						elif 'position()' in idx:
							if '>1' in idx:
								base_part += ':nth-of-type(n+2)'
					except ValueError:
						continue

				css_parts.append(base_part)
			else:
				cssParts = append(cssParts, part)

		baseSelector := strings.Join(cssParts, " > ")
		return baseSelector
}
