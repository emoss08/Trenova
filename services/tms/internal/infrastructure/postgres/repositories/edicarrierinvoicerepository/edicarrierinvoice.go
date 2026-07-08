package edicarrierinvoicerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
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

func New(p Params) repositories.EDICarrierInvoiceRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-carrier-invoice-repository"),
	}
}

func (r *repository) ListCarrierInvoices(
	ctx context.Context,
	req *repositories.ListEDICarrierInvoicesRequest,
) (*pagination.ListResult[*edi.CarrierInvoice], error) {
	entities := make([]*edi.CarrierInvoice, 0, req.Filter.Pagination.SafeLimit())
	cols := buncolgen.CarrierInvoiceColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation("Partner")
	if req.ReconciliationStatus != "" {
		query = query.Where(cols.ReconciliationStatus.Eq(), req.ReconciliationStatus)
	}
	if req.PartnerID.IsNotNil() {
		query = query.Where(cols.EDIPartnerID.Eq(), req.PartnerID)
	}
	if req.ShipmentID.IsNotNil() {
		query = query.Where(cols.ShipmentID.Eq(), req.ShipmentID)
	}

	total, err := querybuilder.ApplyFilters(query, "ecinv", req.Filter, (*edi.CarrierInvoice)(nil)).
		Order(cols.CreatedAt.OrderDesc()).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}
	return &pagination.ListResult[*edi.CarrierInvoice]{Items: entities, Total: total}, nil
}

func (r *repository) GetCarrierInvoiceByID(
	ctx context.Context,
	req repositories.GetEDICarrierInvoiceByIDRequest,
) (*edi.CarrierInvoice, error) {
	entity := new(edi.CarrierInvoice)
	cols := buncolgen.CarrierInvoiceColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation("Partner").
		Where(cols.ID.Eq(), req.ID).
		Apply(buncolgen.CarrierInvoiceApplyTenant(req.TenantInfo))
	if err := query.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "CarrierInvoice")
	}
	return entity, nil
}

func (r *repository) CreateCarrierInvoice(
	ctx context.Context,
	entity *edi.CarrierInvoice,
) (*edi.CarrierInvoice, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) UpdateCarrierInvoice(
	ctx context.Context,
	entity *edi.CarrierInvoice,
) (*edi.CarrierInvoice, error) {
	ov := entity.Version
	entity.Version++
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		entity.Version = ov
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "CarrierInvoice", entity.ID.String()); err != nil {
		return nil, err
	}
	return entity, nil
}
