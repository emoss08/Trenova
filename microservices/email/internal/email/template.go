package email

import (
	"bytes"
	"html/template"
	"path/filepath"
	"sync"

	"github.com/rotisserie/eris"
)

// TemplateService manages the email templates
type TemplateService struct {
	templatesDir string
	templates    map[string]*template.Template
	mutex        sync.RWMutex
}

// NewTemplateService creates a new template service
func NewTemplateService() *TemplateService {
	return &TemplateService{
		templatesDir: "templates",
		templates:    make(map[string]*template.Template),
	}
}

// SetTemplatesDir sets the templates directory
func (s *TemplateService) SetTemplatesDir(dir string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.templatesDir = dir
}

// LoadTemplate loads a template from file
func (s *TemplateService) LoadTemplate(name string) (*template.Template, error) {
	s.mutex.RLock()
	if tmpl, ok := s.templates[name]; ok {
		s.mutex.RUnlock()
		return tmpl, nil
	}
	s.mutex.RUnlock()

	// Template not found in cache, load it
	filePath := filepath.Join(s.templatesDir, name+".html")
	tmpl, err := template.ParseFiles(filePath)
	if err != nil {
		return nil, eris.Wrapf(err, "failed to parse template %s", name)
	}

	// Cache the template
	s.mutex.Lock()
	s.templates[name] = tmpl
	s.mutex.Unlock()

	return tmpl, nil
}

// RenderTemplate renders a template with the given data
func (s *TemplateService) RenderTemplate(name string, data interface{}) (string, error) {
	tmpl, err := s.LoadTemplate(name)
	if err != nil {
		return "", err
	}

	var buffer bytes.Buffer
	if err = tmpl.Execute(&buffer, data); err != nil {
		return "", eris.Wrapf(err, "failed to execute template %s", name)
	}

	return buffer.String(), nil
}

// RenderInlineTemplate renders a template string with the given data
func (s *TemplateService) RenderInlineTemplate(content string, data interface{}) (string, error) {
	tmpl, err := template.New("inline").Parse(content)
	if err != nil {
		return "", eris.Wrap(err, "failed to parse inline template")
	}

	var buffer bytes.Buffer
	if err = tmpl.Execute(&buffer, data); err != nil {
		return "", eris.Wrap(err, "failed to execute inline template")
	}

	return buffer.String(), nil
}

// GetDefaultTemplate returns a default template content for the given type
func (s *TemplateService) GetDefaultTemplate(templateType string) string {
	switch templateType {
	case "welcome":
		return `<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { text-align: center; margin-bottom: 20px; }
        .content { padding: 20px; background-color: #f9f9f9; border-radius: 5px; }
        .footer { text-align: center; margin-top: 20px; font-size: 12px; color: #999; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Welcome to Trenova!</h1>
        </div>
        <div class="content">
            <p>Hello {{.Name}},</p>
            <p>Welcome to Trenova! We're excited to have you on board.</p>
            <p>Your account has been created successfully. You can now log in using your credentials.</p>
            <p>If you have any questions, feel free to contact our support team.</p>
        </div>
        <div class="footer">
            <p>&copy; {{.Year}} Trenova. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`
	case "password-reset":
		return `<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { text-align: center; margin-bottom: 20px; }
        .content { padding: 20px; background-color: #f9f9f9; border-radius: 5px; }
        .button { display: inline-block; padding: 10px 20px; background-color: #007bff; color: white; text-decoration: none; border-radius: 5px; }
        .footer { text-align: center; margin-top: 20px; font-size: 12px; color: #999; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Password Reset</h1>
        </div>
        <div class="content">
            <p>Hello {{.Name}},</p>
            <p>We received a request to reset your password. Click the button below to reset it:</p>
            <p style="text-align: center;">
                <a href="{{.ResetLink}}" class="button">Reset Password</a>
            </p>
            <p>If you didn't request a password reset, please ignore this email or contact support if you have concerns.</p>
            <p>This link will expire in 24 hours.</p>
        </div>
        <div class="footer">
            <p>&copy; {{.Year}} Trenova. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`
	case "invoice":
		return `<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { text-align: center; margin-bottom: 20px; }
        .content { padding: 20px; background-color: #f9f9f9; border-radius: 5px; }
        table { width: 100%; border-collapse: collapse; }
        table th, table td { padding: 8px; text-align: left; border-bottom: 1px solid #ddd; }
        .total { font-weight: bold; text-align: right; margin-top: 20px; }
        .footer { text-align: center; margin-top: 20px; font-size: 12px; color: #999; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Invoice #{{.InvoiceNumber}}</h1>
        </div>
        <div class="content">
            <p>Hello {{.CustomerName}},</p>
            <p>Please find your invoice details below:</p>
            
            <table>
                <tr>
                    <th>Description</th>
                    <th>Quantity</th>
                    <th>Price</th>
                    <th>Amount</th>
                </tr>
                {{range .Items}}
                <tr>
                    <td>{{.Description}}</td>
                    <td>{{.Quantity}}</td>
                    <td>${{.Price}}</td>
                    <td>${{.Amount}}</td>
                </tr>
                {{end}}
            </table>
            
            <div class="total">
                <p>Subtotal: ${{.Subtotal}}</p>
                <p>Tax: ${{.Tax}}</p>
                <p>Total: ${{.Total}}</p>
            </div>
            
            <p>Payment due by: {{.DueDate}}</p>
        </div>
        <div class="footer">
            <p>&copy; {{.Year}} Trenova. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`
	default:
		return `<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { text-align: center; margin-bottom: 20px; }
        .content { padding: 20px; background-color: #f9f9f9; border-radius: 5px; }
        .footer { text-align: center; margin-top: 20px; font-size: 12px; color: #999; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.Subject}}</h1>
        </div>
        <div class="content">
            {{.Body}}
        </div>
        <div class="footer">
            <p>&copy; {{.Year}} Trenova. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`
	}
}
