package capturerepository

import (
	"github.com/emoss08/trenova/pkg/buncolgen/capturecol"
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

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
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); err != nil {
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
		ov := entity.Version
		entity.Version++

		results, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WherePK().
			Where(capturecol.CaptureItemColumns.Version.Eq(), ov).
			Returning("*").
			Exec(ctx)
		if err != nil {
			entity.Version = ov

			return nil, err
		}
		if err = dberror.CheckRowsAffected(results, "Capture item", entity.ID.String()); err != nil {
			entity.Version = ov

			return nil, err
		}

		return entity, nil
	})
}

func (r *itemRepository) GetByID(
	ctx context.Context,
	req captureports.GetCaptureItemByIDRequest,
) (*capture.CaptureItem, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*capture.CaptureItem, error) {
		entity := new(capture.CaptureItem)

		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Apply(capturecol.CaptureItemApplyTenant(req.TenantInfo)).
			Where(capturecol.CaptureItemColumns.ID.Eq(), req.ID).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, "Capture item")
		}

		return entity, nil
	})
}

func (r *itemRepository) ListByBatch(
	ctx context.Context,
	req captureports.ListCaptureItemsRequest,
) ([]*capture.CaptureItem, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*capture.CaptureItem, error) {
		entities := make([]*capture.CaptureItem, 0)
		cols := capturecol.CaptureItemColumns

		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			Apply(capturecol.CaptureItemApplyTenant(req.TenantInfo)).
			Where(cols.BatchID.Eq(), req.BatchID).
			Order(cols.Position.OrderAsc()).
			Scan(ctx); err != nil {
			return nil, err
		}

		return entities, nil
	})
}

func (r *itemRepository) ReplaceOpen(
	ctx context.Context,
	req *captureports.ReplaceOpenCaptureItemsRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := capturecol.CaptureItemColumns

		return r.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
			if _, err := tx.NewDelete().
				Model((*capture.CaptureItem)(nil)).
				WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
					return capturecol.CaptureItemScopeTenantDelete(dq, req.TenantInfo).
						Where(cols.BatchID.Eq(), req.BatchID).
						Where(cols.Status.In(), bun.List(openItemStatuses))
				}).
				Exec(txCtx); err != nil {
				return err
			}

			if len(req.Items) == 0 {
				return nil
			}

			_, err := tx.NewInsert().Model(&req.Items).Returning("*").Exec(txCtx)

			return err
		})
	})
}
