package graphqlread

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const (
	VerbList           = "list"
	ParamQuery         = "query"
	ConnectionEdges    = "edges"
	ConnectionNode     = "node"
	ConnectionPageInfo = "pageInfo"
	PageHasNext        = "hasNextPage"
	typenameField      = "__typename"
	InputFirst         = "first"
	InputAfter         = "after"
	InputQuery         = "query"
	InputFieldFilters  = "fieldFilters"
	InputSort          = "sort"
)

var ErrUnbound = errors.New("no GraphQL read is bound to this tool")

// Referenced types each id parameter by the resource its spec names, drops
// the ones it names none for, and refuses a reference to no id parameter.
func Referenced(
	tool string,
	params []Param,
	references map[string]readspec.Reference,
) ([]Param, error) {
	out := make([]Param, 0, len(params))
	used := 0
	for _, param := range params {
		if param.Kind != ParamReference && param.Kind != ParamReferences {
			out = append(out, param)
			continue
		}
		ref, ok := references[param.Name]
		if !ok {
			continue
		}
		param.Schema = ref.Schema(param.Kind == ParamReferences)
		out = append(out, param)
		used++
	}
	if used == len(references) {
		return out, nil
	}
	for name := range references {
		if !slices.ContainsFunc(out, func(p Param) bool { return p.Name == name }) {
			return nil, fmt.Errorf("%s references %q, which is not one of its id parameters",
				tool, name)
		}
	}

	return out, nil
}

// Annotate appends each note to its parameter's description, and refuses a
// note for a parameter the tool does not take.
func Annotate(
	tool string,
	params []Param,
	notes map[string]string,
) ([]Param, error) {
	matched := 0
	for idx := range params {
		note, ok := notes[params[idx].Name]
		if !ok {
			continue
		}
		description, _ := params[idx].Schema[toolschema.KeyDescription].(string)
		params[idx].Schema = WithDescription(params[idx].Schema, description+" "+note)
		matched++
	}
	if matched == len(notes) {
		return params, nil
	}
	for name := range notes {
		if !slices.ContainsFunc(params, func(p Param) bool { return p.Name == name }) {
			return nil, fmt.Errorf("%s notes %q, which is not one of its parameters", tool, name)
		}
	}

	return params, nil
}

// KeptRecords are the marked records among the rows a cut page kept.
func KeptRecords(marked []agentrecord.RecordRef, items []map[string]any) []agentrecord.RecordRef {
	if len(marked) == 0 {
		return nil
	}
	ids := make(map[string]struct{}, len(items))
	for _, item := range items {
		if id, _ := item[querytool.RecordIDField].(string); id != "" {
			ids[id] = struct{}{}
		}
	}
	out := make([]agentrecord.RecordRef, 0, len(marked))
	for _, ref := range marked {
		if _, ok := ids[ref.ID]; ok {
			out = append(out, ref)
		}
	}

	return out
}

// OutsideRecords are the records among items that show text written outside
// the organization in one of the fields named, each under its kind.
func OutsideRecords(
	resource permission.Resource,
	fields []string,
	items ...map[string]any,
) []agentrecord.RecordRef {
	if len(fields) == 0 {
		return nil
	}
	var out []agentrecord.RecordRef
	for _, item := range items {
		id, _ := item[querytool.RecordIDField].(string)
		if id == "" {
			continue
		}
		for _, field := range fields {
			if text, _ := item[field].(string); strings.TrimSpace(text) != "" {
				out = append(out, agentrecord.RecordRef{EntityType: resource.String(), ID: id})
				break
			}
		}
	}

	return out
}

// ReadFailure turns the operation's errors into the tool's: a refusal the
// screen would have shown, a spent budget, or a read that failed.
func ReadFailure(result *graphqlreadports.GraphQLReadResult, resource permission.Resource) error {
	if len(result.Errors) == 0 {
		return nil
	}
	first := result.Errors[0]
	switch first.Code {
	case graphqlreadports.GraphQLCodeForbidden, graphqlreadports.GraphQLCodeUnauthorized:
		return &graphqlreadports.ReadRefusedError{Resource: resource, Message: first.Message}
	case graphqlreadports.GraphQLCodeBudgetExceeded:
		return fmt.Errorf("the read budget for this person is spent; try again in %s",
			result.RetryAfter.Round(1e9))
	}

	return fmt.Errorf("the read failed: %s", first.Message)
}

// carried reports whether any row holds the field.
func carried(items []map[string]any, name string) bool {
	for _, item := range items {
		if _, ok := item[name]; ok {
			return true
		}
	}

	return false
}

// Compact drops what tells a model nothing: nulls, empty lists and type
// names, in the record and in each record it carries.
func Compact(node map[string]any) map[string]any {
	out := make(map[string]any, len(node))
	for key, value := range node {
		if key == typenameField || value == nil {
			continue
		}
		switch v := value.(type) {
		case []any:
			if len(v) == 0 {
				continue
			}
			for idx, element := range v {
				if object, ok := element.(map[string]any); ok {
					v[idx] = Compact(object)
				}
			}
		case map[string]any:
			value = Compact(v)
		}
		out[key] = value
	}

	return out
}

func SelectedColumns(fields []Field, items []map[string]any) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if !field.Selected {
			continue
		}
		for _, name := range []string{field.Name, PresentedName(field.Name)} {
			if slices.Contains(out, name) || !carried(items, name) {
				continue
			}
			out = append(out, name)
			break
		}
	}

	return out
}
