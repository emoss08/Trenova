package email

import (
	"bytes"
	"context"
	"encoding/json" //nolint:depguard // json is better here incase of errors, sonic errors are terrible.
	htmltemplate "html/template"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"go.uber.org/fx"
)

type templateService struct {
	l         *zerolog.Logger
	r         repositories.EmailTemplateRepository
	funcMap   texttemplate.FuncMap
	templates map[string]*htmltemplate.Template // Cache for compiled templates
}

type TemplateServiceParams struct {
	fx.In

	Logger     *logger.Logger
	Repository repositories.EmailTemplateRepository
}

func NewTemplateService(p TemplateServiceParams) services.EmailTemplateService {
	log := p.Logger.With().
		Str("service", "email_template").
		Logger()

	// Create function map for templates
	funcMap := texttemplate.FuncMap{
		// Date formatting functions
		"formatDate":     formatDate,
		"formatDateTime": formatDateTime,
		"now":            time.Now,

		// String functions
		"upper":     strings.ToUpper,
		"lower":     strings.ToLower,
		"title":     toTitleCase,
		"trim":      strings.TrimSpace,
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,

		// Utility functions
		"default": defaultValue,
		"empty":   isEmpty,
		"json":    toJSON,
	}

	return &templateService{
		l:         &log,
		r:         p.Repository,
		funcMap:   funcMap,
		templates: make(map[string]*htmltemplate.Template),
	}
}

func (s *templateService) Create(
	ctx context.Context,
	template *email.Template,
) (*email.Template, error) {
	log := s.l.With().
		Str("operation", "create_template").
		Str("template_name", template.Name).
		Logger()

	// Validate template syntax
	if err := s.validateTemplateSyntax(template); err != nil {
		log.Error().Err(err).Msg("template syntax validation failed")
		return nil, oops.In("template_service").
			Tags("operation", "validate_syntax").
			Tags("template_name", template.Name).
			Time(time.Now()).
			Wrapf(err, "template syntax validation failed")
	}

	// Create in repository
	created, err := s.r.Create(ctx, template)
	if err != nil {
		log.Error().Err(err).Msg("failed to create template")
		return nil, oops.In("template_service").
			Tags("operation", "create").
			Tags("template_name", template.Name).
			Time(time.Now()).
			Wrapf(err, "failed to create template")
	}

	log.Info().
		Str("template_id", created.ID.String()).
		Msg("template created successfully")

	return created, nil
}

