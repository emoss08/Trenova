package bulkeditrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/bulkedit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/bulkedit/bulkeditdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/bulkeditports"
	"github.com/emoss08/trenova/pkg/dberror"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxListed = 50

var bulkEditErrors = dberror.Mapping{Entity: "Bulk edit"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) bulkeditports.BulkEditRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.bulk-edit-repository")}
}

func (r *repository) Create(
	ctx context.Context,
	entity *bulkedit.BulkEdit,
) (*bulkedit.BulkEdit, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*bulkedit.BulkEdit, error) {
		if err := bulkeditdb.BulkEdits.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create bulk edit", zap.Error(err))
			return nil, err
		}
		return entity, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req *bulkeditports.GetBulkEditRequest,
) (*bulkedit.BulkEdit, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*bulkedit.BulkEdit, error) {
		entity, err := bulkeditdb.BulkEdits.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, bulkEditErrors.Map(err, req.ID.String())
		}
		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *bulkedit.BulkEdit,
) (*bulkedit.BulkEdit, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*bulkedit.BulkEdit, error) {
		if err := bulkeditdb.BulkEdits.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update bulk edit", zap.Error(err))
			return nil, bulkEditErrors.Map(err, entity.ID.String())
		}
		return entity, nil
	})
}

func (r *repository) ListForUser(
	ctx context.Context,
	req *bulkeditports.ListBulkEditsRequest,
) ([]*bulkedit.BulkEdit, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*bulkedit.BulkEdit, error) {
		b := bulkeditdb.BulkEdits
		limit := req.Limit
		if limit <= 0 || limit > maxListed {
			limit = maxListed
		}

		entities, err := b.Omit(b.Targets).
			Where(
				b.UserID.Eq(req.TenantInfo.UserID),
				keel.When(req.Resource != "", b.Resource.Eq(req.Resource)),
			).
			Order(b.CreatedAt.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list bulk edits", zap.Error(err))
			return nil, err
		}
		return entities, nil
	})
}
