package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/schema"
)

// SaveConversation saves conversation history to file
func SaveConversation(inputMessages []*schema.Message, response any, target string, encoding *string) error {
	// Create folders if not exists
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("Failed to create directory %s: %v", dir, err)
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Format conversation
	content, err := formatConversation(inputMessages, response)
	if err != nil {
		return fmt.Errorf("failed to format conversation: %w", err)
	}

	// Write to file
	if err := os.WriteFile(target, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// formatConversation formats the conversation including messages and response
func formatConversation(messages []*schema.Message, response any) (string, error) {
	var lines []string

	// Format messages
	for _, message := range messages {
		lines = append(lines, fmt.Sprintf(" %s ", message.Role))
		lines = append(lines, message.Content)
		lines = append(lines, "") // Empty line after each message
	}

	// Format response
	lines = append(lines, " RESPONSE")
	
	// Serialize response to JSON
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("failed to serialize response: %w", err)
	}

	// Pretty print JSON
	var prettyJSON map[string]interface{}
	if err := json.Unmarshal(responseJSON, &prettyJSON); err != nil {
		return "", fmt.Errorf("failed to parse response JSON: %w", err)
	}

	prettyBytes, err := json.MarshalIndent(prettyJSON, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format response JSON: %w", err)
	}

	lines = append(lines, string(prettyBytes))

	return strings.Join(lines, "\n"), nil
}

// Note: _write_messages_to_file and _write_response_to_file have been merged into formatConversation
// This is more efficient for operations and reduces file I/O
