package aiauditservice

import (
	"context"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/aiaudit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/fieldsensitivity"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	listSegment    = "[]"
	pathSeparator  = "."
	truncatedField = "_truncated"
)

// Masker masks sensitive values: a named field's value, and patterns inside
// free text. auditservice.SensitiveDataManager is the implementation.
type Masker interface {
	MaskField(name, value string) (string, bool)
	MaskText(text string) (string, bool)
}

// ToolPolicies names what a tool acts on, by its name.
type ToolPolicies interface {
	Get(name string) (serviceports.ToolPolicy, bool)
}

// Redactor is what the trail keeps of an argument map: confidential fields
// replaced, sensitive values masked, runtime-only parameters dropped, a size
// bound, and the sensitivity of every path, so a reader's own ceiling can be
// applied when the row is read.
type Redactor struct {
	registry *permission.Registry
	policies ToolPolicies
	masker   Masker
}

func NewRedactor(registry *permission.Registry, policies ToolPolicies, masker Masker) *Redactor {
	if registry == nil {
		registry = permission.NewRegistry()
	}

	return &Redactor{registry: registry, policies: policies, masker: masker}
}

// RedactedArguments is what a row records of one call's arguments.
type RedactedArguments struct {
	Values        map[string]any
	Sensitivity   *aiaudit.ArgumentSensitivity
	RedactedPaths []string
	Truncated     bool
}

// Policy is a tool's policy, when the tool is still registered.
func (r *Redactor) Policy(toolName string) (serviceports.ToolPolicy, bool) {
	if r.policies == nil || toolName == "" {
		return serviceports.ToolPolicy{}, false
	}

	return r.policies.Get(toolName)
}

// Arguments redacts a tool's arguments for the trail.
func (r *Redactor) Arguments(toolName string, args map[string]any) (*RedactedArguments, error) {
	if len(args) == 0 {
		return &RedactedArguments{}, nil
	}

	policy, known := r.Policy(toolName)
	walk := &argumentWalk{
		redactor: r,
		resource: policy.Resource,
		levels:   make(map[string]permission.FieldSensitivity, len(args)),
		redacted: make(map[string]struct{}, 2),
	}

	values := make(map[string]any, len(args))
	for key, value := range args {
		if dropsParameter(policy, known, key) {
			continue
		}
		values[key] = walk.value(key, key, value)
	}

	canonical, err := aiaudit.CanonicalJSONValue(values)
	if err != nil {
		return nil, err
	}
	bounded, truncated, err := boundArguments(canonical.(map[string]any))
	if err != nil {
		return nil, err
	}

	result := &RedactedArguments{
		Values:        bounded,
		RedactedPaths: walk.redactedPaths(),
		Truncated:     truncated,
	}
	if len(walk.levels) > 0 || policy.Resource != "" {
		result.Sensitivity = &aiaudit.ArgumentSensitivity{
			Resource: policy.Resource,
			Levels:   walk.levels,
		}
	}

	return result, nil
}

// dropsParameter reports a parameter the runtime writes for itself rather
// than the model: the owner a self-scoped call is about. An unknown tool's
// is dropped too, since nothing can say it was the model's.
func dropsParameter(policy serviceports.ToolPolicy, known bool, key string) bool {
	if key != serviceports.SelfScopeOwnerParam {
		return false
	}

	return !known || policy.Scope == agent.ToolScopeSelf
}

type argumentWalk struct {
	redactor *Redactor
	resource permission.Resource
	levels   map[string]permission.FieldSensitivity
	redacted map[string]struct{}
}

func (w *argumentWalk) level(key string) permission.FieldSensitivity {
	if w.resource == "" {
		return permission.SensitivityInternal
	}

	return fieldsensitivity.Level(w.redactor.registry, w.resource, key)
}

func (w *argumentWalk) value(path, key string, value any) any {
	level := w.level(key)
	if level != permission.SensitivityPublic && level != "" {
		w.levels[path] = level
	}
	if level == permission.SensitivityConfidential {
		w.redacted[path] = struct{}{}

		return aiaudit.ConfidentialPlaceholder
	}

	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for child, childValue := range typed {
			out[child] = w.value(path+pathSeparator+child, child, childValue)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = w.value(path+listSegment, key, item)
		}
		return out
	case string:
		return w.text(path, key, typed)
	default:
		return typed
	}
}

