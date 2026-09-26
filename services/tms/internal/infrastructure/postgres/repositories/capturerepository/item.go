package capturerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type itemRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewItemRepository(p Params) repositories.CaptureItemRepository {
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
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *itemRepository) Update(
	ctx context.Context,
	entity *capture.CaptureItem,
) (*capture.CaptureItem, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.CaptureItemColumns.Version.Eq(), ov).
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
}

func (r *itemRepository) GetByID(
	ctx context.Context,
	req repositories.GetCaptureItemByIDRequest,
) (*capture.CaptureItem, error) {
	entity := new(capture.CaptureItem)

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Apply(buncolgen.CaptureItemApplyTenant(req.TenantInfo)).
		Where(buncolgen.CaptureItemColumns.ID.Eq(), req.ID).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Capture item")
	}

	return entity, nil
}

func (r *itemRepository) ListByBatch(
	ctx context.Context,
	req repositories.ListCaptureItemsRequest,
) ([]*capture.CaptureItem, error) {
	entities := make([]*capture.CaptureItem, 0)
	cols := buncolgen.CaptureItemColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(buncolgen.CaptureItemApplyTenant(req.TenantInfo)).
		Where(cols.BatchID.Eq(), req.BatchID).
		Order(cols.Position.OrderAsc()).
		Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *itemRepository) ReplaceOpen(
	ctx context.Context,
	req *repositories.ReplaceOpenCaptureItemsRequest,
) error {
	cols := buncolgen.CaptureItemColumns

	return r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().
			Model((*capture.CaptureItem)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.CaptureItemScopeTenantDelete(dq, req.TenantInfo).
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
}
