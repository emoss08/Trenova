//go:build integration

package reportrepository

import (
	"testing"

	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestListSchedulesShowsOwnSchedulesAndThoseOnOwnedReports(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	data := seedtest.SeedFullTestData(t, ctx, db)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	colleague := seedtest.NewUser(data.Organization.ID, data.BusinessUnit.ID).
		WithUsername("schedule_colleague").
		WithEmail("schedule_colleague@example.com").
		WithPassword("password123").
		Build(t, ctx, tx)
	stateB := seedtest.NewState().WithName("Texas").WithAbbreviation("TX").Build(t, ctx, tx)
	buB := seedtest.NewBusinessUnit().WithName("Schedules B BU").WithCode("SCHB").Build(t, ctx, tx)
	orgB := seedtest.NewOrganization(buB.ID, stateB.ID).
		WithName("Schedules Org B").
		WithScacCode("SCHB").
		WithBucketName("schedules-bucket-b").
		Build(t, ctx, tx)
	userB := seedtest.NewUser(orgB.ID, buB.ID).
		WithUsername("schedules_b_user").
		WithEmail("schedules_b@example.com").
		WithPassword("password123").
		Build(t, ctx, tx)
	require.NoError(t, tx.Commit())

	conn := postgres.NewTestConnection(db)
	definitions := NewDefinitionRepository(DefinitionParams{DB: conn, Logger: zap.NewNop()})
	schedules := NewScheduleRepository(ScheduleParams{DB: conn, Logger: zap.NewNop()})

	owner := pagination.TenantInfo{
		OrgID:  data.Organization.ID,
		BuID:   data.BusinessUnit.ID,
		UserID: data.User.ID,
	}
	peer := pagination.TenantInfo{OrgID: owner.OrgID, BuID: owner.BuID, UserID: colleague.ID}
	other := pagination.TenantInfo{OrgID: orgB.ID, BuID: buB.ID, UserID: userB.ID}

	createDefinition := func(tenant pagination.TenantInfo, name string) *report.ReportDefinition {
		created, createErr := definitions.Create(ctx, &report.ReportDefinition{
			BusinessUnitID: tenant.BuID,
			OrganizationID: tenant.OrgID,
			Name:           name,
			Kind:           report.DefinitionKindCustom,
			OwnerID:        tenant.UserID,
			Visibility:     report.VisibilityShared,
			Status:         report.DefinitionStatusActive,
			CatalogVersion: "integration-catalog",
			Definition: &ir.Definition{
				Version: ir.Version,
				Entity:  "shipment",
				Columns: []ir.Column{{ID: "pro", Dimension: &ir.Dimension{Field: "proNumber"}}},
			},
			DefaultFormat: report.FormatCSV,
		}, tenant.UserID)
		require.NoError(t, createErr)
		return created
	}
	createSchedule := func(
		tenant pagination.TenantInfo,
		definitionID pulid.ID,
	) *report.ReportSchedule {
		created, createErr := schedules.Create(ctx, &report.ReportSchedule{
			ID:             pulid.MustNew("rsch_"),
			BusinessUnitID: tenant.BuID,
			OrganizationID: tenant.OrgID,
			DefinitionID:   definitionID,
			CronExpression: "0 8 * * 1",
			Timezone:       "UTC",
			Formats:        []string{string(report.FormatCSV)},
			Delivery: &report.ScheduleDelivery{
				EmailRecipients: []string{"ops@example.com"},
			},
			Enabled: true,
			RunAsID: tenant.UserID,
		})
		require.NoError(t, createErr)
		return created
	}

	ownersReport := createDefinition(owner, "Owner's report")
	peersReport := createDefinition(peer, "Colleague's report")
	otherReport := createDefinition(other, "Other tenant's report")

	peerOnOwnersReport := createSchedule(peer, ownersReport.ID)
	peerOnPeersReport := createSchedule(peer, peersReport.ID)
	ownerOnPeersReport := createSchedule(owner, peersReport.ID)
	otherOnOtherReport := createSchedule(other, otherReport.ID)

	list := func(tenant pagination.TenantInfo, definitionID pulid.ID) []pulid.ID {
		listed, listErr := schedules.List(ctx, &reportports.ListReportSchedulesRequest{
			TenantInfo:   tenant,
			VisibleTo:    tenant.UserID,
			DefinitionID: definitionID,
			Limit:        100,
		})
		require.NoError(t, listErr)
		ids := make([]pulid.ID, 0, len(listed))
		for _, schedule := range listed {
			ids = append(ids, schedule.ID)
		}
		return ids
	}

	assert.ElementsMatch(t,
		[]pulid.ID{peerOnOwnersReport.ID, ownerOnPeersReport.ID},
		list(owner, pulid.Nil),
		"the owner sees their own schedule and every schedule on the report they own")
	assert.ElementsMatch(t,
		[]pulid.ID{ownerOnPeersReport.ID},
		list(owner, peersReport.ID),
		"on a colleague's shared report, the owner sees only their own schedule")
	assert.ElementsMatch(t,
		[]pulid.ID{peerOnOwnersReport.ID, peerOnPeersReport.ID, ownerOnPeersReport.ID},
		list(peer, pulid.Nil),
		"the colleague sees their own schedules and every schedule on their own report")
	assert.ElementsMatch(t,
		[]pulid.ID{otherOnOtherReport.ID},
		list(other, pulid.Nil),
		"another organization sees only its own schedules")
}
