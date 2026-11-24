package recordquerytools

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentcapability"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	paramEntity    = "entity"
	paramWhere     = "where"
	paramOr        = "or"
	paramSort      = "sort"
	paramInclude   = "include"
	paramCountOnly = "countOnly"
	paramGroupBy   = "groupBy"
	paramIDs       = "ids"
	paramNumber    = "number"
	includeAll     = "all"

	defaultLimit     = 25
	maxLimit         = 100
	maxGroupRows     = 500
	groupPageSize    = 100
	maxIncludedRows  = 25
	maxGetIDs        = 25
	maxDescribedKeys = 150
)

type Readers interface {
	Get(name string) (agenttoolports.AgentQueryTool, bool)
}

type linked struct {
	readers Readers
}

func (l *linked) Link(readers Readers) { l.readers = readers }

func (l *linked) reader(name string) (agenttoolports.AgentQueryTool, error) {
	if l.readers == nil {
		return nil, errors.New("records are not linked to their readers")
	}
	tool, ok := l.readers.Get(name)
	if !ok {
		return nil, fmt.Errorf("no reader named %q is registered", name)
	}

	return tool, nil
}

type Linker interface {
	Link(readers Readers)
}

func Backing(name string, args map[string]any) (string, error) {
	verb := agentcapability.VerbList
	switch name {
	case agentcapability.RecordGet:
		verb = agentcapability.VerbGet
	case agentcapability.RecordsQuery, agentcapability.DescribeEntity:
	default:
		return "", fmt.Errorf("%q reads no entity", name)
	}
	raw, _ := args[paramEntity].(string)
	entity, ok := agentcapability.EntityNamed(strings.TrimSpace(raw))
	if !ok {
		return "", errortypes.NewValidationError(paramEntity, errortypes.ErrInvalid,
			fmt.Sprintf("%q is not an entity; use one of %s", raw,
				strings.Join(entityNames(verb), ", ")))
	}
	if name == agentcapability.DescribeEntity {
		if entity.List != "" {
			return entity.List, nil
		}

		return entity.Get, nil
	}
	backing := entity.List
	if verb == agentcapability.VerbGet {
		backing = entity.Get
	}
	if backing == "" {
		other := agentcapability.RecordGet
		if verb == agentcapability.VerbGet {
			other = agentcapability.RecordsQuery
		}

		return "", errortypes.NewValidationError(paramEntity, errortypes.ErrInvalid,
			fmt.Sprintf("%s cannot read %s; use %s", name, entity.Name, other))
	}

	return backing, nil
}

func entityNames(verb agentcapability.Verb) []string {
	out := make([]string, 0, 128)
	for _, entity := range agentcapability.Entities() {
		if (verb == agentcapability.VerbGet && entity.Get != "") ||
			(verb != agentcapability.VerbGet && entity.List != "") {
			out = append(out, entity.Name)
		}
	}
	slices.Sort(out)

	return out
}

func readPolicy(name, rationale string) agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(name, &querytool.ReadSpec{
		Resource:  permission.ResourceAssistant,
		Reads:     agenttooltrust.ExternalReadMarked,
		Rationale: rationale,
	})
}

func zoneOf(params *agenttoolports.QueryToolParams) *time.Location {
	if params.Timezone == "" {
		return time.UTC
	}
	zone, err := time.LoadLocation(params.Timezone)
	if err != nil {
		return time.UTC
	}

	return zone
}

func fieldsOf(reader agenttoolports.AgentQueryTool, entity *agentcapability.Entity) *fieldSet {
	set := &fieldSet{aliases: entity.Aliases}
	if fielded, ok := reader.(querytool.WhereReader); ok {
		set.config = fielded.WhereFields()
	}
	if extended, ok := reader.(querytool.WhereExtender); ok {
		set.extras = extended.WhereExtras()
	}

	return set
}

