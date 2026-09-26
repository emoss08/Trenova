package aitrainingrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	defaultHistoryLimit   = 50
	maxHistoryLimit       = 200
	defaultWithdrawnLimit = 1000
	maxWithdrawnLimit     = 10000
	insertBatchSize       = 500
)

type recordRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewRecords(p Params) repositories.AITrainingRecordRepository {
	return &recordRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.aitraining-record-repository"),
	}
}

func (r *recordRepository) ReplaceForOrganization(
	ctx context.Context,
	req *repositories.ReplaceAITrainingRecordsRequest,
) error {
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
		if err := deleteRecords(txCtx, tx, req.ExportID, req.TenantInfo); err != nil {
			return err
		}
		for start := 0; start < len(req.Records); start += insertBatchSize {
			batch := req.Records[start:min(start+insertBatchSize, len(req.Records))]
			for _, record := range batch {
				record.ExportID = req.ExportID
				record.OrganizationID = req.TenantInfo.OrgID
				record.BusinessUnitID = req.TenantInfo.BuID
			}
			if _, err := tx.NewInsert().Model(&batch).Exec(txCtx); err != nil {
				return fmt.Errorf("insert training export records: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		r.l.Error("failed to replace training export records", zap.Error(err))

		return err
	}

	return nil
}

func (r *recordRepository) DeleteForOrganization(
	ctx context.Context,
	exportID pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	if err := deleteRecords(ctx, r.db.DBForContext(ctx), exportID, tenantInfo); err != nil {
		r.l.Error("failed to delete training export records", zap.Error(err))

		return err
	}

	return nil
}

func deleteRecords(
	ctx context.Context,
	db bun.IDB,
	exportID pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	_, err := db.NewDelete().
		Model((*aitraining.TrainingExportRecord)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.TrainingExportRecordScopeTenantDelete(dq, tenantInfo).
				Where(buncolgen.TrainingExportRecordColumns.ExportID.Eq(), exportID)
		}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete training export records: %w", err)
	}

	return nil
}

func (r *recordRepository) ListHistory(
	ctx context.Context,
	req repositories.ListAITrainingHistoryRequest,
) ([]*aitraining.ExportHistoryEntry, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultHistoryLimit
	}
	limit = min(limit, maxHistoryLimit)

	records := buncolgen.TrainingExportRecordColumns
	exports := buncolgen.TrainingExportColumns
	table := buncolgen.TrainingExportTable
	entries := make([]*aitraining.ExportHistoryEntry, 0, limit)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*aitraining.TrainingExportRecord)(nil)).
		Column(records.ExportID.Bare()).
		ColumnExpr("COUNT(*) AS examples").
		ColumnExpr("COUNT(*) FILTER (WHERE ? = ?) AS train_examples",
			bun.Safe(records.Split.Qualified()), aitraining.SplitTrain).
		ColumnExpr("COUNT(*) FILTER (WHERE ? = ?) AS validation_examples",
			bun.Safe(records.Split.Qualified()), aitraining.SplitValidation).
		ColumnExpr("MAX(?) AS consent_granted_at", bun.Safe(records.ConsentGrantedAt.Qualified())).
		ColumnExpr("MAX(?) AS exported_at", bun.Safe(records.ExportedAt.Qualified())).
		Join(
			"JOIN ? AS ? ON ?",
			bun.Ident(table.Name),
			bun.Ident(table.Alias),
			bun.Safe(exports.ID.EqColumn(records.ExportID)),
		).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.TrainingExportRecordScopeTenant(sq, req.TenantInfo)
		}).
		GroupExpr("?", bun.Safe(records.ExportID.Qualified())).
		OrderExpr("exported_at DESC").
		Limit(limit).
		Scan(ctx, &entries)
	if err != nil {
		r.l.Error("failed to list training export history", zap.Error(err))

		return nil, fmt.Errorf("list training export history: %w", err)
	}

	return entries, nil
}

func (r *recordRepository) ListWithdrawn(
	ctx context.Context,
	req repositories.ListWithdrawnTrainingExamplesRequest,
) ([]repositories.WithdrawnTrainingExample, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultWithdrawnLimit
	}
	limit = min(limit, maxWithdrawnLimit)

	records := buncolgen.TrainingExportRecordColumns
	controls := buncolgen.AgentControlColumns
	table := buncolgen.AgentControlTable
	examples := make([]repositories.WithdrawnTrainingExample, 0, min(limit, defaultWithdrawnLimit))
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model((*aitraining.TrainingExportRecord)(nil)).
		Column(records.ID.Bare(), records.ExampleID.Bare(), records.Split.Bare()).
		Join(
			"LEFT JOIN ? AS ? ON ? AND ?",
			bun.Ident(table.Name),
			bun.Ident(table.Alias),
			bun.Safe(controls.OrganizationID.EqColumn(records.OrganizationID)),
			bun.Safe(controls.BusinessUnitID.EqColumn(records.BusinessUnitID)),
		).
		Where(records.ExportID.Eq(), req.ExportID).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(controls.OrganizationID.IsNull()).
				WhereOr(controls.AITrainingConsent.IsFalse()).
				WhereOr("COALESCE(?, 0) > ?",
					bun.Safe(controls.AITrainingConsentChangedAt.Qualified()),
					bun.Safe(records.ConsentGrantedAt.Qualified()),
				)
		})
	if req.AfterID.IsNotNil() {
		query = query.Where(records.ID.Gt(), req.AfterID)
	}
	err := query.
		Order(records.ID.OrderAsc()).
		Limit(limit).
		Scan(ctx, &examples)
	if err != nil {
		r.l.Error("failed to list withdrawn training examples", zap.Error(err))

		return nil, fmt.Errorf("list withdrawn training examples: %w", err)
	}

	return examples, nil
}
