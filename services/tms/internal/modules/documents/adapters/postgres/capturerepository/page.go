package capturerepository

import (
	"context"

	"github.com/bytedance/sonic"
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture/capturedb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

var pageErrors = dberror.Mapping{Entity: "Capture page"}

type pageRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewPageRepository(p Params) captureports.CapturePageRepository {
	return &pageRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.capture-page-repository"),
	}
}

// Insert relies on the (batch, sequence) unique index rather than a read
// first: two retries of the same page racing each other both reach the index,
// and exactly one of them wins.
func (r *pageRepository) Insert(
	ctx context.Context,
	entity *capture.CapturePage,
) (*capture.CapturePage, bool, error) {
	return dbtx.Write2(ctx, r.db, func(ctx context.Context) (*capture.CapturePage, bool, error) {
		p := capturedb.CapturePages
		db := r.db.Keel(ctx)
		inserted, err := p.CreateOrSkip(
			ctx,
			db,
			[]*capture.CapturePage{entity},
			p.BatchID,
			p.BusinessUnitID,
			p.OrganizationID,
			p.Sequence,
		)
		if err != nil {
			return nil, false, err
		}
		if inserted > 0 {
			return entity, true, nil
		}

		existing, err := bySequence(entity.BatchID, entity.Sequence).One(ctx, db)
		if err != nil {
			return nil, false, pageErrors.Map(err, entity.BatchID.String())
		}

		return existing, false, nil
	})
}

func (r *pageRepository) Update(
	ctx context.Context,
	entity *capture.CapturePage,
) (*capture.CapturePage, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*capture.CapturePage, error) {
		if err := capturedb.CapturePages.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, pageErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *pageRepository) RecordInspection(
	ctx context.Context,
	entity *capture.CapturePage,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		markers, err := sonic.MarshalString(entity.Markers)
		if err != nil {
			return false, err
		}

		p := capturedb.CapturePages
		blankScore := p.BlankScore.ToNull()
		if entity.BlankScore != nil {
			blankScore = p.BlankScore.To(*entity.BlankScore)
		}

		n, err := p.Where(p.ID.Eq(entity.ID), p.Status.Eq(capture.PageReceived)).Update(
			ctx,
			r.db.Keel(ctx),
			p.Status.To(entity.Status),
			p.WidthPx.To(int64(entity.WidthPx)),
			p.HeightPx.To(int64(entity.HeightPx)),
			p.ThumbnailPath.To(entity.ThumbnailPath),
			blankScore,
			p.Markers.ToExpr(keel.DynOf(markers).Cast(keel.TypeJSON)),
			p.FailureMessage.To(entity.FailureMessage),
		)
		if err != nil {
			return false, err
		}

		return n == 1, nil
	})
}

func (r *pageRepository) GetByID(
	ctx context.Context,
	req captureports.GetCapturePageByIDRequest,
) (*capture.CapturePage, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*capture.CapturePage, error) {
		entity, err := capturedb.CapturePages.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, pageErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *pageRepository) GetBySequence(
	ctx context.Context,
	req captureports.GetCapturePageBySequenceRequest,
) (*capture.CapturePage, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*capture.CapturePage, error) {
		entity, err := bySequence(req.BatchID, req.Sequence).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, pageErrors.Map(err, req.BatchID.String())
		}

		return entity, nil
	})
}

func bySequence(batchID pulid.ID, sequence int) *keel.Query[capture.CapturePage] {
	p := capturedb.CapturePages

	return p.Where(p.BatchID.Eq(batchID), p.Sequence.Eq(int64(sequence)))
}

func (r *pageRepository) ListByBatch(
	ctx context.Context,
	req captureports.ListCapturePagesRequest,
) ([]*capture.CapturePage, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*capture.CapturePage, error) {
		p := capturedb.CapturePages
		entities, err := p.Where(p.BatchID.Eq(req.BatchID)).
			Order(p.Sequence.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}
		if entities == nil {
			entities = []*capture.CapturePage{}
		}

		return entities, nil
	})
}
