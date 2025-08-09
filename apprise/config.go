package apprise

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the configuration structure
type Config struct {
	URLs    []URLConfig `yaml:"urls"`
	Version string      `yaml:"version,omitempty"`
}

// URLConfig represents a single URL configuration entry
type URLConfig struct {
	URL  string   `yaml:"url"`
	Tags []string `yaml:"tag,omitempty"`
}

// AppriseConfig manages configuration loading and parsing
type AppriseConfig struct {
	configs []Config
	apprise *Apprise
}

// NewAppriseConfig creates a new configuration manager
func NewAppriseConfig(apprise *Apprise) *AppriseConfig {
	return &AppriseConfig{
		configs: make([]Config, 0),
		apprise: apprise,
	}
}

// AddFromFile loads configuration from a local file
func (ac *AppriseConfig) AddFromFile(configPath string) error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	return ac.parseConfig(string(content), configPath)
}

// AddFromURL loads configuration from a remote URL
func (ac *AppriseConfig) AddFromURL(configURL string) error {
	resp, err := http.Get(configURL)
	if err != nil {
		return fmt.Errorf("failed to fetch config from %s: %w", configURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error %d when fetching config from %s", resp.StatusCode, configURL)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read config response from %s: %w", configURL, err)
	}

	return ac.parseConfig(string(content), configURL)
}

// LoadDefaultConfigs loads configuration from default locations
func (ac *AppriseConfig) LoadDefaultConfigs() error {
	defaultPaths := getDefaultConfigPaths()

	for _, path := range defaultPaths {
		if _, err := os.Stat(path); err == nil {
			if err := ac.AddFromFile(path); err != nil {
				// Log error but continue with other config files
				fmt.Printf("Warning: failed to load config from %s: %v\n", path, err)
			}
		}
	}

	return nil
}

// ApplyToApprise applies all loaded configurations to the Apprise instance
func (ac *AppriseConfig) ApplyToApprise() error {
	for _, config := range ac.configs {
		for _, urlConfig := range config.URLs {
			if err := ac.apprise.Add(urlConfig.URL, urlConfig.Tags...); err != nil {
				return fmt.Errorf("failed to add URL %s: %w", urlConfig.URL, err)
			}
		}
	}
	return nil
}

// parseConfig determines the format and parses the configuration content
func (ac *AppriseConfig) parseConfig(content, source string) error {
	content = strings.TrimSpace(content)

	// Try to determine if it's YAML or text format
	if ac.isYAMLFormat(content) {
		return ac.parseYAMLConfig(content, source)
	}

	return ac.parseTextConfig(content, source)
}

// isYAMLFormat attempts to determine if content is in YAML format
func (ac *AppriseConfig) isYAMLFormat(content string) bool {
	// Simple heuristics to detect YAML
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Look for YAML structure indicators
		if strings.Contains(line, "version:") ||
			strings.Contains(line, "urls:") ||
			strings.HasPrefix(line, "- url:") ||
			strings.HasPrefix(line, "  url:") {
			return true
		}

		// If we see a bare URL without YAML structure, it's probably text format
		if strings.Contains(line, "://") && !strings.Contains(line, ":") {
			return false
		}
	}

	return false
}

// parseYAMLConfig parses YAML format configuration
func (ac *AppriseConfig) parseYAMLConfig(content, source string) error {
	var config Config

	if err := yaml.Unmarshal([]byte(content), &config); err != nil {
		return fmt.Errorf("failed to parse YAML config from %s: %w", source, err)
	}

	ac.configs = append(ac.configs, config)
	return nil
}

// parseTextConfig parses simple text format configuration
func (ac *AppriseConfig) parseTextConfig(content, source string) error {
	config := Config{
		URLs: make([]URLConfig, 0),
	}

	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse URL and optional tags
		urlConfig := ac.parseTextLine(line)
		if urlConfig.URL != "" {
			config.URLs = append(config.URLs, urlConfig)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading config from %s: %w", source, err)
	}

	ac.configs = append(ac.configs, config)
	return nil
}

