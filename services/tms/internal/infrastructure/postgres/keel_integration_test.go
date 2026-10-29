//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/config/databaseconfig"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/internal/testutil/keeltest/keeltestdb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errRollBack = errors.New("roll back")

func (f *rlsFixture) codes(t *testing.T, tenant dbscope.Tenant) []string {
	t.Helper()

	codes, err := queryStrings(t.Context(), f.admin,
		"SELECT code FROM fleet_codes WHERE organization_id = $1 ORDER BY code",
		tenant.OrganizationID.String(),
	)
	require.NoError(t, err)

	return codes
}

func keelCodes(ctx context.Context, db *keel.DB) ([]string, error) {
	return keel.QueryAll(ctx, db, pgx.RowTo[string], "SELECT code FROM fleet_codes ORDER BY code")
}

const insertFleetCodeSQL = "INSERT INTO fleet_codes (id, code, organization_id, business_unit_id, manager_id) " +
	"VALUES ($1, $2, $3, $4, $5)"

func rawInsertFleetCode(
	ctx context.Context,
	conn *Connection,
	tenant dbscope.Tenant,
	code string,
) error {
	_, err := keel.Exec(ctx, conn.Keel(ctx), insertFleetCodeSQL,
		pulid.MustNew("fc_").String(), code, tenant.OrganizationID.String(),
		tenant.BusinessUnitID.String(), tenant.UserID.String(),
	)

	return err
}

func keelFleetCode(tenant dbscope.Tenant, code string) *keeltest.FleetCode {
	return &keeltest.FleetCode{ManagerID: tenant.UserID.String(), Code: code, Status: "Active"}
}

func TestKeel_RawAndTypedWritesInOneDbtxWriteCommitTogether(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)
	ctx := dbscope.WithTenant(t.Context(), f.tenantA)

	created, err := dbtx.Write(ctx, conn, func(ctx context.Context) (*keeltest.FleetCode, error) {
		if err := rawInsertFleetCode(ctx, conn, f.tenantA, "RAW"); err != nil {
			return nil, err
		}

		row := keelFleetCode(f.tenantA, "KEEL")
		if err := keeltestdb.FleetCodes.Create(ctx, conn.Keel(ctx), row); err != nil {
			return nil, err
		}

		seen, err := keelCodes(ctx, conn.Keel(ctx))
		if err != nil {
			return nil, err
		}
		assert.Equal(
			t,
			[]string{"FCA", "KEEL", "RAW"},
			seen,
			"a typed read sees the raw write of its transaction",
		)

		return row, nil
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"FCA", "KEEL", "RAW"}, f.codes(t, f.tenantA))
	assert.Regexp(t, `^fc_[0-9A-Z]{26}$`, created.ID)
	assert.Equal(t, f.tenantA.OrganizationID.String(), created.OrganizationID)
	assert.Equal(t, f.tenantA.BusinessUnitID.String(), created.BusinessUnitID)
	assert.Positive(t, created.CreatedAt)
	assert.Equal(t, created.CreatedAt, created.UpdatedAt)
}

func TestKeel_RawAndTypedWritesInOneDbtxWriteRollBackTogether(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)
	ctx := dbscope.WithTenant(t.Context(), f.tenantA)

	committed := false
	_, err := dbtx.Write(ctx, conn, func(ctx context.Context) (struct{}, error) {
		keel.AfterCommit(ctx, func(context.Context) { committed = true })
		kernelports.AfterCommit(ctx, func(context.Context) { committed = true })

		if err := keeltestdb.FleetCodes.Create(ctx, conn.Keel(ctx), keelFleetCode(f.tenantA, "KEEL")); err != nil {
			return struct{}{}, err
		}
		if err := rawInsertFleetCode(ctx, conn, f.tenantA, "RAW"); err != nil {
			return struct{}{}, err
		}

		return struct{}{}, errRollBack
	})
	require.ErrorIs(t, err, errRollBack)

	assert.Equal(t, []string{"FCA"}, f.codes(t, f.tenantA))
	assert.False(t, committed)
}

