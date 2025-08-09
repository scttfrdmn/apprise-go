package apprise

import (
	"runtime"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	version := GetVersion()
	if version != "1.9.4-1" {
		t.Errorf("Expected version 1.9.4-1, got %s", version)
	}
}

func TestUpstreamVersion(t *testing.T) {
	upstreamVersion := GetUpstreamVersion()
	if upstreamVersion != "1.9.4" {
		t.Errorf("Expected upstream version 1.9.4, got %s", upstreamVersion)
	}
}

func TestVersionInfo(t *testing.T) {
	info := GetVersionInfo()
	
	if info.Version != "1.9.4-1" {
		t.Errorf("Expected version 1.9.4-1, got %s", info.Version)
	}
	
	if info.UpstreamVersion != "1.9.4" {
		t.Errorf("Expected upstream version 1.9.4, got %s", info.UpstreamVersion)
	}
	
	if info.PortVersion != "1" {
		t.Errorf("Expected port version 1, got %s", info.PortVersion)
	}
	
	if info.GoVersion == "" {
		t.Error("Go version should not be empty")
	}
	
	if info.Platform == "" {
		t.Error("Platform should not be empty")
	}
	
	if info.Architecture == "" {
		t.Error("Architecture should not be empty")
	}
}

func TestGetUserAgent(t *testing.T) {
	userAgent := GetUserAgent()
	
	// Should contain our version
	if !strings.Contains(userAgent, "Apprise-Go/1.9.4-1") {
		t.Errorf("User agent should contain Apprise-Go/1.9.4-1, got %s", userAgent)
	}
	
	// Should contain Go version
	if !strings.Contains(userAgent, runtime.Version()) {
		t.Errorf("User agent should contain Go version, got %s", userAgent)
	}
	
	// Should contain upstream version
	if !strings.Contains(userAgent, "based-on-Apprise/1.9.4") {
		t.Errorf("User agent should contain upstream version, got %s", userAgent)
	}
	
	// Should contain platform info
	expectedPlatform := runtime.GOOS + "/" + runtime.GOARCH
	if !strings.Contains(userAgent, expectedPlatform) {
		t.Errorf("User agent should contain platform info %s, got %s", expectedPlatform, userAgent)
	}
}

func TestVersionInfoString(t *testing.T) {
	info := GetVersionInfo()
	versionString := info.String()
	
	// Should contain all key information
	if !strings.Contains(versionString, "Apprise-Go 1.9.4-1") {
		t.Errorf("Version string should contain Apprise-Go 1.9.4-1, got %s", versionString)
	}
	
	if !strings.Contains(versionString, "based on Apprise 1.9.4") {
		t.Errorf("Version string should contain upstream version, got %s", versionString)
	}
	
	if !strings.Contains(versionString, "port revision 1") {
		t.Errorf("Version string should contain port revision, got %s", versionString)
	}
}

func TestVersionConstants(t *testing.T) {
	// Test that constants are correctly defined
	if Version != "1.9.4-1" {
		t.Errorf("Version constant should be 1.9.4-1, got %s", Version)
	}
	
	if UpstreamVersion != "1.9.4" {
		t.Errorf("UpstreamVersion constant should be 1.9.4, got %s", UpstreamVersion)
	}
	
	if PortVersion != "1" {
		t.Errorf("PortVersion constant should be 1, got %s", PortVersion)
	}
}