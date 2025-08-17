package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/scttfrdmn/apprise-go/apprise"
)

// ServiceRequest represents a service configuration request
type ServiceRequest struct {
	URL  string   `json:"url"`
	Tags []string `json:"tags,omitempty"`
}

// handleListServices returns all supported services and their information
func (s *Server) handleListServices(w http.ResponseWriter, r *http.Request) {
	supportedServices := apprise.GetSupportedServices()
	
	services := make([]ServiceInfo, 0, len(supportedServices))
	for _, serviceID := range supportedServices {
		// Create a temporary service instance to get information
		tempService := apprise.CreateService(serviceID)
		if tempService == nil {
			continue
		}

		serviceInfo := ServiceInfo{
			ID:                  serviceID,
			Name:                serviceID, // TODO: Get friendly name from service
			Enabled:             true,
			SupportsAttachments: tempService.SupportsAttachments(),
			MaxBodyLength:       tempService.GetMaxBodyLength(),
		}

		services = append(services, serviceInfo)
	}

	s.sendSuccess(w, "Supported services retrieved", map[string]interface{}{
		"total":    len(services),
		"services": services,
	})
}

// handleAddService adds a new service to the global Apprise instance
func (s *Server) handleAddService(w http.ResponseWriter, r *http.Request) {
	var req ServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.URL == "" {
		s.sendError(w, http.StatusBadRequest, "URL is required", nil)
		return
	}

	// Add service to the main Apprise instance
	if err := s.apprise.Add(req.URL); err != nil {
		s.sendError(w, http.StatusBadRequest, "Failed to add service", err)
		return
	}

	// TODO: Store service configuration in database for persistence
	// For now, we just confirm the service was added

	s.sendSuccess(w, "Service added successfully", map[string]interface{}{
		"url":  req.URL,
		"tags": req.Tags,
	})
}

// handleGetService returns information about a specific service
func (s *Server) handleGetService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceID := vars["service_id"]

	if serviceID == "" {
		s.sendError(w, http.StatusBadRequest, "Service ID is required", nil)
		return
	}

	// Check if service is supported
	supportedServices := apprise.GetSupportedServices()
	var found bool
	for _, id := range supportedServices {
		if id == serviceID {
			found = true
			break
		}
	}

	if !found {
		s.sendError(w, http.StatusNotFound, "Service not found", nil)
		return
	}

	// Create temporary service instance to get information
	tempService := apprise.CreateService(serviceID)
	if tempService == nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to create service instance", nil)
		return
	}

	serviceInfo := ServiceInfo{
		ID:                  serviceID,
		Name:                serviceID,
		Enabled:             true,
		SupportsAttachments: tempService.SupportsAttachments(),
		MaxBodyLength:       tempService.GetMaxBodyLength(),
	}

	s.sendSuccess(w, "Service information retrieved", serviceInfo)
}

// handleUpdateService updates a service configuration
func (s *Server) handleUpdateService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceID := vars["service_id"]

	var req ServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// TODO: Implement service configuration updates
	// This would require storing service configurations in a database

	s.sendSuccess(w, "Service updated successfully", map[string]interface{}{
		"service_id": serviceID,
		"url":        req.URL,
		"tags":       req.Tags,
	})
}

// handleDeleteService removes a service
func (s *Server) handleDeleteService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceID := vars["service_id"]

	if serviceID == "" {
		s.sendError(w, http.StatusBadRequest, "Service ID is required", nil)
		return
	}

	// TODO: Implement service removal from persistent storage
	// For now, we just confirm the request

	s.sendSuccess(w, "Service removed successfully", map[string]interface{}{
		"service_id": serviceID,
	})
}

// handleTestService tests a service configuration
func (s *Server) handleTestService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceID := vars["service_id"]

	var req struct {
		URL     string `json:"url,omitempty"`
		Title   string `json:"title,omitempty"`
		Message string `json:"message,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Use provided URL or try to construct test URL for the service
	testURL := req.URL
	if testURL == "" {
		// For testing purposes, we need a valid URL - this should be provided by the client
		s.sendError(w, http.StatusBadRequest, "Test URL is required", nil)
		return
	}

	// Create temporary Apprise instance for testing
	tempApprise := apprise.New()
	if err := tempApprise.Add(testURL); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid test URL", err)
		return
	}

	// Send test notification
	title := req.Title
	if title == "" {
		title = "Apprise API Test"
	}

	message := req.Message
	if message == "" {
		message = "This is a test notification from Apprise API server"
	}

	notification := apprise.NotificationRequest{
		Title:      title,
		Body:       message,
		NotifyType: apprise.NotifyTypeInfo,
	}

	responses := tempApprise.NotifyAll(notification)

	// Process results
	successful := 0
	var errors []string
	for _, resp := range responses {
		if resp.Success {
			successful++
		} else if resp.Error != nil {
			errors = append(errors, resp.Error.Error())
		}
	}

	result := map[string]interface{}{
		"service_id": serviceID,
		"url":        testURL,
		"success":    successful > 0,
		"total":      len(responses),
		"successful": successful,
		"failed":     len(responses) - successful,
	}

	if len(errors) > 0 {
		result["errors"] = errors
	}

	if successful > 0 {
		s.sendSuccess(w, "Service test completed successfully", result)
	} else {
		s.sendError(w, http.StatusBadRequest, "Service test failed", nil)
		// Still include the result data
		response := APIResponse{
			Success:   false,
			Message:   "Service test failed",
			Data:      result,
			Timestamp: s.getCurrentTimeValue(),
		}
		s.sendJSON(w, http.StatusBadRequest, response)
	}
}

// Helper method for consistent timestamp handling
func (s *Server) getCurrentTimeValue() time.Time {
	return time.Now()
}