package carrierinvoicematchrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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

func New(p Params) repositories.CarrierInvoiceMatchRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-invoice-match-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListCarrierInvoiceMatchesRequest,
) (*pagination.ListResult[*carriersettlement.InvoiceMatch], error) {
	cols := buncolgen.InvoiceMatchColumns
	rel := buncolgen.InvoiceMatchRelations
	limit := req.Filter.Pagination.SafeLimit()
	items := make([]*carriersettlement.InvoiceMatch, 0, limit)

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Apply(buncolgen.InvoiceMatchApplyTenant(req.Filter.TenantInfo)).
		Relation(rel.Carrier).
		Relation(rel.CarrierAssignment).
		Order(cols.CreatedAt.OrderDesc()).
		Limit(limit).
		Offset(req.Filter.Pagination.SafeOffset())

	if req.Filter.Query != "" {
		query = query.Where(cols.InvoiceNumber.ILike(), "%"+req.Filter.Query+"%")
	}
	if !req.CarrierID.IsNil() {
		query = query.Where(cols.CarrierID.Eq(), req.CarrierID)
	}
	if req.Status != "" {
		query = query.Where(cols.Status.Eq(), req.Status)
	}

	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("list carrier invoice matches: %w", err)
	}

	return &pagination.ListResult[*carriersettlement.InvoiceMatch]{
		Items: items,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetCarrierInvoiceMatchByIDRequest,
) (*carriersettlement.InvoiceMatch, error) {
	cols := buncolgen.InvoiceMatchColumns
	rel := buncolgen.InvoiceMatchRelations
	entity := new(carriersettlement.InvoiceMatch)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.InvoiceMatchScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Relation(rel.Carrier).
		Relation(rel.CarrierAssignment).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "CarrierInvoiceMatch")
	}
	return entity, nil
}

func (r *repository) GetOpenByEDIInvoiceID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	invoiceID pulid.ID,
) (*carriersettlement.InvoiceMatch, error) {
	cols := buncolgen.InvoiceMatchColumns
	entity := new(carriersettlement.InvoiceMatch)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.InvoiceMatchScopeTenant(sq, tenantInfo).
				Where(cols.EDICarrierInvoiceID.Eq(), invoiceID).
				Where(cols.Status.NotEq(), carriersettlement.InvoiceMatchStatusRejected)
		}).
		Order(cols.CreatedAt.OrderDesc()).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil //nolint:nilnil // nil match represents an optional absence
		}
		return nil, fmt.Errorf("get open carrier invoice match: %w", err)
	}
	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *carriersettlement.InvoiceMatch,
) (*carriersettlement.InvoiceMatch, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("cim_")
	}
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create carrier invoice match: %w", err)
	}
	return r.GetByID(ctx, repositories.GetCarrierInvoiceMatchByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *carriersettlement.InvoiceMatch,
) (*carriersettlement.InvoiceMatch, error) {
	cols := buncolgen.InvoiceMatchColumns
	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.InvoiceMatchScopeTenantUpdate(uq, pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			}).
				Where(cols.ID.Eq(), entity.ID).
				Where(cols.Version.Eq(), entity.Version)
		}).
		Set(cols.Status.Set(), entity.Status).
		Set(cols.CarrierSettlementID.Set(), entity.CarrierSettlementID).
		Set(cols.AdjustmentCostEventID.Set(), entity.AdjustmentCostEventID).
		Set(cols.InvoiceTotalMinor.Set(), entity.InvoiceTotalMinor).
		Set(cols.ExpectedTotalMinor.Set(), entity.ExpectedTotalMinor).
		Set(cols.VarianceMinor.Set(), entity.VarianceMinor).
		Set(cols.ResolutionNote.Set(), entity.ResolutionNote).
		Set(cols.ResolvedByID.Set(), entity.ResolvedByID).
		Set(cols.ResolvedAt.Set(), entity.ResolvedAt).
		Set(cols.Version.Inc(1)).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update carrier invoice match: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, "CarrierInvoiceMatch", entity.ID.String()); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, repositories.GetCarrierInvoiceMatchByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
}
