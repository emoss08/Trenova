package detentionrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
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

func NewEvidenceRepository(p EvidenceParams) repositories.DetentionEvidenceRepository {
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
	req *repositories.AppendEvidenceRequest,
) (*detention.DetentionEvidence, error) {
	entry := req.Entry
	log := r.l.With(
		zap.String("operation", "Append"),
		zap.String("occurrenceId", entry.DetentionOccurrenceID.String()),
	)

	tenantInfo := pagination.TenantInfo{
		OrgID: entry.OrganizationID,
		BuID:  entry.BusinessUnitID,
	}

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		head, sequence, hErr := r.headForUpdate(c, entry.DetentionOccurrenceID, tenantInfo)
		if hErr != nil {
			return hErr
		}

		entry.PrevHash = head
		entry.Sequence = sequence
		entry.Seal()

		if _, iErr := r.db.DBForContext(c).
			NewInsert().
			Model(entry).
			Returning("*").
			Exec(c); iErr != nil {
			return iErr
		}

		occCols := buncolgen.DetentionOccurrenceColumns
		_, uErr := r.db.DBForContext(c).
			NewUpdate().
			Model((*detention.DetentionOccurrence)(nil)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.DetentionOccurrenceScopeTenantUpdate(uq, tenantInfo).
					Where(occCols.ID.Eq(), entry.DetentionOccurrenceID)
			}).
			Set(occCols.EvidenceHead.Set(), entry.Hash).
			Exec(c)

		return uErr
	})
	if err != nil {
		log.Error("failed to append detention evidence", zap.Error(err))
		return nil, err
	}

	return entry, nil
}

func (r *evidenceRepository) headForUpdate(
	ctx context.Context,
	occurrenceID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (string, int32, error) {
	cols := buncolgen.DetentionEvidenceColumns
	latest := new(detention.DetentionEvidence)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(latest).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DetentionEvidenceScopeTenant(sq, tenantInfo).
				Where(cols.DetentionOccurrenceID.Eq(), occurrenceID)
		}).
		Order(cols.Sequence.OrderDesc()).
		Limit(1).
		For("UPDATE").
		Scan(ctx)
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
	req *repositories.ListEvidenceRequest,
) ([]*detention.DetentionEvidence, error) {
	cols := buncolgen.DetentionEvidenceColumns
	entities := make([]*detention.DetentionEvidence, 0, 16)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DetentionEvidenceScopeTenant(sq, req.TenantInfo).
				Where(cols.DetentionOccurrenceID.Eq(), req.OccurrenceID)
		}).
		Order(cols.Sequence.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list detention evidence", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *evidenceRepository) Head(
	ctx context.Context,
	req *repositories.ListEvidenceRequest,
) (string, int32, error) {
	cols := buncolgen.DetentionEvidenceColumns
	latest := new(detention.DetentionEvidence)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(latest).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DetentionEvidenceScopeTenant(sq, req.TenantInfo).
				Where(cols.DetentionOccurrenceID.Eq(), req.OccurrenceID)
		}).
		Order(cols.Sequence.OrderDesc()).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return detention.GenesisHash, 0, nil
		}
		return "", 0, err
	}

	return latest.Hash, latest.Sequence + 1, nil
}
