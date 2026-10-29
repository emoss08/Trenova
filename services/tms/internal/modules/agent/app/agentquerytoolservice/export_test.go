package agentquerytoolservice

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/gettool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqllisttool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqlread"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/inboxquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/textcase"
)

// convertedListSpecs are every list read through GraphQL by a spec: those
// registered by the spec alone, and the inbox, whose search by words stays on
// the retrieval index beside its GraphQL read.
func convertedListSpecs() []readspec.List {
	return append(graphQLListSpecs(), inboxquerytools.InboundMessagesListSpec())
}

// ConvertedListNames are the list tools read through GraphQL by a spec.
func ConvertedListNames() []string {
	specs := convertedListSpecs()
	names := make([]string, 0, len(specs))
	for idx := range specs {
		names = append(names, specs[idx].Name)
	}

	return names
}

// ConvertedListResource is the resource a converted list reads.
func ConvertedListResource(name string) permission.Resource {
	specs := convertedListSpecs()
	for idx := range specs {
		if specs[idx].Name == name {
			return specs[idx].Resource
		}
	}

	return ""
}

// ConvertedListTool is the registered converted list of that name.
func ConvertedListTool(
	name string,
	reader graphqlreadports.GraphQLReader,
	permissions permissionports.PermissionEngine,
) (agenttoolports.AgentQueryTool, error) {
	specs := convertedListSpecs()
	for idx := range specs {
		if specs[idx].Name == name {
			return graphqllisttool.NewRead(&specs[idx], reader, permissions)
		}
	}

	return nil, graphqlread.ErrUnbound
}

// ConvertedGetNames are the gets that read through GraphQL.
func ConvertedGetNames() []string {
	specs := graphQLGetSpecs()
	names := make([]string, 0, len(specs))
	for idx := range specs {
		names = append(names, specs[idx].Name)
	}

	return names
}

// ConvertedGetSelfScoped reports whether a converted get reads the caller's
// own record through a resolver that checks no resource, so only dispatch's
// authorize of the tool's own policy can refuse it.
func ConvertedGetSelfScoped(name string) bool {
	return graphqlread.Reads[name].SelfScoped
}

// ConvertedGetResource is the resource a converted get reads.
func ConvertedGetResource(name string) permission.Resource {
	specs := graphQLGetSpecs()
	for idx := range specs {
		if specs[idx].Name == name {
			return specs[idx].Resource
		}
	}

	return ""
}

// ConvertedGetCall is the call a converted get takes for that id.
func ConvertedGetCall(tool agenttoolports.AgentQueryTool, id pulid.ID) map[string]any {
	get, ok := tool.(*gettool.GraphQLTool)
	if !ok {
		return nil
	}
	call := requiredCall(get.Params, get.Required)
	switch {
	case get.ByID && get.Spec.Batch > 0:
		call[get.Spec.ParamName] = []any{id.String()}
	case get.ByID:
		call[get.Spec.ParamName] = id.String()
	}

	return call
}

// requiredCall names a value for each required argument: an enum's first
// value, or an id.
func requiredCall(params []graphqlread.Param, required []string) map[string]any {
	call := make(map[string]any, len(required)+1)
	for _, param := range params {
		if !slices.Contains(required, param.Name) {
			continue
		}
		if values, enumerated := param.Schema[toolschema.KeyEnum].([]string); enumerated {
			call[param.Name] = values[0]
			continue
		}
		call[param.Name] = requiredID(param)
	}

	return call
}

// ConvertedGetTool is the registered converted get of that name.
func ConvertedGetTool(
	name string,
	reader graphqlreadports.GraphQLReader,
	permissions permissionports.PermissionEngine,
) (agenttoolports.AgentQueryTool, error) {
	specs := graphQLGetSpecs()
	for idx := range specs {
		if specs[idx].Name == name {
			return gettool.NewGraphQL(&specs[idx], reader, permissions)
		}
	}

	return nil, graphqlread.ErrUnbound
}

// MinimalCall is the smallest call a converted list takes: a page of five,
// or, for a list keyed by a record, that record's id.
func MinimalCall(tool agenttoolports.AgentQueryTool) map[string]any {
	if list, isList := tool.(*graphqllisttool.Tool); isList {
		call := requiredCall(list.Params, list.Required)
		call["limit"] = 5
		return call
	}
	keyed, ok := tool.(*graphqllisttool.UnpagedTool)
	if !ok {
		return map[string]any{"limit": 5}
	}
	call := make(map[string]any, len(keyed.Required))
	for _, name := range keyed.Required {
		call[name] = pulid.MustNew("wrk_").String()
		for _, param := range keyed.Params {
			if param.Name == name && param.Kind == graphqlread.ParamReferences {
				call[name] = []any{pulid.MustNew("wrk_").String()}
			}
		}
	}

	return call
}

// requiredID is an id for a required argument, or a list of one for an
// argument that takes several.
func requiredID(param graphqlread.Param) any {
	if param.Kind == graphqlread.ParamReferences {
		return []any{pulid.MustNew("x_").String()}
	}

	return pulid.MustNew("x_").String()
}

// ExpectedMention is how a converted list's description must name one of its
// parameters, and whether it must name it at all: a parameter whose field is
// hidden at the default data access, or a range's upper end, it need not.
func ExpectedMention(tool agenttoolports.AgentQueryTool, name string) (string, bool) {
	list, ok := tool.(*graphqllisttool.Tool)
	if !ok {
		return "", false
	}
	for _, param := range list.Params {
		if param.Name != name {
			continue
		}
		mention, described := graphqllisttool.DescribedName(param)
		if !described || !list.Describable(param) {
			return "", false
		}
		if parent, _, nested := strings.Cut(param.Field, "."); nested {
			mention = textcase.LowerFirst(strings.TrimPrefix(mention, parent))
		}

		return mention, true
	}

	return "", false
}

// RunView reads a composed view through a list tool, as compose does.
func RunView(
	ctx context.Context,
	tool agenttoolports.AgentQueryTool,
	params *agenttoolports.QueryToolParams,
	filters []domaintypes.FieldFilter,
	sorting []domaintypes.SortField,
) error {
	_, err := tool.(querytool.ViewRunner).RunView(ctx, params, "", filters, sorting, 5)

	return err
}
