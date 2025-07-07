package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	INVALID_FILENAME_ERROR_MESSAGE = "Error: Invalid filename format. Must be alphanumeric with supported extension."
	DEFAULT_FILE_SYSTEM_PATH       = "browseruse_agent_data"
)

// FileSystemError represents custom errors for file system operations
type FileSystemError struct {
	Message string
}

func (e *FileSystemError) Error() string {
	return e.Message
}

// BaseFile interface for all file types
type BaseFile interface {
	GetName() string
	GetContent() string
	GetExtension() string
	GetFullName() string
	GetSize() int
	GetLineCount() int
	SetContent(content string)
	WriteFileContent(content string)
	AppendFileContent(content string)
	SyncToDiskSync(path string) error
	Write(content string, path string) error
	Append(content string, path string) error
	Read() string
}

// baseFileImpl provides common implementation for all file types
type baseFileImpl struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

func (f *baseFileImpl) GetName() string {
	return f.Name
}

func (f *baseFileImpl) GetContent() string {
	return f.Content
}

func (f *baseFileImpl) SetContent(content string) {
	f.Content = content
}

func (f *baseFileImpl) WriteFileContent(content string) {
	f.SetContent(content)
}

func (f *baseFileImpl) AppendFileContent(content string) {
	f.SetContent(f.Content + content)
}

func (f *baseFileImpl) GetSize() int {
	return len(f.Content)
}

func (f *baseFileImpl) GetLineCount() int {
	return len(strings.Split(f.Content, "\n"))
}

func (f *baseFileImpl) Read() string {
	return f.Content
}

func (f *baseFileImpl) syncToDiskSync(path string, extension string) error {
	filePath := filepath.Join(path, f.GetName()+"."+extension)
	return os.WriteFile(filePath, []byte(f.Content), 0644)
}

func (f *baseFileImpl) write(content string, path string, extension string) error {
	f.WriteFileContent(content)
	return f.syncToDiskSync(path, extension)
}

func (f *baseFileImpl) append(content string, path string, extension string) error {
	f.AppendFileContent(content)
	return f.syncToDiskSync(path, extension)
}

// MarkdownFile implementation
type MarkdownFile struct {
	baseFileImpl
}

func NewMarkdownFile(name string) *MarkdownFile {
	return &MarkdownFile{
		baseFileImpl: baseFileImpl{Name: name, Content: ""},
	}
}

func (f *MarkdownFile) GetExtension() string {
	return "md"
}

func (f *MarkdownFile) GetFullName() string {
	return f.Name + "." + f.GetExtension()
}

func (f *MarkdownFile) SyncToDiskSync(path string) error {
	return f.syncToDiskSync(path, f.GetExtension())
}

func (f *MarkdownFile) Write(content string, path string) error {
	return f.write(content, path, f.GetExtension())
}

func (f *MarkdownFile) Append(content string, path string) error {
	return f.append(content, path, f.GetExtension())
}

// TxtFile implementation
type TxtFile struct {
	baseFileImpl
}

func NewTxtFile(name string) *TxtFile {
	return &TxtFile{
		baseFileImpl: baseFileImpl{Name: name, Content: ""},
	}
}

func (f *TxtFile) GetExtension() string {
	return "txt"
}

func (f *TxtFile) GetFullName() string {
	return f.Name + "." + f.GetExtension()
}

func (f *TxtFile) SyncToDiskSync(path string) error {
	return f.syncToDiskSync(path, f.GetExtension())
}

func (f *TxtFile) Write(content string, path string) error {
	return f.write(content, path, f.GetExtension())
}

func (f *TxtFile) Append(content string, path string) error {
	return f.append(content, path, f.GetExtension())
}

// JsonFile implementation
type JsonFile struct {
	baseFileImpl
}

func NewJsonFile(name string) *JsonFile {
	return &JsonFile{
		baseFileImpl: baseFileImpl{Name: name, Content: ""},
	}
}

func (f *JsonFile) GetExtension() string {
	return "json"
}