func TestKeel_AfterCommitRunsOnceTheSharedTransactionCommits(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)
	ctx := dbscope.WithTenant(t.Context(), f.tenantA)

	var seen []string
	err := dbtx.WriteErr(ctx, conn, func(ctx context.Context) error {
		keel.AfterCommit(ctx, func(ctx context.Context) {
			codes, err := fleetCodes(ctx, conn)
			require.NoError(t, err)
			seen = codes
		})

		return keeltestdb.FleetCodes.Create(ctx, conn.Keel(ctx), keelFleetCode(f.tenantA, "KEEL"))
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"FCA", "KEEL"}, seen)
}

func TestKeel_NestedDbtxCallsShareTheOuterTransaction(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)
	ctx := dbscope.WithTenant(t.Context(), f.tenantA)

	err := dbtx.WriteErr(ctx, conn, func(ctx context.Context) error {
		if err := keeltestdb.FleetCodes.Create(ctx, conn.Keel(ctx), keelFleetCode(f.tenantA, "KEEL")); err != nil {
			return err
		}

		inner, err := dbtx.Read(ctx, conn, func(ctx context.Context) ([]string, error) {
			assert.True(t, keel.InTx(ctx, conn.Keel(ctx)))
			return keelCodes(ctx, conn.Keel(ctx))
		})
		if err != nil {
			return err
		}
		assert.Equal(t, []string{"FCA", "KEEL"}, inner, "a nested dbtx.Read sees the outer write")

		return errRollBack
	})
	require.ErrorIs(t, err, errRollBack)
	assert.Equal(t, []string{"FCA"}, f.codes(t, f.tenantA))
}

func TestKeel_SavepointKeepsTheSharedTransactionUsable(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)
	ctx := dbscope.WithTenant(t.Context(), f.tenantA)

	err := dbtx.WriteErr(ctx, conn, func(ctx context.Context) error {
		first := keelFleetCode(f.tenantA, "ONE")
		if err := keeltestdb.FleetCodes.Create(ctx, conn.Keel(ctx), first); err != nil {
			return err
		}

		duplicate := keelFleetCode(f.tenantA, "DUP")
		duplicate.ID = first.ID
		err := dbtx.Savepoint(ctx, conn, func(ctx context.Context) error {
			return keeltestdb.FleetCodes.Create(ctx, conn.Keel(ctx), duplicate)
		})
		require.True(t, keel.IsUnique(err, "pk_fleet_codes"), "%v", err)

		return rawInsertFleetCode(ctx, conn, f.tenantA, "RAW")
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"FCA", "ONE", "RAW"}, f.codes(t, f.tenantA))
}

func TestKeel_InsideDbtxIsRowLevelSecurityScoped(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)
	ctx := dbscope.WithTenant(t.Context(), f.tenantA)

	codes, err := dbtx.Read(ctx, conn, func(ctx context.Context) ([]string, error) {
		return keel.QueryAll(
			ctx, conn.Keel(ctx), pgx.RowTo[string],
			"SELECT code FROM fleet_codes WHERE organization_id = $1",
			f.tenantB.OrganizationID.String(),
		)
	})
	require.NoError(t, err)
	assert.Empty(t, codes, "an unfiltered statement still cannot see another tenant's rows")

	err = dbtx.WriteErr(ctx, conn, func(ctx context.Context) error {
		_, err := keel.Exec(
			ctx,
			conn.Keel(ctx),
			"INSERT INTO fleet_codes (id, code, organization_id, business_unit_id, manager_id) VALUES ($1, 'X', $2, $3, $4)",
			pulid.MustNew("fc_").String(),
			f.tenantB.OrganizationID.String(),
			f.tenantB.BusinessUnitID.String(),
			f.tenantB.UserID.String(),
		)
		return err
	})
	require.Error(t, err)
	assert.True(t, dberror.IsRowLevelSecurityViolation(err), "%v", err)
}

