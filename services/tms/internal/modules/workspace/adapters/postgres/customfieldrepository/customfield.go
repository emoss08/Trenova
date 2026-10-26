package customfieldrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/customfield"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/customfield/customfielddb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/customfieldports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/resultcache"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var definitionErrors = dberror.Mapping{Entity: "CustomFieldDefinition"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) customfieldports.CustomFieldDefinitionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.custom-field-definition-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *customfieldports.ListCustomFieldDefinitionsRequest,
) (*pagination.ListResult[*customfield.CustomFieldDefinition], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*customfield.CustomFieldDefinition], error) {
			d := customfielddb.CustomFieldDefinitions
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				d.Where(keel.When(req.ResourceType != "", d.ResourceType.Eq(req.ResourceType))),
				&listquery.Request[customfield.CustomFieldDefinition]{
					Filter: req.Filter,
					Then:   []keel.Order{d.DisplayOrder.Asc(), d.CreatedAt.Desc()},
				},
			)
			if err != nil {
				r.l.Error("failed to list custom field definitions", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *customfieldports.ListCustomFieldDefinitionConnectionRequest,
) (*pagination.CursorListResult[*customfield.CustomFieldDefinition], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*customfield.CustomFieldDefinition], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				customfielddb.CustomFieldDefinitions.Query(),
				&listquery.Request[customfield.CustomFieldDefinition]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.CustomFieldDefinitionColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list custom field definition connection", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req customfieldports.GetCustomFieldDefinitionByIDRequest,
) (*customfield.CustomFieldDefinition, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*customfield.CustomFieldDefinition, error) {
			entity, err := customfielddb.CustomFieldDefinitions.Get(ctx, r.db.Keel(ctx), req.ID)
			if err != nil {
				r.l.Error(
					"failed to get custom field definition",
					zap.String("id", req.ID.String()),
					zap.Error(err),
				)
				return nil, definitionErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) GetActiveByResourceType(
	ctx context.Context,
	req customfieldports.GetActiveByResourceTypeRequest,
) ([]*customfield.CustomFieldDefinition, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*customfield.CustomFieldDefinition, error) {
			d := customfielddb.CustomFieldDefinitions
			entities, err := d.Where(d.ResourceType.Eq(req.ResourceType), d.IsActive.IsTrue()).
				Order(d.DisplayOrder.Asc()).
				Cache(resultcache.Settings()...).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error(
					"failed to get active custom field definitions by resource type",
					zap.String("resourceType", req.ResourceType),
					zap.Error(err),
				)
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *customfield.CustomFieldDefinition,
) (*customfield.CustomFieldDefinition, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*customfield.CustomFieldDefinition, error) {
			err := customfielddb.CustomFieldDefinitions.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error(
					"failed to create custom field definition",
					zap.String("name", entity.Name),
					zap.String("resourceType", entity.ResourceType),
					zap.Error(err),
				)
				return nil, err
			}

			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *customfield.CustomFieldDefinition,
) (*customfield.CustomFieldDefinition, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*customfield.CustomFieldDefinition, error) {
			err := customfielddb.CustomFieldDefinitions.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error(
					"failed to update custom field definition",
					zap.String("id", entity.ID.String()),
					zap.Error(err),
				)
				return nil, definitionErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Delete(
	ctx context.Context,
	req customfieldports.GetCustomFieldDefinitionByIDRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		d := customfielddb.CustomFieldDefinitions
		rows, err := d.Where(d.ID.Eq(req.ID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error(
				"failed to delete custom field definition",
				zap.String("id", req.ID.String()),
				zap.Error(err),
			)
			return err
		}

		return dberror.CheckAffected(rows, "CustomFieldDefinition", req.ID.String())
	})
}

func (r *repository) CountByResourceType(
	ctx context.Context,
	req customfieldports.CountByResourceTypeRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		d := customfielddb.CustomFieldDefinitions
		count, err := d.Where(d.ResourceType.Eq(req.ResourceType)).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error(
				"failed to count custom field definitions by resource type",
				zap.String("resourceType", req.ResourceType),
				zap.Error(err),
			)
			return 0, err
		}

		return int(count), nil
	})
}
