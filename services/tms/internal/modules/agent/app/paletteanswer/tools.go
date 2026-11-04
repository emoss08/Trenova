package paletteanswer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/argcheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolsummary"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/toolcoerce"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/timeutils"
)

// reader is the person's reads: every call is checked against what they may
// read, exactly as the Desk checks a query tool, and nothing here writes.
type reader struct {
	catalog     *catalog
	validator   *toolschema.Validator
	search      kernelports.GlobalSearchService
	permissions permissionports.PermissionEngine
	actor       *permissionports.RequestActor
	timezone    string
}

func (r *reader) tools() []llmports.NovaTool {
	tools := make([]llmports.NovaTool, 0, 3)
	tools = append(tools, &searchTool{reader: r})

	return append(tools, &getTool{reader: r}, &listTool{reader: r})
}

type searchTool struct{ reader *reader }

func (t *searchTool) Spec() kernelports.ToolSpec {
	return kernelports.ToolSpec{
		Name: "search_records",
		Description: "Find records by a name, number or phrase: a PRO number, a customer, " +
			"a driver, a location. Returns each match's id, kind and title.",
		Parameters: map[string]any{
			toolschema.KeyType: toolschema.TypeObject,
			toolschema.KeyProperties: map[string]any{
				"query": map[string]any{
					toolschema.KeyType:        toolschema.TypeString,
					toolschema.KeyDescription: "What to look for, as the person wrote it.",
				},
			},
			toolschema.KeyRequired: []string{"query"},
		},
	}
}

type searchHit struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
}

func (t *searchTool) Call(ctx context.Context, arguments map[string]any) llmports.NovaToolResult {
	query, _ := arguments["query"].(string)
	query = strings.TrimSpace(query)
	if query == "" {
		return refused("search_records needs a query.")
	}

	actor := t.reader.actor
	result, err := t.reader.search.Search(ctx, &kernelports.GlobalSearchRequest{
		Query:      query,
		TenantInfo: actor.TenantInfo(),
		Principal: kernelports.PrincipalInfo{
			Type:     actor.PrincipalType,
			ID:       actor.PrincipalID,
			UserID:   actor.UserID,
			APIKeyID: actor.APIKeyID,
		},
		Limit: searchLimit,
	})
	if err != nil {
		return failed("The search failed: " + err.Error())
	}

	hits := make([]searchHit, 0, searchLimit)
	records := make([]agentpagecontext.EntityRef, 0, searchLimit)
	for _, group := range result.Groups {
		for _, hit := range group.Hits {
			kind := string(hit.EntityType)
			if resource, ok := resourceOf(hit.ID); ok {
				kind = resource.String()
				records = append(records, agentpagecontext.EntityRef{
					Type: kind, ID: hit.ID, Label: hit.Title,
				})
			}
			hits = append(hits, searchHit{
				ID: hit.ID, Kind: kind, Title: hit.Title, Subtitle: hit.Subtitle,
			})
		}
	}

	encoded := t.reader.encode("search_records", hits)
	encoded.Records = records

	return encoded
}

type getTool struct{ reader *reader }

func (t *getTool) Spec() kernelports.ToolSpec {
	return kernelports.ToolSpec{
		Name:        "get_record",
		Description: "Read one record in full by its id, such as an id search_records or list_records returned.",
		Parameters: map[string]any{
			toolschema.KeyType: toolschema.TypeObject,
			toolschema.KeyProperties: map[string]any{
				"id": map[string]any{
					toolschema.KeyType:        toolschema.TypeString,
					toolschema.KeyDescription: "The record's id, such as shp_01J….",
				},
			},
			toolschema.KeyRequired: []string{"id"},
		},
	}
}

func (t *getTool) Call(ctx context.Context, arguments map[string]any) llmports.NovaToolResult {
	id, _ := arguments["id"].(string)
	id = strings.TrimSpace(id)
	resource, ok := resourceOf(id)
	if !ok {
		return refused(
			fmt.Sprintf("%q is not a record id. Find the record with search_records first.", id),
		)
	}
	tool, ok := t.reader.catalog.get(resource)
	if !ok {
		return refused(fmt.Sprintf("A %s cannot be read here.", resource.String()))
	}
	params, ok := agenttoolports.RecordIDParams(tool.ParamSchema(), id)
	if !ok {
		return refused(fmt.Sprintf("A %s cannot be read by its id here.", resource.String()))
	}

	return t.reader.query(ctx, tool, params, false)
}

type listTool struct{ reader *reader }

func (t *listTool) Spec() kernelports.ToolSpec {
	return kernelports.ToolSpec{
		Name: "list_records",
		Description: "List records of one kind, filtered, with how many there are in all. " +
			"The kinds and the filters each takes:\n" + t.reader.catalog.listGuide(),
		Parameters: map[string]any{
			toolschema.KeyType: toolschema.TypeObject,
			toolschema.KeyProperties: map[string]any{
				"kind": map[string]any{
					toolschema.KeyType:        toolschema.TypeString,
					toolschema.KeyEnum:        t.reader.catalog.kinds,
					toolschema.KeyDescription: "The kind of record to list.",
				},
				"filters": map[string]any{
					toolschema.KeyType:        toolschema.TypeObject,
					toolschema.KeyDescription: "The kind's filters by name; leave out the ones you do not need. A filter named beside kind is read as one of these.",
				},
			},
			toolschema.KeyRequired: []string{"kind"},
		},
	}
}

