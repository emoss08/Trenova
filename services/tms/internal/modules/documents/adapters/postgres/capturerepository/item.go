package capturerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture/capturedb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/pkg/dberror"
	"go.uber.org/zap"
)

var itemErrors = dberror.Mapping{Entity: "Capture item"}

type itemRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewItemRepository(p Params) captureports.CaptureItemRepository {
	return &itemRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.capture-item-repository"),
	}
}

var openItemStatuses = []capture.ItemStatus{capture.ItemProposed, capture.ItemFailed}

func (r *itemRepository) Create(
	ctx context.Context,
	entity *capture.CaptureItem,
) (*capture.CaptureItem, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*capture.CaptureItem, error) {
		if err := capturedb.CaptureItems.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *itemRepository) Update(
	ctx context.Context,
	entity *capture.CaptureItem,
) (*capture.CaptureItem, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*capture.CaptureItem, error) {
		if err := capturedb.CaptureItems.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, itemErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *itemRepository) GetByID(
	ctx context.Context,
	req captureports.GetCaptureItemByIDRequest,
) (*capture.CaptureItem, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*capture.CaptureItem, error) {
		entity, err := capturedb.CaptureItems.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, itemErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *itemRepository) ListByBatch(
	ctx context.Context,
	req captureports.ListCaptureItemsRequest,
) ([]*capture.CaptureItem, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*capture.CaptureItem, error) {
		i := capturedb.CaptureItems
		entities, err := i.Where(i.BatchID.Eq(req.BatchID)).
			Order(i.Position.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}
		if entities == nil {
			entities = []*capture.CaptureItem{}
		}

		return entities, nil
	})
}

func (r *itemRepository) ReplaceOpen(
	ctx context.Context,
	req *captureports.ReplaceOpenCaptureItemsRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		i := capturedb.CaptureItems
		if _, err := i.Where(i.BatchID.Eq(req.BatchID), i.Status.In(openItemStatuses...)).
			Delete(ctx, db); err != nil {
			return err
		}

		if len(req.Items) == 0 {
			return nil
		}

		return i.CreateMany(ctx, db, req.Items)
	})
}
