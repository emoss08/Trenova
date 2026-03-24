package customfieldvaluerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customfield"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
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

func New(p Params) repositories.CustomFieldValueRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.custom-field-value-repository"),
	}
}

func (r *repository) GetByResource(
	ctx context.Context,
	req *repositories.GetCustomFieldValuesByResourceRequest,
) ([]*customfield.CustomFieldValue, error) {
	log := r.l.With(
		zap.String("operation", "GetByResource"),
		zap.String("resourceType", req.ResourceType),
		zap.String("resourceID", req.ResourceID),
	)

	values := make([]*customfield.CustomFieldValue, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&values).
		Relation("Definition").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("cfv.organization_id = ?", req.TenantInfo.OrgID).
				Where("cfv.business_unit_id = ?", req.TenantInfo.BuID).
				Where("cfv.resource_type = ?", req.ResourceType).
				Where("cfv.resource_id = ?", req.ResourceID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get custom field values", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Custom field values are busy. Retry the request.",
		)
	}

	return values, nil
}

func (r *repository) GetByResources(
	ctx context.Context,
	req *repositories.GetCustomFieldValuesByResourcesRequest,
) (map[string][]*customfield.CustomFieldValue, error) {
	log := r.l.With(
		zap.String("operation", "GetByResources"),
		zap.String("resourceType", req.ResourceType),
		zap.Int("resourceCount", len(req.ResourceIDs)),
	)

	if len(req.ResourceIDs) == 0 {
		return make(map[string][]*customfield.CustomFieldValue), nil
	}

	values := make([]*customfield.CustomFieldValue, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&values).
		Relation("Definition").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("cfv.organization_id = ?", req.TenantInfo.OrgID).
				Where("cfv.business_unit_id = ?", req.TenantInfo.BuID).
				Where("cfv.resource_type = ?", req.ResourceType).
				Where("cfv.resource_id IN (?)", bun.List(req.ResourceIDs))
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get custom field values for resources", zap.Error(err))
		return nil, err
	}

	result := make(map[string][]*customfield.CustomFieldValue)
	for _, v := range values {
		result[v.ResourceID] = append(result[v.ResourceID], v)
	}

	return result, nil
}

func (r *repository) Upsert(
	ctx context.Context,
	req *repositories.UpsertCustomFieldValuesRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Upsert"),
		zap.String("resourceType", req.ResourceType),
		zap.String("resourceID", req.ResourceID),
		zap.Int("valueCount", len(req.Values)),
	)

	if len(req.Values) == 0 {
		if err := r.DeleteByResource(ctx, &repositories.GetCustomFieldValuesByResourceRequest{
			TenantInfo:   req.TenantInfo,
			ResourceType: req.ResourceType,
			ResourceID:   req.ResourceID,
		}); err != nil {
			return err
		}
		return nil
	}

	return dberror.MapRetryableTransactionError(
		r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
			_, err := r.db.DBForContext(c).NewDelete().
				Model((*customfield.CustomFieldValue)(nil)).
				WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
					return dq.
						Where("organization_id = ?", req.TenantInfo.OrgID).
						Where("business_unit_id = ?", req.TenantInfo.BuID).
						Where("resource_type = ?", req.ResourceType).
						Where("resource_id = ?", req.ResourceID)
				}).
				Exec(c)
			if err != nil {
				log.Error("failed to delete existing custom field values", zap.Error(err))
				return err
			}

			valuesToInsert := make([]*customfield.CustomFieldValue, 0, len(req.Values))
			for definitionID, value := range req.Values {
				defID, parseErr := pulid.Parse(definitionID)
				if parseErr != nil {
					log.Warn(
						"invalid definition ID, skipping",
						zap.String("definitionID", definitionID),
					)
					continue
				}

				valuesToInsert = append(valuesToInsert, &customfield.CustomFieldValue{
					OrganizationID: req.TenantInfo.OrgID,
					BusinessUnitID: req.TenantInfo.BuID,
					DefinitionID:   defID,
					ResourceType:   req.ResourceType,
					ResourceID:     req.ResourceID,
					Value:          value,
				})
			}

			if len(valuesToInsert) > 0 {
				if _, insertErr := r.db.DBForContext(c).NewInsert().
					Model(&valuesToInsert).
					Exec(c); insertErr != nil {
					log.Error("failed to insert custom field values", zap.Error(insertErr))
					return insertErr
				}
			}

			return nil
		}),
		"Custom field values are busy. Retry the request.",
	)
}

func (r *repository) DeleteByResource(
	ctx context.Context,
	req *repositories.GetCustomFieldValuesByResourceRequest,
) error {
	log := r.l.With(
		zap.String("operation", "DeleteByResource"),
		zap.String("resourceType", req.ResourceType),
		zap.String("resourceID", req.ResourceID),
	)

	_, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*customfield.CustomFieldValue)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return dq.
				Where("organization_id = ?", req.TenantInfo.OrgID).
				Where("business_unit_id = ?", req.TenantInfo.BuID).
				Where("resource_type = ?", req.ResourceType).
				Where("resource_id = ?", req.ResourceID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete custom field values", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) CountByDefinition(
	ctx context.Context,
	req *repositories.GetValuesByDefinitionRequest,
) (int, error) {
	log := r.l.With(
		zap.String("operation", "CountByDefinition"),
		zap.String("definitionID", req.DefinitionID.String()),
	)

	count, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*customfield.CustomFieldValue)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("organization_id = ?", req.TenantInfo.OrgID).
				Where("business_unit_id = ?", req.TenantInfo.BuID).
				Where("definition_id = ?", req.DefinitionID)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count custom field values by definition", zap.Error(err))
		return 0, err
	}

	return count, nil
}

func (r *repository) CountResourcesByDefinition(
	ctx context.Context,
	req *repositories.GetValuesByDefinitionRequest,
) (int, error) {
	log := r.l.With(
		zap.String("operation", "CountResourcesByDefinition"),
		zap.String("definitionID", req.DefinitionID.String()),
	)

	var count int
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*customfield.CustomFieldValue)(nil)).
		ColumnExpr("COUNT(DISTINCT resource_id)").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("organization_id = ?", req.TenantInfo.OrgID).
				Where("business_unit_id = ?", req.TenantInfo.BuID).
				Where("definition_id = ?", req.DefinitionID)
		}).
		Scan(ctx, &count)
	if err != nil {
		log.Error("failed to count resources by definition", zap.Error(err))
		return 0, err
	}

	return count, nil
}

func (r *repository) GetOptionUsageCounts(
	ctx context.Context,
	req *repositories.GetOptionUsageRequest,
) (map[string]int, error) {
	log := r.l.With(
		zap.String("operation", "GetOptionUsageCounts"),
		zap.String("definitionID", req.DefinitionID.String()),
	)

	type optionCount struct {
		Value string `bun:"value"`
		Count int    `bun:"count"`
	}

	var counts []optionCount
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*customfield.CustomFieldValue)(nil)).
		ColumnExpr("trim(both '\"' from value::text) as value, COUNT(*) as count").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("organization_id = ?", req.TenantInfo.OrgID).
				Where("business_unit_id = ?", req.TenantInfo.BuID).
				Where("definition_id = ?", req.DefinitionID)
		}).
		Group("value").
		Scan(ctx, &counts)
	if err != nil {
		log.Error("failed to get option usage counts", zap.Error(err))
		return nil, err
	}

	result := make(map[string]int)
	for _, c := range counts {
		result[c.Value] = c.Count
	}

	return result, nil
}
