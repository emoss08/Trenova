package tenantcopyrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/cloud/archiveplan"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/jackc/pgx/v5"
)

const (
	tenantExistsQuery = `SELECT EXISTS (SELECT 1 FROM public.organizations WHERE id = $1)
    OR EXISTS (SELECT 1 FROM public.business_units WHERE id = $2)`
	takenIdentitiesQuery = `SELECT EXISTS (
    SELECT 1 FROM public.users u
    WHERE lower(u.email_address::text) = ANY($1::text[]) OR lower(u.username::text) = ANY($2::text[])
)`
	takenSlugsQuery = `SELECT EXISTS (
    SELECT 1 FROM public.organizations o WHERE o.login_slug = ANY($1::text[])
)`
	ensurePartitionsQuery = `SELECT audit_partitions.ensure_partitions($1::bigint, 24)`
)

func (r *repository) read(
	ctx context.Context,
	fn func(ctx context.Context, db *keel.DB) error,
) error {
	ctx = dbscope.WithSystem(ctx, preflightReason)
	return dbtx.ReadErr(ctx, r.db, func(ctx context.Context) error {
		return fn(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) TenantExists(ctx context.Context, tenant dbscope.Tenant) (bool, error) {
	var exists bool
	err := r.read(ctx, func(ctx context.Context, db *keel.DB) error {
		var err error
		exists, err = keel.QueryValue[bool](ctx, db, tenantExistsQuery,
			tenant.OrganizationID.String(), tenant.BusinessUnitID.String())
		return err
	})
	if err != nil {
		return false, fmt.Errorf("check whether the tenant is on the server: %w", err)
	}
	return exists, nil
}

func (r *repository) Holding(
	ctx context.Context,
	plan *archiveplan.Plan,
	tenant dbscope.Tenant,
) ([]string, error) {
	var holding []string
	err := r.read(ctx, func(ctx context.Context, db *keel.DB) error {
		var err error
		holding, err = keel.QueryAll(ctx, db, pgx.RowTo[string], plan.HoldingQuery(tenant))
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("find the tables that hold the tenant: %w", err)
	}
	return holding, nil
}

func (r *repository) ExistingKeys(
	ctx context.Context,
	table *archiveplan.PlannedTable,
	keys [][]string,
) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}
	columns := make([][]string, len(table.PrimaryKey))
	for _, key := range keys {
		for i := range columns {
			columns[i] = append(columns[i], key[i])
		}
	}
	args := make([]any, 0, len(columns))
	for _, c := range columns {
		args = append(args, c)
	}
	var found int64
	err := r.read(ctx, func(ctx context.Context, db *keel.DB) error {
		var err error
		found, err = keel.QueryValue[int64](ctx, db, table.ExistingKeysQuery(), args...)
		return err
	})
	if err != nil {
		return 0, fmt.Errorf("check %s for the archive's keys: %w", table.Name, err)
	}
	return found, nil
}

func (r *repository) Resolve(
	ctx context.Context,
	table string,
	key []string,
	values [][]string,
) (map[string]string, error) {
	columns := make([][]string, len(key))
	for _, value := range values {
		if len(value) != len(key) {
			return nil, tenantarchive.Refuse(tenantarchive.RefusalManifest,
				"a reference to "+table+" names the wrong number of key values")
		}
		for i := range columns {
			columns[i] = append(columns[i], value[i])
		}
	}
	args := make([]any, 0, len(columns))
	for _, c := range columns {
		args = append(args, c)
	}
	rows, err := r.referenceRows(ctx, table, key, args)
	if err != nil {
		return nil, fmt.Errorf("find the %s the archive refers to: %w", table, err)
	}
	byKey := make(map[string]string, len(rows))
	for _, row := range rows {
		byKey[strings.Join(row.Key, "\x1f")] = row.ID
	}
	out := make(map[string]string, len(values))
	missing := make([]string, 0)
	for _, value := range values {
		joined := strings.Join(value, "\x1f")
		id, ok := byKey[joined]
		if !ok {
			missing = append(missing, strings.Join(value, "/"))
			continue
		}
		out[joined] = id
	}
	if len(missing) > 0 {
		return nil, tenantarchive.Refuse(tenantarchive.RefusalReference, fmt.Sprintf(
			"the archive refers to %s %s, which this server does not have",
			table, strings.Join(missing, ", ")))
	}
	return out, nil
}

func (r *repository) referenceRows(
	ctx context.Context,
	table string,
	key []string,
	args []any,
) ([]referenceRow, error) {
	ctx = dbscope.WithSystem(ctx, referencesReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]referenceRow, error) {
		return keel.QueryAll(ctx, r.db.Keel(ctx), pgx.RowToStructByName[referenceRow],
			archiveplan.ResolveQuery(table, key), args...)
	})
}

func (r *repository) TakenIdentities(
	ctx context.Context,
	emails, usernames []string,
) (bool, error) {
	var taken bool
	err := r.read(ctx, func(ctx context.Context, db *keel.DB) error {
		var err error
		taken, err = keel.QueryValue[bool](ctx, db, takenIdentitiesQuery, emails, usernames)
		return err
	})
	if err != nil {
		return false, fmt.Errorf("check the archive's addresses and usernames: %w", err)
	}
	return taken, nil
}

func (r *repository) TakenSlugs(ctx context.Context, slugs []string) (bool, error) {
	if len(slugs) == 0 {
		return false, nil
	}
	var taken bool
	err := r.read(ctx, func(ctx context.Context, db *keel.DB) error {
		var err error
		taken, err = keel.QueryValue[bool](ctx, db, takenSlugsQuery, slugs)
		return err
	})
	if err != nil {
		return false, fmt.Errorf("check the archive's sign-in addresses: %w", err)
	}
	return taken, nil
}

func (r *repository) EnsureAuditPartitions(ctx context.Context, from int64) error {
	ctx = dbscope.WithSystem(ctx, partitionsReason)
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if _, err := keel.Exec(ctx, r.db.Keel(ctx), ensurePartitionsQuery, from); err != nil {
			return fmt.Errorf("make the audit partitions the archive needs: %w", err)
		}
		return nil
	})
}
