package benchscenario

import (
	"strings"
)

type FormulaPage struct {
	Template string `yaml:"template" json:"template,omitempty"`
	Library  string `yaml:"library"  json:"library,omitempty"`
}

func (p *FormulaPage) EditsLibrary() bool {
	return strings.TrimSpace(p.Library) != ""
}