func (w *argumentWalk) text(path, key, value string) string {
	value = cleanText(value)
	if value == "" || pulid.LooksLike(value) || w.redactor.masker == nil {
		return value
	}

	masked, changed := w.redactor.masker.MaskField(key, value)
	if changed {
		w.redacted[path] = struct{}{}
	}

	return masked
}

func (w *argumentWalk) redactedPaths() []string {
	if len(w.redacted) == 0 {
		return []string{}
	}

	paths := make([]string, 0, len(w.redacted))
	for path := range w.redacted {
		paths = append(paths, path)
	}
	slices.Sort(paths)

	return paths
}

// boundArguments keeps an argument map inside the size the trail records.
// Top-level values are kept in key order while they fit; the rest are
// replaced with a placeholder, and when even the placeholders do not fit the
// remaining keys are dropped and the map says it was cut.
func boundArguments(values map[string]any) (map[string]any, bool, error) {
	encoded, err := canonicalJSON.Marshal(values)
	if err != nil {
		return nil, false, err
	}
	if len(encoded) <= aiaudit.MaxArgumentsBytes {
		return values, false, nil
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	const envelope = 64
	budget := aiaudit.MaxArgumentsBytes - envelope
	out := make(map[string]any, len(values))
	used := 2
	for _, key := range keys {
		entry, marshalErr := canonicalJSON.Marshal(map[string]any{key: values[key]})
		if marshalErr != nil {
			return nil, false, marshalErr
		}
		if used+len(entry) <= budget {
			out[key] = values[key]
			used += len(entry)

			continue
		}

		placeholder, marshalErr := canonicalJSON.Marshal(
			map[string]any{key: aiaudit.TruncatedPlaceholder},
		)
		if marshalErr != nil {
			return nil, false, marshalErr
		}
		if used+len(placeholder) > budget {
			break
		}
		out[key] = aiaudit.TruncatedPlaceholder
		used += len(placeholder)
	}
	out[truncatedField] = true

	return out, true, nil
}

// Text is free text fit for the trail: sensitive patterns masked inside it,
// control characters a jsonb or text column refuses stripped, and cut on a
// character boundary to fit its column.
func (r *Redactor) Text(value string, limit int) string {
	value = cleanText(value)
	if value == "" {
		return ""
	}
	if r.masker != nil {
		value, _ = r.masker.MaskText(value)
	}

	return truncateRunes(value, limit)
}

func cleanText(value string) string {
	if !utf8.ValidString(value) {
		value = strings.ToValidUTF8(value, "�")
	}
	if strings.IndexByte(value, 0) >= 0 {
		value = strings.ReplaceAll(value, "\x00", "")
	}

	return value
}

func truncateRunes(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}

	cut := limit
	for cut > 0 && !utf8.RuneStart(value[cut]) {
		cut--
	}

	return value[:cut]
}

// ReaderArguments withholds every recorded path the reader's ceiling on the
// tool's resource does not reach. A row recorded without a sensitivity map
// is shown as recorded: it held nothing above Internal.
func ReaderArguments(
	ctx context.Context,
	event *aiaudit.AIAuditEvent,
	ceilings serviceports.FieldCeilings,
) map[string]any {
	if event == nil {
		return nil
	}
	sensitivity := event.ArgumentSensitivity
	if len(event.Arguments) == 0 || sensitivity == nil || len(sensitivity.Levels) == 0 ||
		sensitivity.Resource == "" {
		return event.Arguments
	}

	ceiling := permission.SensitivityInternal
	if ceilings != nil {
		ceiling = ceilings.For(ctx, sensitivity.Resource)
	}
	hidden := make(map[string]struct{}, len(sensitivity.Levels))
	for path, level := range sensitivity.Levels {
		if !fieldsensitivity.VisibleAt(level, ceiling) {
			hidden[path] = struct{}{}
		}
	}
	if len(hidden) == 0 {
		return event.Arguments
	}

	masked, _ := withhold("", event.Arguments, hidden).(map[string]any)

	return masked
}

func withhold(path string, value any, hidden map[string]struct{}) any {
	if path != "" {
		if _, ok := hidden[path]; ok {
			return aiaudit.WithheldPlaceholder
		}
	}

	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			childPath := key
			if path != "" {
				childPath = path + pathSeparator + key
			}
			out[key] = withhold(childPath, child, hidden)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = withhold(path+listSegment, item, hidden)
		}
		return out
	default:
		return typed
	}
}

var canonicalJSON = sonic.Config{SortMapKeys: true, UseNumber: true}.Froze()
