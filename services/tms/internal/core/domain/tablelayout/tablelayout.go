package tablelayout

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tableconfiguration"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	MaxResourceLength = 100
	MaxColumnIDLength = 100
	MaxColumns        = 200
	MaxFormatRules    = 50
	MaxViewIDLength   = 100
	MinColumnWidth    = 24
	MaxColumnWidth    = 2000
	MaxLayoutsPerUser = 500
	MaxPinnedRows     = 100
	MaxRowIDLength    = 100

	DensityComfortable = "comfortable"
	DensityCompact     = "compact"
)

var resourcePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]*$`)

var (
	_ bun.BeforeAppendModelHook          = (*TableLayout)(nil)
	_ validationframework.TenantedEntity = (*TableLayout)(nil)
)

type Layout struct {
	ColumnVisibility          map[string]bool                   `json:"columnVisibility"`
	ColumnOrder               []string                          `json:"columnOrder"`
	ColumnSizing              map[string]float64                `json:"columnSizing"`
	ColumnPinning             *tableconfiguration.ColumnPinning `json:"columnPinning"`
	Density                   string                            `json:"density"`
	FormatRules               []tableconfiguration.FormatRule   `json:"formatRules"`
	ActiveViewID              string                            `json:"activeViewId"`
	PinnedRowIDs              []string                          `json:"pinnedRowIds"`
	PinnedRowsCollapsed       bool                              `json:"pinnedRowsCollapsed"`
	LastSeenAt                int64                             `json:"lastSeenAt"`
	HideChangesSinceLastVisit bool                              `json:"hideChangesSinceLastVisit"`
	HideTotals                bool                              `json:"hideTotals"`
	Virtualized               bool                              `json:"virtualized"`
}

func (l *Layout) Validate(multiErr *errortypes.MultiError) {
	l.validateColumns(multiErr)
	l.validateColumnPinning(multiErr)
	l.validateRows(multiErr)
}

func (l *Layout) validateColumns(multiErr *errortypes.MultiError) {
	if len(l.ColumnVisibility) > MaxColumns {
		multiErr.Add(
			"layout.columnVisibility",
			errortypes.ErrInvalid,
			fmt.Sprintf("A layout can describe at most %d columns", MaxColumns),
		)
	}
	for id := range l.ColumnVisibility {
		validateColumnID(multiErr, "layout.columnVisibility", id)
	}

	if len(l.ColumnOrder) > MaxColumns {
		multiErr.Add(
			"layout.columnOrder",
			errortypes.ErrInvalid,
			fmt.Sprintf("A layout can order at most %d columns", MaxColumns),
		)
	}
	seen := make(map[string]struct{}, len(l.ColumnOrder))
	for idx, id := range l.ColumnOrder {
		field := fmt.Sprintf("layout.columnOrder[%d]", idx)
		validateColumnID(multiErr, field, id)
		if _, dup := seen[id]; dup {
			multiErr.Add(field, errortypes.ErrDuplicate, "A column can appear in the order once")
		}
		seen[id] = struct{}{}
	}

	if len(l.ColumnSizing) > MaxColumns {
		multiErr.Add(
			"layout.columnSizing",
			errortypes.ErrInvalid,
			fmt.Sprintf("A layout can size at most %d columns", MaxColumns),
		)
	}
	for id, width := range l.ColumnSizing {
		validateColumnID(multiErr, "layout.columnSizing", id)
		if width < MinColumnWidth || width > MaxColumnWidth {
			multiErr.Add(
				"layout.columnSizing",
				errortypes.ErrInvalid,
				fmt.Sprintf(
					"Column widths must be between %d and %d pixels",
					MinColumnWidth,
					MaxColumnWidth,
				),
			)
		}
	}
}

func (l *Layout) validateColumnPinning(multiErr *errortypes.MultiError) {
	if l.ColumnPinning != nil {
		if len(l.ColumnPinning.Left)+len(l.ColumnPinning.Right) > MaxColumns {
			multiErr.Add(
				"layout.columnPinning",
				errortypes.ErrInvalid,
				fmt.Sprintf("A layout can pin at most %d columns", MaxColumns),
			)
		}
		pinned := make(map[string]struct{}, len(l.ColumnPinning.Left)+len(l.ColumnPinning.Right))
		for _, side := range [][]string{l.ColumnPinning.Left, l.ColumnPinning.Right} {
			for _, id := range side {
				validateColumnID(multiErr, "layout.columnPinning", id)
				if _, dup := pinned[id]; dup {
					multiErr.Add(
						"layout.columnPinning",
						errortypes.ErrDuplicate,
						"A column can be pinned to one side only",
					)
				}
				pinned[id] = struct{}{}
			}
		}
	}
}

func (l *Layout) validateRows(multiErr *errortypes.MultiError) {
	if l.Density != "" && l.Density != DensityComfortable && l.Density != DensityCompact {
		multiErr.Add(
			"layout.density",
			errortypes.ErrInvalid,
			"Density must be comfortable or compact",
		)
	}

	if len(l.FormatRules) > MaxFormatRules {
		multiErr.Add(
			"layout.formatRules",
			errortypes.ErrInvalid,
			fmt.Sprintf("A layout can hold at most %d format rules", MaxFormatRules),
		)
	}

	if len(l.PinnedRowIDs) > MaxPinnedRows {
		multiErr.Add(
			"layout.pinnedRowIds",
			errortypes.ErrInvalid,
			fmt.Sprintf("A table can keep at most %d pinned rows", MaxPinnedRows),
		)
	}
	pinnedRows := make(map[string]struct{}, len(l.PinnedRowIDs))
	for idx, id := range l.PinnedRowIDs {
		field := fmt.Sprintf("layout.pinnedRowIds[%d]", idx)
		if strings.TrimSpace(id) == "" || len(id) > MaxRowIDLength {
			multiErr.Add(
				field,
				errortypes.ErrInvalid,
				fmt.Sprintf("Row IDs must be 1 to %d characters", MaxRowIDLength),
			)
		}
		if _, dup := pinnedRows[id]; dup {
			multiErr.Add(field, errortypes.ErrDuplicate, "A row can be pinned once")
		}
		pinnedRows[id] = struct{}{}
	}

	if l.LastSeenAt < 0 {
		multiErr.Add(
			"layout.lastSeenAt",
			errortypes.ErrInvalid,
			"The last visit cannot be negative",
		)
	}

	if len(l.ActiveViewID) > MaxViewIDLength {
		multiErr.Add("layout.activeViewId", errortypes.ErrInvalid, "The active view ID is too long")
	}
}

func validateColumnID(multiErr *errortypes.MultiError, field, id string) {
	if strings.TrimSpace(id) == "" || len(id) > MaxColumnIDLength {
		multiErr.Add(
			field,
			errortypes.ErrInvalid,
			fmt.Sprintf("Column IDs must be 1 to %d characters", MaxColumnIDLength),
		)
	}
}

type TableLayout struct {
	bun.BaseModel `bun:"table:table_layouts,alias:tl" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	UserID         pulid.ID `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	Resource       string   `json:"resource"       bun:"resource,type:VARCHAR(100),notnull"`
	Layout         *Layout  `json:"layout"         bun:"layout,type:JSONB,notnull"`
	Version        int64    `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	User         *tenant.User         `json:"user,omitempty"         bun:"rel:belongs-to,join:user_id=id"`
}

