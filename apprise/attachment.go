package apprise

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AttachmentType represents the type of attachment
type AttachmentType int

const (
	AttachmentTypeFile AttachmentType = iota
	AttachmentTypeHTTP
	AttachmentTypeMemory
)

// AttachmentInterface defines the interface that all attachments must implement
type AttachmentInterface interface {
	// GetName returns the attachment name
	GetName() string
	
	// GetMimeType returns the MIME type of the attachment
	GetMimeType() string
	
	// GetSize returns the size of the attachment in bytes
	GetSize() int64
	
	// Exists checks if the attachment is available
	Exists() bool
	
	// Open returns a reader for the attachment content
	Open() (io.ReadCloser, error)
	
	// Base64 returns the attachment content as base64 string
	Base64() (string, error)
	
	// Hash returns an MD5 hash of the attachment
	Hash() (string, error)
	
	// GetType returns the attachment type
	GetType() AttachmentType
}

// AttachmentManager manages multiple attachments
type AttachmentManager struct {
	attachments []AttachmentInterface
	maxSize     int64
	timeout     time.Duration
}

// NewAttachmentManager creates a new attachment manager
func NewAttachmentManager() *AttachmentManager {
	return &AttachmentManager{
		attachments: make([]AttachmentInterface, 0),
		maxSize:     100 * 1024 * 1024, // 100MB default max size
		timeout:     30 * time.Second,   // 30 second default timeout
	}
}

// SetMaxSize sets the maximum allowed attachment size
func (am *AttachmentManager) SetMaxSize(size int64) {
	am.maxSize = size
}

// SetTimeout sets the timeout for HTTP attachments
func (am *AttachmentManager) SetTimeout(timeout time.Duration) {
	am.timeout = timeout
}

// Add adds an attachment from various sources
func (am *AttachmentManager) Add(source string, name ...string) error {
	var attachment AttachmentInterface
	var err error
	
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		attachment, err = NewHTTPAttachment(source, am.timeout)
	} else if strings.HasPrefix(source, "data:") {
		attachment, err = NewMemoryAttachmentFromDataURL(source)
	} else {
		// Assume it's a file path
		attachment, err = NewFileAttachment(source)
	}
	
	if err != nil {
		return err
	}
	
	// Set custom name if provided
	if len(name) > 0 && name[0] != "" {
		if fa, ok := attachment.(*FileAttachment); ok {
			fa.customName = name[0]
		} else if ha, ok := attachment.(*HTTPAttachment); ok {
			ha.customName = name[0]
		} else if ma, ok := attachment.(*MemoryAttachment); ok {
			ma.customName = name[0]
		}
	}
	
	// Check size limit
	if attachment.GetSize() > am.maxSize {
		return fmt.Errorf("attachment size (%d bytes) exceeds maximum (%d bytes)", 
			attachment.GetSize(), am.maxSize)
	}
	
	am.attachments = append(am.attachments, attachment)
	return nil
}

// AddData adds a memory attachment from raw data
func (am *AttachmentManager) AddData(data []byte, filename, mimeType string) error {
	attachment := NewMemoryAttachment(data, filename, mimeType)
	
	if attachment.GetSize() > am.maxSize {
		return fmt.Errorf("attachment size (%d bytes) exceeds maximum (%d bytes)", 
			attachment.GetSize(), am.maxSize)
	}
	
	am.attachments = append(am.attachments, attachment)
	return nil
}

// GetAll returns all attachments
func (am *AttachmentManager) GetAll() []AttachmentInterface {
	return am.attachments
}

// Count returns the number of attachments
func (am *AttachmentManager) Count() int {
	return len(am.attachments)
}

// Clear removes all attachments
func (am *AttachmentManager) Clear() {
	am.attachments = am.attachments[:0]
}

// TotalSize returns the total size of all attachments
func (am *AttachmentManager) TotalSize() int64 {
	var total int64
	for _, attachment := range am.attachments {
		total += attachment.GetSize()
	}
	return total
}

// FileAttachment represents a file-based attachment
type FileAttachment struct {
	path       string
	customName string
	mimeType   string
	size       int64
	exists     bool
}

// NewFileAttachment creates a new file attachment
func NewFileAttachment(path string) (*FileAttachment, error) {
	info, err := os.Stat(path)
	if err != nil {
		return &FileAttachment{
			path:   path,
			exists: false,
		}, nil // Don't error on non-existent files, check with Exists()
	}
	
	// Detect MIME type
	mimeType := mime.TypeByExtension(filepath.Ext(path))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	
	return &FileAttachment{
		path:     path,
		mimeType: mimeType,
		size:     info.Size(),
		exists:   true,
	}, nil
}

func (f *FileAttachment) GetName() string {
	if f.customName != "" {
		return f.customName
	}
	return filepath.Base(f.path)
}

func (f *FileAttachment) GetMimeType() string {
	return f.mimeType
}

func (f *FileAttachment) GetSize() int64 {
	return f.size
}

func (f *FileAttachment) Exists() bool {
	return f.exists
}

func (f *FileAttachment) Open() (io.ReadCloser, error) {
	if !f.exists {
		return nil, fmt.Errorf("file does not exist: %s", f.path)
	}
	return os.Open(f.path)
}

