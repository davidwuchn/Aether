package colony

import (
	"fmt"
	"strings"
)

const maxSignalContentLength = 500

// SanitizeSignalContent validates and sanitizes pheromone signal content.
//
// Rules applied in order:
//  1. Trim whitespace
//  2. Check max length (500 characters)
//  3. Reject XML structural tags
//  4. Reject prompt injection patterns
//  5. Reject shell injection patterns
//  6. Escape remaining angle brackets
//
// Returns the sanitized content and an error if the content was rejected.
func SanitizeSignalContent(content string) (string, error) {
	content = strings.TrimSpace(content)

	// Rule 1: Max length check
	if len(content) > maxSignalContentLength {
		return "", fmt.Errorf("content exceeds maximum length of %d characters (%d)", maxSignalContentLength, len(content))
	}

	// Rules 2-4: Reject integrity violations using the shared detector.
	if findings := DetectPromptIntegrityFindings(content); len(findings) > 0 {
		first := findings[0]
		switch first.Kind {
		case "xml_tag":
			return "", fmt.Errorf("content contains XML structural tags which are not allowed")
		case "prompt_injection":
			return "", fmt.Errorf("content contains prompt injection patterns which are not allowed")
		case "shell_injection":
			return "", fmt.Errorf("%s", first.Message)
		default:
			return "", fmt.Errorf("%s", first.Message)
		}
	}

	// Rule 5: Escape remaining angle brackets
	content = strings.ReplaceAll(content, "<", "&lt;")
	content = strings.ReplaceAll(content, ">", "&gt;")

	return content, nil
}
