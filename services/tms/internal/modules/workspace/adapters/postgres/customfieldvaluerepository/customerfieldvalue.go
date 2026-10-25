package customfieldvaluerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/customfield"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/customfield/customfielddb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/customfieldports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const busyMessage = "Custom field values are busy. Retry the request."

type optionCount struct {
	Value string
	Count int64
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) customfieldports.CustomFieldValueRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.custom-field-value-repository"),
	}
}

func resourceValues(resourceType, resourceID string) *keel.Query[*customfield.CustomFieldValue] {
	v := customfielddb.CustomFieldValues
	return v.Where(v.ResourceType.Eq(resourceType), v.ResourceID.Eq(resourceID))
}

func (r *repository) GetByResource(
	ctx context.Context,
	req *customfieldports.GetCustomFieldValuesByResourceRequest,
) ([]*customfield.CustomFieldValue, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*customfield.CustomFieldValue, error) {
		v := customfielddb.CustomFieldValues
		values, err := resourceValues(req.ResourceType, req.ResourceID).
			Load(v.Definition()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error(
				"failed to get custom field values",
				zap.String("resourceType", req.ResourceType),
				zap.String("resourceID", req.ResourceID),
				zap.Error(err),
			)
			return nil, dberror.MapRetryableTransactionError(err, busyMessage)
		}

		return values, nil
	})
}

func (r *repository) GetByResources(
	ctx context.Context,
	req *customfieldports.GetCustomFieldValuesByResourcesRequest,
) (map[string][]*customfield.CustomFieldValue, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[string][]*customfield.CustomFieldValue, error) {
			if len(req.ResourceIDs) == 0 {
				return make(map[string][]*customfield.CustomFieldValue), nil
			}

			v := customfielddb.CustomFieldValues
			values, err := v.Where(
				v.ResourceType.Eq(req.ResourceType),
				v.ResourceID.In(req.ResourceIDs...),
			).
				Load(v.Definition()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error(
					"failed to get custom field values for resources",
					zap.String("resourceType", req.ResourceType),
					zap.Int("resourceCount", len(req.ResourceIDs)),
					zap.Error(err),
				)
				return nil, err
			}

			result := make(map[string][]*customfield.CustomFieldValue)
			for _, value := range values {
				result[value.ResourceID] = append(result[value.ResourceID], value)
			}

			return result, nil
		},
	)
}

func (r *repository) Upsert(
	ctx context.Context,
	req *customfieldports.UpsertCustomFieldValuesRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		log := r.l.With(
			zap.String("resourceType", req.ResourceType),
			zap.String("resourceID", req.ResourceID),
			zap.Int("valueCount", len(req.Values)),
		)
		db := r.db.Keel(ctx)

		if _, err := resourceValues(req.ResourceType, req.ResourceID).Delete(ctx, db); err != nil {
			log.Error("failed to delete existing custom field values", zap.Error(err))
			return dberror.MapRetryableTransactionError(err, busyMessage)
		}

		values := make([]*customfield.CustomFieldValue, 0, len(req.Values))
		for definitionID, value := range req.Values {
			defID, parseErr := pulid.Parse(definitionID)
			if parseErr != nil {
				log.Warn(
					"invalid definition ID, skipping",
					zap.String("definitionID", definitionID),
				)
				continue
			}

			values = append(values, &customfield.CustomFieldValue{
				OrganizationID: req.TenantInfo.OrgID,
				BusinessUnitID: req.TenantInfo.BuID,
				DefinitionID:   defID,
				ResourceType:   req.ResourceType,
				ResourceID:     req.ResourceID,
				Value:          value,
			})
		}

		if err := customfielddb.CustomFieldValues.CreateMany(ctx, db, values); err != nil {
			log.Error("failed to insert custom field values", zap.Error(err))
			return dberror.MapRetryableTransactionError(err, busyMessage)
		}

		return nil
	})
}

func (r *repository) DeleteByResource(
	ctx context.Context,
	req *customfieldports.GetCustomFieldValuesByResourceRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		_, err := resourceValues(req.ResourceType, req.ResourceID).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error(
				"failed to delete custom field values",
				zap.String("resourceType", req.ResourceType),
				zap.String("resourceID", req.ResourceID),
				zap.Error(err),
			)
			return err
		}

		return nil
	})
}

func (r *repository) CountByDefinition(
	ctx context.Context,
	req *customfieldports.GetValuesByDefinitionRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		v := customfielddb.CustomFieldValues
		count, err := v.Where(v.DefinitionID.Eq(req.DefinitionID)).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error(
				"failed to count custom field values by definition",
				zap.String("definitionID", req.DefinitionID.String()),
				zap.Error(err),
			)
			return 0, err
		}

		return int(count), nil
	})
}

func (r *repository) CountResourcesByDefinition(
	ctx context.Context,
	req *customfieldports.GetValuesByDefinitionRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		v := customfielddb.CustomFieldValues
		count, err := keel.IntoOne[int64](
			ctx,
			r.db.Keel(ctx),
			v.Pick(v.ResourceID.CountDistinct()).Where(v.DefinitionID.Eq(req.DefinitionID)),
		)
		if err != nil {
			r.l.Error(
				"failed to count resources by definition",
				zap.String("definitionID", req.DefinitionID.String()),
				zap.Error(err),
			)
			return 0, err
		}

		return int(count), nil
	})
}

func (r *repository) GetOptionUsageCounts(
	ctx context.Context,
	req *customfieldports.GetOptionUsageRequest,
) (map[string]int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[string]int, error) {
		v := customfielddb.CustomFieldValues
		counts, err := keel.Into[optionCount](
			ctx,
			r.db.Keel(ctx),
			v.Where(v.DefinitionID.Eq(req.DefinitionID)).
				Group(v.Value).
				Pick(keel.CastText(v.Value).TrimChars(`"`), keel.Count()),
		)
		if err != nil {
			r.l.Error(
				"failed to get option usage counts",
				zap.String("definitionID", req.DefinitionID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		result := make(map[string]int, len(counts))
		for _, c := range counts {
			result[c.Value] = int(c.Count)
		}

		return result, nil
	})
}