func conditions(raw any) ([]condition, error) {
	if raw == nil {
		return nil, nil
	}
	encoded, err := sonic.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var decoded []struct {
		Field  string `json:"field"`
		Op     string `json:"op"`
		Value  any    `json:"value"`
		Values []any  `json:"values"`
		Days   int    `json:"days"`
	}
	if err = sonic.Unmarshal(encoded, &decoded); err != nil {
		return nil, fmt.Errorf("%w: conditions are {field, op, value | values | days}", ErrBadArgument)
	}
	out := make([]condition, 0, len(decoded))
	for _, item := range decoded {
		out = append(out, condition{
			Field: strings.TrimSpace(item.Field), Op: strings.ToLower(strings.TrimSpace(item.Op)),
			Value: item.Value, Values: item.Values, Days: item.Days,
		})
	}

	return out, nil
}

type sortField struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

func sortFields(raw any) ([]sortField, error) {
	if raw == nil {
		return nil, nil
	}
	encoded, err := sonic.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var out []sortField
	if err = sonic.Unmarshal(encoded, &out); err != nil {
		return nil, fmt.Errorf("%w: sort is a list of {field, direction}", ErrBadArgument)
	}

	return out, nil
}

func rowIDs(items any) []string {
	rows, ok := items.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		object, isObject := row.(map[string]any)
		if !isObject {
			continue
		}
		if id, hasID := object["id"].(string); hasID {
			out = append(out, id)
		}
	}

	return out
}

func readerParams(
	params *agenttoolports.QueryToolParams,
	values map[string]any,
) *agenttoolports.QueryToolParams {
	copied := *params
	copied.Params = values

	return &copied
}

func idParam(schema map[string]any) (name string, list bool) {
	properties, _ := schema["properties"].(map[string]any)
	keys := slices.Sorted(maps.Keys(properties))
	for _, key := range keys {
		if strings.HasSuffix(key, "Ids") {
			return key, true
		}
	}
	for _, key := range keys {
		if strings.HasSuffix(key, "Id") || key == "id" {
			return key, false
		}
	}

	return "", false
}

func numberParam(schema map[string]any) string {
	properties, _ := schema["properties"].(map[string]any)
	for _, key := range slices.Sorted(maps.Keys(properties)) {
		if key == "number" || strings.HasSuffix(key, "Number") {
			return key
		}
	}

	return ""
}

func includeNames(raw any, entity *agentcapability.Entity) ([]string, error) {
	asked := querytool.OptionalStrings(map[string]any{paramInclude: raw}, paramInclude)
	if slices.Contains(asked, includeAll) {
		return entity.IncludeNames(), nil
	}
	for _, name := range asked {
		if _, ok := entity.Include(name); !ok {
			valid := entity.IncludeNames()
			if len(valid) == 0 {
				return nil, fmt.Errorf("%w: %s has no related sections to include", ErrBadArgument, entity.Name)
			}

			return nil, fmt.Errorf("%w: %q is not a section of %s; use any of %s or all",
				ErrBadArgument, name, entity.Name, strings.Join(valid, ", "))
		}
	}

	return asked, nil
}

func groupCounts(items any, field string, counts map[string]int) {
	rows, ok := items.([]any)
	if !ok {
		return
	}
	for _, row := range rows {
		object, isObject := row.(map[string]any)
		if !isObject {
			continue
		}
		counts[fmt.Sprint(nested(object, field))]++
	}
}

func nested(object map[string]any, path string) any {
	var current any = object
	for _, part := range strings.Split(path, ".") {
		next, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = next[part]
	}

	return current
}

func sortedGroups(counts map[string]int) []Group {
	out := make([]Group, 0, len(counts))
	for value, count := range counts {
		out = append(out, Group{Value: value, Count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}

		return out[i].Value < out[j].Value
	})

	return out
}

func queryInner(
	ctx context.Context,
	reader agenttoolports.AgentQueryTool,
	params *agenttoolports.QueryToolParams,
	values map[string]any,
) (any, error) {
	return reader.Query(ctx, readerParams(params, values))
}
