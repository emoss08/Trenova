//go:build integration

package formulatemplaterepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/bootstrap/crossref"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate/formulatemplatedb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest/ratingdb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fixture struct {
	db   *ratingdb.Database
	repo formulatemplateports.FormulaTemplateRepository
	a    ratingdb.Tenant
	b    ratingdb.Tenant
	ctxA context.Context
	ctxB context.Context
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	db := ratingdb.NewDatabase(t)

	return &fixture{
		db: db,
		repo: New(Params{
			DB:        db.Conn,
			Logger:    zap.NewNop(),
			Consumers: crossref.TemplateConsumers(),
		}),
		a:    db.A,
		b:    db.B,
		ctxA: db.A.Ctx,
		ctxB: db.B.Ctx,
	}
}

func createTemplate(
	t *testing.T,
	ctx context.Context,
	repo formulatemplateports.FormulaTemplateRepository,
	tenantInfo pagination.TenantInfo,
	name string,
	status formulatemplate.Status,
) *formulatemplate.FormulaTemplate {
	t.Helper()

	created, err := repo.Create(ctx, formulatemplate.Seed{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Name:           name,
		Source:         "tariff \"" + name + "\"\n\ncharge \"Flat\" = $10\n",
		Status:         status,
	}.Build())
	require.NoError(t, err)
	return created
}

func TestCountUsagesCountsTheBillingFallback(t *testing.T) {
	f := newFixture(t)

	template := createTemplate(
		t,
		f.ctxA,
		f.repo,
		f.a.Info,
		"Fallback",
		formulatemplate.StatusActive,
	)
	bc := tenantdb.BillingControls
	updated, err := bc.Where(keel.True()).
		Update(f.ctxA, f.db.Admin, bc.FallbackFormulaTemplateID.To(template.ID))
	require.NoError(t, err)
	if updated == 0 {
		control := billingcontrol.NewDefaultBillingControl(f.a.Info.OrgID, f.a.Info.BuID)
		control.FallbackFormulaTemplateID = &template.ID
		require.NoError(t, bc.Create(f.ctxA, f.db.Admin, control))
	}

	usage, err := f.repo.CountUsages(f.ctxA, &formulatemplateports.GetTemplateUsageRequest{
		TemplateID: template.ID,
		TenantInfo: f.a.Info,
	})
	require.NoError(t, err)
	assert.True(t, usage.InUse, "the organization's fallback rates every uncovered shipment")
	assert.Contains(t, usage.Usages, formulatemplateports.TemplateUsageCount{
		Type:  "billing_control_fallback",
		Count: 1,
	})

	stats, err := f.repo.CountStatsByIDs(
		f.ctxA,
		&formulatemplateports.GetFormulaTemplateStatsRequest{
			TenantInfo:  f.a.Info,
			TemplateIDs: []pulid.ID{template.ID},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, 0, stats[template.ID].UsageCount, "the fallback is not a shipment or a rule")
}

func TestCountUsagesOfAnUnknownTemplateIsEmpty(t *testing.T) {
	f := newFixture(t)

	usage, err := f.repo.CountUsages(f.ctxA, &formulatemplateports.GetTemplateUsageRequest{
		TemplateID: pulid.MustNew("ft_"),
		TenantInfo: f.a.Info,
	})
	require.NoError(t, err)
	assert.False(t, usage.InUse)
	assert.Empty(t, usage.Usages)
}

func TestBulkUpdateStatusRefusesATemplateThatLeftActiveAndBumpsTheVersion(t *testing.T) {
	f := newFixture(t)

	active := createTemplate(t, f.ctxA, f.repo, f.a.Info, "Active", formulatemplate.StatusActive)
	inReview := createTemplate(
		t,
		f.ctxA,
		f.repo,
		f.a.Info,
		"In review",
		formulatemplate.StatusInReview,
	)

	_, err := f.repo.BulkUpdateStatus(
		f.ctxA,
		&formulatemplateports.BulkUpdateFormulaTemplateStatusRequest{
			TenantInfo:  f.a.Info,
			TemplateIDs: []pulid.ID{active.ID, inReview.ID},
			Status:      formulatemplate.StatusInactive,
		},
	)
	require.Error(t, err, "a template that went into review after the check is not archived")

	updated, err := f.repo.BulkUpdateStatus(
		f.ctxA,
		&formulatemplateports.BulkUpdateFormulaTemplateStatusRequest{
			TenantInfo:  f.a.Info,
			TemplateIDs: []pulid.ID{active.ID},
			Status:      formulatemplate.StatusInactive,
		},
	)
	require.NoError(t, err)
	require.Len(t, updated, 1)
	assert.Equal(t, formulatemplate.StatusInactive, updated[0].Status)
	assert.Equal(t, active.Version+1, updated[0].Version)

	stored, err := formulatemplatedb.FormulaTemplates.Get(f.ctxA, f.db.Admin, inReview.ID)
	require.NoError(t, err)
	assert.Equal(t, formulatemplate.StatusInReview, stored.Status)
}

func TestUpdateChecksTheVersionAndTheName(t *testing.T) {
	f := newFixture(t)

	first := createTemplate(t, f.ctxA, f.repo, f.a.Info, "First", formulatemplate.StatusDraft)
	second := createTemplate(t, f.ctxA, f.repo, f.a.Info, "Second", formulatemplate.StatusDraft)

	stale := *first
	first.Description = "Edited"
	saved, err := f.repo.Update(f.ctxA, first)
	require.NoError(t, err)
	assert.Equal(t, stale.Version+1, saved.Version)

	stale.Description = "Lost"
	_, err = f.repo.Update(f.ctxA, &stale)
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err), "a stale write is a version conflict")

	second.Name = "First"
	_, err = f.repo.Update(f.ctxA, second)
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err), "a taken name is a validation error")
}

