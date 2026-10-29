package graphqlread

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
)

// getKey is the variable a get's operation takes the record's id in: $id,
// or the one id it requires, for a record read by the id of the one it
// belongs to (a driver's hours by $workerId), as the generator proved.
func (r *Read) GetKey() string {
	key := ""
	for _, variable := range r.Variables {
		if variable.Name == IDVariable {
			return IDVariable
		}
		if variable.NonNull && !variable.List && variable.Type == TypeID {
			key = variable.Name
		}
	}
	if key == "" {
		return IDVariable
	}

	return key
}

// pagedInput is the input variable a list's operation pages inside, when its
// root takes no page of its own and no table input: a bespoke input that
// holds first and after (InboundMessagesInput, CaptureBatchesInput).
func (r *Read) PagedInput() string {
	if _, _, tabled := r.TableInput(); tabled || r.PagesByCursor() {
		return ""
	}
	for _, variable := range r.Variables {
		if variable.Kind != KindInput {
			continue
		}
		var first, after bool
		for _, field := range Inputs[variable.Type] {
			first = first || (field.Name == InputFirst && field.Type == TypeInt)
			after = after || (field.Name == InputAfter && field.Type == TypeString)
		}
		if first && after {
			return variable.Name
		}
	}

	return ""
}

// pagesByCursor reports whether a list's root takes its own first and after,
// as a connection paged without a table input does.
func (r *Read) PagesByCursor() bool {
	var first, after bool
	for _, variable := range r.Variables {
		first = first || (variable.Name == InputFirst && variable.Type == TypeInt)
		after = after || (variable.Name == InputAfter && variable.Type == TypeString)
	}

	return first && after
}

// declares reports whether the read's operation takes a variable.
func (r *Read) Declares(name string) bool {
	return slices.ContainsFunc(r.Variables, func(v Variable) bool { return v.Name == name })
}

// pagesByOffset reports whether an unpaged list's root takes a limit and an
// offset, so the tool can page it by cursor.
func (r *Read) PagesByOffset() bool {
	var limit, offset bool
	for _, variable := range r.Variables {
		limit = limit || (variable.Name == querytool.ParamLimit && variable.Type == TypeInt)
		offset = offset || (variable.Name == querytool.ParamOffset && variable.Type == TypeInt)
	}

	return limit && offset
}

// argumentParams are an unpaged list's parameters: each of its operation's
// variables, typed by the root field argument it is sent as and sent as
// that variable, and the names of those the operation requires. The offset
// of a list that pages by cursor is the tool's to send, never a parameter.
func (r *Read) ArgumentParams(tool string) (params []Param, required []string,
	err error,
) {
	paged := r.PagesByOffset()
	for _, variable := range r.Variables {
		if variable.Kind == KindInput || (paged && variable.Name == querytool.ParamOffset) {
			continue
		}
		field := Field{
			Name:        variable.Name,
			Type:        variable.Type,
			Kind:        variable.Kind,
			List:        variable.List,
			NonNull:     variable.NonNull,
			Enum:        variable.Enum,
			Description: variable.Description,
		}
		derived := ArgumentParam(&field)
		if len(derived) == 0 {
			return nil, nil, fmt.Errorf("%s: %s takes $%s, which no parameter can be derived "+
				"from; describe the argument in the schema", tool, r.Operation, variable.Name)
		}
		for _, param := range derived {
			param.OnInput = true
			if said := variable.Description; said != "" && !variable.List &&
				variable.Type == TypeString {
				param.Schema = WithDescription(param.Schema, said)
			}
			params = append(params, param)
		}
		if variable.NonNull {
			required = append(required, variable.Name)
		}
	}
	slices.SortFunc(params, func(a, b Param) int { return strings.Compare(a.Name, b.Name) })

	return params, required, nil
}
