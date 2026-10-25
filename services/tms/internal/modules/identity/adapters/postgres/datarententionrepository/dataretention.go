package datarententionrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var dataRetentionErrors = dberror.Mapping{Entity: "Data Retention"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) tenantports.DataRetentionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.data-retention-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
) (*pagination.ListResult[*tenant.DataRetention], error) {
	ctx = dbscope.WithSystem(
		ctx,
		"read every organization's retention settings for retention sweeps",
	)
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*tenant.DataRetention], error) {
			entities, err := tenantdb.DataRetentions.AllTenants().All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list data retentions", zap.Error(err))
				return nil, err
			}

			return &pagination.ListResult[*tenant.DataRetention]{
				Items: entities,
				Total: len(entities),
			}, nil
		},
	)
}

func (r *repository) Get(
	ctx context.Context,
	_ tenantports.GetDataRetentionRequest,
) (*tenant.DataRetention, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tenant.DataRetention, error) {
		entity, err := tenantdb.DataRetentions.Query().One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, dataRetentionErrors.Map(err, "")
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *tenant.DataRetention,
) (*tenant.DataRetention, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*tenant.DataRetention, error) {
		if err := tenantdb.DataRetentions.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to update data retention",
				zap.String("orgID", entity.OrganizationID.String()),
				zap.Error(err),
			)
			return nil, dataRetentionErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Upsert(
	ctx context.Context,
	entity *tenant.DataRetention,
) (*tenant.DataRetention, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*tenant.DataRetention, error) {
		existing, err := r.Get(ctx, tenantports.GetDataRetentionRequest{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		})
		if err == nil {
			existing.AuditRetentionPeriod = entity.AuditRetentionPeriod
			existing.EDIInboundFileRetentionPeriod = entity.EDIInboundFileRetentionPeriod
			existing.EDIMessageRetentionPeriod = entity.EDIMessageRetentionPeriod
			existing.AgentEvalCaseRetentionPeriod = entity.AgentEvalCaseRetentionPeriod
			existing.AIFeedbackRetentionPeriod = entity.AIFeedbackRetentionPeriod
			existing.AIAuditRetentionPeriod = entity.AIAuditRetentionPeriod
			existing.AICorrectionRetentionPeriod = entity.AICorrectionRetentionPeriod
			return r.Update(ctx, existing)
		}
		if !dberror.IsNotFoundError(err) {
			return nil, err
		}

		dr := tenantdb.DataRetentions
		if _, err = dr.Upsert(
			ctx,
			r.db.Keel(ctx),
			[]*tenant.DataRetention{entity},
			keel.OnConflict(dr.OrganizationID).Update(
				dr.AuditRetentionPeriod,
				dr.EDIInboundFileRetentionPeriod,
				dr.EDIMessageRetentionPeriod,
				dr.AgentEvalCaseRetentionPeriod,
				dr.AIFeedbackRetentionPeriod,
				dr.AIAuditRetentionPeriod,
				dr.AICorrectionRetentionPeriod,
			),
		); err != nil {
			r.l.Error("failed to upsert data retention", zap.Error(err))
			return nil, dataRetentionErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}
