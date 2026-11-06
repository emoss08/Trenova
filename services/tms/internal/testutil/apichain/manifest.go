package apichain

import (
	"fmt"
	"os"

	"github.com/bytedance/sonic"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

type Operation struct {
	Name     string
	Kind     ast.Operation
	Hash     string
	Document string
}

func (o Operation) IsQuery() bool {
	return o.Kind == ast.Query
}

func ParseOperation(name, document string) (Operation, error) {
	parsed, err := parser.ParseQuery(&ast.Source{Input: document})
	if err != nil {
		return Operation{}, fmt.Errorf("parse the document of %s: %w", name, err)
	}
	for _, operation := range parsed.Operations {
		if operation.Name == name {
			return Operation{Name: name, Kind: operation.Operation, Document: document}, nil
		}
	}

	return Operation{}, fmt.Errorf("the document holds no operation named %s", name)
}

type Manifest struct {
	byName map[string]Operation
}

func LoadManifest(path string) (*Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	documents := map[string]string{}
	if err = sonic.Unmarshal(raw, &documents); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	manifest := &Manifest{byName: make(map[string]Operation, len(documents))}
	for hash, document := range documents {
		parsed, parseErr := parser.ParseQuery(&ast.Source{Input: document})
		if parseErr != nil {
			return nil, fmt.Errorf("%s: %s: %w", path, hash, parseErr)
		}
		for _, operation := range parsed.Operations {
			if operation.Name == "" {
				continue
			}
			if existing, ok := manifest.byName[operation.Name]; ok && existing.Document != document {
				return nil, fmt.Errorf("%s: operation %s has two documents", path, operation.Name)
			}
			manifest.byName[operation.Name] = Operation{
				Name:     operation.Name,
				Kind:     operation.Operation,
				Hash:     hash,
				Document: document,
			}
		}
	}

	return manifest, nil
}

func (m *Manifest) Operation(name string) (Operation, bool) {
	operation, ok := m.byName[name]
	return operation, ok
}
