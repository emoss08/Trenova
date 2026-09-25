package toolpreview

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/fieldsensitivity"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

var ErrNoRecord = errors.New("the record to preview was not loaded")

const maxDiffDepth = 10

// bookkeepingKeys are written by the system on every save and say nothing
// about what a write means to the person deciding it.
var bookkeepingKeys = map[string]struct{}{
	"createdAt":    {},
	"updatedAt":    {},
	"searchVector": {},
	"rank":         {},
}

// Update previews a change to an existing record: before is the record as it
// is, and mutate is the tool's own plan function, applied to a copy. The copy
// is made by way of JSON, so what mutate reads must be carried by the
// record's JSON tags; a tool whose plan needs more computes the after state
// itself and calls Changed.
func Update[T any](
	rec Record,
	before *T,
	mutate func(*T) error,
	opts ...Option,
) (*agent.RecordChange, error) {
	return mutated(agent.PreviewOperationUpdate, rec, before, mutate, opts)
}

// Archive is Update for a write that retires the record rather than editing
// it: a status moved to inactive, a hold released.
func Archive[T any](
	rec Record,
	before *T,
	mutate func(*T) error,
	opts ...Option,
) (*agent.RecordChange, error) {
	return mutated(agent.PreviewOperationArchive, rec, before, mutate, opts)
}

func mutated[T any](
	operation agent.PreviewOperation,
	rec Record,
	before *T,
	mutate func(*T) error,
	opts []Option,
) (*agent.RecordChange, error) {
	if before == nil {
		return nil, ErrNoRecord
	}

	after := new(T)
	if err := jsonutils.Convert(before, after); err != nil {
		return nil, fmt.Errorf("copy the record to preview: %w", err)
	}
	if mutate != nil {
		if err := mutate(after); err != nil {
			return nil, err
		}
	}

	change, err := Changed(rec, before, after, opts...)
	if err != nil {
		return nil, err
	}
	change.Operation = operation

	return change, nil
}

// Changed previews the difference between two states of one record the tool
// computed itself.
func Changed[T any](rec Record, before, after *T, opts ...Option) (*agent.RecordChange, error) {
	if before == nil || after == nil {
		return nil, ErrNoRecord
	}

	return build(agent.PreviewOperationUpdate, &rec, before, after, newOptions(opts))
}

// Create previews a record the write would make. Every value it would hold
// is shown, or only the paths named with Only.
func Create[T any](rec Record, created *T, opts ...Option) (*agent.RecordChange, error) {
	if created == nil {
		return nil, ErrNoRecord
	}

	return build(agent.PreviewOperationCreate, &rec, nil, created, newOptions(opts))
}

// Delete previews a record the write would remove, with the values it holds
// now when before is given.
func Delete[T any](rec Record, before *T, opts ...Option) (*agent.RecordChange, error) {
	if before == nil {
		change := newChange(agent.PreviewOperationDelete, &rec)

		return change, nil
	}

	return build(agent.PreviewOperationDelete, &rec, before, nil, newOptions(opts))
}

func newChange(operation agent.PreviewOperation, rec *Record) *agent.RecordChange {
	return &agent.RecordChange{
		Resource:  rec.Resource,
		EntityID:  rec.ID,
		Label:     strings.TrimSpace(rec.Label),
		Operation: operation,
		Version:   rec.Version,
	}
}

func build(
	operation agent.PreviewOperation,
	rec *Record,
	before, after any,
	o *options,
) (*agent.RecordChange, error) {
	beforeMap, beforeOrder, err := decode(before)
	if err != nil {
		return nil, fmt.Errorf("read the record before the change: %w", err)
	}
	afterMap, afterOrder, err := decode(after)
	if err != nil {
		return nil, fmt.Errorf("read the record after the change: %w", err)
	}

	diff, err := jsonutils.JSONDiff(beforeMap, afterMap, &jsonutils.DiffOptions{
		IgnoreFields:    []string{},
		CustomComparors: map[string]jsonutils.Comparator{},
		MaxDepth:        maxDiffDepth,
	})
	if err != nil {
		return nil, fmt.Errorf("compare the record before and after: %w", err)
	}

	change := newChange(operation, rec)
	if change.Label == "" {
		change.Label = stringutils.FirstNonEmpty(
			assistantartifact.RecordLabel(afterMap),
			assistantartifact.RecordLabel(beforeMap),
		)
	}

	order := afterOrder
	if len(order) == 0 {
		order = beforeOrder
	}
	change.Fields = o.fields(rec.Resource, diff, beforeMap, afterMap, order)

	return change, nil
}

func decode(value any) (decoded map[string]any, order []string, err error) {
	if value == nil {
		return map[string]any{}, nil, nil
	}

	encoded, err := sonic.Marshal(value)
	if err != nil {
		return nil, nil, err
	}
	decoded = make(map[string]any)
	if err = sonic.Unmarshal(encoded, &decoded); err != nil {
		return nil, nil, err
	}

	return decoded, jsonutils.ObjectKeyOrder(encoded), nil
}