func (f *JsonFile) GetFullName() string {
	return f.Name + "." + f.GetExtension()
}

func (f *JsonFile) SyncToDiskSync(path string) error {
	return f.syncToDiskSync(path, f.GetExtension())
}

func (f *JsonFile) Write(content string, path string) error {
	return f.write(content, path, f.GetExtension())
}

func (f *JsonFile) Append(content string, path string) error {
	return f.append(content, path, f.GetExtension())
}

// CsvFile implementation
type CsvFile struct {
	baseFileImpl
}

func NewCsvFile(name string) *CsvFile {
	return &CsvFile{
		baseFileImpl: baseFileImpl{Name: name, Content: ""},
	}
}

func (f *CsvFile) GetExtension() string {
	return "csv"
}

func (f *CsvFile) GetFullName() string {
	return f.Name + "." + f.GetExtension()
}

func (f *CsvFile) SyncToDiskSync(path string) error {
	return f.syncToDiskSync(path, f.GetExtension())
}

func (f *CsvFile) Write(content string, path string) error {
	return f.write(content, path, f.GetExtension())
}

func (f *CsvFile) Append(content string, path string) error {
	return f.append(content, path, f.GetExtension())
}

// FileSystemState represents serializable state of the file system
type FileSystemState struct {
	Files                 map[string]map[string]interface{} `json:"files"`
	BaseDir               string                            `json:"base_dir"`
	ExtractedContentCount int                               `json:"extracted_content_count"`
}

// FileSystem enhanced file system with in-memory storage and multiple file type support
type FileSystem struct {
	baseDir               string
	dataDir               string
	files                 map[string]BaseFile
	fileTypes             map[string]func(string) BaseFile
	defaultFiles          []string
	extractedContentCount int
}

