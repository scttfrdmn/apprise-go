package apprise

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMSTeamsService_AttachmentSupport(t *testing.T) {
	service := NewMSTeamsService().(*MSTeamsService)
	service.teamName = "test-team"
	service.tokenA = "token-a"
	service.tokenB = "token-b"
	service.tokenC = "token-c"
	service.version = 2
	service.webhookURL = service.buildWebhookURL()

	// Test message creation without attachments
	req := NotificationRequest{
		Title:      "Test Without Attachments",
		Body:       "This is a test message without attachments",
		NotifyType: NotifyTypeInfo,
	}

	// Should use standard MessageCard format
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Mock server to capture payload
	var receivedPayload MSTeamsPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&receivedPayload); err != nil {
			t.Fatalf("Failed to decode payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("1"))
	}))
	defer server.Close()

	service.webhookURL = server.URL
	err := service.Send(ctx, req)
	if err != nil {
		t.Fatalf("Failed to send message without attachments: %v", err)
	}

	// Should be standard MessageCard format
	if receivedPayload.Type != "MessageCard" {
		t.Errorf("Expected MessageCard type for message without attachments, got %s", receivedPayload.Type)
	}

	if len(receivedPayload.Attachments) > 0 {
		t.Error("Message without attachments should not have attachments array")
	}
}

func TestMSTeamsService_SendWithAttachments(t *testing.T) {
	service := NewMSTeamsService().(*MSTeamsService)
	service.teamName = "test-team"
	service.tokenA = "token-a"
	service.tokenB = "token-b"
	service.tokenC = "token-c"
	service.version = 2
	service.webhookURL = service.buildWebhookURL()

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

	// Mock server to capture payload
	var receivedPayload MSTeamsPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&receivedPayload); err != nil {
			t.Fatalf("Failed to decode payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("1"))
	}))
	defer server.Close()

	service.webhookURL = server.URL

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = service.Send(ctx, req)
	if err != nil {
		t.Fatalf("Failed to send message with attachments: %v", err)
	}

	// Should use message format with attachments
	if receivedPayload.Type != "message" {
		t.Errorf("Expected message type for message with attachments, got %s", receivedPayload.Type)
	}

	if len(receivedPayload.Attachments) == 0 {
		t.Error("Message with attachments should have attachments array")
	}

	// Should have Adaptive Card attachment
	found := false
	for _, attachment := range receivedPayload.Attachments {
		if attachment.ContentType == "application/vnd.microsoft.card.adaptive" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected Adaptive Card attachment")
	}
}

func TestMSTeamsService_SendWithImageAttachment(t *testing.T) {
	service := NewMSTeamsService().(*MSTeamsService)
	service.teamName = "test-team"
	service.tokenA = "token-a"
	service.tokenB = "token-b"
	service.tokenC = "token-c"
	service.version = 2
	service.webhookURL = service.buildWebhookURL()

	// Create attachment manager with test image data
	attachmentMgr := NewAttachmentManager()
	err := attachmentMgr.AddData([]byte("fake-image-data"), "image.png", "image/png")
	if err != nil {
		t.Fatalf("Failed to add test image: %v", err)
	}

	req := NotificationRequest{
		Title:         "Test With Image",
		Body:          "This message has an image attachment",
		NotifyType:    NotifyTypeSuccess,
		AttachmentMgr: attachmentMgr,
	}

	// Mock server to capture payload
	var receivedPayload MSTeamsPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&receivedPayload); err != nil {
			t.Fatalf("Failed to decode payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("1"))
	}))
	defer server.Close()

	service.webhookURL = server.URL

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = service.Send(ctx, req)
	if err != nil {
		t.Fatalf("Failed to send message with image: %v", err)
	}

	// Check for image attachment with data URL
	foundImage := false
	for _, attachment := range receivedPayload.Attachments {
		if attachment.ContentType == "image/png" {
			foundImage = true
			if attachment.Name != "image.png" {
				t.Errorf("Expected attachment name 'image.png', got '%s'", attachment.Name)
			}
			if !strings.HasPrefix(attachment.ContentURL, "data:image/png;base64,") {
				t.Error("Expected image to have data URL")
			}
			break
		}
	}
	if !foundImage {
		t.Error("Expected to find image attachment")
	}
}