func TestKeel_ChangingScopeInsideASharedTransactionOpensANewOne(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)

	err := dbtx.WriteErr(
		dbscope.WithTenant(t.Context(), f.tenantA),
		conn,
		func(ctx context.Context) error {
			other, err := dbtx.Read(
				dbscope.WithTenant(ctx, f.tenantB),
				conn,
				func(ctx context.Context) ([]string, error) {
					return keelCodes(ctx, conn.Keel(ctx))
				},
			)
			if err != nil {
				return err
			}
			assert.Equal(t, []string{"FCB"}, other)

			outer, err := keelCodes(ctx, conn.Keel(ctx))
			if err != nil {
				return err
			}
			assert.Equal(t, []string{"FCA"}, outer)

			return nil
		},
	)
	require.NoError(t, err)
}

func TestKeel_WriteWithAScopedContextIsRowLevelSecurityScoped(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)
	ctx := dbscope.WithTenant(t.Context(), f.tenantA)
	db := conn.Keel(ctx)

	var codes []string
	err := keel.Write(ctx, db, func(ctx context.Context) error {
		if err := keeltestdb.FleetCodes.Create(ctx, db, keelFleetCode(f.tenantA, "KEEL")); err != nil {
			return err
		}

		var err error
		codes, err = keelCodes(ctx, db)
		return err
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"FCA", "KEEL"}, codes)
	assert.Equal(t, []string{"FCA", "KEEL"}, f.codes(t, f.tenantA))

	implicit, err := keelCodes(dbscope.WithTenant(t.Context(), f.tenantB), db)
	require.NoError(t, err)
	assert.Equal(
		t,
		[]string{"FCB"},
		implicit,
		"a statement outside a transaction runs in an implicit scoped one",
	)

	err = keel.Write(ctx, db, func(ctx context.Context) error {
		_, err := keel.Exec(ctx, db,
			"UPDATE fleet_codes SET code = 'STOLEN' WHERE organization_id = $1",
			f.tenantB.OrganizationID.String(),
		)
		return err
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"FCB"}, f.codes(t, f.tenantB))

	_, err = keelCodes(t.Context(), db)
	require.ErrorIs(t, err, ErrNoTenantScope)
}

func TestKeel_ReadScopeTravelsWithTheTenantScope(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)

	manager := pulid.MustNew("usr_")
	ctx := readscope.With(
		dbscope.WithTenant(t.Context(), f.tenantA),
		func(context.Context) (readscope.Grants, error) {
			return readscope.Grants{
				"worker": {Scope: readscope.ScopeTeam, Team: []pulid.ID{manager}},
			}, nil
		},
	)

	scope, err := keel.QueryValue[string](
		ctx,
		conn.Keel(ctx),
		"SELECT trenova_rls.read_scope('worker')",
	)
	require.NoError(t, err)
	assert.Equal(t, "team", scope)

	plain := dbscope.WithTenant(t.Context(), f.tenantA)
	scope, err = keel.QueryValue[string](
		plain,
		conn.Keel(plain),
		"SELECT coalesce(trenova_rls.read_scope('worker'), '')",
	)
	require.NoError(t, err)
	assert.Empty(t, scope)
}

func TestKeel_SystemScopeUsesTheBypassPool(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)

	ctx := dbscope.WithSystem(t.Context(), "integration test sweep")
	require.Same(t, conn.systemKeel, conn.Keel(ctx))

	codes, err := keelCodes(ctx, conn.Keel(ctx))
	require.NoError(t, err)
	assert.Equal(t, []string{"FCA", "FCB"}, codes)

	rows, err := dbtx.Read(ctx, conn, func(ctx context.Context) ([]*keeltest.FleetCode, error) {
		s := keeltestdb.FleetCodes
		return s.AllTenants().Order(s.Code.Asc()).Limit(10).List(ctx, conn.Keel(ctx))
	})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, f.tenantA.OrganizationID.String(), rows[0].OrganizationID)
	assert.Equal(t, f.tenantB.OrganizationID.String(), rows[1].OrganizationID)

	_, err = keeltestdb.FleetCodes.Query().Limit(10).List(ctx, conn.Keel(ctx))
	require.ErrorIs(t, err, keel.ErrAllTenantsRequired)

	tenant := dbscope.WithTenant(t.Context(), f.tenantA)
	_, err = keelCodes(tenant, conn.systemKeel)
	require.ErrorIs(t, err, ErrSystemScopeRequired)
}