// NewFileSystem creates a new file system instance
func NewFileSystem(baseDir string, createDefaultFiles bool) (*FileSystem, error) {
	fs := &FileSystem{
		baseDir:               baseDir,
		dataDir:               filepath.Join(baseDir, DEFAULT_FILE_SYSTEM_PATH),
		files:                 make(map[string]BaseFile),
		defaultFiles:          []string{"todo.md"},
		extractedContentCount: 0,
	}

	// Initialize file types
	fs.fileTypes = map[string]func(string) BaseFile{
		"md":   func(name string) BaseFile { return NewMarkdownFile(name) },
		"txt":  func(name string) BaseFile { return NewTxtFile(name) },
		"json": func(name string) BaseFile { return NewJsonFile(name) },
		"csv":  func(name string) BaseFile { return NewCsvFile(name) },
	}

	// Create directories
	if err := os.MkdirAll(fs.baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	// Clean and create data directory
	if _, err := os.Stat(fs.dataDir); err == nil {
		if err := os.RemoveAll(fs.dataDir); err != nil {
			return nil, fmt.Errorf("failed to remove existing data directory: %w", err)
		}
	}

	if err := os.MkdirAll(fs.dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	if createDefaultFiles {
		if err := fs.createDefaultFiles(); err != nil {
			return nil, fmt.Errorf("failed to create default files: %w", err)
		}
	}

	return fs, nil
}

// GetAllowedExtensions returns the list of allowed file extensions
func (fs *FileSystem) GetAllowedExtensions() []string {
	extensions := make([]string, 0, len(fs.fileTypes))
	for ext := range fs.fileTypes {
		extensions = append(extensions, ext)
	}
	return extensions
}

// getFileTypeClass returns the appropriate file constructor for an extension
func (fs *FileSystem) getFileTypeClass(extension string) func(string) BaseFile {
	return fs.fileTypes[strings.ToLower(extension)]
}

// createDefaultFiles creates default files in the file system
func (fs *FileSystem) createDefaultFiles() error {
	for _, fullFilename := range fs.defaultFiles {
		name, extension, err := fs.parseFilename(fullFilename)
		if err != nil {
			return fmt.Errorf("invalid default file name %s: %w", fullFilename, err)
		}

		fileClass := fs.getFileTypeClass(extension)
		if fileClass == nil {
			return fmt.Errorf("invalid file extension '%s' for file '%s'", extension, fullFilename)
		}

		fileObj := fileClass(name)
		fs.files[fullFilename] = fileObj
		if err := fileObj.SyncToDiskSync(fs.dataDir); err != nil {
			return fmt.Errorf("failed to sync file %s to disk: %w", fullFilename, err)
		}
	}
	return nil
}

// isValidFilename checks if filename matches the required pattern
func (fs *FileSystem) isValidFilename(fileName string) bool {
	// Build extensions pattern from file types
	extensions := make([]string, 0, len(fs.fileTypes))
	for ext := range fs.fileTypes {
		extensions = append(extensions, ext)
	}
	extensionPattern := strings.Join(extensions, "|")
	pattern := fmt.Sprintf(`^[a-zA-Z0-9_\-]+\.(%s)$`, extensionPattern)

	matched, err := regexp.MatchString(pattern, fileName)
	if err != nil {
		return false
	}
	return matched
}

// parseFilename parses filename into name and extension
func (fs *FileSystem) parseFilename(filename string) (string, string, error) {
	if !fs.isValidFilename(filename) {
		return "", "", fmt.Errorf("invalid filename format")
	}

	parts := strings.Split(filename, ".")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("filename must have extension")
	}

	extension := strings.ToLower(parts[len(parts)-1])
	name := strings.Join(parts[:len(parts)-1], ".")
	return name, extension, nil
}

// GetDir returns the file system directory path
func (fs *FileSystem) GetDir() string {
	return fs.dataDir
}

// GetFile returns a file object by full filename
func (fs *FileSystem) GetFile(fullFilename string) BaseFile {
	if !fs.isValidFilename(fullFilename) {
		return nil
	}
	return fs.files[fullFilename]
}

// ListFiles returns a list of all files in the system
func (fs *FileSystem) ListFiles() []string {
	files := make([]string, 0, len(fs.files))
	for _, fileObj := range fs.files {
		files = append(files, fileObj.GetFullName())
	}
	return files
}

// DisplayFile displays file content
func (fs *FileSystem) DisplayFile(fullFilename string) *string {
	if !fs.isValidFilename(fullFilename) {
		return nil
	}

	fileObj := fs.GetFile(fullFilename)
	if fileObj == nil {
		return nil
	}

	content := fileObj.Read()
	return &content
}

// ReadFile reads file content and returns appropriate message
func (fs *FileSystem) ReadFile(fullFilename string, externalFile bool) string {
	if externalFile {
		// Handle external file reading
		_, extension, err := fs.parseFilename(fullFilename)
		if err != nil {
			return fmt.Sprintf("Error: Invalid filename format %s. Must be alphanumeric with a supported extension.", fullFilename)
		}

		supportedExts := map[string]bool{"md": true, "txt": true, "json": true, "csv": true}
		if !supportedExts[extension] {
			return fmt.Sprintf("Error: Cannot read file %s as %s extension is not supported.", fullFilename, extension)
		}

		content, err := os.ReadFile(fullFilename)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Sprintf("Error: File '%s' not found.", fullFilename)
			}
			if os.IsPermission(err) {
				return fmt.Sprintf("Error: Permission denied to read file '%s'.", fullFilename)
			}
			return fmt.Sprintf("Error: Could not read file '%s'.", fullFilename)
		}

		return fmt.Sprintf("Read from file %s.\n<content>\n%s\n</content>", fullFilename, string(content))
	}

	if !fs.isValidFilename(fullFilename) {
		return INVALID_FILENAME_ERROR_MESSAGE
	}

	fileObj := fs.GetFile(fullFilename)
	if fileObj == nil {
		return fmt.Sprintf("File '%s' not found.", fullFilename)
	}

	content := fileObj.Read()
	return fmt.Sprintf("Read from file %s.\n<content>\n%s\n</content>", fullFilename, content)
}

