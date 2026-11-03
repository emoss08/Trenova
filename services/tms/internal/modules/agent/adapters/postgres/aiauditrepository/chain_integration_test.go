//go:build integration

package aiauditrepository

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiaudit"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiauditports"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRecordVerificationWritesTheTenantsChainHead(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	var orgID, buID pulid.ID
	require.NoError(t, keel.Pool(db).QueryRow(ctx,
		"SELECT id, business_unit_id FROM organizations ORDER BY id LIMIT 1",
	).Scan(&orgID, &buID))
	_, err := keel.Pool(db).Exec(ctx,
		`INSERT INTO ai_audit_chain_heads (organization_id, business_unit_id, last_seq)
		VALUES ($1, $2, 7) ON CONFLICT (organization_id, business_unit_id) DO UPDATE SET last_seq = 7`,
		orgID, buID,
	)
	require.NoError(t, err)

	tenant := pagination.TenantInfo{OrgID: orgID, BuID: buID}
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())
	repo := New(Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()})

	head, err := repo.RecordVerification(ctx, &aiauditports.RecordAIAuditVerificationRequest{
		TenantInfo:  tenant,
		Status:      aiaudit.VerificationVerified,
		VerifiedSeq: 7,
		At:          1_800_000_000,
	})

	require.NoError(t, err)
	assert.Equal(t, orgID, head.OrganizationID)
	assert.Equal(t, aiaudit.VerificationVerified, head.LastVerificationStatus)
	assert.Equal(t, int64(7), head.LastVerifiedSeq)
}
