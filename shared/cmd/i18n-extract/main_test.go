package main

import (
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func extractFrom(t *testing.T, files map[string]string) []string {
	t.Helper()
	root := t.TempDir()
	for name, src := range files {
		path := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(src), 0o644))
	}
	e := &extractor{repoRoot: root, fset: token.NewFileSet(), unknown: map[string][]string{}}
	require.NoError(t, e.walkDir(root))
	messages := make([]string, 0, len(e.entries))
	for _, entry := range e.entries {
		messages = append(messages, entry.Message)
	}
	return messages
}

func TestNonMessageTypesKeepTheirFieldsOut(t *testing.T) {
	t.Parallel()

	messages := extractFrom(t, map[string]string{
		"tools/tools.go": `package tools

var create = &masterdatakit.CreateSpec[int]{Description: "Add a carrier"}

var reads = []readspec.Get{{Summary: "Read one invoice"}}

var byName = map[string]*readspec.List{"x": {Summary: "List invoices"}}

var shown = Card{Title: "Shown to a person"}
`,
		"ediconnectionfield/catalog.go": `package ediconnectionfield

var catalog = []*Spec{{Label: "Partner ISA ID"}}
`,
	})

	assert.Equal(t, []string{"Shown to a person"}, messages)
}

func TestTestSupportPackagesAreSkipped(t *testing.T) {
	t.Parallel()

	messages := extractFrom(t, map[string]string{
		"billing/billingtest/fakes.go": `package billingtest

var err = errortypes.NewNotFoundError("Invoice not found")
`,
		"billing/service.go": `package billing

var err = errortypes.NewNotFoundError("Invoice could not be found")
`,
	})

	assert.Equal(t, []string{"Invoice could not be found"}, messages)
}

func TestLabelMethodsAreMessages(t *testing.T) {
	t.Parallel()

	messages := extractFrom(t, map[string]string{
		"agenttemplate/labels.go": `package agenttemplate

func (t Template) Label() string {
	return "Billing assistant"
}

func Label(t Template) string {
	return "Not a method"
}
`,
	})

	assert.Equal(t, []string{"Billing assistant"}, messages)
}

func TestLabelMapsReadByLabelMethodsAreMessages(t *testing.T) {
	t.Parallel()

	messages := extractFrom(t, map[string]string{
		"agenttemplate/labels.go": `package agenttemplate

var templateLabels = map[Template]string{
	TemplateBilling:  "Billing assistant",
	TemplateDispatch: "Dispatch assistant",
}

var templateDescriptions = map[Template]string{
	TemplateBilling: "Not a label",
}

func (t Template) Label() string {
	if label, ok := templateLabels[t]; ok {
		return label
	}
	return string(t)
}

func (t Template) Description() string {
	return templateDescriptions[t]
}
`,
	})

	assert.Equal(t, []string{"Billing assistant", "Dispatch assistant"}, messages)
}
