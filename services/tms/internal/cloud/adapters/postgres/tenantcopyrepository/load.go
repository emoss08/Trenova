package tenantcopyrepository

import (
	"context"
	"fmt"
	"io"

	"github.com/emoss08/trenova/internal/cloud/archiveplan"
	"github.com/emoss08/trenova/internal/cloud/ports/archiveports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/jackc/pgx/v5"
)

const markForeignCodes = `UPDATE public.mfa_recovery_codes SET hash_version = $2
WHERE organization_id = $1 AND used_at IS NULL AND hash_version = $3`

type load struct {
	repo   *repository
	conn   *pgx.Conn
	tx     pgx.Tx
	plan   *archiveplan.Plan
	tenant dbscope.Tenant
	staged []*archiveplan.PlannedTable
	beat   func()
	done   bool
}

func (r *repository) OpenLoad(
	ctx context.Context,
	req *archiveports.LoadRequest,
) (archiveports.Load, error) {
	conn, err := r.connect(ctx)
	if err != nil {
		return nil, err
	}
	l := &load{repo: r, conn: conn, tenant: req.Tenant, beat: req.Beat}
	if err = l.open(ctx, req); err != nil {
		l.Rollback(ctx)
		return nil, err
	}
	return l, nil
}

func (l *load) open(ctx context.Context, req *archiveports.LoadRequest) error {
	tx, err := l.conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin the archive load: %w", err)
	}
	l.tx = tx
	if err = settle(
		ctx,
		tx,
		statementBudget(ctx),
		[2]string{archiveLoadSetting, "on"},
	); err != nil {
		return err
	}
	cat, err := readCatalog(ctx, txRunner(tx))
	if err != nil {
		return err
	}
	if l.plan, err = archiveplan.Build(&archiveplan.Input{
		Catalog:  cat,
		Purpose:  req.Manifest.Purpose,
		Envelope: req.Envelope,
	}); err != nil {
		return err
	}
	return l.plan.Matches(req.Manifest)
}

func (l *load) Plan() *archiveplan.Plan {
	return l.plan
}

func (l *load) CopyIn(
	ctx context.Context,
	table *archiveplan.PlannedTable,
	src io.Reader,
) (int64, error) {
	if _, err := l.tx.Exec(ctx, table.CreateStage()); err != nil {
		return 0, fmt.Errorf("stage %s: %w", table.Name, err)
	}
	tag, err := l.tx.Conn().PgConn().CopyFrom(ctx, src, table.CopyIn())
	if err != nil {
		return 0, fmt.Errorf("load %s: %w", table.Name, err)
	}
	rows := tag.RowsAffected()
	inserted, err := l.tx.Exec(ctx, table.InsertFromStage())
	if err != nil {
		return 0, fmt.Errorf("load %s from its stage: %w", table.Name, err)
	}
	if inserted.RowsAffected() != rows {
		return 0, fmt.Errorf("load %s: staged %d rows and inserted %d", table.Name, rows,
			inserted.RowsAffected())
	}
	if table.Staged() {
		l.staged = append(l.staged, table)
	} else if _, err = l.tx.Exec(ctx, table.DropStage()); err != nil {
		return 0, fmt.Errorf("drop the stage of %s: %w", table.Name, err)
	}
	if l.beat != nil {
		l.beat()
	}
	return rows, nil
}

func (l *load) Finish(ctx context.Context) error {
	for _, table := range l.staged {
		if _, err := l.tx.Exec(ctx, table.RestoreDeferred()); err != nil {
			return fmt.Errorf("restore %s's references: %w", table.Name, err)
		}
		if l.beat != nil {
			l.beat()
		}
	}
	for _, table := range l.plan.Tables {
		for _, stmt := range table.AdvanceSequences() {
			if _, err := l.tx.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("advance %s's sequence: %w", table.Name, err)
			}
		}
	}
	return nil
}

func (l *load) Counts(ctx context.Context) ([]archiveports.TableCount, error) {
	rows, err := collect(ctx, txRunner(l.tx), l.plan.CountQuery(l.tenant),
		pgx.RowToStructByName[archiveports.TableCount])
	if err != nil {
		return nil, fmt.Errorf("count the loaded rows: %w", err)
	}
	return rows, nil
}

func (l *load) Dangling(ctx context.Context) ([]archiveports.Dangling, error) {
	return dangling(ctx, txRunner(l.tx), l.plan.DanglingQueries(l.tenant, danglingBatch))
}

func (l *load) MarkForeignRecoveryCodes(ctx context.Context) (int64, error) {
	if _, ok := l.plan.Table("mfa_recovery_codes"); !ok {
		return 0, nil
	}
	tag, err := l.tx.Exec(ctx, markForeignCodes, l.tenant.OrganizationID.String(),
		iam.RecoveryCodeHashForeign, iam.RecoveryCodeHashHMAC)
	if err != nil {
		return 0, fmt.Errorf("mark the recovery codes made under another server's pepper: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (l *load) heartbeat() {
	if l.beat != nil {
		l.beat()
	}
}

func (l *load) Commit(ctx context.Context) error {
	err := l.tx.Commit(ctx)
	l.done = true
	_ = l.conn.Close(context.WithoutCancel(ctx))
	if err != nil {
		return fmt.Errorf("commit the archive load: %w", err)
	}
	return nil
}

func (l *load) Rollback(ctx context.Context) {
	if l.done {
		return
	}
	l.done = true
	if l.tx != nil {
		_ = l.tx.Rollback(context.WithoutCancel(ctx))
	}
	_ = l.conn.Close(context.WithoutCancel(ctx))
}
