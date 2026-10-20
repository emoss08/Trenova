package agentproposalrepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestListByPlansReadsTheStepsOfEveryPlanInStepOrder(t *testing.T) {
	t.Parallel()

	conn, sqlMock := dbtest.NewSQLMock(t)
	repo := &repository{db: conn, l: zap.NewNop()}
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	first, second := pulid.MustNew("apl_"), pulid.MustNew("apl_")
	stepOne, stepTwo := pulid.MustNew("aprop_"), pulid.MustNew("aprop_")

	sqlMock.ExpectQuery(
		`SELECT .* FROM "agent_proposals" AS "ap" WHERE \(\(ap.organization_id = '` +
			tenantInfo.OrgID.String() + `'\) AND \(ap.business_unit_id = '` +
			tenantInfo.BuID.String() + `'\) AND \(ap.plan_id IN \('` + first.String() +
			`', '` + second.String() + `'\)\)\) ORDER BY ap.plan_step ASC, ` +
			`"ap"."created_at" ASC, "ap"."id" ASC`,
	).WillReturnRows(
		sqlmock.NewRows([]string{"id", "plan_id", "plan_step"}).
			AddRow(stepOne.String(), first.String(), 1).
			AddRow(stepTwo.String(), second.String(), 1),
	)

	steps, err := repo.ListByPlans(t.Context(), agentports.ListAgentProposalsByPlansRequest{
		PlanIDs:    []pulid.ID{first, second},
		TenantInfo: tenantInfo,
	})

	require.NoError(t, err)
	require.Len(t, steps, 2)
	assert.Equal(t, stepOne, steps[0].ID)
	assert.Equal(t, first, *steps[0].PlanID)
	assert.Equal(t, second, *steps[1].PlanID)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestListByPlansAsksNothingForNoPlans(t *testing.T) {
	t.Parallel()

	conn, sqlMock := dbtest.NewSQLMock(t)
	repo := &repository{db: conn, l: zap.NewNop()}

	steps, err := repo.ListByPlans(t.Context(), agentports.ListAgentProposalsByPlansRequest{})

	require.NoError(t, err)
	assert.Empty(t, steps)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}