func (tl *TableLayout) Validate(multiErr *errortypes.MultiError) {
	tl.ValidateKey(multiErr)
	multiErr.AddOzzoError(validation.ValidateStruct(tl,
		validation.Field(&tl.Layout, validation.NotNil.Error("Layout is required")),
	))

	if tl.Layout != nil {
		tl.Layout.Validate(multiErr)
	}
}

func (tl *TableLayout) ValidateKey(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(tl,
		validation.Field(&tl.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&tl.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&tl.UserID, validation.Required.Error("User is required")),
		validation.Field(&tl.Resource,
			validation.Required.Error("Resource is required"),
			validation.Length(1, MaxResourceLength),
			validation.Match(resourcePattern).Error(
				"Resource may hold letters, digits, dots, colons, dashes and underscores only",
			),
		),
	))
}

func (tl *TableLayout) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if tl.ID.IsNil() {
			tl.ID = pulid.MustNew("tl_")
		}
		tl.CreatedAt = now
		tl.UpdatedAt = now
	case *bun.UpdateQuery:
		tl.UpdatedAt = now
	}

	return nil
}

func (tl *TableLayout) GetID() pulid.ID {
	return tl.ID
}

func (tl *TableLayout) GetOrganizationID() pulid.ID {
	return tl.OrganizationID
}

func (tl *TableLayout) GetBusinessUnitID() pulid.ID {
	return tl.BusinessUnitID
}

func (tl *TableLayout) GetTableName() string {
	return "table_layouts"
}