func (o *options) fields(
	resource permission.Resource,
	diff map[string]jsonutils.FieldChange,
	beforeMap, afterMap map[string]any,
	order []string,
) []agent.PreviewFieldChange {
	fields := make([]agent.PreviewFieldChange, 0, len(diff))
	for path, entry := range diff {
		field, keep := o.field(resource, path, &entry, beforeMap, afterMap)
		if keep {
			fields = append(fields, field)
		}
	}

	position := make(map[string]int, len(order))
	for i, key := range order {
		position[key] = i
	}
	slices.SortFunc(fields, func(a, b agent.PreviewFieldChange) int {
		ap, aok := position[topLevel(a.Path)]
		bp, bok := position[topLevel(b.Path)]
		switch {
		case aok && bok && ap != bp:
			return ap - bp
		case aok != bok:
			if aok {
				return -1
			}
			return 1
		default:
			return strings.Compare(a.Path, b.Path)
		}
	})

	return fields
}

func (o *options) field(
	resource permission.Resource,
	path string,
	entry *jsonutils.FieldChange,
	beforeMap, afterMap map[string]any,
) (agent.PreviewFieldChange, bool) {
	if matches(o.ignore, path) || (len(o.only) > 0 && !matches(o.only, path)) {
		return agent.PreviewFieldChange{}, false
	}

	top := topLevel(path)
	explicit := o.explicit(path)
	if !explicit && o.hidden(path, beforeMap[top], afterMap[top]) {
		return agent.PreviewFieldChange{}, false
	}

	sensitivity := fieldsensitivity.Level(o.registry, resource, top)
	if sensitivity == permission.SensitivityConfidential {
		return agent.PreviewFieldChange{}, false
	}

	field := agent.PreviewFieldChange{
		Path:        path,
		Sensitivity: sensitivity,
		Volatile:    matches(o.volatile, path),
	}

	if target, isRef := o.ref(path); isRef {
		field.Type = assistantartifact.DisplayText
		field.Before, field.BeforeRef = refValue(target, entry.From)
		field.After, field.AfterRef = refValue(target, entry.To)
		field.Label = o.fieldLabel(path, field.Type)

		return field, true
	}

	displayType, shown := o.classify(path, entry.From, entry.To)
	if !shown {
		if !explicit {
			return agent.PreviewFieldChange{}, false
		}
		displayType = assistantartifact.DisplayText
	}

	field.Type = displayType
	field.Before = projected(displayType, entry.From)
	field.After = projected(displayType, entry.To)
	field.Label = o.fieldLabel(path, displayType)

	return field, true
}

func (o *options) hidden(path string, beforeTop, afterTop any) bool {
	top := topLevel(path)
	if _, bookkeeping := bookkeepingKeys[top]; bookkeeping {
		return true
	}
	if strings.HasPrefix(top, "_") {
		return true
	}
	if isRelation(beforeTop) || isRelation(afterTop) {
		return true
	}

	segments := strings.Split(path, ".")
	for _, segment := range segments {
		if assistantartifact.HiddenKey(segment) {
			return true
		}
	}

	return false
}

// isRelation is a value that is another record rather than a value of this
// one: an object with its own id, or a list of them.
func isRelation(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		_, identified := typed["id"]

		return identified
	case []any:
		for _, item := range typed {
			if isRelation(item) {
				return true
			}
		}

		return false
	default:
		return false
	}
}

func (o *options) classify(path string, before, after any) (assistantartifact.DisplayType, bool) {
	if displayType, ok := o.displayType(path); ok {
		return displayType, true
	}

	return assistantartifact.ClassifyValues(camelPath(path), []any{before, after}, true)
}

func (o *options) fieldLabel(path string, displayType assistantartifact.DisplayType) string {
	if label, ok := o.label(path); ok {
		return label
	}

	return assistantartifact.DisplayLabel(camelPath(path), displayType)
}

// camelPath joins a nested path into one camel-case key, so "address.city"
// reads, and classifies, as "addressCity".
func camelPath(path string) string {
	if !strings.Contains(path, ".") {
		return path
	}

	segments := strings.Split(path, ".")
	var builder strings.Builder
	builder.Grow(len(path))
	for i, segment := range segments {
		if i == 0 {
			builder.WriteString(segment)
			continue
		}
		builder.WriteString(stringutils.CapitalizeFirst(segment))
	}

	return builder.String()
}

func projected(displayType assistantartifact.DisplayType, value any) any {
	if value == nil {
		return nil
	}
	if out, ok := assistantartifact.ProjectValue(displayType, value); ok {
		return out
	}

	return nil
}

func refValue(resource permission.Resource, value any) (any, *agent.PreviewRef) {
	text, ok := value.(string)
	if !ok {
		return nil, nil
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}

	id, err := pulid.Parse(text)
	if err != nil {
		return nil, nil
	}

	return text, &agent.PreviewRef{Resource: resource, ID: id}
}
