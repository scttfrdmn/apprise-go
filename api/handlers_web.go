package api

import (
	"embed"
	"net/http"
	"path/filepath"
	"strings"
)

//go:embed web/*
var webFiles embed.FS

// handleDashboard serves the web dashboard
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	// Handle different dashboard routes
	requestPath := r.URL.Path
	var filename string
	
	switch requestPath {
	case "/dashboard", "/dashboard/", "/dashboard.html":
		filename = "web/dashboard.html"
	case "/dashboard.js":
		filename = "web/dashboard.js"
	default:
		// Security: prevent path traversal
		if strings.Contains(requestPath, "..") {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}
		
		// For other paths under /dashboard/, try to serve the file
		path := strings.TrimPrefix(requestPath, "/dashboard")
		if path == "" || path == "/" {
			filename = "web/dashboard.html"
		} else {
			filename = "web" + path
		}
	}
	
	// Try to serve the file from embedded files
	data, err := webFiles.ReadFile(filename)
	if err != nil {
		// File not found, serve dashboard.html as fallback for SPA routing
		data, err = webFiles.ReadFile("web/dashboard.html")
		if err != nil {
			http.Error(w, "Dashboard not found", http.StatusNotFound)
			return
		}
		filename = "web/dashboard.html"
	}
	
	// Set content type based on file extension
	contentType := getContentType(filename)
	w.Header().Set("Content-Type", contentType)
	
	// Set cache headers for static assets
	if strings.HasSuffix(filename, ".js") || strings.HasSuffix(filename, ".css") {
		w.Header().Set("Cache-Control", "public, max-age=31536000") // 1 year
	}
	
	w.Write(data)
}

// getContentType returns the appropriate content type for a file
func getContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	default:
		return "text/plain; charset=utf-8"
	}
}