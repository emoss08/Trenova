package emailjobs

import (
	"bytes"
	"embed"
	"fmt"
	htmltemplate "html/template"
	"strings"
	texttemplate "text/template"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
)

//go:embed templates/user/* templates/shipment/*
var templateFS embed.FS

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

type RenderedSystemTemplate struct {
	Subject  string
	HTMLBody string
	TextBody string
}

type TemplateManager struct {
	templates map[services.SystemTemplateKey]*SystemTemplate
	funcMap   texttemplate.FuncMap
}

func NewTemplateManager() (*TemplateManager, error) {
	tm := &TemplateManager{
		templates: make(map[services.SystemTemplateKey]*SystemTemplate),
		funcMap:   getTemplateFuncMap(),
	}

	tm.loadTemplates()

	return tm, nil
}

func getTemplateFuncMap() texttemplate.FuncMap {
	return texttemplate.FuncMap{
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": func(s string) string {
			words := strings.Fields(s)
			for i, word := range words {
				if word != "" {
					words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
				}
			}
			return strings.Join(words, " ")
		},
		"trim":      strings.TrimSpace,
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"default": func(defaultVal, value any) any {
			if value == nil || value == "" {
				return defaultVal
			}
			return value
		},
	}
}

func (tm *TemplateManager) loadTemplates() {
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
				"LoginURL",
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
}

func (tm *TemplateManager) loadTemplateFile(path string) (string, error) {
	content, err := templateFS.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func (tm *TemplateManager) RenderTemplate(
	key services.SystemTemplateKey,
	vars map[string]any,
) (*RenderedSystemTemplate, error) {
	template, exists := tm.templates[key]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", key)
	}

	if err := tm.ValidateTemplateVars(template, vars); err != nil {
		return nil, err
	}

	subject, err := tm.renderString(template.SubjectTemplate, vars)
	if err != nil {
		return nil, err
	}

	htmlBody, err := tm.renderHTML(template.HTMLTemplate, vars)
	if err != nil {
		return nil, err
	}

	textBody := ""
	if template.TextTemplate != "" {
		textBody, err = tm.renderString(template.TextTemplate, vars)
		if err != nil {
			textBody = ""
		}
	}

	return &RenderedSystemTemplate{
		Subject:  subject,
		HTMLBody: htmlBody,
		TextBody: textBody,
	}, nil
}

func (tm *TemplateManager) renderString(templateStr string, vars map[string]any) (string, error) {
	tmpl, err := texttemplate.New("template").Funcs(tm.funcMap).Parse(templateStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, vars); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (tm *TemplateManager) renderHTML(templateStr string, vars map[string]any) (string, error) {
	tmpl, err := htmltemplate.New("template").Funcs(tm.funcMap).Parse(templateStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, vars); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (tm *TemplateManager) ValidateTemplateVars(
	template *SystemTemplate,
	vars map[string]any,
) error {
	for _, required := range template.RequiredVars {
		if _, exists := vars[required]; !exists {
			return errortypes.NewValidationError(
				"template_variables",
				errortypes.ErrInvalid,
				fmt.Sprintf("Required template variable '%s' is missing", required),
			)
		}
	}
	return nil
}
