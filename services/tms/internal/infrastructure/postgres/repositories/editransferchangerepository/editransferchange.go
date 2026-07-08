package editransferchangerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
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

func New(p Params) repositories.EDITransferChangeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-transfer-change-repository"),
	}
}

func (r *repository) ListTransferChanges(
	ctx context.Context,
	req *repositories.ListEDITransferChangesRequest,
) (*pagination.ListResult[*edi.TransferChange], error) {
	entities := make([]*edi.TransferChange, 0, req.Filter.Pagination.SafeLimit())
	cols := buncolgen.TransferChangeColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Join(transferChangeShipmentLinkJoin()).
		Where(cols.BusinessUnitID.Eq(), req.Filter.TenantInfo.BuID).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			linkCols := buncolgen.ShipmentLinkColumns
			return sq.WhereOr(linkCols.SourceOrganizationID.Eq(), req.Filter.TenantInfo.OrgID).
				WhereOr(linkCols.TargetOrganizationID.Eq(), req.Filter.TenantInfo.OrgID)
		})
	if req.ShipmentLinkID.IsNotNil() {
		query = query.Where(cols.ShipmentLinkID.Eq(), req.ShipmentLinkID)
	}
	total, err := querybuilder.ApplyFiltersWithoutTenantScope(query, "etc", req.Filter, (*edi.TransferChange)(nil)).
		Order(cols.CreatedAt.OrderDesc()).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*edi.TransferChange]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetTransferChangeByID(
	ctx context.Context,
	req repositories.GetEDITransferChangeByIDRequest,
) (*edi.TransferChange, error) {
	entity := new(edi.TransferChange)
	cols := buncolgen.TransferChangeColumns

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Join(transferChangeShipmentLinkJoin()).
		Where(cols.ID.Eq(), req.ID).
		Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			linkCols := buncolgen.ShipmentLinkColumns
			return sq.WhereOr(linkCols.SourceOrganizationID.Eq(), req.TenantInfo.OrgID).
				WhereOr(linkCols.TargetOrganizationID.Eq(), req.TenantInfo.OrgID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDITransferChange")
	}

	return entity, nil
}

func (r *repository) CreateTransferChange(
	ctx context.Context,
	entity *edi.TransferChange,
) (*edi.TransferChange, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) CreateTransferChangeIdempotent(
	ctx context.Context,
	entity *edi.TransferChange,
) (*repositories.CreateEDITransferChangeIdempotentResult, error) {
	cols := buncolgen.TransferChangeColumns

	results, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		On("CONFLICT (shipment_link_id, business_unit_id, direction, change_type, idempotency_key) DO NOTHING").
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := results.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected > 0 {
		return &repositories.CreateEDITransferChangeIdempotentResult{
			TransferChange: entity,
			Created:        true,
		}, nil
	}

	existing := new(edi.TransferChange)
	if err = r.db.DBForContext(ctx).
		NewSelect().
		Model(existing).
		Where(cols.ShipmentLinkID.Eq(), entity.ShipmentLinkID).
		Where(cols.BusinessUnitID.Eq(), entity.BusinessUnitID).
		Where(cols.Direction.Eq(), entity.Direction).
		Where(cols.ChangeType.Eq(), entity.ChangeType).
		Where(cols.IdempotencyKey.Eq(), entity.IdempotencyKey).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDITransferChange")
	}

	return &repositories.CreateEDITransferChangeIdempotentResult{
		TransferChange: existing,
		Created:        false,
	}, nil
}

func (r *repository) UpdateTransferChange(
	ctx context.Context,
	entity *edi.TransferChange,
) (*edi.TransferChange, error) {
	ov := entity.Version
	entity.Version++
	cols := buncolgen.TransferChangeColumns

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(cols.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(
		results,
		"EDITransferChange",
		entity.ID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}

func transferChangeShipmentLinkJoin() string {
	return "JOIN " + buncolgen.ShipmentLinkTable.As(buncolgen.ShipmentLinkTable.Alias) +
		" ON " + buncolgen.TransferChangeColumns.ShipmentLinkID.EqColumn(buncolgen.ShipmentLinkColumns.ID)
}