func (f *FileAttachment) Base64() (string, error) {
	reader, err := f.Open()
	if err != nil {
		return "", err
	}
	defer reader.Close()
	
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	
	return base64.StdEncoding.EncodeToString(data), nil
}

func (f *FileAttachment) Hash() (string, error) {
	reader, err := f.Open()
	if err != nil {
		return "", err
	}
	defer reader.Close()
	
	hash := md5.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}
	
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func (f *FileAttachment) GetType() AttachmentType {
	return AttachmentTypeFile
}

// HTTPAttachment represents an HTTP-based attachment
type HTTPAttachment struct {
	url        string
	customName string
	mimeType   string
	size       int64
	client     *http.Client
	exists     bool
}

// NewHTTPAttachment creates a new HTTP attachment
func NewHTTPAttachment(url string, timeout time.Duration) (*HTTPAttachment, error) {
	client := &http.Client{Timeout: timeout}
	
	// HEAD request to get metadata
	resp, err := client.Head(url)
	if err != nil {
		return &HTTPAttachment{
			url:    url,
			client: client,
			exists: false,
		}, nil // Don't error on network issues, check with Exists()
	}
	defer resp.Body.Close()
	
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	
	// Try to get size
	var size int64
	if resp.ContentLength > 0 {
		size = resp.ContentLength
	}
	
	return &HTTPAttachment{
		url:      url,
		mimeType: mimeType,
		size:     size,
		client:   client,
		exists:   resp.StatusCode >= 200 && resp.StatusCode < 300,
	}, nil
}

func (h *HTTPAttachment) GetName() string {
	if h.customName != "" {
		return h.customName
	}
	// Extract filename from URL
	parts := strings.Split(h.url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "attachment"
}

func (h *HTTPAttachment) GetMimeType() string {
	return h.mimeType
}

func (h *HTTPAttachment) GetSize() int64 {
	return h.size
}

func (h *HTTPAttachment) Exists() bool {
	return h.exists
}

func (h *HTTPAttachment) Open() (io.ReadCloser, error) {
	if !h.exists {
		return nil, fmt.Errorf("HTTP resource not available: %s", h.url)
	}
	
	resp, err := h.client.Get(h.url)
	if err != nil {
		return nil, err
	}
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP error %d for %s", resp.StatusCode, h.url)
	}
	
	return resp.Body, nil
}

func (h *HTTPAttachment) Base64() (string, error) {
	reader, err := h.Open()
	if err != nil {
		return "", err
	}
	defer reader.Close()
	
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	
	return base64.StdEncoding.EncodeToString(data), nil
}

func (h *HTTPAttachment) Hash() (string, error) {
	reader, err := h.Open()
	if err != nil {
		return "", err
	}
	defer reader.Close()
	
	hash := md5.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}
	
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func (h *HTTPAttachment) GetType() AttachmentType {
	return AttachmentTypeHTTP
}

// MemoryAttachment represents an in-memory attachment
type MemoryAttachment struct {
	data       []byte
	filename   string
	customName string
	mimeType   string
}

// NewMemoryAttachment creates a new memory attachment
func NewMemoryAttachment(data []byte, filename, mimeType string) *MemoryAttachment {
	if mimeType == "" {
		// Try to detect from filename
		mimeType = mime.TypeByExtension(filepath.Ext(filename))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
	}
	
	return &MemoryAttachment{
		data:     data,
		filename: filename,
		mimeType: mimeType,
	}
}

// NewMemoryAttachmentFromDataURL creates a memory attachment from a data URL
func NewMemoryAttachmentFromDataURL(dataURL string) (*MemoryAttachment, error) {
	if !strings.HasPrefix(dataURL, "data:") {
		return nil, fmt.Errorf("invalid data URL")
	}
	
	// Parse data URL: data:mime/type;base64,data
	parts := strings.SplitN(dataURL[5:], ",", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid data URL format")
	}
	
	header := parts[0]
	dataString := parts[1]
	
	// Extract MIME type
	mimeType := "application/octet-stream"
	if idx := strings.Index(header, ";"); idx != -1 {
		mimeType = header[:idx]
	} else {
		mimeType = header
	}
	
	// Decode data (assume base64 for now)
	data, err := base64.StdEncoding.DecodeString(dataString)
	if err != nil {
		return nil, fmt.Errorf("failed to decode data URL: %w", err)
	}
	
	return &MemoryAttachment{
		data:     data,
		filename: "data-attachment",
		mimeType: mimeType,
	}, nil
}

func (m *MemoryAttachment) GetName() string {
	if m.customName != "" {
		return m.customName
	}
	return m.filename
}

func (m *MemoryAttachment) GetMimeType() string {
	return m.mimeType
}

func (m *MemoryAttachment) GetSize() int64 {
	return int64(len(m.data))
}

func (m *MemoryAttachment) Exists() bool {
	return len(m.data) > 0
}

func (m *MemoryAttachment) Open() (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(string(m.data))), nil
}

func (m *MemoryAttachment) Base64() (string, error) {
	return base64.StdEncoding.EncodeToString(m.data), nil
}

func (m *MemoryAttachment) Hash() (string, error) {
	hash := md5.New()
	hash.Write(m.data)
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func (m *MemoryAttachment) GetType() AttachmentType {
	return AttachmentTypeMemory
}