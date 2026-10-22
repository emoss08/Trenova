package formulatemplate_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestSeedBuild_DefaultsEveryNewTemplateTheSameWay(t *testing.T) {
	t.Parallel()

	built := formulatemplate.Seed{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Name:           "Per Mile",
		Source:         "tariff \"Per Mile\"\n",
	}.Build()

	assert.Equal(t, formulatemplate.StatusDraft, built.Status, "new templates start as drafts")
	assert.EqualValues(t, 1, built.CurrentVersionNumber)
	assert.NotNil(t, built.Inputs, "empty, never nil, so JSON says []")
	assert.NotNil(t, built.Imports)
	assert.Nil(t, built.SourceTemplateID)
	assert.Nil(t, built.SubmittedByID, "a seed never carries review stamps")
}

func TestSeedFromTemplate_CopiesContentAndPointsAtTheSource(t *testing.T) {
	t.Parallel()

	submitter := pulid.MustNew("usr_")
	source := &formulatemplate.FormulaTemplate{
		ID:                   pulid.MustNew("ft_"),
		OrganizationID:       pulid.MustNew("org_"),
		BusinessUnitID:       pulid.MustNew("bu_"),
		Name:                 "Per Mile",
		Source:               "tariff \"Per Mile\"\n",
		Inputs:               []tarifftypes.InputDecl{{Name: "rate", Required: true}},
		Imports:              []tarifftypes.Import{{Path: "org.accessorials", VersionNumber: 2}},
		TestCount:            3,
		Status:               formulatemplate.StatusActive,
		SubmittedByID:        &submitter,
		CurrentVersionNumber: 4,
		Metadata:             map[string]any{"owner": "pricing"},
	}

	built := formulatemplate.SeedFromTemplate(source).Build()
	assert.Equal(t, source.Source, built.Source)
	assert.Equal(t, source.Inputs, built.Inputs)
	assert.Equal(t, source.Imports, built.Imports)
	assert.EqualValues(t, 3, built.TestCount)
	assert.Equal(t, formulatemplate.StatusDraft, built.Status, "a copy is a draft")
	assert.Nil(t, built.SubmittedByID)
	assert.Equal(t, source.ID, *built.SourceTemplateID)
	assert.EqualValues(t, 4, *built.SourceVersionNumber)

	built.Metadata["owner"] = "billing"
	built.Imports[0].VersionNumber = 9
	assert.Equal(t, "pricing", source.Metadata["owner"], "the copy owns its metadata")
	assert.EqualValues(t, 2, source.Imports[0].VersionNumber, "the copy owns its imports")
}

func TestSeedFromVersion_UsesTheSnapshotAndNamesItsVersion(t *testing.T) {
	t.Parallel()

	version := &formulatemplate.FormulaTemplateVersion{
		TemplateID:    pulid.MustNew("ft_"),
		VersionNumber: 2,
		Name:          "Per Mile",
		Source:        "tariff \"Per Mile v2\"\n",
	}

	built := formulatemplate.SeedFromVersion(version).Build()
	assert.Equal(t, version.Source, built.Source)
	assert.Equal(t, version.TemplateID, *built.SourceTemplateID)
	assert.EqualValues(t, 2, *built.SourceVersionNumber)
}
