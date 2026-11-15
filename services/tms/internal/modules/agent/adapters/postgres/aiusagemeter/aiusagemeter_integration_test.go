//go:build integration

package aiusagemeter_test

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/adapters/postgres/aiusagemeter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage/aiusagedb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type organizationRow struct {
	ID             pulid.ID
	BusinessUnitID pulid.ID
}

func TestPagesAreSummedFromTheSourceRowsAndAResentHourAnswersTheSame(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	o := tenantdb.Organizations
	org, err := keel.IntoOne[organizationRow](dbscope.WithSystem(ctx, "test"), db,
		o.AllTenants().Pick(o.ID, o.BusinessUnitID).Limit(1))
	require.NoError(t, err)
	tenantCtx := dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
	})

	record := func(feature aiusage.Feature, succeeded bool, pages int) *aiusage.AIUsageRecord {
		return &aiusage.AIUsageRecord{
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			ProviderKind:   aiprovider.KindAnthropicMessages,
			Model:          "claude-sonnet-5-5",
			Task:           aitask.TaskDocumentExtraction,
			Surface:        aiusage.SurfaceBackground,
			Feature:        feature,
			Succeeded:      succeeded,
			Pages:          pages,
		}
	}

	counter := aiusagemeter.New(aiusagemeter.Params{DB: postgres.WrapDB(db)}).Pages
	now := timeutils.NowUnix()
	current := now - now%3600
	req := &managedusageports.HourCountRequest{Since: current - 3600, Until: current + 2*3600}
	total := func() int64 {
		t.Helper()
		counts, countErr := counter.CountByHour(ctx, req)
		require.NoError(t, countErr)
		var sum int64
		for _, c := range counts {
			sum += c.Quantity
		}
		return sum
	}

	before := total()

	for _, r := range []*aiusage.AIUsageRecord{
		record(aiusage.FeatureDocumentIntelligenceExtract, true, 3),
		record(aiusage.FeatureDocumentIntelligenceExtract, true, 5),
		record(aiusage.FeatureDocumentIntelligenceExtract, false, 9),
		record(aiusage.FeatureDocumentIntelligenceRoute, true, 4),
	} {
		require.NoError(t, aiusagedb.AIUsageRecords.Create(tenantCtx, db, r))
	}

	first := total()
	assert.Equal(t, before+8, first, "only successful extractions count, by their pages")
	assert.Equal(t, first, total(), "reading the same hours again gives the same answer")
}