// WriteFile writes content to file
func (fs *FileSystem) WriteFile(fullFilename string, content string) string {
	if !fs.isValidFilename(fullFilename) {
		return INVALID_FILENAME_ERROR_MESSAGE
	}

	name, extension, err := fs.parseFilename(fullFilename)
	if err != nil {
		return fmt.Sprintf("Error: Invalid filename format %s", fullFilename)
	}

	fileClass := fs.getFileTypeClass(extension)
	if fileClass == nil {
		return fmt.Sprintf("Error: Invalid file extension '%s' for file '%s'.", extension, fullFilename)
	}

	// Create or get existing file
	var fileObj BaseFile
	if existingFile, exists := fs.files[fullFilename]; exists {
		fileObj = existingFile
	} else {
		fileObj = fileClass(name)
		fs.files[fullFilename] = fileObj
	}

	if err := fileObj.Write(content, fs.dataDir); err != nil {
		return fmt.Sprintf("Error: Could not write to file '%s'. %s", fullFilename, err.Error())
	}

	return fmt.Sprintf("Data written to file %s successfully.", fullFilename)
}

// AppendFile appends content to file
func (fs *FileSystem) AppendFile(fullFilename string, content string) string {
	if !fs.isValidFilename(fullFilename) {
		return INVALID_FILENAME_ERROR_MESSAGE
	}

	fileObj := fs.GetFile(fullFilename)
	if fileObj == nil {
		return fmt.Sprintf("File '%s' not found.", fullFilename)
	}

	if err := fileObj.Append(content, fs.dataDir); err != nil {
		return fmt.Sprintf("Error: Could not append to file '%s'. %s", fullFilename, err.Error())
	}

	return fmt.Sprintf("Data appended to file %s successfully.", fullFilename)
}

// SaveExtractedContent saves extracted content to a numbered file
func (fs *FileSystem) SaveExtractedContent(content string) string {
	initialFilename := fmt.Sprintf("extracted_content_%d", fs.extractedContentCount)
	extractedFilename := fmt.Sprintf("%s.md", initialFilename)
	fileObj := NewMarkdownFile(initialFilename)

	if err := fileObj.Write(content, fs.dataDir); err != nil {
		return fmt.Sprintf("Error: Could not save extracted content. %s", err.Error())
	}

	fs.files[extractedFilename] = fileObj
	fs.extractedContentCount++
	return fmt.Sprintf("Extracted content saved to file %s successfully.", extractedFilename)
}

// Describe lists all files with their content information
func (fs *FileSystem) Describe() string {
	const DISPLAY_CHARS = 400
	var description strings.Builder

	for _, fileObj := range fs.files {
		// Skip todo.md from description
		if fileObj.GetFullName() == "todo.md" {
			continue
		}

		content := fileObj.Read()

		// Handle empty files
		if content == "" {
			description.WriteString(fmt.Sprintf("<file>\n%s - [empty file]\n</file>\n", fileObj.GetFullName()))
			continue
		}

		lines := strings.Split(content, "\n")
		lineCount := len(lines)

		// For small files, display the entire content
		wholeFileDescription := fmt.Sprintf("<file>\n%s - %d lines\n<content>\n%s\n</content>\n</file>\n", fileObj.GetFullName(), lineCount, content)
		if len(content) < int(1.5*DISPLAY_CHARS) {
			description.WriteString(wholeFileDescription)
			continue
		}

		// For larger files, display start and end previews
		halfDisplayChars := DISPLAY_CHARS / 2

		// Get start preview
		var startPreview strings.Builder
		startLineCount := 0
		charsCount := 0
		for _, line := range lines {
			if charsCount+len(line)+1 > halfDisplayChars {
				break
			}
			startPreview.WriteString(line + "\n")
			charsCount += len(line) + 1
			startLineCount++
		}

		// Get end preview
		var endPreview strings.Builder
		endLineCount := 0
		charsCount = 0
		for i := len(lines) - 1; i >= 0; i-- {
			line := lines[i]
			if charsCount+len(line)+1 > halfDisplayChars {
				break
			}
			endPreview.WriteString(line + "\n")
			charsCount += len(line) + 1
			endLineCount++
		}

		// Calculate lines in between
		middleLineCount := lineCount - startLineCount - endLineCount
		if middleLineCount <= 0 {
			description.WriteString(wholeFileDescription)
			continue
		}

		startPreviewStr := strings.TrimRight(startPreview.String(), "\n")
		endPreviewStr := strings.TrimRight(endPreview.String(), "\n")

		// Format output
		if startPreviewStr == "" && endPreviewStr == "" {
			description.WriteString(fmt.Sprintf("<file>\n%s - %d lines\n<content>\n%d lines...\n</content>\n</file>\n", fileObj.GetFullName(), lineCount, middleLineCount))
		} else {
			description.WriteString(fmt.Sprintf("<file>\n%s - %d lines\n<content>\n%s\n", fileObj.GetFullName(), lineCount, startPreviewStr))
			description.WriteString(fmt.Sprintf("... %d more lines ...\n", middleLineCount))
			description.WriteString(fmt.Sprintf("%s\n", endPreviewStr))
			description.WriteString("</content>\n</file>\n")
		}
	}

	return strings.TrimRight(description.String(), "\n")
}

