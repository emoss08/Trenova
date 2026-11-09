package validationtest

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
)

var (
	errTableRequired = errors.New("validationtest: a uniqueness check names no table")
	errFieldRequired = errors.New("validationtest: a uniqueness check names no fields")
	errNoReference   = errors.New("validationtest: a reference check names no table or id")
)

type takenValues struct {
	owner  pulid.ID
	values map[string]any
}

type referenceKey struct {
	table string
	id    pulid.ID
}

type Integrity struct {
	mu       sync.Mutex
	taken    []takenValues
	missing  map[referenceKey]struct{}
	failure  error
	requests []validationframework.IntegrityRequest
}

func NewIntegrity() *Integrity {
	return &Integrity{missing: make(map[referenceKey]struct{})}
}

func (i *Integrity) Taken(values map[string]any) *Integrity {
	return i.TakenBy(pulid.Nil, values)
}

func (i *Integrity) TakenBy(owner pulid.ID, values map[string]any) *Integrity {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.taken = append(i.taken, takenValues{owner: owner, values: values})

	return i
}

func (i *Integrity) Missing(table string, id pulid.ID) *Integrity {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.missing[referenceKey{table: table, id: id}] = struct{}{}

	return i
}

func (i *Integrity) FailWith(err error) *Integrity {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.failure = err

	return i
}

func (i *Integrity) Requests() []validationframework.IntegrityRequest {
	i.mu.Lock()
	defer i.mu.Unlock()

	return slices.Clone(i.requests)
}

func (i *Integrity) CheckIntegrity(
	_ context.Context,
	req *validationframework.IntegrityRequest,
) (*validationframework.IntegrityResult, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.requests = append(i.requests, copyRequest(req))
	if i.failure != nil {
		return nil, i.failure
	}

	result := &validationframework.IntegrityResult{
		Duplicate: make([]bool, len(req.Unique)),
		Exists:    make([]bool, len(req.References)),
	}
	if len(req.Unique) > 0 && req.Table == "" {
		return nil, errTableRequired
	}
	for index, check := range req.Unique {
		if len(check.Fields) == 0 {
			return nil, errFieldRequired
		}
		result.Duplicate[index] = i.isTaken(check, req.ExcludeID)
	}
	for index, reference := range req.References {
		if reference.Table == "" || reference.ID.IsNil() {
			return nil, errNoReference
		}
		_, missing := i.missing[referenceKey{table: reference.Table, id: reference.ID}]
		result.Exists[index] = !missing
	}

	return result, nil
}

func (i *Integrity) isTaken(check validationframework.UniqueCheck, excludeID pulid.ID) bool {
	for _, taken := range i.taken {
		if taken.owner.IsNotNil() && taken.owner == excludeID {
			continue
		}
		if holdsEvery(taken.values, check.Fields) {
			return true
		}
	}

	return false
}

func holdsEvery(values map[string]any, fields []validationframework.FieldCheck) bool {
	if len(values) != len(fields) {
		return false
	}
	for _, field := range fields {
		value, ok := values[field.Column]
		if !ok || !sameValue(value, field.Value, field.CaseSensitive) {
			return false
		}
	}

	return true
}

func sameValue(held, checked any, caseSensitive bool) bool {
	heldText, checkedText := textOf(held), textOf(checked)
	if caseSensitive {
		return heldText == checkedText
	}

	return strings.EqualFold(heldText, checkedText)
}

func textOf(value any) string {
	current := reflect.ValueOf(value)
	for current.IsValid() && current.Kind() == reflect.Pointer {
		if current.IsNil() {
			return ""
		}
		current = current.Elem()
	}
	if !current.IsValid() {
		return ""
	}

	return fmt.Sprint(current.Interface())
}

func copyRequest(req *validationframework.IntegrityRequest) validationframework.IntegrityRequest {
	copied := *req
	copied.Unique = make([]validationframework.UniqueCheck, 0, len(req.Unique))
	for _, check := range req.Unique {
		copied.Unique = append(
			copied.Unique,
			validationframework.UniqueCheck{Fields: slices.Clone(check.Fields)},
		)
	}
	copied.References = make([]validationframework.ReferenceCheck, 0, len(req.References))
	for _, reference := range req.References {
		reference.Conditions = slices.Clone(reference.Conditions)
		copied.References = append(copied.References, reference)
	}

	return copied
}
