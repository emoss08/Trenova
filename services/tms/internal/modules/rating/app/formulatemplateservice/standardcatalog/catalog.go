package standardcatalog

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

//go:embed standard_templates.yaml
var catalogYAML []byte

type Template struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Source      string `yaml:"source"`
}

type rawCatalog struct {
	Templates []Template `yaml:"templates"`
}

func Load() ([]Template, error) {
	var raw rawCatalog
	if err := yaml.Unmarshal(catalogYAML, &raw); err != nil {
		return nil, fmt.Errorf("parse standard template catalog: %w", err)
	}
	return raw.Templates, nil
}
