/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package email

import (
	"bytes"
	"embed"
	"fmt"
	htmltemplate "html/template"
	"io/fs"
	"path/filepath"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/samber/oops"
)

//go:embed templates/user/* templates/shipment/*
var templateFS embed.FS

// SystemTemplate represents a predefined email template
type SystemTemplate struct {
	Key             services.SystemTemplateKey
	Name            string
	SubjectTemplate string
	HTMLTemplate    string
	TextTemplate    string
	RequiredVars    []string
	Category        string
	HTMLPath        string
	TextPath        string
}

// RenderedSystemTemplate represents a rendered email template
type RenderedSystemTemplate struct {
	Subject  string
	HTMLBody string
	TextBody string
}

// TemplateManager manages system email templates
type TemplateManager struct {
	templates map[services.SystemTemplateKey]*SystemTemplate
	funcMap   texttemplate.FuncMap
}

// NewTemplateManager creates a new template manager
func NewTemplateManager() (*TemplateManager, error) {
	tm := &TemplateManager{
		templates: make(map[services.SystemTemplateKey]*SystemTemplate),
		funcMap:   getTemplateFuncMap(),
	}

	if err := tm.loadTemplates(); err != nil {
		return nil, err
	}

	return tm, nil
}

// getTemplateFuncMap returns the function map for templates
func getTemplateFuncMap() texttemplate.FuncMap {
	return texttemplate.FuncMap{
		// Date formatting functions
		"formatDate": func(t time.Time) string {
			return t.Format("January 2, 2006")
		},
		"formatDateTime": func(t time.Time) string {
			return t.Format("January 2, 2006 at 3:04 PM")
		},
		"now": time.Now,

		// String functions
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": func(s string) string {
			words := strings.Fields(s)
			for i, word := range words {
				if len(word) > 0 {
					words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
				}
			}
			return strings.Join(words, " ")
		},
		"trim":      strings.TrimSpace,
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,

		// Utility functions
		"default": func(defaultVal, value any) any {
			if value == nil || value == "" {
				return defaultVal
			}
			return value
		},
	}
}

// loadTemplates loads all templates from the embedded filesystem
func (tm *TemplateManager) loadTemplates() error {
	templateConfigs := []struct {
		key          services.SystemTemplateKey
		name         string
		subject      string
		htmlPath     string
		textPath     string
		requiredVars []string
		category     string
	}{
		{
			key:      services.TemplateUserWelcome,
			name:     "User Welcome",
			subject:  "Welcome to Trenova - Account Created",
			htmlPath: "templates/user/welcome.html",
			textPath: "templates/user/welcome.txt",
			requiredVars: []string{
				"UserName",
				"EmailAddress",
				"TemporaryPassword",
				"Year",
			},
			category: "user",
		},
		{
			key:      services.TemplateShipmentOwnershipTransfer,
			name:     "Shipment Ownership Transfer",
			subject:  "Shipment {{.ProNumber}} Ownership Transferred to You",
			htmlPath: "templates/shipment/ownership-transfer.html",
			textPath: "templates/shipment/ownership-transfer.txt",
			requiredVars: []string{
				"NewOwnerName",
				"PreviousOwnerName",
				"ProNumber",
				"TransferDate",
			},
			category: "shipment",
		},
	}

	for _, config := range templateConfigs {
		htmlContent, err := tm.loadTemplateFile(config.htmlPath)
		if err != nil {
			continue
		}

		textContent, err := tm.loadTemplateFile(config.textPath)
		if err != nil {
			textContent = ""
		}

		template := &SystemTemplate{
			Key:             config.key,
			Name:            config.name,
			SubjectTemplate: config.subject,
			HTMLTemplate:    htmlContent,
			TextTemplate:    textContent,
			RequiredVars:    config.requiredVars,
			Category:        config.category,
			HTMLPath:        config.htmlPath,
			TextPath:        config.textPath,
		}

		tm.templates[config.key] = template
	}

	return nil
}

// loadTemplateFile loads a single template file from the embedded filesystem
func (tm *TemplateManager) loadTemplateFile(path string) (string, error) {
	content, err := templateFS.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// GetTemplate returns a system template by key
func (tm *TemplateManager) GetTemplate(key services.SystemTemplateKey) (*SystemTemplate, error) {
	template, exists := tm.templates[key]
	if !exists {
		return nil, oops.In("template_manager").
			Tags("operation", "get_template").
			Tags("template_key", string(key)).
			Time(time.Now()).
			Errorf("template '%s' not found", key)
	}
	return template, nil
}

// RenderTemplate renders a system template with the provided variables
func (tm *TemplateManager) RenderTemplate(
	key services.SystemTemplateKey,
	vars map[string]any,
) (*RenderedSystemTemplate, error) {
	template, err := tm.GetTemplate(key)
	if err != nil {
		return nil, err
	}

	// Validate required variables
	if err := tm.ValidateTemplateVars(template, vars); err != nil {
		return nil, err
	}

	// Render subject
	subject, err := tm.renderString(template.SubjectTemplate, vars)
	if err != nil {
		return nil, oops.In("template_manager").
			Tags("operation", "render_subject").
			Tags("template_key", string(key)).
			Time(time.Now()).
			Wrapf(err, "failed to render subject")
	}

	// Render HTML body
	htmlBody, err := tm.renderHTML(template.HTMLTemplate, vars)
	if err != nil {
		return nil, oops.In("template_manager").
			Tags("operation", "render_html").
			Tags("template_key", string(key)).
			Time(time.Now()).
			Wrapf(err, "failed to render HTML body")
	}

	// Render text body (optional)
	textBody := ""
	if template.TextTemplate != "" {
		textBody, err = tm.renderString(template.TextTemplate, vars)
		if err != nil {
			// Log warning but don't fail
			textBody = ""
		}
	}

	return &RenderedSystemTemplate{
		Subject:  subject,
		HTMLBody: htmlBody,
		TextBody: textBody,
	}, nil
}

// renderString renders a text template string
func (tm *TemplateManager) renderString(templateStr string, vars map[string]any) (string, error) {
	tmpl, err := texttemplate.New("template").Funcs(tm.funcMap).Parse(templateStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// renderHTML renders an HTML template string
func (tm *TemplateManager) renderHTML(templateStr string, vars map[string]any) (string, error) {
	tmpl, err := htmltemplate.New("template").Funcs(tm.funcMap).Parse(templateStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// ValidateTemplateVars checks if all required variables are present
func (tm *TemplateManager) ValidateTemplateVars(
	template *SystemTemplate,
	vars map[string]any,
) error {
	for _, required := range template.RequiredVars {
		if _, exists := vars[required]; !exists {
			return errors.NewValidationError(
				"template_variables",
				errors.ErrInvalid,
				fmt.Sprintf("Required template variable '%s' is missing", required),
			)
		}
	}
	return nil
}

// ListTemplates returns all available system templates
func (tm *TemplateManager) ListTemplates() map[services.SystemTemplateKey]*SystemTemplate {
	return tm.templates
}

// ReloadTemplates reloads all templates from the filesystem
func (tm *TemplateManager) ReloadTemplates() error {
	tm.templates = make(map[services.SystemTemplateKey]*SystemTemplate)
	return tm.loadTemplates()
}

// WalkTemplates walks through all template files in the embedded filesystem
func (tm *TemplateManager) WalkTemplates() ([]string, error) {
	var files []string
	err := fs.WalkDir(templateFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && (filepath.Ext(path) == ".html" || filepath.Ext(path) == ".txt") {
			files = append(files, path)
		}

		return nil
	})
	return files, err
}
