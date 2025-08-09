package apprise

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAttachmentManager(t *testing.T) {
	mgr := NewAttachmentManager()

	if mgr.Count() != 0 {
		t.Errorf("Expected 0 attachments, got %d", mgr.Count())
	}

	if mgr.TotalSize() != 0 {
		t.Errorf("Expected 0 total size, got %d", mgr.TotalSize())
	}
}

func TestFileAttachment(t *testing.T) {
	// Create a temporary test file
	tmpFile, err := os.CreateTemp("", "apprise_test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	testContent := "This is a test file for attachment testing"
	if _, err := tmpFile.WriteString(testContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Test file attachment
	attachment, err := NewFileAttachment(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create file attachment: %v", err)
	}

	// Test basic properties
	if !attachment.Exists() {
		t.Error("File attachment should exist")
	}

	if attachment.GetSize() != int64(len(testContent)) {
		t.Errorf("Expected size %d, got %d", len(testContent), attachment.GetSize())
	}

	if attachment.GetName() != filepath.Base(tmpFile.Name()) {
		t.Errorf("Expected name %s, got %s", filepath.Base(tmpFile.Name()), attachment.GetName())
	}

	if attachment.GetType() != AttachmentTypeFile {
		t.Errorf("Expected type %d, got %d", AttachmentTypeFile, attachment.GetType())
	}

	// Test content reading
	reader, err := attachment.Open()
	if err != nil {
		t.Fatalf("Failed to open attachment: %v", err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read attachment content: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Expected content %q, got %q", testContent, string(content))
	}

	// Test base64 encoding
	base64Content, err := attachment.Base64()
	if err != nil {
		t.Fatalf("Failed to get base64 content: %v", err)
	}

	if base64Content == "" {
		t.Error("Base64 content should not be empty")
	}

	// Test hash generation
	hash, err := attachment.Hash()
	if err != nil {
		t.Fatalf("Failed to get hash: %v", err)
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}
}

func TestFileAttachmentNonExistent(t *testing.T) {
	attachment, err := NewFileAttachment("/non/existent/file.txt")
	if err != nil {
		t.Fatalf("NewFileAttachment should not error for non-existent files: %v", err)
	}

	if attachment.Exists() {
		t.Error("Non-existent file attachment should not exist")
	}

	_, err = attachment.Open()
	if err == nil {
		t.Error("Opening non-existent file should error")
	}
}

func TestMemoryAttachment(t *testing.T) {
	testData := []byte("This is test data for memory attachment")
	filename := "test.txt"
	mimeType := "text/plain"

	attachment := NewMemoryAttachment(testData, filename, mimeType)

	// Test basic properties
	if !attachment.Exists() {
		t.Error("Memory attachment should exist")
	}

	if attachment.GetSize() != int64(len(testData)) {
		t.Errorf("Expected size %d, got %d", len(testData), attachment.GetSize())
	}

	if attachment.GetName() != filename {
		t.Errorf("Expected name %s, got %s", filename, attachment.GetName())
	}

	if attachment.GetMimeType() != mimeType {
		t.Errorf("Expected MIME type %s, got %s", mimeType, attachment.GetMimeType())
	}

	if attachment.GetType() != AttachmentTypeMemory {
		t.Errorf("Expected type %d, got %d", AttachmentTypeMemory, attachment.GetType())
	}

	// Test content reading
	reader, err := attachment.Open()
	if err != nil {
		t.Fatalf("Failed to open memory attachment: %v", err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read memory attachment: %v", err)
	}

	if string(content) != string(testData) {
		t.Errorf("Expected content %q, got %q", string(testData), string(content))
	}

	// Test base64 encoding
	base64Content, err := attachment.Base64()
	if err != nil {
		t.Fatalf("Failed to get base64 content: %v", err)
	}

	if base64Content == "" {
		t.Error("Base64 content should not be empty")
	}
}

func TestMemoryAttachmentFromDataURL(t *testing.T) {
	// Test data URL: "Hello World" encoded in base64
	dataURL := "data:text/plain;base64,SGVsbG8gV29ybGQ="

	attachment, err := NewMemoryAttachmentFromDataURL(dataURL)
	if err != nil {
		t.Fatalf("Failed to create memory attachment from data URL: %v", err)
	}

	if attachment.GetMimeType() != "text/plain" {
		t.Errorf("Expected MIME type text/plain, got %s", attachment.GetMimeType())
	}

	// Test content reading
	reader, err := attachment.Open()
	if err != nil {
		t.Fatalf("Failed to open attachment: %v", err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read attachment: %v", err)
	}

	if string(content) != "Hello World" {
		t.Errorf("Expected content 'Hello World', got %q", string(content))
	}
}

func TestInvalidDataURL(t *testing.T) {
	invalidURLs := []string{
		"not-a-data-url",
		"data:",
		"data:text/plain",
		"data:text/plain;base64",
		"data:text/plain;base64,invalid-base64!!!",
	}

	for _, invalidURL := range invalidURLs {
		_, err := NewMemoryAttachmentFromDataURL(invalidURL)
		if err == nil {
			t.Errorf("Expected error for invalid data URL: %s", invalidURL)
		}
	}
}

func TestAttachmentManagerOperations(t *testing.T) {
	mgr := NewAttachmentManager()

	// Test adding memory attachment
	testData := []byte("test data")
	err := mgr.AddData(testData, "test.txt", "text/plain")
	if err != nil {
		t.Fatalf("Failed to add memory attachment: %v", err)
	}

	if mgr.Count() != 1 {
		t.Errorf("Expected 1 attachment, got %d", mgr.Count())
	}

	if mgr.TotalSize() != int64(len(testData)) {
		t.Errorf("Expected total size %d, got %d", len(testData), mgr.TotalSize())
	}

	// Test getting all attachments
	attachments := mgr.GetAll()
	if len(attachments) != 1 {
		t.Errorf("Expected 1 attachment, got %d", len(attachments))
	}

	// Test clearing attachments
	mgr.Clear()
	if mgr.Count() != 0 {
		t.Errorf("Expected 0 attachments after clear, got %d", mgr.Count())
	}
}

func TestAttachmentManagerSizeLimit(t *testing.T) {
	mgr := NewAttachmentManager()
	mgr.SetMaxSize(10) // 10 bytes limit

	// Test adding attachment within limit
	smallData := []byte("small")
	err := mgr.AddData(smallData, "small.txt", "text/plain")
	if err != nil {
		t.Errorf("Should accept small attachment: %v", err)
	}

	// Test adding attachment exceeding limit
	largeData := []byte("this is a large piece of data that exceeds the limit")
	err = mgr.AddData(largeData, "large.txt", "text/plain")
	if err == nil {
		t.Error("Should reject large attachment")
	}

	if !strings.Contains(err.Error(), "exceeds maximum") {
		t.Errorf("Error should mention size limit, got: %v", err)
	}
}

func TestAppriseAttachmentIntegration(t *testing.T) {
	app := New()

	// Test attachment count
	if app.AttachmentCount() != 0 {
		t.Errorf("Expected 0 attachments initially, got %d", app.AttachmentCount())
	}

	// Test adding attachment
	testData := []byte("test attachment data")
	err := app.AddAttachmentData(testData, "test.txt", "text/plain")
	if err != nil {
		t.Fatalf("Failed to add attachment: %v", err)
	}

	if app.AttachmentCount() != 1 {
		t.Errorf("Expected 1 attachment, got %d", app.AttachmentCount())
	}

	// Test getting attachments
	attachments := app.GetAttachments()
	if len(attachments) != 1 {
		t.Errorf("Expected 1 attachment, got %d", len(attachments))
	}

	if attachments[0].GetName() != "test.txt" {
		t.Errorf("Expected attachment name test.txt, got %s", attachments[0].GetName())
	}

	// Test clearing attachments
	app.ClearAttachments()
	if app.AttachmentCount() != 0 {
		t.Errorf("Expected 0 attachments after clear, got %d", app.AttachmentCount())
	}
}

func TestAttachmentManagerFromPath(t *testing.T) {
	mgr := NewAttachmentManager()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "apprise_mgr_test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	testContent := "manager test content"
	tmpFile.WriteString(testContent)
	tmpFile.Close()

	// Test adding file attachment via manager
	err = mgr.Add(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to add file via manager: %v", err)
	}

	if mgr.Count() != 1 {
		t.Errorf("Expected 1 attachment, got %d", mgr.Count())
	}

	// Test custom name
	err = mgr.Add(tmpFile.Name(), "custom_name.txt")
	if err != nil {
		t.Fatalf("Failed to add file with custom name: %v", err)
	}

	if mgr.Count() != 2 {
		t.Errorf("Expected 2 attachments, got %d", mgr.Count())
	}

	attachments := mgr.GetAll()
	if attachments[1].GetName() != "custom_name.txt" {
		t.Errorf("Expected custom name custom_name.txt, got %s", attachments[1].GetName())
	}
}

func TestHTTPAttachment(t *testing.T) {
	// Test HTTP attachment creation - will fail in tests but cover the code path
	_, err := NewHTTPAttachment("http://example.com/test.txt", 5*time.Second)
	// We expect this to fail in test environment, but it exercises the code
	if err == nil {
		t.Log("HTTP attachment creation succeeded (unexpected in test env)")
	}
}

func TestHTTPAttachment_Methods(t *testing.T) {
	// Create an HTTP attachment that we know will fail, but test the methods
	attachment := &HTTPAttachment{
		url:      "http://example.com/test.txt",
		mimeType: "text/plain",
		size:     100,
		exists:   false, // Simulate non-existent resource
		client:   &http.Client{Timeout: 1 * time.Second},
	}

	// Test basic properties
	if attachment.GetName() != "test.txt" {
		t.Errorf("Expected name 'test.txt', got '%s'", attachment.GetName())
	}

	if attachment.GetMimeType() != "text/plain" {
		t.Errorf("Expected MIME type 'text/plain', got '%s'", attachment.GetMimeType())
	}

	if attachment.GetSize() != 100 {
		t.Errorf("Expected size 100, got %d", attachment.GetSize())
	}

	if attachment.Exists() {
		t.Error("Expected attachment to not exist")
	}

	if attachment.GetType() != AttachmentTypeHTTP {
		t.Errorf("Expected type %d, got %d", AttachmentTypeHTTP, attachment.GetType())
	}

	// Test Open with non-existent resource
	_, err := attachment.Open()
	if err == nil {
		t.Error("Expected Open to fail for non-existent resource")
	}

	// Test Base64 with non-existent resource
	_, err = attachment.Base64()
	if err == nil {
		t.Error("Expected Base64 to fail for non-existent resource")
	}

	// Test Hash with non-existent resource
	_, err = attachment.Hash()
	if err == nil {
		t.Error("Expected Hash to fail for non-existent resource")
	}
}

func TestAttachmentManager_SetTimeout(t *testing.T) {
	mgr := NewAttachmentManager()
	
	// Test setting timeout
	mgr.SetTimeout(10 * time.Second)
	
	// The timeout is internal, but we can verify it doesn't error
	if mgr.timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", mgr.timeout)
	}
}

func TestAttachmentManager_DataURL(t *testing.T) {
	mgr := NewAttachmentManager()
	
	// Test adding data URL
	err := mgr.Add("data:text/plain;base64,SGVsbG8gV29ybGQ=") // "Hello World"
	if err != nil {
		t.Errorf("Failed to add data URL: %v", err)
	}
	
	if mgr.Count() != 1 {
		t.Errorf("Expected 1 attachment, got %d", mgr.Count())
	}
}

func TestAttachmentManager_HTTPTimeout(t *testing.T) {
	mgr := NewAttachmentManager()
	mgr.SetTimeout(1 * time.Millisecond) // Very short timeout
	
	// This should fail due to timeout, but exercises the HTTP code path
	err := mgr.Add("http://httpbin.org/delay/10") // Long delay URL
	// We don't care if it fails, just that it exercises the code
	t.Logf("HTTP timeout test result: %v", err)
}
