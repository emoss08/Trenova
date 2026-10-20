package agentplanrepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUpdateStatus_KeepsTheApprovalWithTheWindowAndClearsItOnReopen(t *testing.T) {
	t.Parallel()

	conn, sqlMock := dbtest.NewSQLMock(t)
	repo := &repository{db: conn, l: zap.NewNop()}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	planID := pulid.MustNew("apl_")
	planRow := sqlmock.NewRows([]string{"id", "status"}).AddRow(planID.String(), "Approving")

	sqlMock.ExpectQuery(`UPDATE "agent_plans" AS "apl" SET status = 'Approving', updated_at = \d+, ` +
		`version = version \+ 1, commits_at = 5, approval_reason_code = 'looks right', ` +
		`approval_note = NULL, approval_preview_digest = 'abc', undone_at = NULL, ` +
		`undone_by_user_id = NULL WHERE`).
		WillReturnRows(planRow)
	_, err := repo.UpdateStatus(t.Context(), agentports.UpdateAgentPlanStatusRequest{
		ID:         planID,
		TenantInfo: tenant,
		Status:     agent.PlanStatusApproving,
		CommitsAt:  5,
		Approval:   agentports.PlanApproval{ReasonCode: "looks right", PreviewDigest: "abc"},
	})
	require.NoError(t, err)

	sqlMock.ExpectQuery(`UPDATE "agent_plans" AS "apl" SET status = 'Pending', updated_at = \d+, ` +
		`version = version \+ 1, commits_at = NULL, approval_reason_code = NULL, ` +
		`approval_note = NULL, approval_preview_digest = NULL, decided_by_user_id = NULL, ` +
		`decided_at = NULL WHERE`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow(planID.String(), "Pending"))
	_, err = repo.UpdateStatus(t.Context(), agentports.UpdateAgentPlanStatusRequest{
		ID:         planID,
		TenantInfo: tenant,
		Status:     agent.PlanStatusPending,
		Reopen:     true,
	})
	require.NoError(t, err)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestUpdateStatus_ADecisionWithoutAWindowKeepsItsNoteAndDigest(t *testing.T) {
	t.Parallel()

	conn, sqlMock := dbtest.NewSQLMock(t)
	repo := &repository{db: conn, l: zap.NewNop()}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	planID, deciderID := pulid.MustNew("apl_"), pulid.MustNew("usr_")

	sqlMock.ExpectQuery(`UPDATE "agent_plans" AS "apl" SET status = 'Rejected', updated_at = \d+, ` +
		`version = version \+ 1, decided_by_user_id = '` + deciderID.String() + `', ` +
		`decided_at = 9, approval_reason_code = NULL, approval_note = 'Not today', ` +
		`approval_preview_digest = NULL WHERE`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow(planID.String(), "Rejected"))
	_, err := repo.UpdateStatus(t.Context(), agentports.UpdateAgentPlanStatusRequest{
		ID:              planID,
		TenantInfo:      tenant,
		Status:          agent.PlanStatusRejected,
		DecidedByUserID: deciderID,
		DecidedAt:       9,
		Approval:        agentports.PlanApproval{Note: "Not today"},
	})
	require.NoError(t, err)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}