func (t *listTool) Call(ctx context.Context, arguments map[string]any) llmports.NovaToolResult {
	kind, _ := arguments["kind"].(string)
	tool, ok := t.reader.catalog.list(kind)
	if !ok {
		return refused(fmt.Sprintf("%q is not a kind list_records can list.", kind))
	}
	return t.reader.query(ctx, tool, listFilters(arguments), true)
}

// listFilters is what the list is asked for. A model often writes a filter
// beside kind rather than inside filters (`{"kind": "shipment", "late": true}`);
// it is the list's filter either way, so it is read as one, and the list's own
// argument check refuses a name the list does not take. Dropping it would list
// everything and present it as the answer. A filter named in both places is
// read from filters.
func listFilters(arguments map[string]any) map[string]any {
	filters := map[string]any{}
	for name, value := range arguments {
		if name != "kind" && name != "filters" {
			filters[name] = value
		}
	}
	if nested, ok := arguments["filters"].(map[string]any); ok {
		for name, value := range nested {
			filters[name] = value
		}
	}

	return filters
}

// query runs one Desk read as the person asking, and names the read and what
// it was asked on the result, so the call's record says what was tried.
func (r *reader) query(
	ctx context.Context,
	tool agenttoolports.AgentQueryTool,
	params map[string]any,
	list bool,
) llmports.NovaToolResult {
	result, asked := r.read(ctx, tool, params, list)
	result.Read = tool.Name()
	result.Params = asked

	return result
}

func (r *reader) read(
	ctx context.Context,
	tool agenttoolports.AgentQueryTool,
	params map[string]any,
	list bool,
) (llmports.NovaToolResult, map[string]any) {
	resource := tool.Policy().Resource
	check, err := r.permissions.Check(ctx, r.actor.PermissionCheck(resource, permission.OpRead))
	if err != nil {
		return failed(
			"Whether you may read this could not be checked. Try again in a moment.",
		), params
	}
	if !check.Allowed {
		return refused(
			fmt.Sprintf("The person asking cannot read %s records.", resource.String()),
		), params
	}

	schema := tool.ParamSchema()
	today := time.Unix(timeutils.NowUnix(), 0).In(timeutils.LoadLocation(r.timezone))
	contract, err := argcheck.ContractCall(
		r.validator, tool.Name(), schema, params, toolcoerce.WithToday(today),
	)
	if err != nil {
		// The refusal names each argument that does not fit and what the
		// tool wanted there, so the next round can send it right.
		return refused(argcheck.Refusal(tool.Name(), schema, err).Content), params
	}
	params = contract.Args

	data, err := tool.Query(ctx, &agenttoolports.QueryToolParams{
		OrganizationID: r.actor.OrganizationID,
		BusinessUnitID: r.actor.BusinessUnitID,
		Actor:          r.actor,
		Timezone:       r.timezone,
		Params:         params,
		IncludeTotal:   list,
		// The palette reads at the tier any agent reads at, never the
		// person's own when theirs is higher: what Nova reads goes to a
		// model provider.
		DataAccessCeiling: permission.SensitivityInternal,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return failed("The read was stopped."), params
		}

		return failed(fmt.Sprintf("%s failed: %s", tool.Name(), err.Error())), params
	}

	result := r.encode(tool.Name(), data)
	if note := contract.Note(); note != "" && !result.Failed {
		result.Content += "\n\n[" + note + "]"
	}

	return result, params
}

// encode turns a read into what the model reads, cut to fit when it is a long
// list, with the records it describes and a line for the person watching.
func (r *reader) encode(name string, data any) llmports.NovaToolResult {
	now := timeutils.NowUnix()
	encoded, document, err := toolresult.EncodeModelDocument(data, now, r.timezone, nil)
	if err != nil {
		return failed(name + " returned data that could not be read.")
	}
	note := ""
	if fitted, cut := toolresult.FitPage(data, encoded, now, r.timezone, nil); cut {
		encoded, document = fitted.Encoded, fitted.Document
		note = "\n\n" + fitted.Note()
	}

	return llmports.NovaToolResult{
		Content: toolresult.Fence(name, encoded) + note,
		Summary: toolsummary.Result(name, document),
		Records: toolresult.CitedRecords(encoded),
	}
}

func resourceOf(id string) (permission.Resource, bool) {
	separator := strings.IndexByte(id, '_')
	if separator <= 0 {
		return "", false
	}

	return recordkind.ResourceOfIDPrefix(id[:separator+1])
}

// refused is a call turned away before anything was read.
func refused(content string) llmports.NovaToolResult {
	return llmports.NovaToolResult{Content: content, Failed: true, Refused: true}
}

// failed is a call whose read ran, or could not be checked, and went wrong.
func failed(content string) llmports.NovaToolResult {
	return llmports.NovaToolResult{Content: content, Failed: true}
}
