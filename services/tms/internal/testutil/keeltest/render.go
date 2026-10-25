package keeltest

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbpool"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

const (
	unreachableDSN = "postgres://test:test@127.0.0.1:1/none"

	TestOrganizationID = pulid.ID("org_1")
	TestBusinessUnitID = pulid.ID("bu_1")
)

type Renderer interface {
	ToSQL(ctx context.Context, db *keel.DB) (string, []any, error)
}

func RenderDB(t testing.TB) *keel.DB {
	t.Helper()

	cfg := dbpool.KeelConfig(nil, nil)
	cfg.DSN = unreachableDSN
	db, err := keel.Open(t.Context(), cfg)
	require.NoError(t, err)
	t.Cleanup(db.Close)

	return db
}

func TenantContext(t testing.TB) context.Context {
	t.Helper()

	return dbscope.WithTenant(t.Context(), dbscope.Tenant{
		OrganizationID: TestOrganizationID,
		BusinessUnitID: TestBusinessUnitID,
	})
}

func Render(t testing.TB, q Renderer) (string, []any) {
	t.Helper()

	return RenderIn(t, TenantContext(t), q)
}

func RenderIn(t testing.TB, ctx context.Context, q Renderer) (string, []any) {
	t.Helper()

	sql, args, err := q.ToSQL(ctx, RenderDB(t))
	require.NoError(t, err)

	return sql, args
}
