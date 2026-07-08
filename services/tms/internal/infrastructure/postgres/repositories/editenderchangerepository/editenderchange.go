package editenderchangerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
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

func New(p Params) repositories.EDITenderChangeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-tender-change-repository"),
	}
}

func (r *repository) ListTenderChanges(
	ctx context.Context,
	req *repositories.ListEDITenderChangesRequest,
) (*pagination.ListResult[*edi.TenderChange], error) {
	entities := make([]*edi.TenderChange, 0, req.Filter.Pagination.SafeLimit())
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation("Recipient").
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyTenderChangeTenantScope(sq, req.Filter.TenantInfo)
		})
	if req.RecipientID.IsNotNil() {
		query = query.Where("etcg.recipient_id = ?", req.RecipientID)
	}
	if req.SourceShipmentID.IsNotNil() {
		query = query.Where("etcg.source_shipment_id = ?", req.SourceShipmentID)
	}
	if req.Status != "" {
		query = query.Where("etcg.status = ?", req.Status)
	}
	total, err := querybuilder.ApplyFiltersWithoutTenantScope(query, "etcg", req.Filter, (*edi.TenderChange)(nil)).
		Order("etcg.created_at DESC").
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}
	return &pagination.ListResult[*edi.TenderChange]{Items: entities, Total: total}, nil
}

func (r *repository) GetTenderChangeByID(
	ctx context.Context,
	req repositories.GetEDITenderChangeByIDRequest,
) (*edi.TenderChange, error) {
	entity := new(edi.TenderChange)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation("Recipient").
		Where("etcg.id = ?", req.ID).
		Apply(func(query *bun.SelectQuery) *bun.SelectQuery {
			return applyTenderChangeTenantScope(query, req.TenantInfo)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDITenderChange")
	}
	return entity, nil
}

func (r *repository) GetTenderChangeByOutboundMessageID(
	ctx context.Context,
	req repositories.GetEDITenderChangeByOutboundMessageIDRequest,
) (*edi.TenderChange, error) {
	entity := new(edi.TenderChange)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation("Recipient").
		Where("etcg.outbound_message_id = ?", req.OutboundMessageID).
		Apply(func(query *bun.SelectQuery) *bun.SelectQuery {
			return applyTenderChangeTenantScope(query, req.TenantInfo)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDITenderChange")
	}
	return entity, nil
}

func (r *repository) CreateTenderChangeIdempotent(
	ctx context.Context,
	entity *edi.TenderChange,
) (*repositories.CreateEDITenderChangeIdempotentResult, error) {
	results, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		On("CONFLICT (recipient_id, source_shipment_version, new_payload_hash, change_type) DO NOTHING").
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
		return &repositories.CreateEDITenderChangeIdempotentResult{
			TenderChange: entity,
			Created:      true,
		}, nil
	}

	existing := new(edi.TenderChange)
	if err = r.db.DBForContext(ctx).
		NewSelect().
		Model(existing).
		Where("recipient_id = ?", entity.RecipientID).
		Where("source_shipment_version = ?", entity.SourceShipmentVersion).
		Where("new_payload_hash = ?", entity.NewPayloadHash).
		Where("change_type = ?", entity.ChangeType).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDITenderChange")
	}
	return &repositories.CreateEDITenderChangeIdempotentResult{
		TenderChange: existing,
		Created:      false,
	}, nil
}

func (r *repository) SupersedeActionableTenderChanges(
	ctx context.Context,
	req repositories.SupersedeActionableEDITenderChangesRequest,
) error {
	if len(req.Statuses) == 0 {
		req.Statuses = []edi.TenderChangeStatus{
			edi.TenderChangeStatusPendingReview,
			edi.TenderChangeStatusQueued,
		}
	}
	query := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*edi.TenderChange)(nil)).
		Set("status = ?", edi.TenderChangeStatusSuperseded).
		Set("updated_at = extract(epoch from current_timestamp)::bigint").
		Where("recipient_id = ?", req.RecipientID).
		Where("status IN (?)", bun.List(req.Statuses))
	if req.ExcludeChangeID.IsNotNil() {
		query = query.Where("id <> ?", req.ExcludeChangeID)
	}
	_, err := query.Exec(ctx)
	return err
}

func (r *repository) UpdateTenderChange(
	ctx context.Context,
	entity *edi.TenderChange,
) (*edi.TenderChange, error) {
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
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "EDITenderChange", entity.ID.String()); err != nil {
		return nil, err
	}
	return entity, nil
}

func applyTenderChangeTenantScope(
	query *bun.SelectQuery,
	tenantInfo pagination.TenantInfo,
) *bun.SelectQuery {
	return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereOr(
			"(etcg.source_organization_id = ? AND etcg.source_business_unit_id = ?)",
			tenantInfo.OrgID,
			tenantInfo.BuID,
		).WhereOr(
			`EXISTS (
				SELECT 1
				FROM edi_tender_recipients AS etr_scope
				WHERE etr_scope.id = etcg.recipient_id
					AND etr_scope.recipient_organization_id = ?
					AND (
						etr_scope.recipient_business_unit_id = ?
						OR etr_scope.recipient_business_unit_id IS NULL
						OR etr_scope.business_unit_id = ?
					)
			)`,
			tenantInfo.OrgID,
			tenantInfo.BuID,
			tenantInfo.BuID,
		)
	})
}
