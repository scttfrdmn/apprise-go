package apprise

import (
	"strings"
	"testing"
)

func TestEmailService_AttachmentSupport(t *testing.T) {
	service := NewEmailService().(*EmailService)
	service.fromEmail = "test@example.com"
	service.toEmails = []string{"recipient@example.com"}

	// Test message creation without attachments
	req := NotificationRequest{
		Title:      "Test Without Attachments",
		Body:       "This is a test message without attachments",
		NotifyType: NotifyTypeInfo,
	}

	message, err := service.createMessage(req)
	if err != nil {
		t.Fatalf("Failed to create message without attachments: %v", err)
	}

	// Should be simple message without multipart
	if strings.Contains(message, "multipart/mixed") {
		t.Error("Message without attachments should not be multipart/mixed")
	}

	if strings.Contains(message, "boundary=") {
		t.Error("Message without attachments should not have boundary")
	}
}

func TestEmailService_CreateMessageWithAttachments(t *testing.T) {
	service := NewEmailService().(*EmailService)
	service.fromEmail = "test@example.com"
	service.toEmails = []string{"recipient@example.com"}

	// Create attachment manager with test data
	attachmentMgr := NewAttachmentManager()
	err := attachmentMgr.AddData([]byte("Hello, World!"), "test.txt", "text/plain")
	if err != nil {
		t.Fatalf("Failed to add test attachment: %v", err)
	}

	req := NotificationRequest{
		Title:         "Test With Attachments",
		Body:          "This is a test message with attachments",
		NotifyType:    NotifyTypeInfo,
		AttachmentMgr: attachmentMgr,
	}

	message, err := service.createMessage(req)
	if err != nil {
		t.Fatalf("Failed to create message with attachments: %v", err)
	}

	// Should be multipart message
	if !strings.Contains(message, "multipart/mixed") {
		t.Error("Message with attachments should be multipart/mixed")
	}

	if !strings.Contains(message, "boundary=") {
		t.Error("Message with attachments should have boundary")
	}

	// Should contain attachment headers
	if !strings.Contains(message, "Content-Disposition: attachment") {
		t.Error("Message should contain attachment disposition")
	}

	if !strings.Contains(message, "filename=\"test.txt\"") {
		t.Error("Message should contain attachment filename")
	}

	if !strings.Contains(message, "Content-Type: text/plain") {
		t.Error("Message should contain attachment content type")
	}

	if !strings.Contains(message, "Content-Transfer-Encoding: base64") {
		t.Error("Message should contain base64 encoding")
	}
}

func TestEmailService_CreateMessageWithHTMLAndAttachments(t *testing.T) {
	service := NewEmailService().(*EmailService)
	service.fromEmail = "test@example.com"
	service.toEmails = []string{"recipient@example.com"}

	// Create attachment manager with test image data
	attachmentMgr := NewAttachmentManager()
	err := attachmentMgr.AddData([]byte("fake-image-data"), "image.png", "image/png")
	if err != nil {
		t.Fatalf("Failed to add test attachment: %v", err)
	}

	req := NotificationRequest{
		Title:         "HTML Test With Attachments",
		Body:          "<p>This is an <strong>HTML</strong> message with attachments</p>",
		NotifyType:    NotifyTypeWarning,
		BodyFormat:    "html",
		AttachmentMgr: attachmentMgr,
	}

	message, err := service.createMessage(req)
	if err != nil {
		t.Fatalf("Failed to create HTML message with attachments: %v", err)
	}

	// Should contain HTML content type for body
	if !strings.Contains(message, "Content-Type: text/html; charset=UTF-8") {
		t.Error("HTML message should contain HTML content type")
	}

	// Should contain image as inline attachment
	if !strings.Contains(message, "Content-Disposition: inline") {
		t.Error("Image attachment should be inline")
	}

	if !strings.Contains(message, "Content-ID: <image.png>") {
		t.Error("Image attachment should have Content-ID")
	}

	if !strings.Contains(message, "image/png") {
		t.Error("Message should contain image MIME type")
	}
}

func TestEmailService_GenerateBoundary(t *testing.T) {
	service := NewEmailService().(*EmailService)

	// Test boundary generation
	boundary1, err1 := service.generateBoundary()
	boundary2, err2 := service.generateBoundary()

	if err1 != nil {
		t.Fatalf("Failed to generate first boundary: %v", err1)
	}

	if err2 != nil {
		t.Fatalf("Failed to generate second boundary: %v", err2)
	}

	// Boundaries should be different
	if boundary1 == boundary2 {
		t.Error("Generated boundaries should be unique")
	}

	// Boundaries should have correct format
	if !strings.HasPrefix(boundary1, "----=_Part_") {
		t.Errorf("Boundary should have correct prefix, got: %s", boundary1)
	}

	if !strings.HasPrefix(boundary2, "----=_Part_") {
		t.Errorf("Boundary should have correct prefix, got: %s", boundary2)
	}
}

