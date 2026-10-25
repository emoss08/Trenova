//nolint:gocritic // existing value-shaped APIs and hot-path helpers are intentionally stable
package journalsourcerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalsource"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalsource/journalsourcedb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalsourceports"
	"github.com/emoss08/trenova/pkg/dberror"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxSourcesPerObject = 1000

var sourceErrors = dberror.Mapping{Entity: "JournalSource"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) journalsourceports.JournalSourceRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.journal-source-repository")}
}

func (r *repository) GetByObject(
	ctx context.Context,
	req journalsourceports.GetJournalSourceByObjectRequest,
) (*journalsource.Source, error) {
	return r.getByObjectQuery(ctx, req, "")
}

func (r *repository) GetByObjectAndEvent(
	ctx context.Context,
	req journalsourceports.GetJournalSourceByObjectRequest,
	sourceEventType string,
) (*journalsource.Source, error) {
	return r.getByObjectQuery(ctx, req, sourceEventType)
}

func (r *repository) ListByObject(
	ctx context.Context,
	req journalsourceports.GetJournalSourceByObjectRequest,
) ([]*journalsource.Source, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*journalsource.Source, error) {
		js := journalsourcedb.Sources
		return js.Where(
			js.SourceObjectType.Eq(req.SourceObjectType),
			js.SourceObjectID.Eq(req.SourceObjectID),
		).
			Order(js.CreatedAt.Asc()).
			Limit(maxSourcesPerObject).
			List(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) getByObjectQuery(
	ctx context.Context,
	req journalsourceports.GetJournalSourceByObjectRequest,
	sourceEventType string,
) (*journalsource.Source, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*journalsource.Source, error) {
		js := journalsourcedb.Sources
		source, err := js.Where(
			js.SourceObjectType.Eq(req.SourceObjectType),
			js.SourceObjectID.Eq(req.SourceObjectID),
			keel.When(sourceEventType != "", js.SourceEventType.Eq(sourceEventType)),
		).
			Order(js.CreatedAt.Desc()).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, sourceErrors.Map(err, req.SourceObjectID)
		}

		return source, nil
	})
}

func (r *repository) GetByJournalEntry(
	ctx context.Context,
	req journalsourceports.GetJournalSourceByEntryRequest,
) (*journalsource.Source, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*journalsource.Source, error) {
		js := journalsourcedb.Sources
		jb := js.JournalBatch()
		source, err := js.Where(js.JournalEntryID.Eq(req.JournalEntryID)).
			Load(jb.Pick(jb.ID, jb.BatchNumber, jb.BatchType)).
			Order(js.CreatedAt.Desc()).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, sourceErrors.Map(err, req.JournalEntryID.String())
		}

		return source, nil
	})
}
