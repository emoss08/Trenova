package report

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*Dashboard)(nil)
	_ validationframework.TenantedEntity = (*Dashboard)(nil)
)

const (
	DashboardGridColumns = 12
	MaxDashboardTiles    = 24
	MaxTileSpan          = DashboardGridColumns
	MaxTileHeight        = 12
)

type TileKind string

const (
	TileKindTable = TileKind("table")
	TileKindChart = TileKind("chart")
	TileKindKPI   = TileKind("kpi")
	TileKindText  = TileKind("text")
)

func (k TileKind) IsValid() bool {
	switch k {
	case TileKindTable, TileKindChart, TileKindKPI, TileKindText:
		return true
	default:
		return false
	}
}

// NeedsReport reports whether the tile draws from a saved or canned report;
// text tiles are notes on the canvas and query nothing.
func (k TileKind) NeedsReport() bool {
	switch k {
	case TileKindTable, TileKindChart, TileKindKPI:
		return true
	case TileKindText:
		return false
	default:
		return false
	}
}

// DashboardTile places one report on the canvas. A tile never carries its own
// definition: it points at a saved or canned report so the numbers on a
// dashboard are the same numbers that report exports.
type DashboardTile struct {
	ID           string   `json:"id"`
	Kind         TileKind `json:"kind"`
	Title        string   `json:"title,omitempty"`
	DefinitionID pulid.ID `json:"definitionId,omitempty"`
	CannedKey    string   `json:"cannedKey,omitempty"`
	// ChartID selects which of the report's charts this tile draws. Empty on a
	// chart tile means the report's first chart.
	ChartID string `json:"chartId,omitempty"`
	// ColumnID names the measure a KPI tile reads, as an output column id.
	ColumnID string `json:"columnId,omitempty"`
	Text     string `json:"text,omitempty"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	W        int    `json:"w"`
	H        int    `json:"h"`
	Limit    int    `json:"limit,omitempty"`
	// ParamBindings maps a report parameter name to the dashboard parameter
	// that feeds it. Names that match are bound automatically, so this only
	// carries the exceptions.
	ParamBindings map[string]string `json:"paramBindings,omitempty"`
}

func (t *DashboardTile) HasReport() bool {
	return !t.DefinitionID.IsNil() || t.CannedKey != ""
}

// DashboardFilter narrows every tile that queries the same entity, without the
// tile's report having to declare anything. It is the field-level counterpart
// to Parameters, which stay name-bound for values a report asks for directly
// (a lookback window has no field to point at).
type DashboardFilter struct {
	ID       string          `json:"id"`
	Label    string          `json:"label,omitempty"`
	Entity   string          `json:"entity"`
	Ref      FieldRef        `json:"ref"`
	Operator dbtype.Operator `json:"operator"`
	Default  any             `json:"default,omitempty"`
}

type DashboardLayout struct {
	Tiles      []DashboardTile   `json:"tiles"`
	Parameters []ParameterDef    `json:"parameters,omitempty"`
	Filters    []DashboardFilter `json:"filters,omitempty"`
}

func (l *DashboardLayout) Tile(id string) (*DashboardTile, bool) {
	for i := range l.Tiles {
		if l.Tiles[i].ID == id {
			return &l.Tiles[i], true
		}
	}
	return nil, false
}

type Dashboard struct {
	bun.BaseModel             `bun:"table:report_dashboards,alias:rdash" json:"-"`
	pagination.CursorValueSet `bun:",embed"                              json:"-"`

	ID             pulid.ID         `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID         `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID         `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	Name           string           `json:"name"           bun:"name,type:VARCHAR(255),notnull"`
	Description    string           `json:"description"    bun:"description,type:TEXT,nullzero"`
	Category       string           `json:"category"       bun:"category,type:VARCHAR(100),nullzero"`
	Tags           []string         `json:"tags"           bun:"tags,type:TEXT[],array,nullzero"`
	OwnerID        pulid.ID         `json:"ownerId"        bun:"owner_id,type:VARCHAR(100),notnull"`
	Visibility     Visibility       `json:"visibility"     bun:"visibility,type:VARCHAR(20),notnull,default:'private'"`
	Layout         *DashboardLayout `json:"layout"         bun:"layout,type:JSONB,notnull"`
	Version        int64            `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64            `json:"createdAt"      bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64            `json:"updatedAt"      bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Owner        *tenant.User         `json:"owner,omitempty"        bun:"rel:belongs-to,join:owner_id=id"`
}

func (d *Dashboard) Validate(multiErr *errortypes.MultiError) {
	if d.Name == "" {
		multiErr.Add("name", errortypes.ErrRequired, "Name is required")
	}
	if !d.Visibility.IsValid() {
		multiErr.Add("visibility", errortypes.ErrInvalid, "Visibility must be private or shared")
	}
	if d.Layout == nil {
		multiErr.Add("layout", errortypes.ErrRequired, "Layout is required")
		return
	}
	if len(d.Layout.Tiles) > MaxDashboardTiles {
		multiErr.Add("layout.tiles", errortypes.ErrInvalid,
			fmt.Sprintf("A dashboard holds at most %d tiles", MaxDashboardTiles))
		return
	}

	seen := make(map[string]bool, len(d.Layout.Tiles))
	for i := range d.Layout.Tiles {
		validateTile(multiErr, &d.Layout.Tiles[i], seen, fmt.Sprintf("layout.tiles[%d]", i))
	}
	validateDashboardParams(multiErr, d.Layout.Parameters)
	validateDashboardFilterShape(multiErr, d.Layout.Filters)
}

// validateDashboardFilterShape checks what the domain can see on its own; the
// service resolves the field against the catalog.
func validateDashboardFilterShape(
	multiErr *errortypes.MultiError,
	filters []DashboardFilter,
) {
	seen := make(map[string]bool, len(filters))
	for i := range filters {
		filter := &filters[i]
		fieldPath := fmt.Sprintf("layout.filters[%d]", i)

		switch {
		case filter.ID == "":
			multiErr.Add(fieldPath+".id", errortypes.ErrRequired, "Filter ID is required")
			continue
		case seen[filter.ID]:
			multiErr.Add(fieldPath+".id", errortypes.ErrInvalid, "Duplicate filter ID")
			continue
		}
		seen[filter.ID] = true

		if filter.Entity == "" {
			multiErr.Add(fieldPath+".entity", errortypes.ErrRequired,
				"Choose what this filter applies to")
		}
		if filter.Ref.Field == "" {
			multiErr.Add(fieldPath+".ref", errortypes.ErrRequired, "Choose a field to filter on")
		}
		if filter.Operator == "" {
			multiErr.Add(fieldPath+".operator", errortypes.ErrRequired,
				"Choose how the value is compared")
		}
	}
}

func validateTile(
	multiErr *errortypes.MultiError,
	tile *DashboardTile,
	seen map[string]bool,
	fieldPath string,
) {
	switch {
	case tile.ID == "":
		multiErr.Add(fieldPath+".id", errortypes.ErrRequired, "Tile ID is required")
		return
	case seen[tile.ID]:
		multiErr.Add(fieldPath+".id", errortypes.ErrInvalid, "Duplicate tile ID")
		return
	}
	seen[tile.ID] = true

	if !tile.Kind.IsValid() {
		multiErr.Add(fieldPath+".kind", errortypes.ErrInvalid,
			"Tile kind must be table, chart, kpi, or text")
		return
	}

	switch {
	case tile.Kind.NeedsReport() && !tile.HasReport():
		multiErr.Add(fieldPath, errortypes.ErrRequired, "Choose the report this tile shows")
	case tile.Kind.NeedsReport() && !tile.DefinitionID.IsNil() && tile.CannedKey != "":
		multiErr.Add(fieldPath, errortypes.ErrInvalid,
			"A tile reads from a saved report or a canned report, not both")
	case tile.Kind == TileKindText && tile.Text == "":
		multiErr.Add(fieldPath+".text", errortypes.ErrRequired, "Text tiles need content")
	case tile.Kind == TileKindKPI && tile.ColumnID == "":
		multiErr.Add(fieldPath+".columnId", errortypes.ErrRequired,
			"Choose the measure this KPI shows")
	}

	validateTileGeometry(multiErr, tile, fieldPath)
}

func validateTileGeometry(
	multiErr *errortypes.MultiError,
	tile *DashboardTile,
	fieldPath string,
) {
	if tile.X < 0 || tile.Y < 0 {
		multiErr.Add(fieldPath, errortypes.ErrInvalid, "Tile position cannot be negative")
		return
	}
	if tile.W < 1 || tile.W > MaxTileSpan {
		multiErr.Add(fieldPath+".w", errortypes.ErrInvalid,
			fmt.Sprintf("Tile width must be between 1 and %d columns", MaxTileSpan))
		return
	}
	if tile.X+tile.W > DashboardGridColumns {
		multiErr.Add(fieldPath+".x", errortypes.ErrInvalid,
			fmt.Sprintf("Tile runs past the %d-column grid", DashboardGridColumns))
		return
	}
	if tile.H < 1 || tile.H > MaxTileHeight {
		multiErr.Add(fieldPath+".h", errortypes.ErrInvalid,
			fmt.Sprintf("Tile height must be between 1 and %d rows", MaxTileHeight))
	}
}

func validateDashboardParams(multiErr *errortypes.MultiError, params []ParameterDef) {
	seen := make(map[string]bool, len(params))
	for i := range params {
		param := &params[i]
		fieldPath := fmt.Sprintf("layout.parameters[%d]", i)

		if param.Name == "" {
			multiErr.Add(fieldPath+".name", errortypes.ErrRequired, "Parameter name is required")
			continue
		}
		if seen[param.Name] {
			multiErr.Add(fieldPath+".name", errortypes.ErrInvalid, "Duplicate parameter name")
			continue
		}
		seen[param.Name] = true

		if !param.Type.IsValid() {
			multiErr.Add(fieldPath+".type", errortypes.ErrInvalid,
				fmt.Sprintf("Unknown parameter type %q", param.Type))
		}
	}
}

func (d *Dashboard) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if d.ID.IsNil() {
			d.ID = pulid.MustNew("rdb_")
		}
		d.CreatedAt = now
	case *bun.UpdateQuery:
		d.UpdatedAt = now
	}

	return nil
}

func (d *Dashboard) GetCreatedAt() int64 { return d.CreatedAt }

func (d *Dashboard) GetID() pulid.ID { return d.ID }

func (d *Dashboard) GetOrganizationID() pulid.ID { return d.OrganizationID }

func (d *Dashboard) GetBusinessUnitID() pulid.ID { return d.BusinessUnitID }

func (d *Dashboard) GetTableName() string { return "report_dashboards" }

func (d *Dashboard) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "rdash",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
			{Name: "description", Type: domaintypes.FieldTypeText},
			{Name: "category", Type: domaintypes.FieldTypeText},
		},
	}
}
