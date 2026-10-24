//go:build integration

package accountingsyncrepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/accounting/accountingtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type candidateShipment struct {
	ID         pulid.ID
	CustomerID pulid.ID
	ProNumber  string
	BOL        string
}

type seededTenant struct {
	db     *accountingtest.Database
	ctx    context.Context
	kdb    *keel.DB
	conn   *postgres.Connection
	tenant pagination.TenantInfo
	userID pulid.ID
}

func newSeededTenant(t *testing.T) *seededTenant {
	t.Helper()

	db := accountingtest.NewDevelopmentDatabase(t)
	return &seededTenant{
		db:     db,
		ctx:    db.A.Ctx,
		kdb:    db.Admin,
		conn:   db.Conn,
		tenant: pagination.TenantInfo{OrgID: db.A.Info.OrgID, BuID: db.A.Info.BuID},
		userID: db.A.Info.UserID,
	}
}

func seededRows[T any](t *testing.T, s *seededTenant, sql string, args ...any) []T {
	t.Helper()

	rows, err := s.kdb.Pool().Query(s.ctx, sql, args...)
	require.NoError(t, err)
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[T])
	require.NoError(t, err)

	return out
}

func (s *seededTenant) shipments(t *testing.T, n int) []candidateShipment {
	t.Helper()

	shipments := seededRows[candidateShipment](t, s,
		`SELECT id, customer_id, COALESCE(pro_number, ''), COALESCE(bol, '')
		FROM shipments
		WHERE organization_id = $1 AND business_unit_id = $2
		ORDER BY id
		LIMIT $3`,
		s.tenant.OrgID.String(),
		s.tenant.BuID.String(),
		n,
	)
	require.Len(t, shipments, n)

	return shipments
}

type seededParty struct {
	ID   pulid.ID
	Name string
}

func (s *seededTenant) party(t *testing.T, sql string) seededParty {
	t.Helper()

	parties := seededRows[seededParty](t, s, sql, s.tenant.OrgID.String(), s.tenant.BuID.String())
	require.NotEmpty(t, parties)

	return parties[0]
}

func (s *seededTenant) carrier(t *testing.T) seededParty {
	t.Helper()

	return s.party(t, `SELECT id, name FROM carriers
		WHERE organization_id = $1 AND business_unit_id = $2 ORDER BY id LIMIT 1`)
}

func (s *seededTenant) customer(t *testing.T) seededParty {
	t.Helper()

	return s.party(t, `SELECT id, name FROM customers
		WHERE organization_id = $1 AND business_unit_id = $2 ORDER BY id LIMIT 1`)
}

func (s *seededTenant) worker(t *testing.T) seededParty {
	t.Helper()

	return s.party(t, `SELECT id, TRIM(CONCAT(first_name, ' ', last_name)) FROM workers
		WHERE organization_id = $1 AND business_unit_id = $2 ORDER BY id LIMIT 1`)
}