func TestEmailService_WriteBase64Lines(t *testing.T) {
	service := NewEmailService().(*EmailService)
	var message strings.Builder

	// Test base64 line wrapping
	longBase64 := "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXpBQkNERUZHSElKS0xNTk9QUVJTVFVWV1hZWjEyMzQ1Njc4OTA="
	service.writeBase64Lines(&message, longBase64)

	result := message.String()

	// Should contain CRLF line endings
	if !strings.Contains(result, "\r\n") {
		t.Error("Base64 lines should end with CRLF")
	}

	// Lines should not exceed 76 characters (plus CRLF)
	lines := strings.Split(result, "\r\n")
	for i, line := range lines {
		if len(line) > 76 {
			t.Errorf("Line %d exceeds 76 characters: %d chars", i, len(line))
		}
	}
}

func TestEmailService_AddSingleAttachment(t *testing.T) {
	service := NewEmailService().(*EmailService)
	var message strings.Builder
	boundary := "test-boundary-123"

	// Create test attachment
	attachmentMgr := NewAttachmentManager()
	err := attachmentMgr.AddData([]byte("test-content"), "document.pdf", "application/pdf")
	if err != nil {
		t.Fatalf("Failed to create test attachment: %v", err)
	}

	attachments := attachmentMgr.GetAll()
	if len(attachments) != 1 {
		t.Fatalf("Expected 1 attachment, got %d", len(attachments))
	}

	attachment := attachments[0]

	err = service.addSingleAttachment(&message, boundary, attachment)
	if err != nil {
		t.Fatalf("Failed to add single attachment: %v", err)
	}

	result := message.String()

	// Check that attachment is properly formatted
	expectedParts := []string{
		"--test-boundary-123",
		"Content-Type: application/pdf; name=\"document.pdf\"",
		"Content-Transfer-Encoding: base64",
		"Content-Disposition: attachment; filename=\"document.pdf\"",
	}

	for _, expected := range expectedParts {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected to find '%s' in attachment, but didn't find it", expected)
		}
	}
}

func TestEmailService_MultipleAttachments(t *testing.T) {
	service := NewEmailService().(*EmailService)
	service.fromEmail = "test@example.com"
	service.toEmails = []string{"recipient@example.com"}

	// Create attachment manager with multiple attachments
	attachmentMgr := NewAttachmentManager()
	
	err := attachmentMgr.AddData([]byte("Document content"), "doc.txt", "text/plain")
	if err != nil {
		t.Fatalf("Failed to add text attachment: %v", err)
	}

	err = attachmentMgr.AddData([]byte("Image data"), "pic.jpg", "image/jpeg")
	if err != nil {
		t.Fatalf("Failed to add image attachment: %v", err)
	}

	err = attachmentMgr.AddData([]byte("PDF content"), "file.pdf", "application/pdf")
	if err != nil {
		t.Fatalf("Failed to add PDF attachment: %v", err)
	}

	req := NotificationRequest{
		Title:         "Multiple Attachments Test",
		Body:          "This message has multiple attachments",
		NotifyType:    NotifyTypeInfo,
		AttachmentMgr: attachmentMgr,
	}

	message, err := service.createMessage(req)
	if err != nil {
		t.Fatalf("Failed to create message with multiple attachments: %v", err)
	}

	// Should contain all attachment filenames
	attachmentNames := []string{"doc.txt", "pic.jpg", "file.pdf"}
	for _, name := range attachmentNames {
		if !strings.Contains(message, fmt.Sprintf("filename=\"%s\"", name)) {
			t.Errorf("Message should contain attachment: %s", name)
		}
	}

	// Should contain proper MIME types
	mimeTypes := []string{"text/plain", "image/jpeg", "application/pdf"}
	for _, mimeType := range mimeTypes {
		if !strings.Contains(message, mimeType) {
			t.Errorf("Message should contain MIME type: %s", mimeType)
		}
	}

	// Image should be inline, others should be attachment
	if !strings.Contains(message, "Content-Disposition: inline") {
		t.Error("Image should be inline attachment")
	}

	// Count attachment dispositions
	attachmentCount := strings.Count(message, "Content-Disposition: attachment")
	if attachmentCount != 2 { // doc.txt and file.pdf
		t.Errorf("Expected 2 attachment dispositions, got %d", attachmentCount)
	}

	inlineCount := strings.Count(message, "Content-Disposition: inline")
	if inlineCount != 1 { // pic.jpg
		t.Errorf("Expected 1 inline disposition, got %d", inlineCount)
	}
}