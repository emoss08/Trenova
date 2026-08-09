package documenttemplate

import (
	"errors"
	"fmt"
	"html/template"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/emoss08/trenova/pkg/templateengine"
)

// Channel names one output slot of a template version.
type Channel string

const (
	ChannelPDF               = Channel("PDF")
	ChannelEmailHTML         = Channel("EmailHTML")
	ChannelEmailText         = Channel("EmailText")
	ChannelSubject           = Channel("Subject")
	ChannelNotificationTitle = Channel("NotificationTitle")
	ChannelNotificationBody  = Channel("NotificationBody")
)

func (c Channel) String() string { return string(c) }

// Engine maps a domain channel to the engine's, so the engine stays free of
// domain imports and the domain owns the enum the database and GraphQL see.
func (c Channel) Engine() templateengine.Channel {
	return templateengine.Channel(c)
}

// Field names the editor pane and the database column a channel is stored in.
func (c Channel) Field() string {
	switch c {
	case ChannelPDF, ChannelEmailHTML:
		return "bodyHtml"
	case ChannelEmailText:
		return "bodyText"
	case ChannelSubject:
		return "subject"
	case ChannelNotificationTitle:
		return "notificationTitle"
	case ChannelNotificationBody:
		return "notificationMessage"
	default:
		return string(c)
	}
}

// VariableType is what a variable holds, so the editor can describe it and the
// catalog can be checked against the sample struct.
type VariableType string

const (
	VariableString     = VariableType("String")
	VariableMoney      = VariableType("Money")
	VariableDate       = VariableType("Date")
	VariableDateTime   = VariableType("DateTime")
	VariableInt        = VariableType("Int")
	VariableDecimal    = VariableType("Decimal")
	VariableBool       = VariableType("Bool")
	VariableObject     = VariableType("Object")
	VariableCollection = VariableType("Collection")
	VariableStringList = VariableType("StringList")
)

func (t VariableType) String() string { return string(t) }

// VariableDefinition documents one path a template may reference.
type VariableDefinition struct {
	// Path is the dotted chain an author writes, e.g. "BillTo.Name".
	Path string

	// Description says what the value means in business terms, because the editor
	// shows it verbatim and "the invoice total" is more use than "decimal".
	Description string

	Type VariableType

	// Required means a published version must still reference this path.
	//
	// Assignment replaces a whole template, so an organization that customizes an
	// invoice stops receiving our fixes to it. These are the fields whose absence
	// makes the document legally deficient rather than merely ugly — a total, a
	// remit-to address — and publishing without them is refused.
	Required bool

	// Fields describes the element shape for Object and Collection variables.
	Fields []VariableDefinition
}

// KindDefinition is everything the system knows about one authorable template.
type KindDefinition struct {
	Kind        Kind
	DisplayName string
	Description string

	// Category groups kinds in the admin catalog.
	Category string

	// Channels are the output slots this kind produces.
	Channels []Channel

	// Paged means page size, orientation, and margins apply. False for messages.
	Paged bool

	// CustomerScoped means a per-customer assignment is meaningful.
	//
	// Without this flag an administrator can assign a template to a customer for a
	// kind that has no customer in scope — a scheduled report, a driver invite —
	// and the assignment silently never fires.
	CustomerScoped bool

	// Variables is the catalog, and the sample context must match it exactly.
	Variables []VariableDefinition

	// sampleFactory builds the context used to verify a version at publish time
	// and to render a preview before real data is chosen.
	sampleFactory func() any
}

// Registry holds every kind.
type Registry struct {
	mu    sync.RWMutex
	kinds map[Kind]*KindDefinition
}

// NewRegistry builds the populated registry.
func NewRegistry() *Registry {
	r := &Registry{kinds: make(map[Kind]*KindDefinition, len(AllKinds()))}
	r.registerAll()
	return r
}

func (r *Registry) registerAll() {
	r.registerBillingKinds()
	r.registerDetentionKinds()
	r.registerRateConfirmationKinds()
	r.registerReportingKinds()
	r.registerPortalKinds()
	r.registerAgentKinds()
	r.registerNotificationKinds()
}

// Register adds a kind, refusing a duplicate or an incoherent definition.
func (r *Registry) Register(def *KindDefinition) error {
	if def == nil {
		return errors.New("cannot register a nil kind definition")
	}
	if def.Kind == "" {
		return errors.New("kind definition has no key")
	}
	if len(def.Channels) == 0 {
		return fmt.Errorf("kind %q declares no channels", def.Kind)
	}
	if def.sampleFactory == nil {
		return fmt.Errorf("kind %q has no sample context", def.Kind)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.kinds[def.Kind]; exists {
		return fmt.Errorf("kind %q is already registered", def.Kind)
	}
	r.kinds[def.Kind] = def

	return nil
}

// Get returns one kind.
func (r *Registry) Get(kind Kind) (*KindDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, ok := r.kinds[kind]
	return def, ok
}

// All returns every kind, ordered by category then display name so the admin
// catalog is stable between requests.
func (r *Registry) All() []*KindDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]*KindDefinition, 0, len(r.kinds))
	for _, def := range r.kinds {
		out = append(out, def)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Category != out[j].Category {
			return out[i].Category < out[j].Category
		}
		return out[i].DisplayName < out[j].DisplayName
	})

	return out
}

