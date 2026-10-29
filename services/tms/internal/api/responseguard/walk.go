package responseguard

import (
	"context"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
)

// maxDepth bounds the walk. Nothing a handler returns nests this deep; a value
// that does is answered null rather than shown unguarded.
const maxDepth = 128

const recordIDField = "id"

var anyType = reflect.TypeFor[any]()

// decider is one request's answers: which kinds the caller reads, what each
// field of a linked record shows, and which protected field groups they hold.
type decider interface {
	reads(ctx context.Context, kind permission.Resource) bool
	visible(ctx context.Context, kind permission.Resource, field string) bool
	groups() protectedfield.GroupSet
}

// record is the record a value belongs to and how it was reached.
type record struct {
	kind permission.Resource
	// linked is a record carried by another record, which shows its kind's
	// reference view unless the caller reads the kind.
	linked bool
	// part is a record that is part of a linked record (its stop, its move),
	// which shows only its id unless the caller reads the record's kind.
	part bool
}

var (
	rootRecord = &record{}
	// referenceCarrier stands for the record a reference route's records
	// will be linked from: one of no kind, so every record under it is
	// linked.
	referenceCarrier = &record{}
)

type walker struct {
	ctx        context.Context
	decider    decider
	planner    *planner
	registry   *permissioncatalog.Registry
	selfScoped bool

	path     []string
	withheld map[string]struct{}
}

// guard returns the value to encode in place of v, and whether it differs.
// A record of a kind the caller cannot read, reached through another record,
// is replaced by a map of its JSON keys with every key outside what the
// caller may see set to null; every value that holds one is rebuilt the
// same way, and everything else is the caller's own value.
func (w *walker) guard(v reflect.Value, rec *record, depth int) (reflect.Value, bool) {
	if depth > maxDepth {
		w.markWithheld()
		return reflect.Zero(anyType), true
	}

	switch v.Kind() { //nolint:exhaustive // only containers and structs can hold a record
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return v, false
		}
		return w.guard(v.Elem(), rec, depth+1)
	case reflect.Slice:
		if v.IsNil() {
			return v, false
		}
		return w.guardList(v, rec, depth)
	case reflect.Array:
		return w.guardList(v, rec, depth)
	case reflect.Map:
		if v.IsNil() {
			return v, false
		}
		return w.guardMap(v, rec, depth)
	case reflect.Struct:
		return w.guardStruct(v, rec, depth)
	default:
		return v, false
	}
}

func (w *walker) guardList(v reflect.Value, rec *record, depth int) (reflect.Value, bool) {
	if !w.planner.mayGuard(v.Type().Elem()) {
		return v, false
	}

	var out []any
	for i := range v.Len() {
		elem := v.Index(i)
		guarded, changed := w.guard(elem, rec, depth+1)
		if !changed {
			if out != nil {
				out[i] = jsonValue(elem)
			}
			continue
		}
		if out == nil {
			out = make([]any, v.Len())
			for j := range i {
				out[j] = jsonValue(v.Index(j))
			}
		}
		out[i] = interfaceOf(guarded)
	}
	if out == nil {
		return v, false
	}

	return reflect.ValueOf(out), true
}

func (w *walker) guardMap(v reflect.Value, rec *record, depth int) (reflect.Value, bool) {
	if !w.planner.mayGuard(v.Type().Elem()) {
		return v, false
	}

	var out reflect.Value
	iter := v.MapRange()
	for iter.Next() {
		key := iter.Key()
		w.push(mapKeyName(key))
		guarded, changed := w.guard(iter.Value(), rec, depth+1)
		w.pop()
		if !changed {
			continue
		}
		if !out.IsValid() {
			out = reflect.MakeMapWithSize(reflect.MapOf(v.Type().Key(), anyType), v.Len())
			copyMap(out, v)
		}
		out.SetMapIndex(key, guardedOrNull(guarded))
	}
	if !out.IsValid() {
		return v, false
	}

	return out, true
}

