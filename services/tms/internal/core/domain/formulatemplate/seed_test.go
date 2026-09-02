package formulatemplate_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeedBuild_DefaultsEveryNewTemplateTheSameWay(t *testing.T) {
	t.Parallel()

	built := formulatemplate.Seed{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Name:           "Per Mile",
		Type:           formulatemplate.TemplateTypeFreightCharge,
		Expression:     "baseRate * totalDistance",
	}.Build()

	assert.Equal(t, formulatemplate.StatusDraft, built.Status, "new templates start as drafts")
	assert.Equal(t, "shipment", built.SchemaID, "the shipment schema is the default")
	assert.EqualValues(t, 1, built.CurrentVersionNumber)
	assert.NotNil(t, built.VariableDefinitions, "empty, never nil, so JSON says []")
	assert.NotNil(t, built.BreakdownDefinitions)
	assert.Equal(t, ratetypes.RoundingModeHalfUp, built.RoundingMode, "rounding is normalised")
	assert.EqualValues(t, formulatypes.DefaultRoundingPrecision, built.RoundingPrecision)
	assert.Nil(t, built.SourceTemplateID)
	assert.Nil(t, built.SubmittedByID, "a seed never carries review stamps")
}

func TestSeedFromTemplate_CopiesContentAndPointsAtTheSource(t *testing.T) {
	t.Parallel()

	sourceID := pulid.MustNew("ft_")
	approver := pulid.MustNew("usr_")
	source := &formulatemplate.FormulaTemplate{
		ID:                   sourceID,
		OrganizationID:       pulid.MustNew("org_"),
		BusinessUnitID:       pulid.MustNew("bu_"),
		Name:                 "Acme Lane",
		Description:          "Acme's lane rate",
		Type:                 formulatemplate.TemplateTypeFreightCharge,
		Expression:           "lookup('lane', laneCode)",
		Status:               formulatemplate.StatusActive,
		SchemaID:             "shipment",
		BreakdownDefinitions: []*formulatypes.BreakdownDefinition{{Name: "linehaul", Expression: "1"}},
		MinCharge:            decimal.NewNullDecimal(decimal.NewFromInt(250)),
		RoundingMode:         ratetypes.RoundingModeUp,
		RoundingPrecision:    0,
		Metadata:             map[string]any{"region": "SE"},
		ApprovedByID:         &approver,
		CurrentVersionNumber: 7,
	}

	seed := formulatemplate.SeedFromTemplate(source)
	seed.Name = "Acme Lane (Copy)"
	copied := seed.Build()

	assert.Equal(t, "Acme Lane (Copy)", copied.Name)
	assert.Equal(t, source.Expression, copied.Expression)
	assert.Equal(t, source.BreakdownDefinitions, copied.BreakdownDefinitions)
	assert.True(t, copied.MinCharge.Valid)
	assert.Equal(t, ratetypes.RoundingModeUp, copied.RoundingMode)
	require.NotNil(t, copied.SourceTemplateID)
	assert.Equal(t, sourceID, *copied.SourceTemplateID)
	require.NotNil(t, copied.SourceVersionNumber)
	assert.EqualValues(t, 7, *copied.SourceVersionNumber)
	assert.Equal(t, formulatemplate.StatusDraft, copied.Status, "a copy is never Active on arrival")
	assert.Nil(t, copied.ApprovedByID)
	assert.EqualValues(t, 1, copied.CurrentVersionNumber)

	copied.Metadata["region"] = "NE"
	assert.Equal(t, "SE", source.Metadata["region"], "metadata is copied, not shared")
}

func TestSeedFromVersion_UsesTheSnapshotAndNamesItsVersion(t *testing.T) {
	t.Parallel()

	sourceID := pulid.MustNew("ft_")
	snapshot := &formulatemplate.FormulaTemplateVersion{
		TemplateID:    sourceID,
		VersionNumber: 3,
		Description:   "as approved in March",
		Type:          formulatemplate.TemplateTypeAccessorialCharge,
		Expression:    "75",
		SchemaID:      "shipment",
		RoundingMode:  ratetypes.RoundingModeHalfEven,
	}

	seed := formulatemplate.SeedFromVersion(snapshot)
	seed.Name = "Fork of March"
	forked := seed.Build()

	assert.Equal(t, formulatemplate.TemplateTypeAccessorialCharge, forked.Type)
	assert.Equal(t, "75", forked.Expression)
	assert.Equal(t, ratetypes.RoundingModeHalfEven, forked.RoundingMode)
	require.NotNil(t, forked.SourceTemplateID)
	assert.Equal(t, sourceID, *forked.SourceTemplateID)
	require.NotNil(t, forked.SourceVersionNumber)
	assert.EqualValues(t, 3, *forked.SourceVersionNumber)
}
