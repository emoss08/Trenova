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

// ClearTemplateCache removes a template from the cache, forcing it to be reloaded on next use
func (s *TemplateService) ClearTemplateCache(name string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.templates, name)
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
func (s *TemplateService) RenderTemplate(name string, data any) (string, error) {
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
func (s *TemplateService) RenderInlineTemplate(content string, data any) (string, error) {
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
