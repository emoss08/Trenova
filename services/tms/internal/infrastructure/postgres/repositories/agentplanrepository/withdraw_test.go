package agentplanrepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestExpirePendingByDefinitionExpiresOnlyThePendingPlansOfTheAgentsRuns(t *testing.T) {
	t.Parallel()

	conn, sqlMock := dbtest.NewSQLMock(t)
	repo := &repository{db: conn, l: zap.NewNop()}
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	agentID := pulid.MustNew("agdef_")

	sqlMock.ExpectExec(
		`UPDATE "agent_plans" AS "apl" SET status = 'Expired', updated_at = \d+, version = version \+ 1 ` +
			`WHERE \(\(apl.organization_id = '` + tenantInfo.OrgID.String() +
			`'\) AND \(apl.business_unit_id = '` + tenantInfo.BuID.String() +
			`'\) AND \(apl.status = 'Pending'\) AND \(apl.run_id IN \(SELECT "ar"."id" FROM "agent_runs" AS "ar" ` +
			`WHERE \(ar.organization_id = '` + tenantInfo.OrgID.String() +
			`'\) AND \(ar.business_unit_id = '` + tenantInfo.BuID.String() +
			`'\) AND \(ar.agent_definition_id = '` + agentID.String() + `'\)\)\)\)`,
	).WillReturnResult(sqlmock.NewResult(0, 1))

	expired, err := repo.ExpirePendingByDefinition(
		t.Context(),
		repositories.ExpireAgentPlansByDefinitionRequest{
			AgentDefinitionID: agentID,
			TenantInfo:        tenantInfo,
		},
	)

	require.NoError(t, err)
	require.Equal(t, 1, expired)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}