func (s *templateService) Update(
	ctx context.Context,
	template *email.Template,
) (*email.Template, error) {
	log := s.l.With().
		Str("operation", "update_template").
		Str("template_id", template.ID.String()).
		Logger()

	// Validate template syntax
	if err := s.validateTemplateSyntax(template); err != nil {
		log.Error().Err(err).Msg("template syntax validation failed")
		return nil, oops.In("template_service").
			Tags("operation", "validate_syntax").
			Tags("template_id", template.ID.String()).
			Time(time.Now()).
			Wrapf(err, "template syntax validation failed")
	}

	// Clear cached template
	delete(s.templates, template.ID.String())

	// Update in repository
	updated, err := s.r.Update(ctx, template)
	if err != nil {
		log.Error().Err(err).Msg("failed to update template")
		return nil, oops.In("template_service").
			Tags("operation", "update").
			Tags("template_id", template.ID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to update template")
	}

	log.Info().Msg("template updated successfully")
	return updated, nil
}

func (s *templateService) Get(ctx context.Context, id pulid.ID) (*email.Template, error) {
	return s.r.Get(ctx, id)
}

func (s *templateService) GetBySlug(
	ctx context.Context,
	slug string,
	organizationID pulid.ID,
) (*email.Template, error) {
	return s.r.GetBySlug(ctx, slug, organizationID)
}

func (s *templateService) List(
	ctx context.Context,
	filter *ports.QueryOptions,
) (*ports.ListResult[*email.Template], error) {
	return s.r.List(ctx, filter)
}

func (s *templateService) Delete(ctx context.Context, id pulid.ID) error {
	log := s.l.With().
		Str("operation", "delete_template").
		Str("template_id", id.String()).
		Logger()

	// Clear cached template
	delete(s.templates, id.String())

	// Delete from repository
	if err := s.r.Delete(ctx, id); err != nil {
		log.Error().Err(err).Msg("failed to delete template")
		return oops.In("template_service").
			Tags("operation", "delete").
			Tags("template_id", id.String()).
			Time(time.Now()).
			Wrapf(err, "failed to delete template")
	}

	log.Info().Msg("template deleted successfully")
	return nil
}

func (s *templateService) PreviewTemplate(
	ctx context.Context,
	id pulid.ID,
	variables map[string]any,
) (*services.PreviewTemplateResponse, error) {
	log := s.l.With().
		Str("operation", "preview_template").
		Str("template_id", id.String()).
		Logger()

	// Get template
	template, err := s.Get(ctx, id)
	if err != nil {
		log.Error().Err(err).Msg("failed to get template")
		return nil, oops.In("template_service").
			Tags("operation", "get_template").
			Tags("template_id", id.String()).
			Time(time.Now()).
			Wrapf(err, "failed to get template")
	}

	// Add sample data if variables are empty
	if len(variables) == 0 {
		variables = s.getSampleData()
	}

	// Render template
	rendered, err := s.RenderTemplate(ctx, template, variables)
	if err != nil {
		log.Error().Err(err).Msg("failed to render template")
		return nil, oops.In("template_service").
			Tags("operation", "render").
			Tags("template_id", id.String()).
			Time(time.Now()).
			Wrapf(err, "failed to render template")
	}

	return &services.PreviewTemplateResponse{
		Subject:  rendered.Subject,
		HTMLBody: rendered.HTMLBody,
		TextBody: rendered.TextBody,
	}, nil
}

func (s *templateService) ValidateVariables(
	ctx context.Context,
	templateID pulid.ID,
	variables map[string]any,
) error {
	log := s.l.With().
		Str("operation", "validate_variables").
		Str("template_id", templateID.String()).
		Logger()

	// Get template
	template, err := s.Get(ctx, templateID)
	if err != nil {
		return oops.In("template_service").
			Tags("operation", "get_template").
			Tags("template_id", templateID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to get template")
	}

	// Parse variable schema if defined
	if template.VariablesSchema != nil {
		if err := s.validateAgainstSchema(variables, template.VariablesSchema); err != nil {
			log.Error().Err(err).Msg("variable validation failed")
			return oops.In("template_service").
				Tags("operation", "validate_schema").
				Tags("template_id", templateID.String()).
				Time(time.Now()).
				Wrapf(err, "variable validation failed")
		}
	}

	// Try to render template to catch missing variables
	if _, err := s.RenderTemplate(ctx, template, variables); err != nil {
		log.Error().Err(err).Msg("template rendering validation failed")
		return oops.In("template_service").
			Tags("operation", "render_validation").
			Tags("template_id", templateID.String()).
			Time(time.Now()).
			Wrapf(err, "template rendering validation failed")
	}

	return nil
}

func (s *templateService) RenderTemplate(
	ctx context.Context,
	template *email.Template,
	variables map[string]any,
) (*services.RenderedTemplate, error) {
	log := s.l.With().
		Str("operation", "render_template").
		Str("template_id", template.ID.String()).
		Logger()

	// Get or compile template
	compiledTemplate, err := s.getCompiledTemplate(template)
	if err != nil {
		log.Error().Err(err).Msg("failed to compile template")
		return nil, oops.In("template_service").
			Tags("operation", "compile").
			Tags("template_id", template.ID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to compile template")
	}

	// Render subject
	subject, err := s.renderText(compiledTemplate.Lookup("subject"), variables)
	if err != nil {
		log.Error().Err(err).Msg("failed to render subject")
		return nil, oops.In("template_service").
			Tags("operation", "render_subject").
			Tags("template_id", template.ID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to render subject")
	}

	// Render HTML body
	htmlBody, err := s.renderHTML(compiledTemplate.Lookup("html"), variables)
	if err != nil {
		log.Error().Err(err).Msg("failed to render HTML body")
		return nil, oops.In("template_service").
			Tags("operation", "render_html").
			Tags("template_id", template.ID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to render HTML body")
	}

	// Render text body (optional)
	textBody := ""
	if textTemplate := compiledTemplate.Lookup("text"); textTemplate != nil {
		textBody, err = s.renderText(textTemplate, variables)
		if err != nil {
			log.Warn().Err(err).Msg("failed to render text body")
			// Don't fail completely if text body fails
		}
	}

	log.Debug().Msg("template rendered successfully")

	return &services.RenderedTemplate{
		Subject:  subject,
		HTMLBody: htmlBody,
		TextBody: textBody,
	}, nil
}

// Helper functions for template processing

// validateTemplateSyntax validates that template syntax is correct
func (s *templateService) validateTemplateSyntax(tmpl *email.Template) error {
	// Validate subject template
	if _, err := texttemplate.New("subject").Funcs(s.funcMap).Parse(tmpl.SubjectTemplate); err != nil {
		return oops.In("template_service").
			Tags("operation", "validate_subject").
			Tags("template_name", tmpl.Name).
			Time(time.Now()).
			Wrapf(err, "invalid subject template syntax")
	}

	// Validate HTML template
	if _, err := htmltemplate.New("html").Funcs(s.funcMap).Parse(tmpl.HTMLTemplate); err != nil {
		return oops.In("template_service").
			Tags("operation", "validate_html").
			Tags("template_name", tmpl.Name).
			Time(time.Now()).
			Wrapf(err, "invalid HTML template syntax")
	}

	// Validate text template if present
	if tmpl.TextTemplate != "" {
		if _, err := texttemplate.New("text").Funcs(s.funcMap).Parse(tmpl.TextTemplate); err != nil {
			return oops.In("template_service").
				Tags("operation", "validate_text").
				Tags("template_name", tmpl.Name).
				Time(time.Now()).
				Wrapf(err, "invalid text template syntax")
		}
	}

	return nil
}

// getCompiledTemplate gets or compiles a template
func (s *templateService) getCompiledTemplate(
	tmpl *email.Template,
) (*htmltemplate.Template, error) {
	templateID := tmpl.ID.String()

	// Check cache
	if cached, exists := s.templates[templateID]; exists {
		return cached, nil
	}

	// Create new template with function map
	compiled := htmltemplate.New("email").Funcs(s.funcMap)

	// Parse subject template
	if _, err := compiled.New("subject").Parse(tmpl.SubjectTemplate); err != nil {
		return nil, oops.In("template_service").
			Tags("operation", "compile_subject").
			Tags("template_id", templateID).
			Time(time.Now()).
			Wrapf(err, "failed to parse subject template")
	}

	// Parse HTML template
	if _, err := compiled.New("html").Parse(tmpl.HTMLTemplate); err != nil {
		return nil, oops.In("template_service").
			Tags("operation", "compile_html").
			Tags("template_id", templateID).
			Time(time.Now()).
			Wrapf(err, "failed to parse HTML template")
	}

	// Parse text template if present
	if tmpl.TextTemplate != "" {
		if _, err := compiled.New("text").Parse(tmpl.TextTemplate); err != nil {
			return nil, oops.In("template_service").
				Tags("operation", "compile_text").
				Tags("template_id", templateID).
				Time(time.Now()).
				Wrapf(err, "failed to parse text template")
		}
	}

	// Cache the compiled template
	s.templates[templateID] = compiled

	return compiled, nil
}

// renderHTML renders an HTML template
func (s *templateService) renderHTML(
	tmpl *htmltemplate.Template,
	variables map[string]any,
) (string, error) {
	if tmpl == nil {
		return "", oops.In("template_service").
			Tags("operation", "render_html").
			Time(time.Now()).
			Errorf("HTML template is nil")
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, variables); err != nil {
		return "", oops.In("template_service").
			Tags("operation", "execute_html").
			Time(time.Now()).
			Wrapf(err, "failed to execute HTML template")
	}

	return buf.String(), nil
}

// renderText renders a text template
func (s *templateService) renderText(
	tmpl *htmltemplate.Template,
	variables map[string]any,
) (string, error) {
	if tmpl == nil {
		return "", oops.In("template_service").
			Tags("operation", "render_text").
			Time(time.Now()).
			Errorf("text template is nil")
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, variables); err != nil {
		return "", oops.In("template_service").
			Tags("operation", "execute_text").
			Time(time.Now()).
			Wrapf(err, "failed to execute text template")
	}

	return buf.String(), nil
}

// validateAgainstSchema validates variables against a JSON schema
func (s *templateService) validateAgainstSchema(
	variables map[string]any,
	schema map[string]any,
) error {
	// Simple validation - check required fields
	if required, ok := schema["required"].([]any); ok {
		for _, field := range required {
			if fieldName, ok := field.(string); ok {
				if _, exists := variables[fieldName]; !exists {
					return oops.In("template_service").
						Tags("operation", "validate_required").
						Tags("missing_field", fieldName).
						Time(time.Now()).
						Errorf("required field '%s' is missing", fieldName)
				}
			}
		}
	}

	// Validate field types if properties are defined
	if properties, ok := schema["properties"].(map[string]any); ok {
		for fieldName, fieldSchema := range properties {
			if value, exists := variables[fieldName]; exists {
				if err := s.validateFieldType(fieldName, value, fieldSchema); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// validateFieldType validates a field's type
func (s *templateService) validateFieldType(fieldName string, value any, schema any) error {
	schemaMap, ok := schema.(map[string]any)
	if !ok {
		return nil // Skip validation if schema is not a map
	}

	expectedType, ok := schemaMap["type"].(string)
	if !ok {
		return nil // Skip validation if type is not specified
	}

	actualType := getValueType(value)
	if actualType != expectedType {
		return oops.In("template_service").
			Tags("operation", "validate_type").
			Tags("field_name", fieldName).
			Tags("expected_type", expectedType).
			Tags("actual_type", actualType).
			Time(time.Now()).
			Errorf("field '%s' has type '%s' but expected '%s'", fieldName, actualType, expectedType)
	}

	return nil
}

// getSampleData returns sample data for template preview
func (s *templateService) getSampleData() map[string]any {
	return map[string]any{
		"CustomerName": "John Doe",
		"CompanyName":  "Acme Corp",
		"OrderNumber":  "ORD-12345",
		"OrderTotal":   199.99,
		"IsVIP":        true,
		"CreatedAt":    time.Now(),
		"Items": []map[string]any{
			{
				"Name":     "Premium Widget",
				"Price":    99.99,
				"Quantity": 1,
			},
			{
				"Name":     "Standard Widget",
				"Price":    49.99,
				"Quantity": 2,
			},
		},
		"ShippingAddress": map[string]any{
			"Street":  "123 Main St",
			"City":    "Anytown",
			"State":   "CA",
			"ZipCode": "12345",
		},
	}
}

// Template function implementations

// formatDate formats a time value as a date
func formatDate(t time.Time) string {
	return t.Format("January 2, 2006")
}

// formatDateTime formats a time value as date and time
func formatDateTime(t time.Time) string {
	return t.Format("January 2, 2006 at 3:04 PM")
}

// defaultValue returns a default value if the input is empty
func defaultValue(defaultVal, value any) any {
	if isEmpty(value) {
		return defaultVal
	}
	return value
}

// isEmpty checks if a value is empty
func isEmpty(value any) bool {
	if value == nil {
		return true
	}

	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	case []any:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	default:
		return false
	}
}

// toJSON converts a value to JSON string
func toJSON(value any) string {
	if value == nil {
		return "null"
	}

	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}

	return string(data)
}

// toTitleCase converts a string to title case
func toTitleCase(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

// getValueType returns the type name for validation
func getValueType(value any) string {
	if value == nil {
		return "null"
	}

	switch value.(type) {
	case string:
		return "string"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "integer"
	case float32, float64:
		return "number"
	case bool:
		return "boolean"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return "unknown"
	}
}
