package documenttemplate

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/templateengine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The tests here are the reason the context types live in the domain rather than
// inside the services that build them.
//
// They verify the catalog against the sample struct in both directions, by
// reflection. That makes it impossible to add a field to a context without
// documenting it for the editor, and impossible to document a variable that does
// not exist. Either mistake would otherwise surface as a blank space on a customer's
// invoice, months later.

// reflectPaths enumerates every dotted path reachable on a value.
//
// A slice contributes its element's paths under the slice's own name, matching how
// {{range}} rebinds dot: a catalog declares "ChargeRows" once, with the element's
// fields beneath it.
func reflectPaths(t reflect.Type, prefix string, depth int, out *[]string) {
	const maxDepth = 6
	if depth > maxDepth {
		return
	}

	switch t.Kind() {
	case reflect.Ptr:
		reflectPaths(t.Elem(), prefix, depth, out)

	case reflect.Slice, reflect.Array:
		elem := t.Elem()
		// A list of scalars has no sub-paths of its own.
		if elem.Kind() == reflect.Struct {
			reflectPaths(elem, prefix, depth+1, out)
		}

	case reflect.Struct:
		for i := range t.NumField() {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}

			// An embedded struct contributes its fields at this level, because that
			// is how a template resolves them: {{ .FirstName }} finds a promoted
			// field, and {{ .NotificationRecipient.FirstName }} is not what a
			// notification family's catalog should be asking an administrator to
			// type. Flattening here keeps the bijection honest for shared blocks.
			if field.Anonymous && field.Type.Kind() == reflect.Struct {
				reflectPaths(field.Type, prefix, depth+1, out)
				continue
			}

			path := field.Name
			if prefix != "" {
				path = prefix + "." + field.Name
			}
			*out = append(*out, path)
			reflectPaths(field.Type, path, depth+1, out)
		}

	default:
	}
}

func structPathsFor(sample any) []string {
	var out []string
	reflectPaths(reflect.TypeOf(sample), "", 0, &out)
	sort.Strings(out)
	return out
}

// TestCatalogMatchesSampleStruct is the bijection: every documented path must exist
// on the struct, and every field on the struct must be documented.
func TestCatalogMatchesSampleStruct(t *testing.T) {
	registry := NewRegistry()

	for _, def := range registry.All() {
		t.Run(string(def.Kind), func(t *testing.T) {
			sample, err := registry.SampleContext(def.Kind)
			require.NoError(t, err)

			declared := registry.Paths(def.Kind)
			actual := structPathsFor(sample)

			for _, path := range declared {
				assert.Contains(t, actual, path,
					"%s documents %q but the context struct has no such field",
					def.Kind, path)
			}

			for _, path := range actual {
				assert.Contains(t, declared, path,
					"%s has field %q on its context struct but does not document it, "+
						"so nobody authoring a template can discover it",
					def.Kind, path)
			}
		})
	}
}

// probeSource builds a template that reads every documented path in the scope it
// actually lives in.
//
// A path under a collection only exists inside a range — {{ .ChargeRows.Amount }}
// is meaningless at the root — so each collection ancestor contributes a range and
// each object ancestor a with. Rendering the result exercises the whole catalog in
// one pass.
func probeSource(vars []VariableDefinition) string {
	var b strings.Builder

	for i := range vars {
		v := vars[i]
		ref := "." + v.Path

		switch {
		case v.Type == VariableStringList:
			fmt.Fprintf(&b, "{{ range %s }}{{ . }}{{ end }}", ref)

		case v.Type == VariableCollection:
			fmt.Fprintf(&b, "{{ range %s }}%s{{ end }}", ref, probeSource(v.Fields))

		case v.Type == VariableObject:
			fmt.Fprintf(&b, "{{ with %s }}%s{{ end }}", ref, probeSource(v.Fields))

		default:
			fmt.Fprintf(&b, "{{ %s }}", ref)
		}
	}

	return b.String()
}