func TestMSTeamsService_CreateAdaptiveCard(t *testing.T) {
	service := &MSTeamsService{}

	req := NotificationRequest{
		Title:      "Test Card Title",
		Body:       "Test card body content",
		NotifyType: NotifyTypeWarning,
	}

	card := service.createAdaptiveCard(req)

	// Check card structure
	if card.Type != "http://adaptivecards.io/schemas/adaptive-card.json" {
		t.Errorf("Expected Adaptive Card schema, got '%s'", card.Type)
	}

	if card.Version != "1.2" {
		t.Errorf("Expected version '1.2', got '%s'", card.Version)
	}

	if len(card.Body) == 0 {
		t.Error("Expected card to have body elements")
	}

	// Check for title element
	foundTitle := false
	for _, element := range card.Body {
		if element.Type == "TextBlock" && element.Text == req.Title {
			foundTitle = true
			if element.Weight != "Bolder" {
				t.Error("Expected title to be bold")
			}
			break
		}
	}
	if !foundTitle {
		t.Error("Expected to find title element")
	}

	// Check for notification type element
	foundType := false
	for _, element := range card.Body {
		if element.Type == "TextBlock" && strings.Contains(element.Text, "warning") {
			foundType = true
			break
		}
	}
	if !foundType {
		t.Error("Expected to find notification type element")
	}
}

func TestMSTeamsService_CreateFileAttachments(t *testing.T) {
	service := &MSTeamsService{}

	// Create attachment manager with multiple files
	attachmentMgr := NewAttachmentManager()

	err := attachmentMgr.AddData([]byte("Document content"), "doc.txt", "text/plain")
	if err != nil {
		t.Fatalf("Failed to add text attachment: %v", err)
	}

	err = attachmentMgr.AddData([]byte("Image data"), "pic.jpg", "image/jpeg")
	if err != nil {
		t.Fatalf("Failed to add image attachment: %v", err)
	}

	attachments, err := service.createFileAttachments(attachmentMgr)
	if err != nil {
		t.Fatalf("Failed to create file attachments: %v", err)
	}

	if len(attachments) != 2 {
		t.Errorf("Expected 2 attachments, got %d", len(attachments))
	}

	// Check text file attachment
	foundText := false
	for _, attachment := range attachments {
		if attachment.Name == "doc.txt" && attachment.ContentType == "text/plain" {
			foundText = true
			// Should have text representation
			if attachment.Content == nil {
				t.Error("Expected text file to have content representation")
			}
			break
		}
	}
	if !foundText {
		t.Error("Expected to find text file attachment")
	}

	// Check image file attachment
	foundImage := false
	for _, attachment := range attachments {
		if attachment.Name == "pic.jpg" && attachment.ContentType == "image/jpeg" {
			foundImage = true
			// Should have data URL
			if attachment.ContentURL == "" {
				t.Error("Expected image to have content URL")
			}
			if !strings.HasPrefix(attachment.ContentURL, "data:image/jpeg;base64,") {
				t.Error("Expected image to have data URL format")
			}
			break
		}
	}
	if !foundImage {
		t.Error("Expected to find image file attachment")
	}
}

func TestMSTeamsService_GetEmojiForNotifyType(t *testing.T) {
	service := &MSTeamsService{}

	tests := []struct {
		notifyType NotifyType
		expected   string
	}{
		{NotifyTypeSuccess, "✅"},
		{NotifyTypeWarning, "⚠️"},
		{NotifyTypeError, "❌"},
		{NotifyTypeInfo, "ℹ️"},
	}

	for _, test := range tests {
		t.Run(test.notifyType.String(), func(t *testing.T) {
			result := service.getEmojiForNotifyType(test.notifyType)
			if result != test.expected {
				t.Errorf("Expected emoji '%s' for %v, got '%s'", test.expected, test.notifyType, result)
			}
		})
	}
}

func TestMSTeamsService_SendWithMultipleAttachments(t *testing.T) {
	service := NewMSTeamsService().(*MSTeamsService)
	service.teamName = "test-team"
	service.tokenA = "token-a"
	service.tokenB = "token-b"
	service.tokenC = "token-c"
	service.version = 2
	service.webhookURL = service.buildWebhookURL()

	// Create attachment manager with multiple files
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

	// Mock server to capture payload
	var receivedPayload MSTeamsPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&receivedPayload); err != nil {
			t.Fatalf("Failed to decode payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("1"))
	}))
	defer server.Close()

	service.webhookURL = server.URL

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = service.Send(ctx, req)
	if err != nil {
		t.Fatalf("Failed to send message with multiple attachments: %v", err)
	}

	// Should have Adaptive Card + 3 file attachments = 4 total
	expectedAttachments := 4 // Adaptive Card + 3 files
	if len(receivedPayload.Attachments) != expectedAttachments {
		t.Errorf("Expected %d attachments, got %d", expectedAttachments, len(receivedPayload.Attachments))
	}

	// Check that all file types are present
	fileTypes := []string{"text/plain", "image/jpeg", "application/pdf"}
	for _, expectedType := range fileTypes {
		found := false
		for _, attachment := range receivedPayload.Attachments {
			if attachment.ContentType == expectedType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find attachment with type %s", expectedType)
		}
	}
}
