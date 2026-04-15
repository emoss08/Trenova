package bankreceiptbatchrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/bankreceiptbatch"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In
	DB     *postgres.Connection
	Logger *zap.Logger
}
type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.BankReceiptBatchRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.bank-receipt-batch-repository")}
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetBankReceiptBatchByIDRequest,
) (*bankreceiptbatch.Batch, error) {
	entity := new(bankreceiptbatch.Batch)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("brib.id = ?", req.ID).
		Where("brib.organization_id = ?", req.TenantInfo.OrgID).
		Where("brib.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "BankReceiptBatch")
	}
	return entity, nil
}

func (r *repository) List(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*bankreceiptbatch.Batch, error) {
	items := make([]*bankreceiptbatch.Batch, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("brib.organization_id = ?", tenantInfo.OrgID).
		Where("brib.business_unit_id = ?", tenantInfo.BuID).
		Order("brib.created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list bank receipt batches: %w", err)
	}
	return items, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *bankreceiptbatch.Batch,
) (*bankreceiptbatch.Batch, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create bank receipt batch: %w", err)
	}
	return r.GetByID(
		ctx,
		repositories.GetBankReceiptBatchByIDRequest{
			ID: entity.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		},
	)
}

func (r *repository) DistinctSources(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*bankreceiptbatch.SourceOption], error) {
	var sources []string
	q := r.db.DBForContext(ctx).
		NewSelect().
		TableExpr("bank_receipt_import_batches AS brib").
		ColumnExpr("DISTINCT brib.source").
		Where("brib.organization_id = ?", req.TenantInfo.OrgID).
		Where("brib.business_unit_id = ?", req.TenantInfo.BuID).
		Where("brib.source != ''").
		OrderExpr("brib.source ASC")

	if strings.TrimSpace(req.Query) != "" {
		q = q.Where("brib.source ILIKE ?", "%"+strings.TrimSpace(req.Query)+"%")
	}

	err := q.Scan(ctx, &sources)
	if err != nil {
		return nil, fmt.Errorf("distinct sources: %w", err)
	}

	options := make([]*bankreceiptbatch.SourceOption, len(sources))
	for i, s := range sources {
		options[i] = &bankreceiptbatch.SourceOption{Value: s, Label: s}
	}

	limit := req.Pagination.SafeLimit()
	offset := req.Pagination.SafeOffset()
	total := len(options)

	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}

	return &pagination.ListResult[*bankreceiptbatch.SourceOption]{
		Items: options[offset:end],
		Total: total,
	}, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *bankreceiptbatch.Batch,
) (*bankreceiptbatch.Batch, error) {
	entity.UpdatedAt = timeutils.NowUnix()
	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Where("id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Where("version = ?", entity.Version).
		Set("status = ?", entity.Status).
		Set("imported_count = ?", entity.ImportedCount).
		Set("matched_count = ?", entity.MatchedCount).
		Set("exception_count = ?", entity.ExceptionCount).
		Set("imported_amount_minor = ?", entity.ImportedAmountMinor).
		Set("matched_amount_minor = ?", entity.MatchedAmountMinor).
		Set("exception_amount_minor = ?", entity.ExceptionAmountMinor).
		Set("updated_by_id = ?", entity.UpdatedByID).
		Set("updated_at = ?", entity.UpdatedAt).
		Set("version = version + 1").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update bank receipt batch: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, "BankReceiptBatch", entity.ID.String()); err != nil {
		return nil, err
	}
	return r.GetByID(
		ctx,
		repositories.GetBankReceiptBatchByIDRequest{
			ID: entity.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		},
	)
}
