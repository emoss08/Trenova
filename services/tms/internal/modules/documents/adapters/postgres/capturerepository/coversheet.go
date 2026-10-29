package capturerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capturecoversheet"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capturecoversheet/capturecoversheetdb"
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
	entities []*capturecoversheet.CaptureCoverSheet,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(entities) == 0 {
			return nil
		}

		return capturecoversheetdb.CaptureCoverSheets.CreateMany(ctx, r.db.Keel(ctx), entities)
	})
}

func (r *coverSheetRepository) GetByID(
	ctx context.Context,
	req captureports.GetCaptureCoverSheetByIDRequest,
) (*capturecoversheet.CaptureCoverSheet, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*capturecoversheet.CaptureCoverSheet, error) {
			entity, err := capturecoversheetdb.CaptureCoverSheets.Get(ctx, r.db.Keel(ctx), req.ID)
			if err != nil {
				return nil, coverSheetErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *coverSheetRepository) GetByTokenHash(
	ctx context.Context,
	req captureports.GetCaptureCoverSheetByTokenRequest,
) (*capturecoversheet.CaptureCoverSheet, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*capturecoversheet.CaptureCoverSheet, error) {
			c := capturecoversheetdb.CaptureCoverSheets
			entity, err := c.Where(c.TokenHash.Eq(req.TokenHash)).One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, coverSheetErrors.Map(err, "")
			}

			return entity, nil
		},
	)
}

func (r *coverSheetRepository) MarkUsed(
	ctx context.Context,
	req captureports.MarkCaptureCoverSheetUsedRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		c := capturecoversheetdb.CaptureCoverSheets
		_, err := c.Where(c.ID.Eq(req.ID)).
			Update(ctx, r.db.Keel(ctx), c.LastUsedAt.To(req.UsedAt), c.UseCount.Increment(1))

		return err
	})
}
