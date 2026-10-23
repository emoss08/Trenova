//go:build integration

package formulalibraryversionrepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/formulalibraryrepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/formulalibraryversionrepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/formulatemplaterepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/formulatemplateversionrepository"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulalibraryports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type libraryFixture struct {
	ctx       context.Context
	libraries formulalibraryports.FormulaLibraryRepository
	versions  formulalibraryports.FormulaLibraryVersionRepository
	templates formulatemplateports.FormulaTemplateRepository
	tversions formulatemplateports.FormulaTemplateVersionRepository
	tenant    pagination.TenantInfo
	other     pagination.TenantInfo
}

func setup(t *testing.T) *libraryFixture {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	other := seedtest.SeedAdditionalTenant(t, ctx, db, "LB")
	conn := postgres.NewTestConnection(db)

	return &libraryFixture{
		ctx: ctx,
		libraries: formulalibraryrepository.New(formulalibraryrepository.Params{
			DB: conn, Logger: zap.NewNop(),
		}),
		versions: formulalibraryversionrepository.New(formulalibraryversionrepository.Params{
			DB: conn, Logger: zap.NewNop(),
		}),
		templates: formulatemplaterepository.New(formulatemplaterepository.Params{
			DB: conn, Logger: zap.NewNop(),
		}),
		tversions: formulatemplateversionrepository.New(formulatemplateversionrepository.Params{
			DB: conn, Logger: zap.NewNop(),
		}),
		tenant: pagination.TenantInfo{
			OrgID:  data.Organization.ID,
			BuID:   data.BusinessUnit.ID,
			UserID: data.User.ID,
		},
		other: pagination.TenantInfo{
			OrgID:  other.Organization.ID,
			BuID:   other.BusinessUnit.ID,
			UserID: other.User.ID,
		},
	}
}

func (f *libraryFixture) library(
	t *testing.T,
	tenant pagination.TenantInfo,
	name string,
	status formulatemplate.Status,
) *formulalibrary.FormulaLibrary {
	t.Helper()

	created, err := f.libraries.Create(f.ctx, &formulalibrary.FormulaLibrary{
		OrganizationID:       tenant.OrgID,
		BusinessUnitID:       tenant.BuID,
		Name:                 name,
		Source:               "library \"org." + name + "\"\n",
		Status:               status,
		CurrentVersionNumber: 1,
	})
	require.NoError(t, err)
	return created
}

func (f *libraryFixture) version(
	t *testing.T,
	library *formulalibrary.FormulaLibrary,
	number int64,
	status formulatemplate.Status,
	source string,
) *formulalibrary.FormulaLibraryVersion {
	t.Helper()

	library.Source = source
	library.Status = status
	created, err := f.versions.Create(
		f.ctx,
		formulalibrary.NewVersionFromLibrary(library, number, f.userOf(library).UserID, "", nil),
	)
	require.NoError(t, err)
	return created
}

func (f *libraryFixture) userOf(library *formulalibrary.FormulaLibrary) pagination.TenantInfo {
	if library.OrganizationID == f.other.OrgID {
		return f.other
	}
	return f.tenant
}

func TestLibraryNamesAreUniquePerTenantIgnoringCase(t *testing.T) {
	f := setup(t)

	f.library(t, f.tenant, "accessorials", formulatemplate.StatusDraft)
	_, err := f.libraries.Create(f.ctx, &formulalibrary.FormulaLibrary{
		OrganizationID: f.tenant.OrgID,
		BusinessUnitID: f.tenant.BuID,
		Name:           "Accessorials",
		Source:         "library \"x\"\n",
		Status:         formulatemplate.StatusDraft,
	})
	require.Error(t, err)

	f.library(t, f.other, "accessorials", formulatemplate.StatusDraft)
}

func TestLatestApprovedByNamesPicksTheNewestApprovedVersionOfLiveLibraries(t *testing.T) {
	f := setup(t)

	accessorials := f.library(t, f.tenant, "accessorials", formulatemplate.StatusDraft)
	f.version(t, accessorials, 1, formulatemplate.StatusDraft, "library \"v1 draft\"\n")
	f.version(t, accessorials, 2, formulatemplate.StatusActive, "library \"v2 approved\"\n")
	f.version(t, accessorials, 3, formulatemplate.StatusDraft, "library \"v3 draft\"\n")

	archived := f.library(t, f.tenant, "archived", formulatemplate.StatusInactive)
	f.version(t, archived, 1, formulatemplate.StatusActive, "library \"gone\"\n")

	theirs := f.library(t, f.other, "zones", formulatemplate.StatusActive)
	f.version(t, theirs, 1, formulatemplate.StatusActive, "library \"theirs\"\n")

	found, err := f.versions.LatestApprovedByNames(f.ctx, &formulalibraryports.LatestApprovedByNamesRequest{
		TenantInfo: f.tenant,
		Names:      []string{"ACCESSORIALS", "archived", "zones"},
	})
	require.NoError(t, err)
	require.Len(t, found, 1, "archived libraries and other tenants' libraries are not importable")
	assert.EqualValues(t, 2, found[0].VersionNumber)
	assert.Equal(t, "library \"v2 approved\"\n", found[0].Source)

	listed, err := f.versions.ListLatestApproved(f.ctx, &formulalibraryports.ListApprovedLibrariesRequest{
		TenantInfo: f.tenant,
	})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, accessorials.ID, listed[0].LibraryID)
}

