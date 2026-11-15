// Package planfields keeps a write from changing a field the organization's
// plan withholds: a value of an add-on stored on a base record, such as a
// customer's credit limit or a shipment's tariff. Every service that saves a
// base record a person or an agent sent back calls it before it validates,
// with the record as it is stored. Self-hosted, every feature is on and it
// changes nothing.
package planfields

import (
	"context"
	"encoding"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/modules/identity/app/onplan"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/shared/textcase"
)

const domainPackagePrefix = "github.com/emoss08/trenova/internal/modules/"

var _ permissionports.PlanFieldKeeper = (*Keeper)(nil)

// Keeper is the process's keeper. A record type's gated fields are worked out
// once per kind and cached; the set of record types is small and fixed, so
// after the first writes every call is a read.
type Keeper struct {
	registry *permissioncatalog.Registry
	plan     *onplan.Lookup

	mu    sync.RWMutex
	plans map[planKey]*typePlan
}

type planKey struct {
	kind permission.Resource
	typ  reflect.Type
}

// gatedField is one field of a record type that belongs to a feature of its
// own, found by its API name: index is its path from the record, through
// plain structs and pointers to them.
type gatedField struct {
	index   []int
	feature platformcatalog.FeatureKey
}

type typePlan struct {
	fields   []gatedField
	features []platformcatalog.FeatureKey
}

func New(registry *permissioncatalog.Registry, plan *onplan.Lookup) *Keeper {
	return &Keeper{
		registry: registry,
		plan:     plan,
		plans:    make(map[planKey]*typePlan, 16),
	}
}

func (k *Keeper) Keep(
	ctx context.Context,
	kind permission.Resource,
	incoming, stored any,
) error {
	target, err := recordValue(incoming)
	if err != nil {
		return fmt.Errorf("keep the fields the plan withholds on a %s: %w", kind, err)
	}
	source, err := storedValue(stored, target.Type())
	if err != nil {
		return fmt.Errorf("keep the fields the plan withholds on a %s: %w", kind, err)
	}

	plan := k.planFor(kind, target.Type())
	if len(plan.fields) == 0 {
		return nil
	}

	memo := k.plan.Memo()
	off := make(map[platformcatalog.FeatureKey]bool, len(plan.features))
	for _, feature := range plan.features {
		withheld, askErr := memo.FeatureOff(ctx, feature)
		if askErr != nil {
			return askErr
		}
		if withheld {
			off[feature] = true
		}
	}
	if len(off) == 0 {
		return nil
	}

	for i := range plan.fields {
		field := &plan.fields[i]
		if off[field.feature] {
			restore(target, source, field.index)
		}
	}

	return nil
}

func recordValue(incoming any) (reflect.Value, error) {
	v := reflect.ValueOf(incoming)
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("the record must be a pointer to a struct, not %T", incoming)
	}

	return v.Elem(), nil
}

// storedValue is the stored record as a struct value of the incoming record's
// type, or the invalid value when there is none, which restores zero values.
func storedValue(stored any, want reflect.Type) (reflect.Value, error) {
	if stored == nil {
		return reflect.Value{}, nil
	}
	v := reflect.ValueOf(stored)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return reflect.Value{}, nil
		}
		v = v.Elem()
	}
	if v.Type() != want {
		return reflect.Value{}, fmt.Errorf("the stored record is a %s, not a %s", v.Type(), want)
	}

	return v, nil
}

// restore sets the field at index on target to its value on source, or to
// its zero value when source holds none. A nested struct the target does not
// hold was not sent, so nothing under it is written.
func restore(target, source reflect.Value, index []int) {
	for depth, i := range index {
		field := target.Field(i)
		var from reflect.Value
		if source.IsValid() {
			from = source.Field(i)
		}
		if depth == len(index)-1 {
			if from.IsValid() {
				field.Set(from)
			} else {
				field.SetZero()
			}
			return
		}
		if field.Kind() == reflect.Pointer {
			if field.IsNil() {
				return
			}
			field = field.Elem()
			if from.IsValid() {
				if from.IsNil() {
					from = reflect.Value{}
				} else {
					from = from.Elem()
				}
			}
		}
		target, source = field, from
	}
}

