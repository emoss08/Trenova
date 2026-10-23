package formulatemplaterepository

import (
	"database/sql"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

func TestUpdateWritesClearedFields(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(new(sql.DB), pgdialect.New())

	entity := &formulatemplate.FormulaTemplate{
		ID:             pulid.MustNew("ft_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Name:           "Per Mile",
		Source:         "tariff \"Per Mile\"\n",
		Description:    "",
		ReviewComment:  "",
		TestCount:      0,
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
	assert.Contains(t, rendered, `"test_count" = 0`)
	assert.Contains(t, rendered, `"submitted_by_id" = NULL`)
	assert.Contains(t, rendered, `"submitted_at" = NULL`)
	assert.Contains(t, rendered, `"approved_by_id" = NULL`)
	assert.Contains(t, rendered, `"approved_at" = NULL`)
}

func TestUpdateWritesSetValues(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(new(sql.DB), pgdialect.New())

	submitter := pulid.MustNew("usr_")
	submittedAt := int64(1_700_000_000)

	entity := &formulatemplate.FormulaTemplate{
		ID:             pulid.MustNew("ft_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Name:           "Per Mile",
		Source:         "tariff \"Per Mile\"\n",
		Description:    "Prices by distance",
		TestCount:      3,
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
	assert.Contains(t, rendered, `"test_count" = 3`)
	assert.Contains(t, rendered, `"submitted_by_id" = '`+submitter.String()+`'`)
	assert.Contains(t, rendered, `"submitted_at" = 1700000000`)
}

func TestBulkDuplicateEntityCarriesFullContentAsDraft(t *testing.T) {
	t.Parallel()

	source := &formulatemplate.FormulaTemplate{
		ID:                   pulid.MustNew("ft_"),
		OrganizationID:       pulid.MustNew("org_"),
		BusinessUnitID:       pulid.MustNew("bu_"),
		Name:                 "Per Mile",
		Description:          "Prices by distance",
		Source:               "tariff \"Per Mile\"\n",
		Inputs:               []tarifftypes.InputDecl{{Name: "rate", Required: true}},
		Imports:              []tarifftypes.Import{{Path: "org.accessorials", VersionNumber: 2}},
		TestCount:            2,
		Status:               formulatemplate.StatusActive,
		Version:              7,
		CurrentVersionNumber: 4,
	}

	seed := formulatemplate.SeedFromTemplate(source)
	seed.Name = source.Name + " (Copy)"
	duplicate := seed.Build()

	assert.Equal(t, "Per Mile (Copy)", duplicate.Name)
	assert.Equal(t, source.Description, duplicate.Description)
	assert.Equal(t, source.Source, duplicate.Source)
	assert.Equal(t, source.Inputs, duplicate.Inputs)
	assert.Equal(t, source.Imports, duplicate.Imports)
	assert.EqualValues(t, 2, duplicate.TestCount)
	assert.Equal(t, formulatemplate.StatusDraft, duplicate.Status)
	assert.Equal(t, int64(0), duplicate.Version)
	assert.Equal(t, int64(1), duplicate.CurrentVersionNumber)
	require.NotNil(t, duplicate.SourceTemplateID)
	assert.Equal(t, source.ID, *duplicate.SourceTemplateID)
	require.NotNil(t, duplicate.SourceVersionNumber)
	assert.Equal(t, source.CurrentVersionNumber, *duplicate.SourceVersionNumber)
	assert.Nil(t, duplicate.SubmittedByID)
	assert.Nil(t, duplicate.ApprovedByID)
}

func TestFilterQueryRendersTypedFilters(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(new(sql.DB), pgdialect.New())
	repo := &repository{l: zap.NewNop()}

	var entities []*formulatemplate.FormulaTemplate
	req := &formulatemplateports.ListFormulaTemplatesRequest{
		Filter: &pagination.QueryOptions{Pagination: pagination.Info{Limit: 25}},
		Status: "Active",
	}

	query, err := repo.filterQuery(db.NewSelect().Model(&entities), req).
		AppendQuery(db.QueryGen(), nil)
	require.NoError(t, err)

	rendered := string(query)
	assert.Contains(t, rendered, `ft.status = 'Active'`)
	assert.Contains(t, rendered, `ORDER BY "ft"."created_at" DESC`)
	assert.Contains(t, rendered, `LIMIT 25`)
}
