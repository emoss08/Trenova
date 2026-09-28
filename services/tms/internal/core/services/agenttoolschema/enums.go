package agenttoolschema

import (
	"fmt"
	"slices"
	"sync"

	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/sliceutils"
)

// EnumSource is where a tool schema's enum values come from: a name a
// contract test can hold the list to, and the values as the domain types
// them, so the same source serves the schema and the read of the argument.
type EnumSource[T ~string] struct {
	Name   string
	Values []T
}

// Names is the source's values as the schema lists them.
func (s EnumSource[T]) Names() []string {
	return sliceutils.Strings(s.Values)
}

// AsStrings is the same source with its values untyped, for a tool that
// carries several sources of different types in one place.
func (s EnumSource[T]) AsStrings() EnumSource[string] {
	return EnumSource[string]{Name: s.Name, Values: s.Names()}
}

type registration struct {
	values  []string
	derived bool
}

var (
	registryMu sync.RWMutex
	registry   = make(map[string]registration, 128)
)

// Source registers a fixed list under its name and returns it for the tools
// that spend it. A list is fixed when the domain declares it: registering
// the same name with different values is a programming error, and panics
// where the package is initialized rather than where a model first hits it.
func Source[T ~string](name string, values []T) EnumSource[T] {
	source := EnumSource[T]{Name: name, Values: slices.Clone(values)}
	register(name, source.Names(), false)

	return source
}

// Derived registers a list a tool computes from a catalog or a
// configuration rather than from a type, such as the fields of a resource.
// The latest computation stands, since two constructions of the same tool
// may read different catalogs, as a test with a stub does.
func Derived[T ~string](name string, values []T) EnumSource[T] {
	source := EnumSource[T]{Name: name, Values: slices.Clone(values)}
	register(name, source.Names(), true)

	return source
}

func register(name string, names []string, derived bool) {
	if name == "" {
		panic("agenttoolschema: an enum source needs a name")
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	existing, found := registry[name]
	switch {
	case !found:
		registry[name] = registration{values: names, derived: derived}
	case existing.derived != derived:
		panic(fmt.Sprintf(
			"agenttoolschema: enum source %q is registered both as fixed and as derived",
			name,
		))
	case derived:
		registry[name] = registration{values: names, derived: true}
	case !slices.Equal(existing.values, names):
		panic(fmt.Sprintf(
			"agenttoolschema: enum source %q is registered twice with different values",
			name,
		))
	}
}

// Enum is a string property whose values are the source's, marked with the
// source's name so the list can be held to it.
func Enum[T ~string](description string, source EnumSource[T]) map[string]any {
	property := map[string]any{
		toolschema.KeyType:   toolschema.TypeString,
		toolschema.KeyEnum:   source.Names(),
		toolschema.KeyEnumOf: source.Name,
	}
	if description != "" {
		property[toolschema.KeyDescription] = description
	}

	return property
}

// Registered is the values a source was registered with, for the contract
// test that checks a schema's list against it.
func Registered(name string) ([]string, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	entry, found := registry[name]
	if !found {
		return nil, false
	}

	return slices.Clone(entry.values), true
}
