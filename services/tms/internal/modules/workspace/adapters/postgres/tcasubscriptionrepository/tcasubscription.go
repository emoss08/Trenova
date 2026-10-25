package tcasubscriptionrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tablechangealert"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tablechangealert/tablechangealertdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/tablechangealertports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const entityName = "TCASubscription"

var subscriptionErrors = dberror.Mapping{Entity: entityName}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) tablechangealertports.TCASubscriptionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.tca-subscription-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *tablechangealertports.ListTCASubscriptionsRequest,
) (*pagination.ListResult[*tablechangealert.TCASubscription], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*tablechangealert.TCASubscription], error) {
			s := tablechangealertdb.TCASubscriptions
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				s.Where(s.UserID.Eq(req.Filter.TenantInfo.UserID)),
				&listquery.Request[tablechangealert.TCASubscription]{
					Filter: req.Filter,
					Then:   []keel.Order{s.CreatedAt.Desc()},
				},
			)
			if err != nil {
				r.l.Error("failed to list tca subscriptions", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *tablechangealertports.ListTCASubscriptionConnectionRequest,
) (*pagination.CursorListResult[*tablechangealert.TCASubscription], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*tablechangealert.TCASubscription], error) {
			s := tablechangealertdb.TCASubscriptions
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				s.Where(s.UserID.Eq(req.Filter.TenantInfo.UserID)),
				&listquery.Request[tablechangealert.TCASubscription]{
					Filter: req.Filter,
					Cursor: req.Cursor,
				},
			)
			if err != nil {
				r.l.Error("failed to list tca subscription connection", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *tablechangealert.TCASubscription,
) (*tablechangealert.TCASubscription, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*tablechangealert.TCASubscription, error) {
			err := tablechangealertdb.TCASubscriptions.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to create tca subscription", zap.Error(err))
				return nil, err
			}

			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *tablechangealert.TCASubscription,
) (*tablechangealert.TCASubscription, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*tablechangealert.TCASubscription, error) {
			s := tablechangealertdb.TCASubscriptions
			db := r.db.Keel(ctx)

			owned, err := s.Where(s.ID.Eq(entity.ID), s.UserID.Eq(entity.UserID)).Exists(ctx, db)
			if err == nil && !owned {
				return nil, dberror.CreateVersionMismatchError(entityName, entity.ID.String())
			}
			if err == nil {
				err = s.Save(ctx, db, entity)
			}
			if err != nil {
				r.l.Error(
					"failed to update tca subscription",
					zap.String("id", entity.ID.String()),
					zap.Error(err),
				)
				return nil, subscriptionErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req tablechangealertports.GetTCASubscriptionByIDRequest,
) (*tablechangealert.TCASubscription, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*tablechangealert.TCASubscription, error) {
			s := tablechangealertdb.TCASubscriptions
			entity, err := s.Where(
				s.ID.Eq(req.SubscriptionID),
				s.UserID.Eq(req.TenantInfo.UserID),
			).One(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error(
					"failed to get tca subscription",
					zap.String("subscriptionID", req.SubscriptionID.String()),
					zap.Error(err),
				)
				return nil, subscriptionErrors.Map(err, req.SubscriptionID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Delete(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		s := tablechangealertdb.TCASubscriptions
		rows, err := s.Where(s.ID.Eq(id), s.UserID.Eq(tenantInfo.UserID)).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error(
				"failed to delete tca subscription",
				zap.String("id", id.String()),
				zap.Error(err),
			)
			return err
		}

		return dberror.CheckAffected(rows, entityName, id.String())
	})
}

func (r *repository) FindMatchingSubscriptions(
	ctx context.Context,
	req tablechangealertports.FindMatchingTCASubscriptionsRequest,
) ([]*tablechangealert.TCASubscription, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*tablechangealert.TCASubscription, error) {
			if !tablechangealert.ValidEventType(req.Operation) {
				return nil, nil
			}

			s := tablechangealertdb.TCASubscriptions
			anyRecord := keel.Or(s.RecordID.IsNull(), s.RecordID.Eq(""))
			record := anyRecord
			if req.RecordID != "" {
				record = keel.Or(anyRecord, s.RecordID.Eq(req.RecordID))
			}

			entities, err := s.Where(
				s.TableName.Eq(req.TableName),
				s.Status.Eq(tablechangealert.SubscriptionStatusActive),
				s.EventTypes.Contains([]string{req.Operation}),
				record,
			).All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error(
					"failed to find matching subscriptions",
					zap.String("tableName", req.TableName),
					zap.String("operationType", req.Operation),
					zap.Error(err),
				)
				return nil, err
			}

			return entities, nil
		},
	)
}
