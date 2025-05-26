package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplaceSensitiveData(t *testing.T) {
	// Create a Registry instance for testing
	registry := &Registry{}

	tests := []struct {
		name           string
		argumentsInJson string
		sensitiveData  map[string]string
		expected       string
	}{
		{
			name:           "Replace single sensitive data",
			argumentsInJson: `{"username": "user", "password": "<secret>password</secret>"}`,
			sensitiveData:  map[string]string{"password": "secret123"},
			expected:       `{"username": "user", "password": "secret123"}`,
		},
		{
			name:           "Replace multiple sensitive data",
			argumentsInJson: `{"username": "<secret>username</secret>", "password": "<secret>password</secret>"}`,
			sensitiveData:  map[string]string{"username": "admin", "password": "secret123"},
			expected:       `{"username": "admin", "password": "secret123"}`,
		},
		{
			name:           "No sensitive data to replace",
			argumentsInJson: `{"username": "user", "password": "plaintext"}`,
			sensitiveData:  map[string]string{"password": "secret123"},
			expected:       `{"username": "user", "password": "plaintext"}`,
		},
		{
			name:           "Placeholder exists but no matching key in sensitiveData",
			argumentsInJson: `{"username": "user", "password": "<secret>nonexistent</secret>"}`,
			sensitiveData:  map[string]string{"password": "secret123"},
			expected:       `{"username": "user", "password": "<secret>nonexistent</secret>"}`,
		},
		{
			name:           "Multiple occurrences of the same secret",
			argumentsInJson: `{"password1": "<secret>password</secret>", "password2": "<secret>password</secret>"}`,
			sensitiveData:  map[string]string{"password": "secret123"},
			expected:       `{"password1": "secret123", "password2": "secret123"}`,
		},
		{
			name:           "Empty input",
			argumentsInJson: "",
			sensitiveData:  map[string]string{"password": "secret123"},
			expected:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.replaceSensitiveData(tt.argumentsInJson, tt.sensitiveData)
			assert.Equal(t, tt.expected, result, "The replaced string should match the expected output")
		})
	}
}
