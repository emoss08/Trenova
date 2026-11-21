package tenantcopyrepository

import (
	"context"
	"fmt"
	"io"

	"github.com/emoss08/trenova/internal/cloud/archiveplan"
	"github.com/emoss08/trenova/internal/cloud/ports/archiveports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/jackc/pgx/v5"
)

const (
	organizationsOfUnit = `SELECT o.id::text, o.name::text FROM public.organizations o WHERE o.business_unit_id = $1 ORDER BY o.id`
)

type snapshot struct {
	repo       *repository
	conn       *pgx.Conn
	tx         pgx.Tx
	plan       *archiveplan.Plan
	tenant     dbscope.Tenant
	orgName    string
	trails     *archiveports.TrailFacts
	sealedTail bool
}

func (r *repository) OpenSnapshot(
	ctx context.Context,
	req *archiveports.SnapshotRequest,
) (archiveports.Snapshot, error) {
	conn, err := r.connect(ctx)
	if err != nil {
		return nil, err
	}
	s := &snapshot{repo: r, conn: conn, tenant: req.Tenant}
	if err = s.open(ctx, req); err != nil {
		s.Close(ctx)
		return nil, err
	}
	if req.Purpose.Importable() {
		if err = s.readTrails(ctx, req.At); err != nil {
			s.Close(ctx)
			return nil, err
		}
	}
	return s, nil
}

func (s *snapshot) open(ctx context.Context, req *archiveports.SnapshotRequest) error {
	if req.Purpose.Importable() {
		if err := prepareTrailTables(ctx, s.conn); err != nil {
			return err
		}
	}
	tx, err := s.conn.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.RepeatableRead,
		AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return fmt.Errorf("begin the archive snapshot: %w", err)
	}
	s.tx = tx
	if err = settle(ctx, tx, statementBudget(ctx)); err != nil {
		return err
	}
	cat, err := readCatalog(ctx, txRunner(tx))
	if err != nil {
		return err
	}
	if s.plan, err = archiveplan.Build(&archiveplan.Input{
		Catalog:  cat,
		Purpose:  req.Purpose,
		Envelope: req.Envelope,
	}); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, organizationsOfUnit, req.Tenant.BusinessUnitID.String())
	if err != nil {
		return fmt.Errorf("read the business unit's organizations: %w", err)
	}
	type org struct{ id, name string }
	orgs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (org, error) {
		var o org
		return o, row.Scan(&o.id, &o.name)
	})
	if err != nil {
		return fmt.Errorf("read the business unit's organizations: %w", err)
	}
	switch {
	case len(orgs) == 0 || orgs[0].id != req.Tenant.OrganizationID.String():
		return ErrTenantMissing
	case len(orgs) > 1:
		return ErrUnitShared
	}
	s.orgName = orgs[0].name
	return nil
}

func (s *snapshot) Plan() *archiveplan.Plan {
	return s.plan
}

func (s *snapshot) Tenant() dbscope.Tenant {
	return s.tenant
}

func (s *snapshot) OrganizationName() string {
	return s.orgName
}

func (s *snapshot) Trails() *archiveports.TrailFacts {
	return s.trails
}

func (s *snapshot) CopyOut(
	ctx context.Context,
	table *archiveplan.PlannedTable,
	dst io.Writer,
) error {
	if _, err := s.tx.Conn().PgConn().CopyTo(ctx, dst, s.copyOutQuery(table)); err != nil {
		return fmt.Errorf("copy %s out: %w", table.Name, err)
	}
	return nil
}

type referenceRow struct {
	ID  string   `db:"id"`
	Key []string `db:"natural_key"`
}

func (s *snapshot) References(ctx context.Context) ([]archiveports.Reference, error) {
	out := make([]archiveports.Reference, 0, len(s.plan.References))
	for _, ref := range s.plan.References {
		rows, err := collect(ctx, txRunner(s.tx), ref.CollectQuery(s.plan, s.tenant),
			pgx.RowToStructByName[referenceRow])
		if err != nil {
			return nil, fmt.Errorf("read the %s the tenant refers to: %w", ref.Table, err)
		}
		reference := archiveports.Reference{
			Table: ref.Table,
			Key:   ref.Key,
			Rows:  make(map[string][]string, len(rows)),
		}
		for _, row := range rows {
			reference.Rows[row.ID] = row.Key
		}
		out = append(out, reference)
	}
	return out, nil
}

func (s *snapshot) Outside(ctx context.Context) ([]archiveports.Dangling, error) {
	return dangling(ctx, txRunner(s.tx), s.plan.OutsideQueries(s.tenant, danglingBatch))
}

func dangling(ctx context.Context, run runner, queries []string) ([]archiveports.Dangling, error) {
	out := make([]archiveports.Dangling, 0)
	for _, q := range queries {
		rows, err := collect(ctx, run, q, pgx.RowToStructByName[archiveports.Dangling])
		if err != nil {
			return nil, fmt.Errorf("check the tenant's references: %w", err)
		}
		for _, row := range rows {
			if row.Rows > 0 {
				out = append(out, row)
			}
		}
	}
	return out, nil
}

func (s *snapshot) Close(ctx context.Context) {
	if s.tx != nil {
		_ = s.tx.Rollback(context.WithoutCancel(ctx))
	}
	_ = s.conn.Close(context.WithoutCancel(ctx))
}
