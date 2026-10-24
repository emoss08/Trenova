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

var coverSheetErrors = dberror.Mapping{Entity: "Cover sheet"}

type coverSheetRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewCoverSheetRepository(p Params) captureports.CaptureCoverSheetRepository {
	return &coverSheetRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.capture-cover-sheet-repository"),
	}
}

func (r *coverSheetRepository) CreateMany(
	ctx context.Context,
	entities []*capture.CaptureCoverSheet,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(entities) == 0 {
			return nil
		}

		return capturedb.CaptureCoverSheets.CreateMany(ctx, r.db.Keel(ctx), entities)
	})
}

func (r *coverSheetRepository) GetByID(
	ctx context.Context,
	req captureports.GetCaptureCoverSheetByIDRequest,
) (*capture.CaptureCoverSheet, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*capture.CaptureCoverSheet, error) {
		entity, err := capturedb.CaptureCoverSheets.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, coverSheetErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *coverSheetRepository) GetByTokenHash(
	ctx context.Context,
	req captureports.GetCaptureCoverSheetByTokenRequest,
) (*capture.CaptureCoverSheet, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*capture.CaptureCoverSheet, error) {
		c := capturedb.CaptureCoverSheets
		entity, err := c.Where(c.TokenHash.Eq(req.TokenHash)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, coverSheetErrors.Map(err, "")
		}

		return entity, nil
	})
}

func (r *coverSheetRepository) MarkUsed(
	ctx context.Context,
	req captureports.MarkCaptureCoverSheetUsedRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		c := capturedb.CaptureCoverSheets
		_, err := c.Where(c.ID.Eq(req.ID)).
			Update(ctx, r.db.Keel(ctx), c.LastUsedAt.To(req.UsedAt), c.UseCount.Add(1))

		return err
	})
}
