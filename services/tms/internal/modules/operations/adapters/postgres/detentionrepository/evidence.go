package detentionrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention/detentiondb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type EvidenceParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type evidenceRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewEvidenceRepository(p EvidenceParams) detentionports.DetentionEvidenceRepository {
	return &evidenceRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.detention-evidence-repository"),
	}
}

// Append links a new fact onto the occurrence's chain. Reading the head and
// writing the link happen in one transaction so concurrent writers cannot fork
// the chain, and the occurrence's cached head advances with it.
func (r *evidenceRepository) Append(
	ctx context.Context,
	req *detentionports.AppendEvidenceRequest,
) (*detention.DetentionEvidence, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*detention.DetentionEvidence, error) {
		entry := req.Entry
		log := r.l.With(
			zap.String("operation", "Append"),
			zap.String("occurrenceId", entry.DetentionOccurrenceID.String()),
		)

		db := r.db.Keel(ctx)
		head, sequence, err := latestLink(
			ctx,
			db,
			entry.DetentionOccurrenceID,
			keel.ForUpdate,
		)
		if err != nil {
			log.Error("failed to read detention evidence head", zap.Error(err))
			return nil, err
		}

		entry.PrevHash = head
		entry.Sequence = sequence
		entry.Seal()

		if err = detentiondb.DetentionEvidences.Create(ctx, db, entry); err != nil {
			log.Error("failed to append detention evidence", zap.Error(err))
			return nil, evidenceErrors.Map(err, entry.ID.String())
		}

		o := detentiondb.DetentionOccurrences
		if _, err = o.Where(o.ID.Eq(entry.DetentionOccurrenceID)).Update(
			ctx,
			db,
			o.EvidenceHead.To(entry.Hash),
			o.Version.ToExpr(o.Version),
			o.UpdatedAt.ToExpr(o.UpdatedAt),
		); err != nil {
			log.Error("failed to advance detention evidence head", zap.Error(err))
			return nil, err
		}

		return entry, nil
	})
}

func latestLink(
	ctx context.Context,
	db *keel.DB,
	occurrenceID pulid.ID,
	lock ...keel.LockStrength,
) (string, int32, error) {
	e := detentiondb.DetentionEvidences
	q := e.Pick(e.Hash, e.Sequence).
		Where(e.DetentionOccurrenceID.Eq(occurrenceID)).
		Order(e.Sequence.Desc())
	if len(lock) > 0 {
		q = q.Lock(lock[0])
	}

	latest, err := q.First(ctx, db)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return detention.GenesisHash, 0, nil
		}
		return "", 0, err
	}

	return latest.Hash, latest.Sequence + 1, nil
}

func (r *evidenceRepository) List(
	ctx context.Context,
	req *detentionports.ListEvidenceRequest,
) ([]*detention.DetentionEvidence, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*detention.DetentionEvidence, error) {
		e := detentiondb.DetentionEvidences
		entities, err := e.Where(e.DetentionOccurrenceID.Eq(req.OccurrenceID)).
			Order(e.Sequence.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list detention evidence", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *evidenceRepository) Head(
	ctx context.Context,
	req *detentionports.ListEvidenceRequest,
) (string, int32, error) {
	return dbtx.Read2(ctx, r.db, func(ctx context.Context) (string, int32, error) {
		return latestLink(ctx, r.db.Keel(ctx), req.OccurrenceID)
	})
}