// ByCategory returns the kinds in one category.
func (r *Registry) ByCategory(category string) []*KindDefinition {
	all := r.All()
	out := make([]*KindDefinition, 0, len(all))
	for _, def := range all {
		if def.Category == category {
			out = append(out, def)
		}
	}
	return out
}

// Paths returns every referenceable path for a kind, flattened and sorted.
//
// This is what the engine checks a template against and what the editor's
// autocomplete offers, so the two can never disagree.
func (r *Registry) Paths(kind Kind) []string {
	def, ok := r.Get(kind)
	if !ok {
		return nil
	}

	out := make([]string, 0, len(def.Variables)*2)
	collectPaths("", def.Variables, &out)
	sort.Strings(out)

	return out
}

// RequiredPaths returns the paths a published version must still reference.
func (r *Registry) RequiredPaths(kind Kind) []string {
	def, ok := r.Get(kind)
	if !ok {
		return nil
	}

	out := make([]string, 0, 4)
	collectRequiredPaths("", def.Variables, &out)
	sort.Strings(out)

	return out
}

func collectPaths(prefix string, vars []VariableDefinition, out *[]string) {
	for i := range vars {
		path := vars[i].Path
		if prefix != "" {
			path = prefix + "." + path
		}
		*out = append(*out, path)
		collectPaths(path, vars[i].Fields, out)
	}
}

func collectRequiredPaths(prefix string, vars []VariableDefinition, out *[]string) {
	for i := range vars {
		path := vars[i].Path
		if prefix != "" {
			path = prefix + "." + path
		}
		if vars[i].Required {
			*out = append(*out, path)
		}
		collectRequiredPaths(path, vars[i].Fields, out)
	}
}

// SampleContext returns realistic data for a kind.
//
// Publishing renders against this, which is what stops a template that only
// breaks on real data from going live.
func (r *Registry) SampleContext(kind Kind) (any, error) {
	def, ok := r.Get(kind)
	if !ok {
		return nil, fmt.Errorf("unknown template kind %q", kind)
	}
	return def.sampleFactory(), nil
}

// hostileProbeValue is chosen to trip every escaper html/template has: a quote and
// angle brackets for markup contexts, a semicolon and brace for CSS, and a colon
// for a URL scheme position.
const hostileProbeValue = `x";}</style><a:1 'y`

// HostileProbeContext returns the sample context with every string replaced by a
// value designed to be un-escapable.
//
// Rendering against it answers a question benign sample data cannot: does this
// template put a text field somewhere its value will be neutralized? If so the
// template is structurally wrong and will silently break the first time a customer
// name contains an apostrophe. Publishing checks both contexts.
func (r *Registry) HostileProbeContext(kind Kind) (any, error) {
	sample, err := r.SampleContext(kind)
	if err != nil {
		return nil, err
	}

	value := reflect.ValueOf(sample)
	if value.Kind() != reflect.Struct {
		return sample, nil
	}

	clone := reflect.New(value.Type()).Elem()
	clone.Set(value)
	poisonStrings(clone)

	return clone.Interface(), nil
}

// trustedURLType is template.URL, which the escaper passes through unchanged.
var trustedURLType = reflect.TypeOf(template.URL(""))

// poisonStrings walks a value replacing every settable string with the probe.
//
// template.URL fields are skipped. They hold values this system generated — a data:
// URI built from validated image bytes — not text that arrived from a customer
// record, so poisoning one would report a failure that cannot happen and would
// inject the probe unescaped into the output it is meant to be testing.
func poisonStrings(v reflect.Value) {
	if v.Type() == trustedURLType {
		return
	}

	switch v.Kind() {
	case reflect.String:
		if v.CanSet() {
			v.SetString(hostileProbeValue)
		}

	case reflect.Struct:
		for i := range v.NumField() {
			poisonStrings(v.Field(i))
		}

	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			poisonStrings(v.Index(i))
		}

	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			poisonStrings(v.Elem())
		}

	case reflect.Invalid, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr, reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128,
		reflect.Chan, reflect.Func, reflect.Map, reflect.UnsafePointer:
		// Nothing an escaper could reject: only strings reach an output context.

	default:
	}
}

// Categories returns the distinct categories, sorted.
func (r *Registry) Categories() []string {
	seen := make(map[string]struct{}, 8)
	for _, def := range r.All() {
		seen[def.Category] = struct{}{}
	}

	out := make([]string, 0, len(seen))
	for category := range seen {
		out = append(out, category)
	}
	sort.Strings(out)

	return out
}

// HasChannel reports whether a kind produces a given channel.
func (d *KindDefinition) HasChannel(channel Channel) bool {
	for _, c := range d.Channels {
		if c == channel {
			return true
		}
	}
	return false
}

// StarterKey is the filename stem for this kind's built-in assets.
func (d *KindDefinition) StarterKey() string {
	return strings.ReplaceAll(string(d.Kind), ":", ".")
}