// TestEveryDeclaredPathResolvesAtRenderTime goes further than a type comparison: it
// renders every documented path through the real engine, in its correct scope, so a
// path that type-checks but cannot be evaluated still fails.
func TestEveryDeclaredPathResolvesAtRenderTime(t *testing.T) {
	registry := NewRegistry()
	engine, err := templateengine.New(templateengine.Config{})
	require.NoError(t, err)

	for _, def := range registry.All() {
		t.Run(string(def.Kind), func(t *testing.T) {
			sample, sampleErr := registry.SampleContext(def.Kind)
			require.NoError(t, sampleErr)

			source := probeSource(def.Variables)
			require.NotEmpty(t, source)

			compiled, diags := engine.Parse(&templateengine.ParseRequest{
				Field:        "bodyText",
				Kind:         string(def.Kind),
				Channel:      templateengine.ChannelEmailText,
				Source:       source,
				AllowedPaths: registry.Paths(def.Kind),
				Strict:       true,
			})
			require.False(t, diags.HasErrors(),
				"%s: a template reading every documented path must validate: %s",
				def.Kind, diags.Summary())

			_, renderErr := engine.Render(t.Context(), compiled, sample)
			assert.NoError(t, renderErr,
				"%s: a template reading every documented path must render", def.Kind)
		})
	}
}

// TestProbeSourceReferencesEveryPath keeps the test above honest: if the probe
// builder skipped a path, the render would pass without covering it.
func TestProbeSourceReferencesEveryPath(t *testing.T) {
	registry := NewRegistry()

	for _, def := range registry.All() {
		t.Run(string(def.Kind), func(t *testing.T) {
			source := probeSource(def.Variables)

			for _, path := range registry.Paths(def.Kind) {
				leaf := path
				if idx := strings.LastIndex(path, "."); idx >= 0 {
					leaf = path[idx+1:]
				}
				assert.Contains(t, source, "."+leaf,
					"%s: the probe never reads %q", def.Kind, path)
			}
		})
	}
}

// TestRequiredPathsAreDeclared guards against a required path that does not exist:
// it would make every publish fail with a message nobody can act on.
func TestRequiredPathsAreDeclared(t *testing.T) {
	registry := NewRegistry()

	for _, def := range registry.All() {
		t.Run(string(def.Kind), func(t *testing.T) {
			declared := registry.Paths(def.Kind)
			required := registry.RequiredPaths(def.Kind)

			// Notifications are exempt, and the exemption is a consequence of
			// sharing one catalog across a family: no single field is present for
			// every event in a family, so a required field would force a
			// credential name into a load-assignment message. What the check
			// protects against for a document — publishing something empty — is
			// covered for a notification by the version gate, which refuses a
			// version with no content in any channel.
			if def.Category != notificationCategory {
				assert.NotEmpty(t, required,
					"%s declares no required paths; a customer-facing template with no "+
						"mandatory field can be published empty", def.Kind)
			}

			for _, path := range required {
				assert.Contains(t, declared, path)
			}
		})
	}
}

// TestEveryVariableHasADescription enforces the rule that the editor shows these
// verbatim: an undocumented variable is one an administrator has to guess at.
func TestEveryVariableHasADescription(t *testing.T) {
	registry := NewRegistry()

	for _, def := range registry.All() {
		t.Run(string(def.Kind), func(t *testing.T) {
			assertDescriptions(t, string(def.Kind), "", def.Variables)
		})
	}
}

func assertDescriptions(t *testing.T, kind, prefix string, vars []VariableDefinition) {
	t.Helper()

	for i := range vars {
		path := vars[i].Path
		if prefix != "" {
			path = prefix + "." + path
		}

		assert.NotEmpty(t, vars[i].Description,
			"%s: %q has no description, and the editor shows descriptions verbatim",
			kind, path)
		assert.NotEmpty(t, vars[i].Type, "%s: %q has no type", kind, path)

		assertDescriptions(t, kind, path, vars[i].Fields)
	}
}