// parseTextLine parses a single line from text format config
func (ac *AppriseConfig) parseTextLine(line string) URLConfig {
	// Format: URL [tag1,tag2,tag3]
	// or just: URL

	urlConfig := URLConfig{}

	// Look for tags in square brackets at the end
	if idx := strings.LastIndex(line, " ["); idx != -1 && strings.HasSuffix(line, "]") {
		urlConfig.URL = strings.TrimSpace(line[:idx])
		tagsStr := strings.Trim(line[idx+2:len(line)-1], " ")

		if tagsStr != "" {
			tags := strings.Split(tagsStr, ",")
			for _, tag := range tags {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					urlConfig.Tags = append(urlConfig.Tags, tag)
				}
			}
		}
	} else {
		urlConfig.URL = line
	}

	return urlConfig
}

// getDefaultConfigPaths returns default configuration file paths based on OS
func getDefaultConfigPaths() []string {
	var paths []string

	if runtime.GOOS == "windows" {
		// Windows paths
		appData := os.Getenv("APPDATA")
		localAppData := os.Getenv("LOCALAPPDATA")
		allUsersProfile := os.Getenv("ALLUSERSPROFILE")
		programFiles := os.Getenv("PROGRAMFILES")
		commonProgramFiles := os.Getenv("COMMONPROGRAMFILES")

		if appData != "" {
			paths = append(paths,
				filepath.Join(appData, "Apprise", "apprise.conf"),
				filepath.Join(appData, "Apprise", "apprise.yaml"),
				filepath.Join(appData, "Apprise", "apprise.yml"),
			)
		}

		if localAppData != "" {
			paths = append(paths,
				filepath.Join(localAppData, "Apprise", "apprise.conf"),
				filepath.Join(localAppData, "Apprise", "apprise.yaml"),
				filepath.Join(localAppData, "Apprise", "apprise.yml"),
			)
		}

		if allUsersProfile != "" {
			paths = append(paths,
				filepath.Join(allUsersProfile, "Apprise", "apprise.conf"),
				filepath.Join(allUsersProfile, "Apprise", "apprise.yaml"),
			)
		}

		if programFiles != "" {
			paths = append(paths,
				filepath.Join(programFiles, "Apprise", "apprise.conf"),
				filepath.Join(programFiles, "Apprise", "apprise.yaml"),
			)
		}

		if commonProgramFiles != "" {
			paths = append(paths,
				filepath.Join(commonProgramFiles, "Apprise", "apprise.conf"),
				filepath.Join(commonProgramFiles, "Apprise", "apprise.yaml"),
			)
		}
	} else {
		// Unix-like systems (Linux, macOS, etc.)
		homeDir, _ := os.UserHomeDir()

		if homeDir != "" {
			paths = append(paths,
				filepath.Join(homeDir, ".apprise"),
				filepath.Join(homeDir, ".apprise.yaml"),
				filepath.Join(homeDir, ".apprise.yml"),
				filepath.Join(homeDir, ".config", "apprise.conf"),
				filepath.Join(homeDir, ".config", "apprise.yaml"),
				filepath.Join(homeDir, ".config", "apprise.yml"),
				filepath.Join(homeDir, ".config", "apprise", "apprise.conf"),
				filepath.Join(homeDir, ".config", "apprise", "apprise.yaml"),
				filepath.Join(homeDir, ".config", "apprise", "apprise.yml"),
			)
		}

		// System-wide configurations
		paths = append(paths,
			"/etc/apprise.conf",
			"/etc/apprise.yaml",
			"/etc/apprise.yml",
			"/etc/apprise/apprise.conf",
			"/etc/apprise/apprise.yaml",
			"/etc/apprise/apprise.yml",
		)
	}

	return paths
}

// Example YAML configuration format:
/*
version: 1
urls:
  - url: discord://webhook_id/webhook_token
    tag:
      - team
      - alerts
  - url: mailto://user:pass@gmail.com
    tag:
      - admin
  - url: slack://TokenA/TokenB/TokenC/Channel
*/

// Example text configuration format:
/*
# Discord webhook for team notifications
discord://webhook_id/webhook_token [team,alerts]

# Email for admin notifications
mailto://user:pass@gmail.com [admin]

# Slack for general notifications
slack://TokenA/TokenB/TokenC/Channel

# Telegram without tags
tgram://bot_token/chat_id
*/
