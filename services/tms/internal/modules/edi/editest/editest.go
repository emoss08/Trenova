//go:build integration

package editest

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

type Tenant struct {
	Ctx  context.Context
	Info pagination.TenantInfo
}

func FirstTenant(t *testing.T, ctx context.Context, db *keel.DB) Tenant {
	t.Helper()

	var orgID, buID string
	require.NoError(t, db.Pool().QueryRow(ctx,
		"SELECT id, business_unit_id FROM organizations ORDER BY id LIMIT 1",
	).Scan(&orgID, &buID))

	return TenantOf(ctx, pagination.TenantInfo{OrgID: pulid.ID(orgID), BuID: pulid.ID(buID)})
}

func TenantOf(ctx context.Context, info pagination.TenantInfo) Tenant {
	return Tenant{Ctx: dbscope.WithTenant(ctx, info.DBTenant()), Info: info}
}

func Exec(t *testing.T, ctx context.Context, db *keel.DB, sql string, args ...any) {
	t.Helper()

	_, err := db.Pool().Exec(ctx, sql, args...)
	require.NoError(t, err)
}