func (k *Keeper) planFor(kind permission.Resource, t reflect.Type) *typePlan {
	key := planKey{kind: kind, typ: t}
	k.mu.RLock()
	plan, ok := k.plans[key]
	k.mu.RUnlock()
	if ok {
		return plan
	}

	plan = &typePlan{}
	seen := make(map[platformcatalog.FeatureKey]struct{}, 4)
	add := func(index []int, feature platformcatalog.FeatureKey) {
		plan.fields = append(plan.fields, gatedField{index: index, feature: feature})
		if _, dup := seen[feature]; !dup {
			seen[feature] = struct{}{}
			plan.features = append(plan.features, feature)
		}
	}
	k.collect(kind, t, nil, map[reflect.Type]struct{}{t: {}}, add)

	k.mu.Lock()
	k.plans[key] = plan
	k.mu.Unlock()

	return plan
}

// collect finds every field of a record type that belongs to a feature of its
// own: a field named in the kind's FieldFeatures, at the top or inside a plain
// struct the record holds. A struct of a kind of its own is another record and
// is not walked, nor is a list, a map or a type that encodes itself.
func (k *Keeper) collect(
	kind permission.Resource,
	t reflect.Type,
	prefix []int,
	path map[reflect.Type]struct{},
	add func(index []int, feature platformcatalog.FeatureKey),
) {
	for i := range t.NumField() {
		sf := t.Field(i)
		name, skip := jsonName(sf)
		if skip {
			continue
		}
		index := make([]int, len(prefix)+1)
		copy(index, prefix)
		index[len(prefix)] = i

		if !sf.Anonymous || name != "" {
			if feature, declared := k.registry.FieldFeatureOf(kind.String(), name); declared {
				add(index, feature)
				continue
			}
		}

		nested := sf.Type
		if nested.Kind() == reflect.Pointer {
			nested = nested.Elem()
		}
		if nested.Kind() != reflect.Struct || encodesItself(nested) || k.isKind(nested) {
			continue
		}
		if _, cycle := path[nested]; cycle {
			continue
		}
		path[nested] = struct{}{}
		k.collect(kind, nested, index, path, add)
		delete(path, nested)
	}
}

// jsonName is the name a field is written under, and whether it is not
// written at all. An untagged embedded struct reports no name: its fields are
// the record's own.
func jsonName(sf reflect.StructField) (string, bool) {
	tag := sf.Tag.Get("json")
	if tag == "-" || !sf.IsExported() {
		return "", true
	}
	name, _, _ := strings.Cut(tag, ",")
	if sf.Anonymous && name == "" {
		return "", false
	}
	if name == "" {
		name = sf.Name
	}

	return name, false
}

// isKind reports whether a struct is a record of a kind of its own: a domain
// type named for a registered resource, as the response guard names them.
func (k *Keeper) isKind(t reflect.Type) bool {
	rest, ok := strings.CutPrefix(t.PkgPath(), domainPackagePrefix)
	if !ok || !strings.Contains(rest, "/domain/") || t.Name() == "" {
		return false
	}

	return k.registry.HasResource(textcase.ConvertCamelToSnake(t.Name()))
}

var (
	jsonMarshalerType = reflect.TypeFor[interface{ MarshalJSON() ([]byte, error) }]()
	textMarshalerType = reflect.TypeFor[encoding.TextMarshaler]()
)

func encodesItself(t reflect.Type) bool {
	pointer := reflect.PointerTo(t)
	return t.Implements(jsonMarshalerType) || pointer.Implements(jsonMarshalerType) ||
		t.Implements(textMarshalerType) || pointer.Implements(textMarshalerType)
}
