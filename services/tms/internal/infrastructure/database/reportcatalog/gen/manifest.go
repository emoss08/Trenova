package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Manifest struct {
	Entities map[string]*EntityManifest `yaml:"entities"`
}

type EntityManifest struct {
	Struct          string                    `yaml:"struct"`
	Resource        string                    `yaml:"resource"`
	Label           string                    `yaml:"label"`
	PluralLabel     string                    `yaml:"pluralLabel"`
	Description     string                    `yaml:"description"`
	Category        string                    `yaml:"category"`
	OwnershipColumn string                    `yaml:"ownershipColumn"`
	ExcludeFields   []string                  `yaml:"excludeFields"`
	Fields          map[string]*FieldManifest `yaml:"fields"`
	Edges           map[string]*EdgeManifest  `yaml:"edges"`
}

type FieldManifest struct {
	Label        string            `yaml:"label"`
	Description  string            `yaml:"description"`
	Type         string            `yaml:"type"`
	Format       string            `yaml:"format"`
	EnumLabels   map[string]string `yaml:"enumLabels"`
	Aggregations []string          `yaml:"aggregations"`
	Filterable   *bool             `yaml:"filterable"`
	Groupable    *bool             `yaml:"groupable"`
}

type EdgeManifest struct {
	Label       string `yaml:"label"`
	Traversable *bool  `yaml:"traversable"`
}

func LoadManifest(path string) (*Manifest, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read manifest: %w", err)
	}

	var manifest Manifest
	decoder := yaml.NewDecoder(strings.NewReader(string(raw)))
	decoder.KnownFields(true)
	if err = decoder.Decode(&manifest); err != nil {
		return nil, nil, fmt.Errorf("parse manifest: %w", err)
	}

	if len(manifest.Entities) == 0 {
		return nil, nil, fmt.Errorf("manifest declares no entities")
	}

	return &manifest, raw, nil
}

func (m *Manifest) SortedEntityKeys() []string {
	keys := make([]string, 0, len(m.Entities))
	for key := range m.Entities {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (e *EntityManifest) SortedEdgeNames() []string {
	names := make([]string, 0, len(e.Edges))
	for name := range e.Edges {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (e *EdgeManifest) IsTraversable() bool {
	return e.Traversable == nil || *e.Traversable
}
