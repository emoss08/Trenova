package responseguard

import (
	"encoding"
	"reflect"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

// field is one key a struct writes in JSON, as encoding/json names it.
type field struct {
	index     []int
	name      string
	omitEmpty bool
	omitZero  bool
	quoted    bool
	visit     bool
}

// typePlan is what the guard needs to know about one struct type, worked out
// once per type: the kind of record it holds, whether it withholds protected
// fields itself, its JSON keys, and whether anything under it can need
// guarding at all.
type typePlan struct {
	kind       permission.Resource
	withholder bool
	fields     []field
	visit      bool
}

var (
	jsonMarshalerType = reflect.TypeFor[jsonMarshaler]()
	textMarshalerType = reflect.TypeFor[encoding.TextMarshaler]()
	withholderType    = reflect.TypeFor[permission.FieldWithholder]()
)

type jsonMarshaler interface {
	MarshalJSON() ([]byte, error)
}

// planner caches type plans. It is a plain map behind a mutex: the set of
// response types is small and fixed, so after the first requests every call
// is a read.
type planner struct {
	kinds *kindTable

	mu      sync.RWMutex
	plans   map[reflect.Type]*typePlan
	reaches map[reflect.Type]bool
}

func newPlanner(kinds *kindTable) *planner {
	return &planner{
		kinds:   kinds,
		plans:   make(map[reflect.Type]*typePlan, 64),
		reaches: make(map[reflect.Type]bool, 128),
	}
}

func (p *planner) plan(t reflect.Type) *typePlan {
	p.mu.RLock()
	plan, ok := p.plans[t]
	p.mu.RUnlock()
	if ok {
		return plan
	}

	kind, _ := p.kinds.kindOf(t)
	fields := jsonFields(t)
	for i := range fields {
		fields[i].visit = p.mayGuard(t.FieldByIndex(fields[i].index).Type)
	}
	plan = &typePlan{
		kind:       kind,
		withholder: reflect.PointerTo(t).Implements(withholderType),
		fields:     fields,
		visit:      p.mayGuard(t),
	}

	p.mu.Lock()
	p.plans[t] = plan
	p.mu.Unlock()

	return plan
}

// mayGuard reports whether a value of a type can hold anything the guard acts
// on: a record of a kind, an entity that withholds protected fields, or an
// interface whose dynamic value could be either.
func (p *planner) mayGuard(t reflect.Type) bool {
	p.mu.RLock()
	known, ok := p.reaches[t]
	p.mu.RUnlock()
	if ok {
		return known
	}

	positive := make(map[reflect.Type]struct{}, 8)
	result := p.search(t, make(map[reflect.Type]struct{}, 16), positive)

	p.mu.Lock()
	p.reaches[t] = result
	for typ := range positive {
		p.reaches[typ] = true
	}
	p.mu.Unlock()

	return result
}

// search walks the type graph. A type found to reach something is recorded as
// such; a negative answer is only cached for the type first asked about,
// because a type on a cycle is assumed negative while it is being searched.
func (p *planner) search(
	t reflect.Type,
	seen map[reflect.Type]struct{},
	positive map[reflect.Type]struct{},
) bool {
	if _, ok := positive[t]; ok {
		return true
	}
	if _, ok := seen[t]; ok {
		return false
	}
	seen[t] = struct{}{}

	found := false
	switch t.Kind() { //nolint:exhaustive // only these kinds can hold a value
	case reflect.Interface:
		found = true
	case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
		found = p.search(t.Elem(), seen, positive)
	case reflect.Struct:
		found = p.structReaches(t, seen, positive)
	default:
	}
	if found {
		positive[t] = struct{}{}
	}

	return found
}

func (p *planner) structReaches(
	t reflect.Type,
	seen map[reflect.Type]struct{},
	positive map[reflect.Type]struct{},
) bool {
	if _, kinded := p.kinds.kindOf(t); kinded {
		return true
	}
	if reflect.PointerTo(t).Implements(withholderType) {
		return true
	}
	if marshalsItself(t) {
		return false
	}
	for _, f := range jsonFields(t) {
		if p.search(t.FieldByIndex(f.index).Type, seen, positive) {
			return true
		}
	}

	return false
}

// marshalsItself reports whether encoding/json hands a type's encoding to the
// type, so its fields are not what the response holds.
func marshalsItself(t reflect.Type) bool {
	pointer := reflect.PointerTo(t)
	return t.Implements(jsonMarshalerType) || pointer.Implements(jsonMarshalerType) ||
		t.Implements(textMarshalerType) || pointer.Implements(textMarshalerType)
}

// jsonFields lists a struct's JSON keys by encoding/json's rules: exported
// fields under their tag name, "-" skipped, and the fields of an untagged
// embedded struct promoted unless a shallower field has the same name.
func jsonFields(t reflect.Type) []field {
	out := make([]field, 0, t.NumField())
	taken := make(map[string]int, t.NumField())
	collectFields(t, nil, 0, &out, taken)

	return out
}

type fieldDepth struct {
	index int
	depth int
}

func collectFields(
	t reflect.Type,
	prefix []int,
	depth int,
	out *[]field,
	taken map[string]int,
) {
	embedded := make([]fieldDepth, 0, 2)
	for i := range t.NumField() {
		sf := t.Field(i)
		tag := sf.Tag.Get("json")
		if tag == "-" {
			continue
		}
		name, opts := parseTag(tag)
		if sf.Anonymous && name == "" {
			ft := sf.Type
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				embedded = append(embedded, fieldDepth{index: i, depth: depth + 1})
				continue
			}
		}
		if !sf.IsExported() {
			continue
		}
		if name == "" {
			name = sf.Name
		}
		if previous, ok := taken[name]; ok && previous <= depth {
			continue
		}
		taken[name] = depth
		index := make([]int, len(prefix)+1)
		copy(index, prefix)
		index[len(prefix)] = i
		*out = append(*out, field{
			index:     index,
			name:      name,
			omitEmpty: opts.has("omitempty"),
			omitZero:  opts.has("omitzero"),
			quoted:    opts.has("string") && quotable(sf.Type),
		})
	}
	for _, e := range embedded {
		sf := t.Field(e.index)
		ft := sf.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		index := make([]int, len(prefix)+1)
		copy(index, prefix)
		index[len(prefix)] = e.index
		collectFields(ft, index, e.depth, out, taken)
	}
}

type tagOptions string

func (o tagOptions) has(option string) bool {
	for o != "" {
		current, rest, _ := strings.Cut(string(o), ",")
		if current == option {
			return true
		}
		o = tagOptions(rest)
	}

	return false
}

func parseTag(tag string) (string, tagOptions) {
	name, opts, _ := strings.Cut(tag, ",")
	return name, tagOptions(opts)
}

func quotable(t reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() { //nolint:exhaustive // the kinds encoding/json's ,string option quotes
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr, reflect.Float32, reflect.Float64, reflect.String:
		return true
	default:
		return false
	}
}