// TestHostileProbeContextPoisonsEveryString confirms the probe actually reaches
// nested structs and slice elements, since a probe that only replaced top-level
// strings would pass templates that interpolate a nested field into a URL.
func TestHostileProbeContextPoisonsEveryString(t *testing.T) {
	registry := NewRegistry()

	probe, err := registry.HostileProbeContext(KindInvoicePDF)
	require.NoError(t, err)

	invoice, ok := probe.(InvoiceContext)
	require.True(t, ok)

	assert.Equal(t, hostileProbeValue, invoice.InvoiceNumber, "top-level string")
	assert.Equal(t, hostileProbeValue, invoice.BillTo.Name, "nested struct field")
	require.NotEmpty(t, invoice.BillTo.Lines)
	assert.Equal(t, hostileProbeValue, invoice.BillTo.Lines[0], "string slice element")
	require.NotEmpty(t, invoice.ChargeRows)
	assert.Equal(t, hostileProbeValue, invoice.ChargeRows[0].Amount, "struct slice element field")
	require.NotEmpty(t, invoice.BillTo.Details)
	assert.Equal(
		t,
		hostileProbeValue,
		invoice.BillTo.Details[0].Value,
		"nested slice element field",
	)
}

func TestHostileProbeLeavesSampleUntouched(t *testing.T) {
	registry := NewRegistry()

	_, err := registry.HostileProbeContext(KindInvoicePDF)
	require.NoError(t, err)

	sample, err := registry.SampleContext(KindInvoicePDF)
	require.NoError(t, err)

	invoice, ok := sample.(InvoiceContext)
	require.True(t, ok)
	assert.Equal(t, "INV-2026-1042", invoice.InvoiceNumber,
		"the probe must not mutate the shared sample")
}

// TestSampleContextIsRealistic catches placeholder data. A sample of "foo" renders
// without exercising anything: no punctuation to trip an escaper, no second address
// line to reveal a dangling separator, no negative margin to colour.
func TestSampleContextIsRealistic(t *testing.T) {
	registry := NewRegistry()

	for _, def := range registry.All() {
		t.Run(string(def.Kind), func(t *testing.T) {
			sample, err := registry.SampleContext(def.Kind)
			require.NoError(t, err)

			var populated, total int
			countPopulated(reflect.ValueOf(sample), &populated, &total, 0)

			require.Positive(t, total)
			assert.Greater(t, populated*100/total, 70,
				"%s sample leaves most fields empty, so publishing would verify almost nothing",
				def.Kind)
		})
	}
}

func countPopulated(v reflect.Value, populated, total *int, depth int) {
	const maxDepth = 5
	if depth > maxDepth {
		return
	}

	switch v.Kind() {
	case reflect.Struct:
		for i := range v.NumField() {
			field := v.Type().Field(i)
			if !field.IsExported() {
				continue
			}
			*total++
			if !v.Field(i).IsZero() {
				*populated++
			}
			countPopulated(v.Field(i), populated, total, depth+1)
		}

	case reflect.Slice, reflect.Array:
		if v.Len() > 0 {
			countPopulated(v.Index(0), populated, total, depth+1)
		}

	case reflect.Ptr, reflect.Interface:
		if !v.IsNil() {
			countPopulated(v.Elem(), populated, total, depth+1)
		}

	default:
	}
}

func TestPathsAreSortedAndUnique(t *testing.T) {
	registry := NewRegistry()

	for _, def := range registry.All() {
		t.Run(string(def.Kind), func(t *testing.T) {
			paths := registry.Paths(def.Kind)

			assert.True(t, slices.IsSorted(paths),
				"the editor renders this list in order")

			seen := make(map[string]struct{}, len(paths))
			for _, path := range paths {
				_, dup := seen[path]
				assert.False(t, dup, "%q is declared twice", path)
				seen[path] = struct{}{}
			}
		})
	}
}