// GetTodoContents returns the todo file contents
func (fs *FileSystem) GetTodoContents() string {
	todoFile := fs.GetFile("todo.md")
	if todoFile == nil {
		return ""
	}
	return todoFile.Read()
}

// GetState returns serializable state of the file system
func (fs *FileSystem) GetState() (*FileSystemState, error) {
	filesData := make(map[string]map[string]interface{})

	for fullFilename, fileObj := range fs.files {
		// Create a map to represent the file data
		var fileType string
		switch fileObj.(type) {
		case *MarkdownFile:
			fileType = "MarkdownFile"
		case *TxtFile:
			fileType = "TxtFile"
		case *JsonFile:
			fileType = "JsonFile"
		case *CsvFile:
			fileType = "CsvFile"
		default:
			fileType = "BaseFile"
		}

		filesData[fullFilename] = map[string]interface{}{
			"type": fileType,
			"data": map[string]interface{}{
				"name":    fileObj.GetName(),
				"content": fileObj.GetContent(),
			},
		}
	}

	return &FileSystemState{
		Files:                 filesData,
		BaseDir:               fs.baseDir,
		ExtractedContentCount: fs.extractedContentCount,
	}, nil
}

// Nuke deletes the file system directory
func (fs *FileSystem) Nuke() error {
	return os.RemoveAll(fs.dataDir)
}

// FromState restores file system from serializable state
func FromState(state *FileSystemState) (*FileSystem, error) {
	// Create file system without default files
	fs, err := NewFileSystem(state.BaseDir, false)
	if err != nil {
		return nil, err
	}
	fs.extractedContentCount = state.ExtractedContentCount

	// Restore all files
	for fullFilename, fileData := range state.Files {
		fileType := fileData["type"].(string)
		fileInfo := fileData["data"].(map[string]interface{})

		name := fileInfo["name"].(string)
		content := fileInfo["content"].(string)

		var fileObj BaseFile
		switch fileType {
		case "MarkdownFile":
			fileObj = NewMarkdownFile(name)
		case "TxtFile":
			fileObj = NewTxtFile(name)
		case "JsonFile":
			fileObj = NewJsonFile(name)
		case "CsvFile":
			fileObj = NewCsvFile(name)
		default:
			// Skip unknown file types
			continue
		}

		fileObj.SetContent(content)
		fs.files[fullFilename] = fileObj
		if err := fileObj.SyncToDiskSync(fs.dataDir); err != nil {
			return nil, fmt.Errorf("failed to sync file %s to disk: %w", fullFilename, err)
		}
	}

	return fs, nil
}
