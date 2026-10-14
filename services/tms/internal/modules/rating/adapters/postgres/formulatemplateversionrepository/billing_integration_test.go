//go:build integration

package formulatemplateversionrepository

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/formulatemplaterepository"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type versionFixture struct {
	ctx        context.Context
	repo       formulatemplateports.FormulaTemplateVersionRepository
	template   *formulatemplate.FormulaTemplate
	tenantInfo pagination.TenantInfo
	author     pulid.ID
}

func setupVersions(t *testing.T) *versionFixture {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenantInfo := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	conn := postgres.NewTestConnection(db)

	templates := formulatemplaterepository.New(formulatemplaterepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})
	template, err := templates.Create(ctx, formulatemplate.Seed{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Name:           "Windows",
		Type:           formulatemplate.TemplateTypeFreightCharge,
		Expression:     "totalDistance * 2",
		SchemaID:       "shipment",
		Status:         formulatemplate.StatusActive,
	}.Build())
	require.NoError(t, err)

	return &versionFixture{
		ctx:        ctx,
		repo:       New(Params{DB: conn, Logger: zap.NewNop()}),
		template:   template,
		tenantInfo: tenantInfo,
		author:     data.User.ID,
	}
}

func (f *versionFixture) snapshot(
	t *testing.T,
	number int64,
	status formulatemplate.Status,
	effectiveFrom *int64,
	author pulid.ID,
) {
	t.Helper()

	version := formulatemplate.NewVersionFromTemplate(f.template, number, author, "", nil)
	version.Status = status
	version.EffectiveFrom = effectiveFrom
	_, err := f.repo.Create(f.ctx, version)
	require.NoError(t, err)
}

func (f *versionFixture) request() *formulatemplateports.ListScheduledVersionsRequest {
	return &formulatemplateports.ListScheduledVersionsRequest{
		TenantInfo: f.tenantInfo,
		TemplateID: f.template.ID,
	}
}

func numbers(versions []*formulatemplate.FormulaTemplateVersion) []int64 {
	out := make([]int64, 0, len(versions))
	for _, version := range versions {
		out = append(out, version.VersionNumber)
	}
	return out
}

func TestApprovedSnapshotsScheduleAndClearOnlyPendingWindows(t *testing.T) {
	f := setupVersions(t)

	past := timeutils.NowUnix() - 30*86_400
	future := timeutils.NowUnix() + 30*86_400
	f.snapshot(t, 1, formulatemplate.StatusDraft, nil, f.author)
	f.snapshot(t, 2, formulatemplate.StatusActive, &past, f.author)
	f.snapshot(t, 3, formulatemplate.StatusDraft, nil, f.author)
	f.snapshot(t, 4, formulatemplate.StatusActive, &future, f.author)

	approved, err := f.repo.ListApproved(f.ctx, f.request())
	require.NoError(t, err)
	assert.Equal(t, []int64{2, 4}, numbers(approved))

	scheduled, err := f.repo.ListScheduled(f.ctx, f.request())
	require.NoError(t, err)
	assert.Equal(t, []int64{4}, numbers(scheduled), "only windows that have not started")

	cleared, err := f.repo.ClearScheduled(f.ctx, f.request())
	require.NoError(t, err)
	assert.EqualValues(t, 1, cleared)

	approved, err = f.repo.ListApproved(f.ctx, f.request())
	require.NoError(t, err)
	assert.Equal(t, []int64{2}, numbers(approved), "the window in force keeps billing")
}

func TestListAuthorIDsSinceNamesEveryoneWhoWroteAfterTheBase(t *testing.T) {
	f := setupVersions(t)

	other := pulid.MustNew("usr_")
	f.snapshot(t, 1, formulatemplate.StatusActive, nil, f.author)
	f.snapshot(t, 2, formulatemplate.StatusDraft, nil, other)
	f.snapshot(t, 3, formulatemplate.StatusDraft, nil, other)

	authors, err := f.repo.ListAuthorIDsSince(f.ctx, &formulatemplateports.ListVersionAuthorsRequest{
		TenantInfo:   f.tenantInfo,
		TemplateID:   f.template.ID,
		AfterVersion: 1,
	})
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{other}, authors)
}