func TestKeel_ObserveModeAllowsButPostgresRefusesUnscopedAccess(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeObserve)
	conn := f.connect(t)

	count, err := keel.QueryValue[int64](
		t.Context(),
		conn.Keel(t.Context()),
		"SELECT count(*) FROM us_states",
	)
	require.NoError(t, err)
	assert.Positive(t, count)

	_, err = keelCodes(t.Context(), conn.Keel(t.Context()))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no tenant scope")
}

func TestKeel_ErrorsMapThroughTheEntityMapping(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)
	ctx := dbscope.WithTenant(t.Context(), f.tenantA)
	db := conn.Keel(ctx)
	s := keeltestdb.FleetCodes

	mapping := dberror.Mapping{Entity: "Fleet code"}

	_, err := dbtx.Read(ctx, conn, func(ctx context.Context) (*keeltest.FleetCode, error) {
		return s.Get(ctx, db, pulid.MustNew("fc_").String())
	})
	assertNotFound(t, mapping.Map(err, ""), "Fleet code not found within your organization")

	row := keelFleetCode(f.tenantA, "KEEL")
	require.NoError(t, dbtx.WriteErr(ctx, conn, func(ctx context.Context) error {
		return s.Create(ctx, db, row)
	}))

	duplicate := keelFleetCode(f.tenantA, "DUP")
	duplicate.ID = row.ID
	err = dbtx.WriteErr(ctx, conn, func(ctx context.Context) error {
		return s.Create(ctx, db, duplicate)
	})
	assertValidation(t, mapping.Map(err, ""), "id", errortypes.ErrDuplicate)

	rawErr := dbtx.WriteErr(ctx, conn, func(ctx context.Context) error {
		_, err := keel.Exec(ctx, conn.Keel(ctx), insertFleetCodeSQL,
			row.ID, "RAW", f.tenantA.OrganizationID.String(),
			f.tenantA.BusinessUnitID.String(), f.tenantA.UserID.String(),
		)
		return err
	})
	assertValidation(t, mapping.Map(rawErr, ""), "id", errortypes.ErrDuplicate)

	orphan := keelFleetCode(f.tenantA, "ORPHAN")
	orphan.ManagerID = pulid.MustNew("usr_").String()
	err = dbtx.WriteErr(ctx, conn, func(ctx context.Context) error {
		return s.Create(ctx, db, orphan)
	})
	assertValidation(t, mapping.Map(err, ""), "managerId", errortypes.ErrInvalidReference)

	stale := *row
	require.NoError(t, dbtx.WriteErr(ctx, conn, func(ctx context.Context) error {
		row.Code = "NEW"
		return s.Save(ctx, db, row, s.Code)
	}))
	stale.Code = "STALE"
	err = dbtx.WriteErr(ctx, conn, func(ctx context.Context) error {
		return s.Save(ctx, db, &stale, s.Code)
	})
	assertValidation(t, mapping.Map(err, stale.ID), "version", errortypes.ErrVersionMismatch)

	_, err = dbtx.Read(ctx, conn, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, s.Create(ctx, db, keelFleetCode(f.tenantA, "RO"))
	})
	assert.True(t, dberror.IsReadOnlyTransaction(err), "%v", err)

	err = dbtx.WriteErr(ctx, conn, func(ctx context.Context) error {
		other := keelFleetCode(f.tenantB, "OTHER")
		other.OrganizationID = f.tenantB.OrganizationID.String()
		other.BusinessUnitID = f.tenantB.BusinessUnitID.String()
		return s.Create(ctx, db, other)
	})
	assert.True(t, dberror.IsTenantBoundaryViolation(err), "%v", err)
}

func assertNotFound(t *testing.T, err error, message string) {
	t.Helper()

	var notFound *errortypes.NotFoundError
	require.ErrorAs(t, err, &notFound)
	assert.Equal(t, message, notFound.Error())
}

func assertValidation(t *testing.T, err error, field string, code errortypes.ErrorCode) {
	t.Helper()

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, field, validation.Field)
	assert.Equal(t, code, validation.Code)
}
