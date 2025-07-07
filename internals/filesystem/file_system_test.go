package filesystem

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMarkdownFile_Content(t *testing.T) {
	// test to understand what content looks like
	mdFile := NewMarkdownFile("test")
	mdFile.SetContent("Hello, world!")
	
	// Test basic properties
	if mdFile.GetName() != "test" {
		t.Errorf("Expected name 'test', got '%s'", mdFile.GetName())
	}
	
	if mdFile.GetContent() != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got '%s'", mdFile.GetContent())
	}
	
	if mdFile.GetExtension() != "md" {
		t.Errorf("Expected extension 'md', got '%s'", mdFile.GetExtension())
	}
	
	if mdFile.GetFullName() != "test.md" {
		t.Errorf("Expected full name 'test.md', got '%s'", mdFile.GetFullName())
	}
	
	// Test size and line count
	if mdFile.GetSize() != 13 {
		t.Errorf("Expected size 13, got %d", mdFile.GetSize())
	}
	
	if mdFile.GetLineCount() != 1 {
		t.Errorf("Expected line count 1, got %d", mdFile.GetLineCount())
	}
	
	// Print the content for debugging
	fmt.Printf("MarkdownFile content: name=%s, content=%s, extension=%s, full_name=%s\n",
		mdFile.GetName(), mdFile.GetContent(), mdFile.GetExtension(), mdFile.GetFullName())
}

func TestFileSystem_StateOperations(t *testing.T) {
	// test to understand how state looks like
	tempDir := os.TempDir()
	testDir := filepath.Join(tempDir, "browseruse_test_data_"+fmt.Sprintf("%d", time.Now().UnixNano()))
	
	fs, err := NewFileSystem(testDir, true)
	if err != nil {
		t.Fatalf("Failed to create file system: %v", err)
	}
	
	// Clean up at the end
	defer func() {
		if err := fs.Nuke(); err != nil {
			t.Logf("Failed to clean up: %v", err)
		}
	}()
	
	// Get initial state
	state, err := fs.GetState()
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}
	
	// Print state for debugging
	stateJSON, _ := json.MarshalIndent(state, "", "  ")
	fmt.Printf("Initial FileSystem state:\n%s\n", string(stateJSON))
	
	// Test basic state properties
	if state.BaseDir != testDir {
		t.Errorf("Expected base dir '%s', got '%s'", testDir, state.BaseDir)
	}
	
	if state.ExtractedContentCount != 0 {
		t.Errorf("Expected extracted content count 0, got %d", state.ExtractedContentCount)
	}
	
	// Should have default todo.md file
	if len(state.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(state.Files))
	}
	
	if _, exists := state.Files["todo.md"]; !exists {
		t.Error("Expected todo.md file to exist")
	}
}

func TestFileSystem_StateRestoration(t *testing.T) {
	// test to understand creating a filesystem, getting its state, and restoring it
	tempDir := os.TempDir()
	testDir := filepath.Join(tempDir, "browseruse_test_data_"+fmt.Sprintf("%d", time.Now().UnixNano()))
	
	// Create first file system
	fs, err := NewFileSystem(testDir, true)
	if err != nil {
		t.Fatalf("Failed to create file system: %v", err)
	}
	
	// Add some content
	result := fs.WriteFile("test.md", "# Test Content\nThis is a test file.")
	if !strings.Contains(result, "successfully") {
		t.Errorf("Failed to write file: %s", result)
	}
	
	// Get state
	state, err := fs.GetState()
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}
	
	// Print state for debugging
	stateJSON, _ := json.MarshalIndent(state, "", "  ")
	fmt.Printf("FileSystem state before restoration:\n%s\n", string(stateJSON))
	
	// Clean up first file system
	if err := fs.Nuke(); err != nil {
		t.Logf("Failed to clean up first fs: %v", err)
	}
	
	// Restore from state
	fs2, err := FromState(state)
	if err != nil {
		t.Fatalf("Failed to restore from state: %v", err)
	}
	
	// Clean up at the end
	defer func() {
		if err := fs2.Nuke(); err != nil {
			t.Logf("Failed to clean up second fs: %v", err)
		}
	}()
	
	// Get state from restored file system
	state2, err := fs2.GetState()
	if err != nil {
		t.Fatalf("Failed to get state from restored fs: %v", err)
	}
	
	// Print restored state for debugging
	stateJSON2, _ := json.MarshalIndent(state2, "", "  ")
	fmt.Printf("FileSystem state after restoration:\n%s\n", string(stateJSON2))
	
	// Compare states
	if state.BaseDir != state2.BaseDir {
		t.Errorf("Base dir mismatch: %s vs %s", state.BaseDir, state2.BaseDir)
	}
	
	if state.ExtractedContentCount != state2.ExtractedContentCount {
		t.Errorf("Extracted content count mismatch: %d vs %d", 
			state.ExtractedContentCount, state2.ExtractedContentCount)
	}
	
	if len(state.Files) != len(state2.Files) {
		t.Errorf("File count mismatch: %d vs %d", len(state.Files), len(state2.Files))
	}
	
	// Check if files are the same
	for filename := range state.Files {
		if _, exists := state2.Files[filename]; !exists {
			t.Errorf("File %s missing in restored state", filename)
		}
	}
	
	// Test reading the file we created
	content := fs2.ReadFile("test.md", false)
	if !strings.Contains(content, "Test Content") {
		t.Errorf("Expected restored file to contain 'Test Content', got: %s", content)
	}
}

func TestFileSystem_BasicOperations(t *testing.T) {
	tempDir := os.TempDir()
	testDir := filepath.Join(tempDir, "browseruse_test_ops_"+fmt.Sprintf("%d", time.Now().UnixNano()))
	
	fs, err := NewFileSystem(testDir, true)
	if err != nil {
		t.Fatalf("Failed to create file system: %v", err)
	}
	
	defer func() {
		if err := fs.Nuke(); err != nil {
			t.Logf("Failed to clean up: %v", err)
		}
	}()
	
	// Test write and read
	writeResult := fs.WriteFile("example.txt", "Hello, Go!")
	if !strings.Contains(writeResult, "successfully") {
		t.Errorf("Failed to write file: %s", writeResult)
	}
	
	readResult := fs.ReadFile("example.txt", false)
	if !strings.Contains(readResult, "Hello, Go!") {
		t.Errorf("Failed to read correct content: %s", readResult)
	}
	
	// Test append
	appendResult := fs.AppendFile("example.txt", "\nAppended text")
	if !strings.Contains(appendResult, "successfully") {
		t.Errorf("Failed to append to file: %s", appendResult)
	}
	
	readResult2 := fs.ReadFile("example.txt", false)
	if !strings.Contains(readResult2, "Appended text") {
		t.Errorf("Failed to read appended content: %s", readResult2)
	}
	
	// Test list files
	files := fs.ListFiles()
	if len(files) != 2 { // todo.md + example.txt
		t.Errorf("Expected 2 files, got %d: %v", len(files), files)
	}
	
	// Test allowed extensions
	extensions := fs.GetAllowedExtensions()
	expectedExts := []string{"md", "txt", "json", "csv"}
	if len(extensions) != len(expectedExts) {
		t.Errorf("Expected %d extensions, got %d: %v", len(expectedExts), len(extensions), extensions)
	}
}