func TestGetByPinsLoadsExactlyThePinnedVersionsInOneQuery(t *testing.T) {
	f := setup(t)

	accessorials := f.library(t, f.tenant, "accessorials", formulatemplate.StatusActive)
	v1 := f.version(t, accessorials, 1, formulatemplate.StatusActive, "library \"v1\"\n")
	f.version(t, accessorials, 2, formulatemplate.StatusActive, "library \"v2\"\n")
	money := f.library(t, f.tenant, "money", formulatemplate.StatusActive)
	m1 := f.version(t, money, 1, formulatemplate.StatusActive, "library \"m1\"\n")

	pinned, err := f.versions.GetByPins(f.ctx, &formulalibraryports.GetVersionsByPinsRequest{
		TenantInfo: f.tenant,
		Pins:       []tarifftypes.Import{v1.Pin(), m1.Pin()},
	})
	require.NoError(t, err)
	require.Len(t, pinned, 2)
	sources := []string{pinned[0].Source, pinned[1].Source}
	assert.ElementsMatch(t, []string{"library \"v1\"\n", "library \"m1\"\n"}, sources,
		"a newer approved version never replaces the pinned one")

	theirs, err := f.versions.GetByPins(f.ctx, &formulalibraryports.GetVersionsByPinsRequest{
		TenantInfo: f.other,
		Pins:       []tarifftypes.Import{v1.Pin()},
	})
	require.NoError(t, err)
	assert.Empty(t, theirs)
}

func TestTemplateVersionImportsAnswerWhoUsesALibrary(t *testing.T) {
	f := setup(t)

	accessorials := f.library(t, f.tenant, "accessorials", formulatemplate.StatusActive)
	v1 := f.version(t, accessorials, 1, formulatemplate.StatusActive, "library \"v1\"\n")
	v2 := f.version(t, accessorials, 2, formulatemplate.StatusActive, "library \"v2\"\n")

	template, err := f.templates.Create(f.ctx, formulatemplate.Seed{
		OrganizationID: f.tenant.OrgID,
		BusinessUnitID: f.tenant.BuID,
		Name:           "Dry van",
		Source:         "tariff \"Dry van\"\nuse org.accessorials\n",
		Imports:        []tarifftypes.Import{v1.Pin()},
		Status:         formulatemplate.StatusActive,
	}.Build())
	require.NoError(t, err)

	first := formulatemplate.NewVersionFromTemplate(template, 1, f.tenant.UserID, "", nil)
	first.Status = formulatemplate.StatusActive
	_, err = f.tversions.Create(f.ctx, first)
	require.NoError(t, err)

	template.Imports = []tarifftypes.Import{v2.Pin()}
	second := formulatemplate.NewVersionFromTemplate(template, 2, f.tenant.UserID, "", nil)
	_, err = f.tversions.Create(f.ctx, second)
	require.NoError(t, err)

	usages, err := f.tversions.ListLibraryUsages(f.ctx, &formulatemplateports.ListLibraryUsagesRequest{
		TenantInfo: f.tenant,
		LibraryID:  accessorials.ID,
	})
	require.NoError(t, err)
	require.Len(t, usages, 2)
	assert.EqualValues(t, 2, usages[0].LibraryVersionNumber, "newest library version first")
	assert.EqualValues(t, 2, usages[0].TemplateVersion)
	assert.Equal(t, "Dry van", usages[0].TemplateName)
	assert.EqualValues(t, 1, usages[1].LibraryVersionNumber)
	assert.Equal(t, formulatemplate.StatusActive, usages[1].VersionStatus)

	pinnedToV1 := int64(1)
	onlyV1, err := f.tversions.ListLibraryUsages(f.ctx, &formulatemplateports.ListLibraryUsagesRequest{
		TenantInfo:    f.tenant,
		LibraryID:     accessorials.ID,
		VersionNumber: &pinnedToV1,
	})
	require.NoError(t, err)
	require.Len(t, onlyV1, 1)
	assert.EqualValues(t, 1, onlyV1[0].TemplateVersion)

	theirs, err := f.tversions.ListLibraryUsages(f.ctx, &formulatemplateports.ListLibraryUsagesRequest{
		TenantInfo: f.other,
		LibraryID:  accessorials.ID,
	})
	require.NoError(t, err)
	assert.Empty(t, theirs)
}
