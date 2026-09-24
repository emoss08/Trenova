package agentquerytoolservice

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/internal/core/services/tablequeryservice"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/productguide"
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

func (t *composeTableViewTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceReport,
		effect:   agent.ToolEffectPresent,
	})
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
// The page comes from the product guide, which reads it from the app's own
// router: the page whose loader guards on reading the resource. The path used
// to be the entity's plural with a slash in front, which named no page at all
// ("/shipments", "/equipment types").
func tableViewPath(
	resource filtercatalog.Resource,
	composed *tablequeryservice.ComposeResult,
) string {
	page, ok := productguide.Default.PageForResource(resource.Resource.String())
	if !ok {
		return ""
	}

	var b strings.Builder
	b.WriteString(page.Path)

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

// listDashboardsTool exists because add_dashboard_tile needs a dashboard id
// and nothing else hands one out. Without it the only dashboard an agent can
// touch is one it made in the same turn.
type listDashboardsTool struct {
	dashboards dashboardLister
}

type dashboardLister interface {
	ListDashboards(
		ctx context.Context,
		req *reporting.ListDashboardsRequest,
	) ([]*report.Dashboard, error)
}

func newListDashboardsTool(dashboards *reporting.Service) serviceports.AgentQueryTool {
	return &listDashboardsTool{dashboards: dashboards}
}

func (t *listDashboardsTool) Name() string { return "list_dashboards" }

func (t *listDashboardsTool) Description() string {
	return "List the report dashboards under Reports, with the tiles on each. Use it to find " +
		"the dashboard somebody means before add_dashboard_tile, and to check whether a page " +
		"for this already exists before create_dashboard. Not the person's home page — " +
		"get_my_home_layout reads that."
}

func (t *listDashboardsTool) SearchTerms() []string {
	return []string{"report dashboard", "reports page", "board"}
}

func (t *listDashboardsTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": withPaging(map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Text matched against the dashboard's name, description and category.",
			},
		}, defaultDashboardRows, maxDashboardRows),
		"additionalProperties": false,
	}
}

func (t *listDashboardsTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceDashboard,
	})
}

const (
	defaultDashboardRows = 20
	maxDashboardRows     = 50
	// maxDashboardsScanned is how many dashboards one search reads. A search
	// narrows in memory, so it reads the organization's dashboards once
	// rather than a page at a time.
	maxDashboardsScanned = 500
)

type dashboardRow struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Category    string             `json:"category,omitempty"`
	Visibility  string             `json:"visibility"`
	Tiles       []dashboardTileRow `json:"tiles"`
}

// dashboardTileRow is what one tile shows, so "is revenue already on the ops
// page" is answered without opening it.
type dashboardTileRow struct {
	Kind         string `json:"kind"`
	Title        string `json:"title,omitempty"`
	DefinitionID string `json:"definitionId,omitempty"`
	CannedKey    string `json:"cannedKey,omitempty"`
}

func (t *listDashboardsTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	query := strings.ToLower(optionalString(params.Params, "query"))
	criteria := filtercatalog.NewCriteria("dashboards").At(clockFor(params))
	criteria.Text(query)

	dashboards, err := t.dashboards.ListDashboards(ctx, &reporting.ListDashboardsRequest{
		Request: reporting.Request{
			TenantInfo: pagination.TenantInfo{
				OrgID:  params.OrganizationID,
				BuID:   params.BusinessUnitID,
				UserID: params.Actor.UserID,
			},
		},
		Limit: maxDashboardsScanned,
	})
	if err != nil {
		return nil, err
	}

	rows := make([]dashboardRow, 0, len(dashboards))
	for _, dashboard := range dashboards {
		if query != "" && !matchesDashboard(dashboard, query) {
			continue
		}
		rows = append(rows, dashboardRow{
			ID:          dashboard.ID.String(),
			Name:        dashboard.Name,
			Description: dashboard.Description,
			Category:    dashboard.Category,
			Visibility:  string(dashboard.Visibility),
			Tiles:       tileRows(dashboard.Layout),
		})
	}

	window := readPage(params.Params, defaultDashboardRows, maxDashboardRows)
	shown, more := slicePage(window, rows)

	return searchResult(criteria, shown, len(shown)).paged(window, more), nil
}

func matchesDashboard(dashboard *report.Dashboard, needle string) bool {
	return strings.Contains(strings.ToLower(dashboard.Name), needle) ||
		strings.Contains(strings.ToLower(dashboard.Description), needle) ||
		strings.Contains(strings.ToLower(dashboard.Category), needle)
}

func tileRows(layout *report.DashboardLayout) []dashboardTileRow {
	if layout == nil {
		return []dashboardTileRow{}
	}

	rows := make([]dashboardTileRow, 0, len(layout.Tiles))
	for i := range layout.Tiles {
		tile := &layout.Tiles[i]
		row := dashboardTileRow{
			Kind:      string(tile.Kind),
			Title:     tile.Title,
			CannedKey: tile.CannedKey,
		}
		if !tile.DefinitionID.IsNil() {
			row.DefinitionID = tile.DefinitionID.String()
		}
		rows = append(rows, row)
	}

	return rows
}
