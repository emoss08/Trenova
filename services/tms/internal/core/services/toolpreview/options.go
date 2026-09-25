// Package toolpreview builds what a write tool says it would do. A tool's
// Preview loads the record it acts on, applies the same plan function its
// Execute applies to a copy, and hands both states here; the engine diffs
// them, types and labels each value, stamps its sensitivity and drops what
// no preview may carry. It never reads or writes anything itself.
package toolpreview

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/fieldsensitivity"
	"github.com/emoss08/trenova/shared/pulid"
)

// Record names the record a change is about.
type Record struct {
	Resource permission.Resource
	ID       pulid.ID
	Label    string
	Version  *int64
}

// Option adjusts how one change is built.
type Option func(*options)

type options struct {
	refs        map[string]permission.Resource
	ignore      map[string]struct{}
	volatile    map[string]struct{}
	only        map[string]struct{}
	labels      map[string]string
	types       map[string]assistantartifact.DisplayType
	sensitiveAs []string
	registry    *permission.Registry
}

func newOptions(opts []Option) *options {
	o := &options{registry: fieldsensitivity.DefaultRegistry()}
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}

	return o
}

// WithRefs names the paths whose values are ids of another record, and the
// resource each points at. Their values are shown by the record's label,
// resolved once the preview is built, never by the id.
func WithRefs(refs map[string]permission.Resource) Option {
	return func(o *options) {
		if o.refs == nil {
			o.refs = make(map[string]permission.Resource, len(refs))
		}
		for path, resource := range refs {
			o.refs[path] = resource
		}
	}
}

// Ignore leaves paths out of the change: a value the write sets that says
// nothing to the person deciding it.
func Ignore(paths ...string) Option {
	return func(o *options) { o.ignore = addPaths(o.ignore, paths) }
}

// Volatile marks paths whose value moves on its own between two reads, such
// as a computed age. They are shown and left out of the digest.
func Volatile(paths ...string) Option {
	return func(o *options) { o.volatile = addPaths(o.volatile, paths) }
}

// Only keeps just these paths, in any order the record declares them; for a
// create, the values worth reading on a record that has many.
func Only(paths ...string) Option {
	return func(o *options) { o.only = addPaths(o.only, paths) }
}

// Labels overrides the words a path is shown under.
func Labels(labels map[string]string) Option {
	return func(o *options) {
		if o.labels == nil {
			o.labels = make(map[string]string, len(labels))
		}
		for path, label := range labels {
			o.labels[path] = label
		}
	}
}

// Types overrides how a path's value is drawn.
func Types(types map[string]assistantartifact.DisplayType) Option {
	return func(o *options) {
		if o.types == nil {
			o.types = make(map[string]assistantartifact.DisplayType, len(types))
		}
		for path, displayType := range types {
			o.types[path] = displayType
		}
	}
}

// SensitiveAs names the fields a money block is as sensitive as: the block
// takes the highest of their levels on the record's resource.
func SensitiveAs(paths ...string) Option {
	return func(o *options) { o.sensitiveAs = append(o.sensitiveAs, paths...) }
}

// WithRegistry classifies sensitivity against a registry other than the
// process's own.
func WithRegistry(registry *permission.Registry) Option {
	return func(o *options) {
		if registry != nil {
			o.registry = registry
		}
	}
}

func addPaths(set map[string]struct{}, paths []string) map[string]struct{} {
	if set == nil {
		set = make(map[string]struct{}, len(paths))
	}
	for _, path := range paths {
		if path = strings.TrimSpace(path); path != "" {
			set[path] = struct{}{}
		}
	}

	return set
}

func topLevel(path string) string {
	top, _, _ := strings.Cut(path, ".")

	return top
}

func matches(set map[string]struct{}, path string) bool {
	if len(set) == 0 {
		return false
	}
	if _, ok := set[path]; ok {
		return true
	}
	_, ok := set[topLevel(path)]

	return ok
}

func (o *options) ref(path string) (permission.Resource, bool) {
	if resource, ok := o.refs[path]; ok {
		return resource, true
	}
	resource, ok := o.refs[topLevel(path)]

	return resource, ok
}

func (o *options) label(path string) (string, bool) {
	label, ok := o.labels[path]

	return label, ok
}

func (o *options) displayType(path string) (assistantartifact.DisplayType, bool) {
	displayType, ok := o.types[path]

	return displayType, ok
}

func (o *options) explicit(path string) bool {
	if _, ok := o.ref(path); ok {
		return true
	}
	if _, ok := o.labels[path]; ok {
		return true
	}
	if _, ok := o.types[path]; ok {
		return true
	}

	return matches(o.only, path)
}
