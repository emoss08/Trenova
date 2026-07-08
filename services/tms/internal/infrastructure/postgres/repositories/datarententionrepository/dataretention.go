package datarententionrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
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

func New(p Params) repositories.DataRetentionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.data-retention-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
) (*pagination.ListResult[*tenant.DataRetention], error) {
	log := r.l.With(zap.String("operation", "List"))

	entities := make([]*tenant.DataRetention, 0)
	total, err := r.db.DB().NewSelect().Model(&entities).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count data retentions", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*tenant.DataRetention]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) Get(
	ctx context.Context,
	req repositories.GetDataRetentionRequest,
) (*tenant.DataRetention, error) {
	entity := new(tenant.DataRetention)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("dr.organization_id = ?", req.OrgID).
				Where("dr.business_unit_id = ?", req.BuID)
		}).Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Data Retention")
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *tenant.DataRetention,
) (*tenant.DataRetention, error) {
	log := r.l.With(zap.String("operation", "Update"), zap.String("id", entity.ID.String()))

	ov := entity.Version
	entity.Version++

	result, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update data retention", zap.Error(err))
		return nil, err
	}

	roErr := dberror.CheckRowsAffected(result, "DataRetention", entity.ID.String())
	if roErr != nil {
		log.Error("failed to check rows affected", zap.Error(roErr))
		return nil, roErr
	}

	return entity, nil
}

func (r *repository) Upsert(
	ctx context.Context,
	entity *tenant.DataRetention,
) (*tenant.DataRetention, error) {
	log := r.l.With(zap.String("operation", "Upsert"))

	existing, err := r.Get(ctx, repositories.GetDataRetentionRequest{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err == nil {
		existing.AuditRetentionPeriod = entity.AuditRetentionPeriod
		existing.EDIInboundFileRetentionPeriod = entity.EDIInboundFileRetentionPeriod
		existing.EDIMessageRetentionPeriod = entity.EDIMessageRetentionPeriod
		return r.Update(ctx, existing)
	}
	if !dberror.IsNotFoundError(err) {
		return nil, err
	}

	if _, err = r.db.DB().
		NewInsert().
		Model(entity).
		On(`CONFLICT ("organization_id") DO UPDATE`).
		Set("audit_retention_period = EXCLUDED.audit_retention_period").
		Set("edi_inbound_file_retention_period = EXCLUDED.edi_inbound_file_retention_period").
		Set("edi_message_retention_period = EXCLUDED.edi_message_retention_period").
		Returning("*").
		Exec(ctx); err != nil {
		log.Error("failed to upsert data retention", zap.Error(err))
		return nil, err
	}
	return entity, nil
}
