package formulatemplaterepository

import (
	"database/sql"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

// TestUpdateWritesClearedFields guards the OmitZero interaction: nil approval
// stamps, an invalid NullDecimal guardrail, and blanked text are all zero
// values, so without the explicit Value clauses the UPDATE would silently keep
// the old submitter, approver, guardrails, and review comment — a rejected
// template would still show who submitted it, and a removed minimum charge
// would keep clamping every rate.
func TestUpdateWritesClearedFields(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(new(sql.DB), pgdialect.New())

	entity := &formulatemplate.FormulaTemplate{
		ID:             pulid.MustNew("ft_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Name:           "Per Mile",
		Expression:     "baseRate * totalDistance",
		Description:    "",
		ReviewComment:  "",
		MinCharge:      decimal.NullDecimal{},
		MaxCharge:      decimal.NullDecimal{},
		SubmittedByID:  nil,
		SubmittedAt:    nil,
		ApprovedByID:   nil,
		ApprovedAt:     nil,
	}

	base := db.NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", 1).
		OmitZero()

	query, err := applyClearableColumns(base, entity).AppendQuery(db.QueryGen(), nil)
	require.NoError(t, err)

	rendered := string(query)
	assert.Contains(t, rendered, `"description" = ''`)
	assert.Contains(t, rendered, `"review_comment" = ''`)
	assert.Contains(t, rendered, `"min_charge" = NULL`)
	assert.Contains(t, rendered, `"max_charge" = NULL`)
	assert.Contains(t, rendered, `"submitted_by_id" = NULL`)
	assert.Contains(t, rendered, `"submitted_at" = NULL`)
	assert.Contains(t, rendered, `"approved_by_id" = NULL`)
	assert.Contains(t, rendered, `"approved_at" = NULL`)
}

// TestUpdateWritesSetGuardrails is the other half of the clearing contract: a
// valid guardrail must render as its value, not be flattened to NULL by the
// same clauses that allow clearing.
func TestUpdateWritesSetGuardrails(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(new(sql.DB), pgdialect.New())

	submitter := pulid.MustNew("usr_")
	submittedAt := int64(1_700_000_000)

	entity := &formulatemplate.FormulaTemplate{
		ID:             pulid.MustNew("ft_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Name:           "Per Mile",
		Expression:     "baseRate * totalDistance",
		Description:    "Prices by distance",
		MinCharge:      decimal.NullDecimal{Decimal: decimal.NewFromInt(250), Valid: true},
		MaxCharge:      decimal.NullDecimal{Decimal: decimal.NewFromInt(5000), Valid: true},
		SubmittedByID:  &submitter,
		SubmittedAt:    &submittedAt,
	}

	base := db.NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", 1).
		OmitZero()

	query, err := applyClearableColumns(base, entity).AppendQuery(db.QueryGen(), nil)
	require.NoError(t, err)

	rendered := string(query)
	assert.Contains(t, rendered, `"description" = 'Prices by distance'`)
	assert.Contains(t, rendered, `"min_charge" = '250'`)
	assert.Contains(t, rendered, `"max_charge" = '5000'`)
	assert.Contains(t, rendered, `"submitted_by_id" = '`+submitter.String()+`'`)
	assert.Contains(t, rendered, `"submitted_at" = 1700000000`)
}

// TestBulkDuplicateEntityCarriesFullContentAsDraft pins the duplicate policy:
// every content field is copied, the copy lands in Draft with a fresh
// optimistic lock and version counter, approval fields stay empty, and the
// lineage back to the source is recorded.
func TestBulkDuplicateEntityCarriesFullContentAsDraft(t *testing.T) {
	t.Parallel()

	source := &formulatemplate.FormulaTemplate{
		ID:                   pulid.MustNew("ft_"),
		OrganizationID:       pulid.MustNew("org_"),
		BusinessUnitID:       pulid.MustNew("bu_"),
		Name:                 "Per Mile",
		Description:          "Prices by distance",
		Type:                 formulatemplate.TemplateTypeFreightCharge,
		Expression:           "baseRate * totalDistance",
		Status:               formulatemplate.StatusActive,
		SchemaID:             "shipment",
		MinCharge:            decimal.NullDecimal{Decimal: decimal.NewFromInt(250), Valid: true},
		MaxCharge:            decimal.NullDecimal{Decimal: decimal.NewFromInt(5000), Valid: true},
		Version:              7,
		CurrentVersionNumber: 4,
	}

	seed := formulatemplate.SeedFromTemplate(source)
	seed.Name = source.Name + " (Copy)"
	duplicate := seed.Build()

	assert.Equal(t, "Per Mile (Copy)", duplicate.Name)
	assert.Equal(t, source.Description, duplicate.Description)
	assert.Equal(t, source.Expression, duplicate.Expression)
	assert.Equal(t, formulatemplate.StatusDraft, duplicate.Status)
	assert.Equal(t, source.MinCharge, duplicate.MinCharge)
	assert.Equal(t, source.MaxCharge, duplicate.MaxCharge)
	assert.Equal(t, int64(0), duplicate.Version)
	assert.Equal(t, int64(1), duplicate.CurrentVersionNumber)
	require.NotNil(t, duplicate.SourceTemplateID)
	assert.Equal(t, source.ID, *duplicate.SourceTemplateID)
	require.NotNil(t, duplicate.SourceVersionNumber)
	assert.Equal(t, source.CurrentVersionNumber, *duplicate.SourceVersionNumber)
	assert.Nil(t, duplicate.SubmittedByID)
	assert.Nil(t, duplicate.ApprovedByID)
}

func TestNextAvailableName(t *testing.T) {
	t.Parallel()

	taken := map[string]struct{}{
		"Per Mile (Copy)":   {},
		"Per Mile (Copy) 2": {},
	}

	assert.Equal(t, "Flat (Copy)", nextAvailableName(taken, "Flat (Copy)"))
	assert.Equal(t, "Per Mile (Copy) 3", nextAvailableName(taken, "Per Mile (Copy)"))
}

// TestFilterQueryRendersTypedFilters pins the SQL the list endpoint issues, so
// moving the filters onto the generated column helpers cannot change a table
// alias, a column name, or the newest-first order the client relies on.
func TestFilterQueryRendersTypedFilters(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(new(sql.DB), pgdialect.New())
	repo := &repository{l: zap.NewNop()}

	var entities []*formulatemplate.FormulaTemplate
	req := &repositories.ListFormulaTemplatesRequest{
		Filter: &pagination.QueryOptions{Pagination: pagination.Info{Limit: 25}},
		Type:   "FreightCharge",
		Status: "Active",
	}

	query, err := repo.filterQuery(db.NewSelect().Model(&entities), req).
		AppendQuery(db.QueryGen(), nil)
	require.NoError(t, err)

	rendered := string(query)
	assert.Contains(t, rendered, `ft.type = 'FreightCharge'`)
	assert.Contains(t, rendered, `ft.status = 'Active'`)
	assert.Contains(t, rendered, `ORDER BY "ft"."created_at" DESC`)
	assert.Contains(t, rendered, `LIMIT 25`)
}
