package apprise

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"
)

// ConfigTemplate provides configuration templating with environment variables
type ConfigTemplate struct {
	template      *template.Template
	vars          map[string]interface{}
	envVars       map[string]string
	secretsFile   string
	defaultValues map[string]string
	functions     template.FuncMap
}

// ConfigContext contains template context data
type ConfigContext struct {
	Env       map[string]string            `json:"env"`
	Vars      map[string]interface{}       `json:"vars"`
	Defaults  map[string]string            `json:"defaults"`
	Secrets   map[string]string            `json:"secrets"`
	System    SystemInfo                   `json:"system"`
	Runtime   RuntimeInfo                  `json:"runtime"`
	Functions map[string]template.FuncMap `json:"-"`
}

// SystemInfo provides system information for templates
type SystemInfo struct {
	Hostname  string `json:"hostname"`
	User      string `json:"user"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	GoVersion string `json:"go_version"`
	Timestamp string `json:"timestamp"`
	WorkDir   string `json:"work_dir"`
}

// RuntimeInfo provides runtime information for templates
type RuntimeInfo struct {
	StartTime   time.Time `json:"start_time"`
	Uptime      string    `json:"uptime"`
	ConfigFile  string    `json:"config_file"`
	ServiceName string    `json:"service_name"`
	Version     string    `json:"version"`
}

// NewConfigTemplate creates a new configuration template engine
func NewConfigTemplate() *ConfigTemplate {
	ct := &ConfigTemplate{
		vars:          make(map[string]interface{}),
		envVars:       make(map[string]string),
		defaultValues: make(map[string]string),
		functions:     make(template.FuncMap),
	}
	
	// Load environment variables
	ct.loadEnvironmentVariables()
	
	// Register built-in template functions
	ct.registerBuiltinFunctions()
	
	return ct
}

// isEmpty checks if a value is considered empty
func isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}
	
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	case bool:
		return !v
	case int, int8, int16, int32, int64:
		return v == 0
	case uint, uint8, uint16, uint32, uint64:
		return v == 0
	case float32, float64:
		return v == 0.0
	default:
		return false
	}
}

// loadEnvironmentVariables loads all environment variables
func (ct *ConfigTemplate) loadEnvironmentVariables() {
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) == 2 {
			ct.envVars[pair[0]] = pair[1]
		}
	}
}

// registerBuiltinFunctions registers built-in template functions
func (ct *ConfigTemplate) registerBuiltinFunctions() {
	ct.functions = template.FuncMap{
		// String functions
		"upper":    strings.ToUpper,
		"lower":    strings.ToLower,
		"title":    strings.Title,
		"trim":     strings.TrimSpace,
		"replace":  strings.ReplaceAll,
		"contains": strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"split":     strings.Split,
		"joinStr":   strings.Join,
		
		// Environment and variable functions
		"env": func(key string, defaultVal ...string) string {
			if val, exists := ct.envVars[key]; exists && val != "" {
				return val
			}
			if len(defaultVal) > 0 {
				return defaultVal[0]
			}
			return ""
		},
		
		"envRequired": func(key string) (string, error) {
			if val, exists := ct.envVars[key]; exists && val != "" {
				return val, nil
			}
			return "", fmt.Errorf("required environment variable %s not set", key)
		},
		
		"var": func(key string, defaultVal ...interface{}) interface{} {
			if val, exists := ct.vars[key]; exists {
				return val
			}
			if len(defaultVal) > 0 {
				return defaultVal[0]
			}
			return ""
		},
		
		"default": func(key, defaultVal string) string {
			if val, exists := ct.defaultValues[key]; exists {
				return val
			}
			return defaultVal
		},
		
		// File and path functions
		"file": func(path string) (string, error) {
			content, err := os.ReadFile(path)
			if err != nil {
				return "", err
			}
			return string(content), nil
		},
		
		"fileExists": func(path string) bool {
			_, err := os.Stat(path)
			return err == nil
		},
		
		"basename": filepath.Base,
		"dirname":  filepath.Dir,
		"join":     filepath.Join,
		
		// Conditional and comparison functions
		"if": func(condition bool, trueVal, falseVal interface{}) interface{} {
			if condition {
				return trueVal
			}
			return falseVal
		},
		
		"eq": func(a, b interface{}) bool {
			return a == b
		},
		
		"ne": func(a, b interface{}) bool {
			return a != b
		},
		
		"and": func(values ...interface{}) bool {
			for _, v := range values {
				if isEmpty(v) {
					return false
				}
			}
			return len(values) > 0
		},
		
		"or": func(values ...interface{}) bool {
			for _, v := range values {
				if !isEmpty(v) {
					return true
				}
			}
			return false
		},
		
		"not": func(a bool) bool {
			return !a
		},
		
		"empty": isEmpty,
		
		// Date/time functions
		"now":        time.Now,
		"formatTime": func(format string) string { return time.Now().Format(format) },
		"rfc3339":    func() string { return time.Now().Format(time.RFC3339) },
		"unix":       func() int64 { return time.Now().Unix() },
		
		// Utility functions
		"seq": func(start, end int) []int {
			result := make([]int, end-start+1)
			for i := range result {
				result[i] = start + i
			}
			return result
		},
		
		"repeat": func(count int, str string) string {
			return strings.Repeat(str, count)
		},
		
		// URL encoding functions
		"urlEncode": func(str string) string {
			return strings.ReplaceAll(strings.ReplaceAll(str, " ", "%20"), ":", "%3A")
		},
	}
}

// SetVariable sets a template variable
func (ct *ConfigTemplate) SetVariable(key string, value interface{}) {
	ct.vars[key] = value
}

// SetDefault sets a default value
func (ct *ConfigTemplate) SetDefault(key, value string) {
	ct.defaultValues[key] = value
}

// SetSecretsFile sets the path to a secrets file
func (ct *ConfigTemplate) SetSecretsFile(path string) {
	ct.secretsFile = path
}

// AddFunction adds a custom template function
func (ct *ConfigTemplate) AddFunction(name string, fn interface{}) {
	ct.functions[name] = fn
}

// LoadTemplate loads a template from a string
func (ct *ConfigTemplate) LoadTemplate(name, content string) error {
	tmpl, err := template.New(name).Funcs(ct.functions).Parse(content)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}
	ct.template = tmpl
	return nil
}

// LoadTemplateFile loads a template from a file
func (ct *ConfigTemplate) LoadTemplateFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}
	
	name := filepath.Base(path)
	return ct.LoadTemplate(name, string(content))
}

// Execute executes the template with context
func (ct *ConfigTemplate) Execute(writer io.Writer) error {
	if ct.template == nil {
		return fmt.Errorf("no template loaded")
	}
	
	context := ct.createContext()
	return ct.template.Execute(writer, context)
}

// ExecuteToString executes the template and returns the result as a string
func (ct *ConfigTemplate) ExecuteToString() (string, error) {
	var buf strings.Builder
	if err := ct.Execute(&buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// createContext creates the template execution context
func (ct *ConfigTemplate) createContext() *ConfigContext {
	context := &ConfigContext{
		Env:      ct.envVars,
		Vars:     ct.vars,
		Defaults: ct.defaultValues,
		Secrets:  ct.loadSecrets(),
		System:   ct.getSystemInfo(),
		Runtime:  ct.getRuntimeInfo(),
	}
	
	return context
}

// loadSecrets loads secrets from the secrets file
func (ct *ConfigTemplate) loadSecrets() map[string]string {
	secrets := make(map[string]string)
	
	if ct.secretsFile == "" {
		return secrets
	}
	
	file, err := os.Open(ct.secretsFile)
	if err != nil {
		return secrets
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			secrets[key] = value
		}
	}
	
	return secrets
}

// getSystemInfo returns system information
func (ct *ConfigTemplate) getSystemInfo() SystemInfo {
	hostname, _ := os.Hostname()
	user := os.Getenv("USER")
	if user == "" {
		user = os.Getenv("USERNAME")
	}
	workDir, _ := os.Getwd()
	
	return SystemInfo{
		Hostname:  hostname,
		User:      user,
		OS:        os.Getenv("GOOS"),
		Arch:      os.Getenv("GOARCH"),
		GoVersion: os.Getenv("GOVERSION"),
		Timestamp: time.Now().Format(time.RFC3339),
		WorkDir:   workDir,
	}
}

// getRuntimeInfo returns runtime information
func (ct *ConfigTemplate) getRuntimeInfo() RuntimeInfo {
	return RuntimeInfo{
		StartTime:   time.Now(),
		Uptime:      "0s", // Would be calculated from actual start time
		ServiceName: "apprise-go",
		Version:     GetVersion(),
	}
}

// ConfigManager manages configuration files and templates
type ConfigManager struct {
	configDir     string
	templateDir   string
	outputDir     string
	fileWatcher   *FileWatcher
	templates     map[string]*ConfigTemplate
	autoReload    bool
}

// FileWatcher watches for file changes
type FileWatcher struct {
	files     map[string]time.Time
	callbacks map[string]func(string)
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(configDir string) *ConfigManager {
	cm := &ConfigManager{
		configDir:   configDir,
		templateDir: filepath.Join(configDir, "templates"),
		outputDir:   filepath.Join(configDir, "generated"),
		templates:   make(map[string]*ConfigTemplate),
		fileWatcher: &FileWatcher{
			files:     make(map[string]time.Time),
			callbacks: make(map[string]func(string)),
		},
		autoReload: false,
	}
	
	// Ensure directories exist
	os.MkdirAll(cm.templateDir, 0755)
	os.MkdirAll(cm.outputDir, 0755)
	
	return cm
}

// SetAutoReload enables or disables automatic template reloading
func (cm *ConfigManager) SetAutoReload(enabled bool) {
	cm.autoReload = enabled
}

// LoadTemplates loads all templates from the template directory
func (cm *ConfigManager) LoadTemplates() error {
	pattern := filepath.Join(cm.templateDir, "*.tmpl")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to find template files: %w", err)
	}
	
	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), ".tmpl")
		template := NewConfigTemplate()
		
		if err := template.LoadTemplateFile(file); err != nil {
			return fmt.Errorf("failed to load template %s: %w", name, err)
		}
		
		cm.templates[name] = template
		
		if cm.autoReload {
			cm.watchFile(file, func(path string) {
				template.LoadTemplateFile(path)
			})
		}
	}
	
	return nil
}

// SetVariableOnAllTemplates sets a variable on all loaded templates
func (cm *ConfigManager) SetVariableOnAllTemplates(key string, value interface{}) {
	for _, template := range cm.templates {
		template.SetVariable(key, value)
	}
}

// SetDefaultOnAllTemplates sets a default value on all loaded templates
func (cm *ConfigManager) SetDefaultOnAllTemplates(key, value string) {
	for _, template := range cm.templates {
		template.SetDefault(key, value)
	}
}

// GetTemplate returns a template by name
func (cm *ConfigManager) GetTemplate(name string) (*ConfigTemplate, bool) {
	template, exists := cm.templates[name]
	return template, exists
}

// GenerateConfig generates a configuration file from a template
func (cm *ConfigManager) GenerateConfig(templateName, outputName string) error {
	template, exists := cm.templates[templateName]
	if !exists {
		return fmt.Errorf("template %s not found", templateName)
	}
	
	outputPath := filepath.Join(cm.outputDir, outputName)
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()
	
	return template.Execute(file)
}

// GenerateAllConfigs generates all configuration files
func (cm *ConfigManager) GenerateAllConfigs() error {
	for name, template := range cm.templates {
		outputName := name + ".conf"
		outputPath := filepath.Join(cm.outputDir, outputName)
		
		file, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file %s: %w", outputPath, err)
		}
		
		err = template.Execute(file)
		file.Close()
		
		if err != nil {
			return fmt.Errorf("failed to execute template %s: %w", name, err)
		}
	}
	
	return nil
}

// watchFile watches a file for changes
func (cm *ConfigManager) watchFile(path string, callback func(string)) {
	cm.fileWatcher.files[path] = time.Now()
	cm.fileWatcher.callbacks[path] = callback
}

// CheckForChanges checks for file changes and triggers callbacks
func (cm *ConfigManager) CheckForChanges() error {
	for path, lastMod := range cm.fileWatcher.files {
		stat, err := os.Stat(path)
		if err != nil {
			continue
		}
		
		if stat.ModTime().After(lastMod) {
			cm.fileWatcher.files[path] = stat.ModTime()
			if callback, exists := cm.fileWatcher.callbacks[path]; exists {
				callback(path)
			}
		}
	}
	
	return nil
}

// EnvironmentLoader provides environment-specific configuration loading
type EnvironmentLoader struct {
	environment string
	configPaths []string
	envFiles    []string
	overrides   map[string]string
}

// NewEnvironmentLoader creates a new environment loader
func NewEnvironmentLoader(environment string) *EnvironmentLoader {
	el := &EnvironmentLoader{
		environment: environment,
		overrides:   make(map[string]string),
	}
	
	// Default environment file paths
	el.envFiles = []string{
		".env",
		fmt.Sprintf(".env.%s", environment),
		fmt.Sprintf("config/%s.env", environment),
	}
	
	return el
}

// AddConfigPath adds a configuration file path
func (el *EnvironmentLoader) AddConfigPath(path string) {
	el.configPaths = append(el.configPaths, path)
}

// AddEnvFile adds an environment file path
func (el *EnvironmentLoader) AddEnvFile(path string) {
	el.envFiles = append(el.envFiles, path)
}

// SetOverride sets an environment variable override
func (el *EnvironmentLoader) SetOverride(key, value string) {
	el.overrides[key] = value
}

// LoadEnvironment loads environment variables from files and overrides
func (el *EnvironmentLoader) LoadEnvironment() error {
	// Load environment files
	for _, envFile := range el.envFiles {
		if err := el.loadEnvFile(envFile); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to load environment file %s: %w", envFile, err)
		}
	}
	
	// Apply overrides
	for key, value := range el.overrides {
		os.Setenv(key, value)
	}
	
	return nil
}

// loadEnvFile loads environment variables from a file
func (el *EnvironmentLoader) loadEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Regular expression to match environment variable assignments
	envRegex := regexp.MustCompile(`^([A-Z_][A-Z0-9_]*)=(.*)$`)
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Parse environment variable assignment
		matches := envRegex.FindStringSubmatch(line)
		if len(matches) == 3 {
			key := matches[1]
			value := matches[2]
			
			// Remove quotes if present
			if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || 
				(value[0] == '\'' && value[len(value)-1] == '\'')) {
				value = value[1 : len(value)-1]
			}
			
			// Set the environment variable (later files override earlier ones)
			os.Setenv(key, value)
		}
	}
	
	return scanner.Err()
}

// GetEnvironment returns the current environment
func (el *EnvironmentLoader) GetEnvironment() string {
	return el.environment
}

// ValidateRequired validates that required environment variables are set
func (el *EnvironmentLoader) ValidateRequired(required []string) error {
	var missing []string
	
	for _, key := range required {
		if os.Getenv(key) == "" {
			missing = append(missing, key)
		}
	}
	
	if len(missing) > 0 {
		return fmt.Errorf("required environment variables not set: %v", missing)
	}
	
	return nil
}