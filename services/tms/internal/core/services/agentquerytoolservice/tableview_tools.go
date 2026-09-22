package agentquerytoolservice

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/tablequeryservice"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
)

/*
The tools behind "show me a table of ...".

They exist because the assistant and the Ask input on a data table are the same
question asked in two places, and answering it twice would mean two vocabularies
and one of them tested. Both go through tablequeryservice, which goes through
filtercatalog, which is also what the list tools compile against.

What comes back is a view, not rows: the filters, the sort, and the path that
opens the real table with them applied. That matters because a table the person
opens is live, sortable, exportable and permission-checked on every page, where
rows pasted into a conversation are a snapshot nobody can act on.
*/

// tableComposer is the slice of the compose service these tools use.
type tableComposer interface {
	Compose(
		ctx context.Context,
		req *tablequeryservice.ComposeRequest,
	) (*tablequeryservice.ComposeResult, error)
}

type composeTableViewTool struct {
	composer tableComposer
	catalog  *filtercatalog.Catalog
}

func newComposeTableViewTool(
	composer *tablequeryservice.Service,
	catalog *filtercatalog.Catalog,
) serviceports.AgentQueryTool {
	return &composeTableViewTool{composer: composer, catalog: catalog}
}

func (t *composeTableViewTool) Name() string { return "compose_table_view" }

func (t *composeTableViewTool) Description() string {
	return "Turn a description of what someone wants to see into a filtered, sorted view " +
		"of one of the application's tables, and give back the link that opens it. Use it " +
		"when the answer is a list a person should work from rather than a few rows to " +
		"read out: \"the shipments still in transit that were due yesterday\", \"drivers " +
		"whose medical card lapses this month\". The view opens live, so it stays correct " +
		"after you have stopped talking. Anything the description asked for that the " +
		"table cannot express comes back named, so say so rather than implying it applied."
}

func (t *composeTableViewTool) ParamSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"entity", "description"},
		"properties": map[string]any{
			"entity": map[string]any{
				"type":        "string",
				"enum":        t.entities(),
				"description": "Which table to build the view of.",
			},
			"description": map[string]any{
				"type": "string",
				"description": "What the view should show, in plain words. " +
					"Name the conditions, not the column names.",
			},
		},
		"additionalProperties": false,
	}
}

func (t *composeTableViewTool) entities() []string {
	resources := t.catalog.Resources()
	names := make([]string, 0, len(resources))
	for _, resource := range resources {
		names = append(names, resource.Entity)
	}

	return names
}

// PermissionResource is the catalog's own resource rather than any one table's.
//
// The tool spans every catalogued entity, so there is no single table to name
// here; the read that matters is checked per call by the compose service,
// against whichever table was asked for.
func (t *composeTableViewTool) PermissionResource() permission.Resource {
	return permission.ResourceReport
}

type tableViewResult struct {
	Entity      string   `json:"entity"`
	Resource    string   `json:"resource"`
	Path        string   `json:"path"`
	Explanation string   `json:"explanation"`
	Terms       []string `json:"terms,omitempty"`
	// Unresolved is what the description asked for that the table cannot
	// express. It is returned rather than dropped because a view that quietly
	// lost a condition looks like an answer.
	Unresolved  []unresolvedTermRow `json:"unresolved,omitempty"`
	FilterCount int                 `json:"filterCount"`
}

type unresolvedTermRow struct {
	Phrase string `json:"phrase"`
	Reason string `json:"reason"`
}

func (t *composeTableViewTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	entity := optionalString(params.Params, "entity")
	resource, ok := t.catalog.ByEntity(entity)
	if !ok {
		return nil, fmt.Errorf(
			"%q is not a table that can be described; the tables are: %s",
			entity, strings.Join(t.entities(), ", "),
		)
	}

	composed, err := t.composer.Compose(ctx, &tablequeryservice.ComposeRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  params.OrganizationID,
			BuID:   params.BusinessUnitID,
			UserID: params.Actor.UserID,
		},
		Actor:    params.Actor,
		Resource: resource.Resource,
		Prompt:   optionalString(params.Params, "description"),
		Timezone: params.Timezone,
	})
	if err != nil {
		return nil, err
	}

	result := tableViewResult{
		Entity:      resource.Entity,
		Resource:    resource.Resource.String(),
		Path:        tableViewPath(resource, composed),
		Explanation: composed.Explanation,
		Terms:       composed.Terms,
		FilterCount: len(composed.FieldFilters),
	}
	for _, unresolved := range composed.Unresolved {
		result.Unresolved = append(result.Unresolved, unresolvedTermRow{
			Phrase: unresolved.Phrase,
			Reason: unresolved.Reason,
		})
	}

	return result, nil
}

// tableViewPath builds the link that opens the real table with the view
// applied, in the query parameters the data table already reads.
//
// The path is built here rather than by the model for the same reason the
// filters are: a fabricated link is indistinguishable from a real one until
// somebody clicks it.
func tableViewPath(
	resource filtercatalog.Resource,
	composed *tablequeryservice.ComposeResult,
) string {
	var b strings.Builder
	b.WriteString("/")
	b.WriteString(strings.ReplaceAll(resource.Entity, "_", "-"))

	params := make([]string, 0, 3)
	if composed.Query != "" {
		params = append(params, "query="+urlValue(composed.Query))
	}
	if len(composed.FieldFilters) > 0 {
		params = append(params, "fieldFilters="+urlValue(encodeJSON(composed.FieldFilters)))
	}
	if len(composed.Sort) > 0 {
		params = append(params, "sort="+urlValue(encodeJSON(composed.Sort)))
	}
	if len(params) > 0 {
		b.WriteString("?")
		b.WriteString(strings.Join(params, "&"))
	}

	return b.String()
}

func encodeJSON(value any) string {
	encoded, err := sonic.MarshalString(value)
	if err != nil {
		return ""
	}

	return encoded
}

func urlValue(raw string) string {
	return url.QueryEscape(raw)
}
