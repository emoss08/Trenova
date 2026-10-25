package formulatemplaterepository

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate/formulatemplatedb"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestStatusFilterComparesTheStatus(t *testing.T) {
	t.Parallel()

	pred, err := statusFilter("Active")
	require.NoError(t, err)

	ft := formulatemplatedb.FormulaTemplates
	sql, args := keeltest.Render(t, ft.Where(pred).Limit(25))

	assert.Contains(t, sql, `ft."status" = 'Active'`)
	assert.Equal(t, []any{"org_1", "bu_1"}, args)
}

func TestStatusFilterRefusesAnUnknownStatus(t *testing.T) {
	t.Parallel()

	_, err := statusFilter("Nope")
	require.Error(t, err)

	pred, err := statusFilter("")
	require.NoError(t, err)
	assert.Nil(t, pred)
}

func TestUsageQueryCountsEveryConsumerInOneStatement(t *testing.T) {
	t.Parallel()

	id := pulid.ID("ft_1")
	sql, args := keeltest.Render(t, usageQuery(id))

	assert.Contains(
		t,
		sql,
		`(SELECT count(*) FROM shipments sp WHERE sp.formula_template_id = ft."id" `+
			`AND sp.organization_id = ft."organization_id" AND sp.business_unit_id = ft."business_unit_id")`,
	)
	assert.Contains(t, sql, `FROM "rate_agreement_rules"`)
	assert.Contains(t, sql, `FROM "billing_controls"`)
	assert.Equal(t, 1, strings.Count(sql, `FROM "formula_templates" ft`))
	assert.Contains(t, args, "ft_1")
}
