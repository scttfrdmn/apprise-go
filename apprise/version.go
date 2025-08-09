package apprise

import (
	"fmt"
	"runtime"
)

const (
	// Version of this Go port
	Version = "1.9.4-1"
	
	// UpstreamVersion is the version of the original Apprise project this port is based on
	UpstreamVersion = "1.9.4"
	
	// PortVersion is the Go port revision number for this upstream version
	PortVersion = "1"
)

// VersionInfo contains detailed version information
type VersionInfo struct {
	Version         string `json:"version"`
	UpstreamVersion string `json:"upstream_version"`
	PortVersion     string `json:"port_version"`
	GoVersion       string `json:"go_version"`
	Platform        string `json:"platform"`
	Architecture    string `json:"architecture"`
}

// GetVersionInfo returns detailed version information
func GetVersionInfo() VersionInfo {
	return VersionInfo{
		Version:         Version,
		UpstreamVersion: UpstreamVersion,
		PortVersion:     PortVersion,
		GoVersion:       runtime.Version(),
		Platform:        runtime.GOOS,
		Architecture:    runtime.GOARCH,
	}
}

// GetVersion returns the current version string
func GetVersion() string {
	return Version
}

// GetUpstreamVersion returns the upstream Apprise version this port is based on
func GetUpstreamVersion() string {
	return UpstreamVersion
}

// GetUserAgent returns a user agent string for HTTP requests
func GetUserAgent() string {
	return fmt.Sprintf("Apprise-Go/%s (Go %s; %s/%s) based-on-Apprise/%s", 
		Version, runtime.Version(), runtime.GOOS, runtime.GOARCH, UpstreamVersion)
}

// String returns a human-readable version string
func (v VersionInfo) String() string {
	return fmt.Sprintf("Apprise-Go %s (based on Apprise %s, port revision %s)\nGo %s on %s/%s",
		v.Version, v.UpstreamVersion, v.PortVersion, v.GoVersion, v.Platform, v.Architecture)
}