func TestReadsStayInTheTenant(t *testing.T) {
	f := newFixture(t)

	template := createTemplate(t, f.ctxA, f.repo, f.a.Info, "Mine", formulatemplate.StatusActive)

	_, err := f.repo.GetByID(f.ctxB, formulatemplateports.GetFormulaTemplateByIDRequest{
		TemplateID: template.ID,
		TenantInfo: f.b.Info,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
	assert.False(t, dberror.IsTenantBoundaryViolation(err))

	byNames, err := f.repo.FindByNames(
		f.ctxB,
		formulatemplateports.GetFormulaTemplatesByNamesRequest{
			TenantInfo: f.b.Info,
			Names:      []string{"Mine"},
		},
	)
	require.NoError(t, err)
	assert.Empty(t, byNames)

	taken, err := f.repo.TakenNames(f.ctxA, &formulatemplateports.ListTakenNamesRequest{
		TenantInfo: f.a.Info,
		Bases:      []string{"Mi"},
	})
	require.NoError(t, err)
	assert.Contains(t, taken, "Mine")

	cursor, err := pagination.NewCursorInfo(10, "")
	require.NoError(t, err)
	page, err := f.repo.ListConnection(
		f.ctxB,
		&formulatemplateports.ListFormulaTemplateConnectionRequest{
			Filter: &pagination.QueryOptions{TenantInfo: f.b.Info},
			Cursor: cursor,
		},
	)
	require.NoError(t, err)
	assert.Empty(t, page.Items)
	require.NotNil(t, page.TotalCount)
	assert.Zero(t, *page.TotalCount)
}

func TestListConnectionPagesThroughEveryTemplate(t *testing.T) {
	f := newFixture(t)

	for _, name := range []string{"One", "Two", "Three"} {
		createTemplate(t, f.ctxA, f.repo, f.a.Info, name, formulatemplate.StatusActive)
	}

	seen := map[pulid.ID]struct{}{}
	cursor, err := pagination.NewCursorInfo(2, "")
	require.NoError(t, err)
	for range 3 {
		var page *pagination.CursorListResult[*formulatemplate.FormulaTemplate]
		page, err = f.repo.ListConnection(
			f.ctxA,
			&formulatemplateports.ListFormulaTemplateConnectionRequest{
				Filter: &pagination.QueryOptions{TenantInfo: f.a.Info},
				Cursor: cursor,
			},
		)
		require.NoError(t, err)
		for _, item := range page.Items {
			seen[item.ID] = struct{}{}
		}
		if !page.HasNextPage {
			break
		}
		cursor, err = pagination.NewCursorInfo(2, page.EdgeCursors[len(page.EdgeCursors)-1])
		require.NoError(t, err)
	}

	assert.GreaterOrEqual(t, len(seen), 3)
}
