package apprise

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"
)

// NotificationTemplate represents a reusable notification template
type NotificationTemplate struct {
	ID          int64             `json:"id" db:"id"`
	Name        string            `json:"name" db:"name"`
	Title       string            `json:"title" db:"title"`
	Body        string            `json:"body" db:"body"`
	Variables   map[string]string `json:"variables" db:"variables"`
	Description string            `json:"description" db:"description"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
}

// TemplateManager manages notification templates
type TemplateManager struct {
	db     *sql.DB
}

// NewTemplateManager creates a new template manager
func NewTemplateManager(db *sql.DB) *TemplateManager {
	return &TemplateManager{
		db: db,
	}
}

// AddTemplate adds a new notification template
func (tm *TemplateManager) AddTemplate(template NotificationTemplate) (*NotificationTemplate, error) {
	now := time.Now()
	template.CreatedAt = now
	template.UpdatedAt = now

	variablesJSON, err := json.Marshal(template.Variables)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal variables: %w", err)
	}

	query := `INSERT INTO notification_templates (name, title, body, variables, description, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?)`

	result, err := tm.db.Exec(query, template.Name, template.Title, template.Body,
		string(variablesJSON), template.Description, template.CreatedAt, template.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert template: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get template ID: %w", err)
	}
	template.ID = id

	return &template, nil
}

// GetTemplate retrieves a template by name
func (tm *TemplateManager) GetTemplate(name string) (*NotificationTemplate, error) {
	query := `SELECT id, name, title, body, variables, description, created_at, updated_at
			  FROM notification_templates WHERE name = ?`

	row := tm.db.QueryRow(query, name)
	return tm.scanTemplateRow(row)
}

// GetTemplates retrieves all templates
func (tm *TemplateManager) GetTemplates() ([]NotificationTemplate, error) {
	query := `SELECT id, name, title, body, variables, description, created_at, updated_at
			  FROM notification_templates ORDER BY name`

	rows, err := tm.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query templates: %w", err)
	}
	defer rows.Close()

	var templates []NotificationTemplate
	for rows.Next() {
		template, err := tm.scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, template)
	}

	return templates, nil
}

// UpdateTemplate updates an existing template
func (tm *TemplateManager) UpdateTemplate(template NotificationTemplate) error {
	template.UpdatedAt = time.Now()

	variablesJSON, err := json.Marshal(template.Variables)
	if err != nil {
		return fmt.Errorf("failed to marshal variables: %w", err)
	}

	query := `UPDATE notification_templates 
			  SET title = ?, body = ?, variables = ?, description = ?, updated_at = ?
			  WHERE name = ?`

	result, err := tm.db.Exec(query, template.Title, template.Body,
		string(variablesJSON), template.Description, template.UpdatedAt, template.Name)
	if err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("template '%s' not found", template.Name)
	}

	return nil
}

// DeleteTemplate removes a template
func (tm *TemplateManager) DeleteTemplate(name string) error {
	query := `DELETE FROM notification_templates WHERE name = ?`

	result, err := tm.db.Exec(query, name)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("template '%s' not found", name)
	}

	return nil
}

// RenderTemplate renders a template with the provided variables
func (tm *TemplateManager) RenderTemplate(templateName string, variables map[string]interface{}) (*NotificationRequest, error) {
	tmpl, err := tm.GetTemplate(templateName)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Merge template variables with provided variables
	allVars := make(map[string]interface{})
	
	// Add template defaults
	for key, value := range tmpl.Variables {
		allVars[key] = value
	}
	
	// Override with provided variables
	for key, value := range variables {
		allVars[key] = value
	}

	// Add system variables
	allVars["timestamp"] = time.Now().Format(time.RFC3339)
	allVars["date"] = time.Now().Format("2006-01-02")
	allVars["time"] = time.Now().Format("15:04:05")

	// Render title
	titleTmpl, err := template.New("title").Parse(tmpl.Title)
	if err != nil {
		return nil, fmt.Errorf("failed to parse title template: %w", err)
	}

	var titleBuf strings.Builder
	if err := titleTmpl.Execute(&titleBuf, allVars); err != nil {
		return nil, fmt.Errorf("failed to render title template: %w", err)
	}

	// Render body
	bodyTmpl, err := template.New("body").Parse(tmpl.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse body template: %w", err)
	}

	var bodyBuf strings.Builder
	if err := bodyTmpl.Execute(&bodyBuf, allVars); err != nil {
		return nil, fmt.Errorf("failed to render body template: %w", err)
	}

	return &NotificationRequest{
		Title:      titleBuf.String(),
		Body:       bodyBuf.String(),
		NotifyType: NotifyTypeInfo, // Default, can be overridden
	}, nil
}

// ValidateTemplate validates a template's syntax
func (tm *TemplateManager) ValidateTemplate(tmpl NotificationTemplate) error {
	// Validate title template
	if _, err := template.New("title").Parse(tmpl.Title); err != nil {
		return fmt.Errorf("invalid title template: %w", err)
	}

	// Validate body template
	if _, err := template.New("body").Parse(tmpl.Body); err != nil {
		return fmt.Errorf("invalid body template: %w", err)
	}

	return nil
}

// scanTemplate scans a template from database rows
func (tm *TemplateManager) scanTemplate(rows *sql.Rows) (NotificationTemplate, error) {
	var template NotificationTemplate
	var variablesJSON string

	err := rows.Scan(&template.ID, &template.Name, &template.Title, &template.Body,
		&variablesJSON, &template.Description, &template.CreatedAt, &template.UpdatedAt)
	if err != nil {
		return template, fmt.Errorf("failed to scan template: %w", err)
	}

	// Parse variables JSON
	if err := json.Unmarshal([]byte(variablesJSON), &template.Variables); err != nil {
		return template, fmt.Errorf("failed to parse variables: %w", err)
	}

	return template, nil
}

// scanTemplateRow scans a template from a single database row
func (tm *TemplateManager) scanTemplateRow(row *sql.Row) (*NotificationTemplate, error) {
	var template NotificationTemplate
	var variablesJSON string

	err := row.Scan(&template.ID, &template.Name, &template.Title, &template.Body,
		&variablesJSON, &template.Description, &template.CreatedAt, &template.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("template not found")
		}
		return nil, fmt.Errorf("failed to scan template: %w", err)
	}

	// Parse variables JSON
	if err := json.Unmarshal([]byte(variablesJSON), &template.Variables); err != nil {
		return nil, fmt.Errorf("failed to parse variables: %w", err)
	}

	return &template, nil
}

// GetTemplateManager returns the template manager for the scheduler
func (s *NotificationScheduler) GetTemplateManager() *TemplateManager {
	return NewTemplateManager(s.db)
}


// CreateDefaultTemplates creates some useful default templates
func (tm *TemplateManager) CreateDefaultTemplates() error {
	templates := []NotificationTemplate{
		{
			Name:        "system-alert",
			Title:       "🚨 System Alert: {{.alert_type | default \"Unknown\"}}",
			Body:        "Alert Details:\n• System: {{.system | default \"Unknown\"}}\n• Severity: {{.severity | default \"Medium\"}}\n• Message: {{.message}}\n• Timestamp: {{.timestamp}}",
			Variables:   map[string]string{"severity": "Medium", "alert_type": "System Alert"},
			Description: "Template for system alerts and notifications",
		},
		{
			Name:        "deployment-status",
			Title:       "🚀 Deployment {{.status | default \"Update\"}}",
			Body:        "Deployment Information:\n• Application: {{.app_name}}\n• Version: {{.version}}\n• Environment: {{.environment | default \"production\"}}\n• Status: {{.status}}\n• Time: {{.timestamp}}",
			Variables:   map[string]string{"environment": "production", "status": "completed"},
			Description: "Template for deployment status notifications",
		},
		{
			Name:        "monitoring-report",
			Title:       "📊 {{.report_type | default \"Monitoring\"}} Report",
			Body:        "Report Summary:\n• Period: {{.period | default \"Last 24 hours\"}}\n• Metrics: {{.metrics}}\n• Status: {{.overall_status | default \"Normal\"}}\n• Details: {{.details}}\n• Generated: {{.timestamp}}",
			Variables:   map[string]string{"period": "Last 24 hours", "overall_status": "Normal"},
			Description: "Template for monitoring and health reports",
		},
		{
			Name:        "backup-status",
			Title:       "💾 Backup {{.status | default \"Completed\"}}",
			Body:        "Backup Details:\n• Database: {{.database}}\n• Size: {{.backup_size | default \"Unknown\"}}\n• Duration: {{.duration | default \"Unknown\"}}\n• Status: {{.status}}\n• Location: {{.backup_location}}\n• Time: {{.timestamp}}",
			Variables:   map[string]string{"status": "completed", "backup_size": "Unknown", "duration": "Unknown"},
			Description: "Template for database backup status notifications",
		},
	}

	for _, template := range templates {
		// Check if template already exists
		existing, err := tm.GetTemplate(template.Name)
		if err == nil && existing != nil {
			continue // Skip if exists
		}

		if _, err := tm.AddTemplate(template); err != nil {
			return fmt.Errorf("failed to create default template %s: %w", template.Name, err)
		}
	}

	return nil
}