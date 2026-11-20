package reply

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/goccy/go-yaml"
)

const (
	generateDir   = "generate"
	guideDir      = "guide"
	yamlExtension = ".yaml"
)

var errNotCarrier = errors.New("reply: the partner is not a carrier")

type Template struct {
	Set      string          `yaml:"set"`
	Segments []partner.Entry `yaml:"segments"`
}

type Carrier struct {
	Definition *partner.Definition
	Guides     *oracle.GuideSet
	templates  map[string]*Template
}

func Open(definition *partner.Definition) (*Carrier, error) {
	if definition.Partner.Role != partner.RoleCarrier {
		return nil, fmt.Errorf("%w: %s", errNotCarrier, definition.Name())
	}
	guides, err := oracle.LoadGuides(filepath.Join(definition.Dir, guideDir))
	if err != nil {
		return nil, err
	}
	carrier := &Carrier{
		Definition: definition,
		Guides:     guides,
		templates:  map[string]*Template{},
	}
	for _, set := range partner.ReplySets {
		if !definition.Partner.SendsSet(set) {
			continue
		}
		path := filepath.Join(definition.Dir, generateDir, set+yamlExtension)
		template, loadErr := loadTemplate(path, set)
		if loadErr != nil {
			return nil, loadErr
		}
		carrier.templates[set] = template
	}
	return carrier, nil
}

func loadTemplate(path, set string) (*Template, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	template := &Template{}
	if err := yaml.UnmarshalWithOptions(raw, template, yaml.DisallowUnknownField()); err != nil {
		return nil, partner.Problems{{File: path, Message: yaml.FormatError(err, false, true)}}
	}
	if template.Set != set {
		return nil, partner.Problems{{
			File:    path,
			Path:    "set",
			Message: fmt.Sprintf("%q must match the file name %q", template.Set, set),
		}}
	}
	if len(template.Segments) == 0 {
		return nil, partner.Problems{{File: path, Path: "segments", Message: "lists no segment"}}
	}
	if where := variationEntry(template.Segments, "segments"); where != "" {
		return nil, partner.Problems{{
			File:    path,
			Path:    where,
			Message: "a reply is not a variation of a tender: on and optional belong to 204 templates",
		}}
	}
	if err := partner.CompileEntries(path, template.Segments); err != nil {
		return nil, err
	}
	return template, nil
}

func variationEntry(entries []partner.Entry, path string) string {
	for index := range entries {
		where := fmt.Sprintf("%s[%d]", path, index)
		if len(entries[index].On) > 0 || entries[index].Optional {
			return where
		}
		if inner := variationEntry(entries[index].Body, where+".body"); inner != "" {
			return inner
		}
	}
	return ""
}

func (c *Carrier) render(set string, vars map[string]any) (partner.Transaction, error) {
	template, ok := c.templates[set]
	if !ok {
		return partner.Transaction{}, fmt.Errorf(
			"reply: %s does not send %s",
			c.Definition.Name(),
			set,
		)
	}
	segments, err := partner.RenderEntries(template.Segments, vars)
	if err != nil {
		return partner.Transaction{}, fmt.Errorf("reply: %s %s: %w", c.Definition.Name(), set, err)
	}
	return partner.Transaction{Set: set, Segments: segments}, nil
}

func (c *Carrier) Write(reply Reply, controls partner.Controls) (partner.Written, error) {
	written, err := c.Definition.Partner.Writer(controls).
		Write(reply.At, []partner.Transaction{reply.Transaction})
	if err != nil {
		return partner.Written{}, err
	}
	return written[0], nil
}
