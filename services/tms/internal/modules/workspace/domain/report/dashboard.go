package report

import (
	"context"
	"fmt"

	"github.com/emoss08/keel/analytics/ir"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
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

func (t *DashboardTile) Params(values map[string]any) ir.Params {
	return ir.Params{Values: values, Bindings: t.ParamBindings}
}

func FilterValues(filters []ir.DashboardFilter, values map[string]any) []ir.FilterValue {
	out := make([]ir.FilterValue, 0, len(filters))
	for i := range filters {
		out = append(out, ir.FilterValue{Filter: filters[i], Value: values[filters[i].ID]})
	}
	return out
}

type DashboardLayout struct {
	Tiles      []DashboardTile      `json:"tiles"`
	Parameters []ir.Parameter       `json:"parameters,omitempty"`
	Filters    []ir.DashboardFilter `json:"filters,omitempty"`
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
	keel.Table `keel:"report_dashboards"`

	ID             pulid.ID         `json:"id"             keel:"id,pk,autoid:rdb_"`
	BusinessUnitID pulid.ID         `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID         `json:"organizationId" keel:"organization_id,pk,tenant"`
	Name           string           `json:"name"           keel:"name"`
	Description    string           `json:"description"    keel:"description,nullzero"`
	Category       string           `json:"category"       keel:"category,nullzero"`
	Tags           []string         `json:"tags"           keel:"tags,nullzero"`
	OwnerID        pulid.ID         `json:"ownerId"        keel:"owner_id"`
	Visibility     Visibility       `json:"visibility"     keel:"visibility"`
	Layout         *DashboardLayout `json:"layout"         keel:"layout,jsonb"`
	Version        int64            `json:"version"        keel:"version,version"`
	CreatedAt      int64            `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64            `json:"updatedAt"      keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	Owner        *tenant.User         `json:"owner,omitempty"        keel:"belongs:owner_id"`
}

func (d *Dashboard) BeforeCreate(_ context.Context) error {
	if d.Visibility == "" {
		d.Visibility = VisibilityPrivate
	}

	return nil
}

func (d *Dashboard) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(d,
		validation.Field(&d.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&d.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&d.OwnerID, validation.Required.Error("Owner is required")),
		validation.Field(&d.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxDefinitionNameLength).
				Error("Name cannot be longer than 255 characters"),
		),
		validation.Field(&d.Category,
			validation.Length(0, maxDefinitionCategoryLength).
				Error("Category cannot be longer than 100 characters"),
		),
		validation.Field(&d.Visibility,
			validation.Required.Error("Visibility must be private or shared"),
			domainvalidation.ValidEnum[Visibility]("Visibility must be private or shared"),
		),
		validation.Field(&d.Layout, validation.Required.Error("Layout is required")),
	))

	if d.Layout == nil {
		return
	}

	if len(d.Layout.Tiles) > MaxDashboardTiles {
		multiErr.Add("layout.tiles", errortypes.ErrInvalid,
			"A dashboard holds at most {0} tiles", MaxDashboardTiles)
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
	filters []ir.DashboardFilter,
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
		if filter.Field == "" {
			multiErr.Add(fieldPath+".field", errortypes.ErrRequired, "Choose a field to filter on")
		}
		if filter.Op == "" {
			multiErr.Add(fieldPath+".op", errortypes.ErrRequired,
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
			"Tile width must be between 1 and {0} columns", MaxTileSpan)
		return
	}
	if tile.X+tile.W > DashboardGridColumns {
		multiErr.Add(fieldPath+".x", errortypes.ErrInvalid,
			"Tile runs past the {0}-column grid", DashboardGridColumns)
		return
	}
	if tile.H < 1 || tile.H > MaxTileHeight {
		multiErr.Add(fieldPath+".h", errortypes.ErrInvalid,
			"Tile height must be between 1 and {0} rows", MaxTileHeight)
	}
}

func validateDashboardParams(multiErr *errortypes.MultiError, params []ir.Parameter) {
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

		if !param.Type.Valid() {
			multiErr.Add(fieldPath+".type", errortypes.ErrInvalid,
				"Unknown parameter type \"{0}\"", param.Type)
		}
	}
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