func (w *walker) guardStruct(v reflect.Value, carrier *record, depth int) (reflect.Value, bool) {
	t := v.Type()
	if marshalsItself(t) {
		return v, false
	}
	plan := w.planner.plan(t)
	if !plan.visit {
		return v, false
	}
	replaced := false
	if plan.withholder {
		v, replaced = w.withhold(v)
	}

	rec := w.recordOf(plan.kind, carrier, isEnvelope(t))
	gate := w.gate(plan.kind, rec)

	var overrides map[string]any
	set := func(name string, value any) {
		if overrides == nil {
			overrides = make(map[string]any, 4)
		}
		overrides[name] = value
	}
	for i := range plan.fields {
		f := &plan.fields[i]
		fv, ok := fieldByIndex(v, f.index)
		if !ok {
			continue
		}
		if gate != gateNone && !w.fieldVisible(gate, rec.kind, f.name) {
			if !f.omitted(fv) && !isNull(fv) {
				w.markWithheld()
				set(f.name, nil)
			}
			continue
		}
		if !f.visit {
			continue
		}
		w.push(f.name)
		guarded, changed := w.guard(fv, rec, depth+1)
		w.pop()
		if changed {
			set(f.name, interfaceOf(guarded))
		}
	}

	if overrides == nil {
		if replaced {
			return v, true
		}
		return v, false
	}

	return reflect.ValueOf(encodeStruct(v, plan, overrides)), true
}

// recordOf is the record a struct belongs to. A struct of a kind reached
// through no record, or under a route that returns only the caller's own
// records, is a root record. One reached through another record is linked,
// unless it is part of that record (a shipment's stop) or of its kind, when it
// follows it: a part of a root record is the root's, and a part of a linked
// one shows only its id without the linked kind's read. A struct of no kind
// belongs to the record that carries it; at the top of a response, a struct
// that is not a list envelope is the record itself.
func (w *walker) recordOf(kind permission.Resource, carrier *record, envelope bool) *record {
	if kind == "" {
		if carrier == nil && !envelope {
			return rootRecord
		}
		return carrier
	}
	if carrier == nil || w.selfScoped {
		return &record{kind: kind}
	}
	if carrier.kind != "" && w.registry.PartOf(kind.String(), carrier.kind.String()) {
		if kind == carrier.kind || (!carrier.linked && !carrier.part) {
			return carrier
		}
		return &record{kind: carrier.kind, part: true}
	}

	return &record{kind: kind, linked: true}
}

type gateKind uint8

const (
	gateNone gateKind = iota
	gateView
	gatePart
)

// gate is how a struct's own fields are decided. Only the linked record
// itself is gated: a struct of no kind inside it was reached through one of
// its fields the caller may see.
func (w *walker) gate(kind permission.Resource, rec *record) gateKind {
	if kind == "" || rec == nil || w.selfScoped {
		return gateNone
	}
	switch {
	case rec.part:
		if w.decider.reads(w.ctx, rec.kind) {
			return gateNone
		}
		return gatePart
	case rec.linked:
		if w.decider.reads(w.ctx, rec.kind) {
			return gateNone
		}
		return gateView
	default:
		return gateNone
	}
}

func (w *walker) fieldVisible(gate gateKind, kind permission.Resource, name string) bool {
	if name == recordIDField {
		return true
	}
	if gate == gatePart {
		return false
	}

	return w.decider.visible(w.ctx, kind, name)
}

// withhold clears the protected fields the caller's groups do not cover, in
// place as the REST routes always have, or on a copy when the value cannot be
// addressed.
func (w *walker) withhold(v reflect.Value) (reflect.Value, bool) {
	groups := w.decider.groups()
	if groups.HasAll() {
		return v, false
	}
	if v.CanAddr() {
		if withholder, ok := v.Addr().Interface().(protectedfield.Withholder); ok {
			withholder.WithholdFields(groups)
		}
		return v, false
	}
	cp := reflect.New(v.Type())
	cp.Elem().Set(v)
	if withholder, ok := cp.Interface().(protectedfield.Withholder); ok {
		withholder.WithholdFields(groups)
	}

	return cp.Elem(), true
}

func (w *walker) push(name string) { w.path = append(w.path, name) }

func (w *walker) pop() { w.path = w.path[:len(w.path)-1] }

func (w *walker) markWithheld() {
	if w.withheld == nil {
		w.withheld = make(map[string]struct{}, 4)
	}
	w.withheld[strings.Join(w.path, ".")] = struct{}{}
}

func (w *walker) withheldRecords() []string {
	if len(w.withheld) == 0 {
		return nil
	}
	out := make([]string, 0, len(w.withheld))
	for path := range w.withheld {
		out = append(out, path)
	}
	slices.Sort(out)

	return out
}

