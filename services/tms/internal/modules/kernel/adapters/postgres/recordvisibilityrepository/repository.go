package recordvisibilityrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/keelkinds"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

var ErrUnscopedKind = errors.New("record kind has no database read scope")

type Params struct {
	fx.In

	DB      *postgres.Connection
	Sources kernelports.VisibleRecordSources
}

type repository struct {
	db      *postgres.Connection
	sources map[string]keelkinds.PulidIDCol
}

type visibleRecord struct {
	ID pulid.ID
}

func New(p Params) kernelports.RecordVisibility {
	sources := make(map[string]keelkinds.PulidIDCol, len(p.Sources))
	for _, source := range p.Sources {
		sources[source.Kind] = source.ID
	}
	return &repository{db: p.DB, sources: sources}
}

func (r *repository) Scoped(kind string) bool {
	_, ok := r.sources[kind]
	return ok
}

func (r *repository) VisibleIDs(
	ctx context.Context,
	req *kernelports.ListVisibleRecordsRequest,
) ([]pulid.ID, error) {
	id, ok := r.sources[req.Kind]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnscopedKind, req.Kind)
	}
	if len(req.IDs) == 0 {
		return []pulid.ID{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		records, err := keel.Into[visibleRecord](
			ctx,
			r.db.Keel(ctx),
			keel.Handle[*keel.Row]{Ref: id.Ref}.Pick(id).Where(id.In(req.IDs...)),
		)
		if err != nil {
			return nil, fmt.Errorf("check which %s records are visible: %w", req.Kind, err)
		}
		visible := make([]pulid.ID, 0, len(records))
		for _, record := range records {
			visible = append(visible, record.ID)
		}
		return visible, nil
	})
}