// encodeStruct writes a struct as the map encoding/json would have written,
// with the overridden keys in place of the struct's own values.
func encodeStruct(v reflect.Value, plan *typePlan, overrides map[string]any) map[string]any {
	out := make(map[string]any, len(plan.fields))
	for i := range plan.fields {
		f := &plan.fields[i]
		if value, ok := overrides[f.name]; ok {
			out[f.name] = value
			continue
		}
		fv, ok := fieldByIndex(v, f.index)
		if !ok {
			continue
		}
		if f.omitted(fv) {
			continue
		}
		if f.quoted {
			out[f.name] = quote(fv)
			continue
		}
		out[f.name] = jsonValue(fv)
	}

	return out
}

// omitted reports whether encoding/json leaves a field's key out, as it does
// an empty value tagged omitempty or a zero one tagged omitzero.
func (f *field) omitted(v reflect.Value) bool {
	return (f.omitEmpty && isEmptyValue(v)) || (f.omitZero && isZeroValue(v))
}

// isNull reports whether a value already encodes as null, so withholding it
// hides nothing.
func isNull(v reflect.Value) bool {
	switch v.Kind() { //nolint:exhaustive // only nilable kinds encode as null
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

// fieldByIndex follows an index through embedded pointers, and reports false
// where one is nil, as encoding/json skips the fields behind it.
func fieldByIndex(v reflect.Value, index []int) (reflect.Value, bool) {
	for i, x := range index {
		if i > 0 && v.Kind() == reflect.Pointer {
			if v.IsNil() {
				return reflect.Value{}, false
			}
			v = v.Elem()
		}
		v = v.Field(x)
	}

	return v, true
}

// jsonValue is a value as encoding/json would see it in place: a field whose
// pointer marshals itself is handed over by pointer, so the method still runs.
func jsonValue(v reflect.Value) any {
	switch v.Kind() { //nolint:exhaustive // only reference kinds are handed over as they are
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice:
		return v.Interface()
	default:
	}
	t := v.Type()
	pointer := reflect.PointerTo(t)
	needsPointer := (pointer.Implements(jsonMarshalerType) && !t.Implements(jsonMarshalerType)) ||
		(pointer.Implements(textMarshalerType) && !t.Implements(textMarshalerType))
	if !needsPointer {
		return v.Interface()
	}
	if v.CanAddr() {
		return v.Addr().Interface()
	}
	cp := reflect.New(t)
	cp.Elem().Set(v)

	return cp.Interface()
}

func interfaceOf(v reflect.Value) any {
	if !v.IsValid() {
		return nil
	}
	return jsonValue(v)
}

func guardedOrNull(v reflect.Value) reflect.Value {
	if !v.IsValid() || (v.Kind() == reflect.Interface && v.IsNil()) {
		return reflect.Zero(anyType)
	}
	if v.Type() == anyType {
		return v
	}
	out := reflect.New(anyType).Elem()
	out.Set(v)

	return out
}

func copyMap(dst, src reflect.Value) {
	iter := src.MapRange()
	for iter.Next() {
		dst.SetMapIndex(iter.Key(), guardedOrNull(reflect.ValueOf(jsonValue(iter.Value()))))
	}
}

func mapKeyName(key reflect.Value) string {
	if key.Kind() == reflect.String {
		return key.String()
	}
	return "*"
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() { //nolint:exhaustive // the kinds encoding/json's omitempty treats as empty
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.Interface, reflect.Pointer:
		return v.IsZero()
	default:
		return false
	}
}

type zeroReporter interface {
	IsZero() bool
}

func isZeroValue(v reflect.Value) bool {
	if v.Kind() == reflect.Pointer && v.IsNil() {
		return true
	}
	if reporter, ok := v.Interface().(zeroReporter); ok {
		return reporter.IsZero()
	}
	if v.CanAddr() {
		if reporter, ok := v.Addr().Interface().(zeroReporter); ok {
			return reporter.IsZero()
		}
	}

	return v.IsZero()
}

func quote(v reflect.Value) any {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	switch v.Kind() { //nolint:exhaustive // the kinds encoding/json's ,string option quotes
	case reflect.String:
		return strconv.Quote(v.String())
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32:
		return strconv.FormatFloat(v.Float(), 'g', -1, 32)
	case reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'g', -1, 64)
	default:
		return v.Interface()
	}
}
