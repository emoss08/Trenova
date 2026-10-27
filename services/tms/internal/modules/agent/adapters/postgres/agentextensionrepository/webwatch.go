package agentextensionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension/agentextensiondb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	webWatchEntityName   = "Web watch"
	defaultWebWatches    = 50
	maxWebWatches        = 200
	defaultWatchFindings = 20
	maxWatchFindings     = 100
	maxUndelivered       = 50
)

var webWatchErrors = dberror.Mapping{Entity: webWatchEntityName}

type webWatchRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewWebWatchRepository(p Params) agentextensionports.WebWatchRepository {
	return &webWatchRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agent-web-watch-repository"),
	}
}

func (r *webWatchRepository) Create(
	ctx context.Context,
	watch *agentextension.WebWatch,
) (*agentextension.WebWatch, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentextension.WebWatch, error) {
		if err := agentextensiondb.WebWatches.Create(ctx, r.db.Keel(ctx), watch); err != nil {
			r.l.Error("failed to create web watch", zap.Error(err))
			return nil, fmt.Errorf("create web watch: %w", err)
		}

		return watch, nil
	})
}

func (r *webWatchRepository) Get(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (*agentextension.WebWatch, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentextension.WebWatch, error) {
		watch, err := agentextensiondb.WebWatches.Get(ctx, r.db.Keel(ctx), id)
		if err != nil {
			return nil, webWatchErrors.Map(err, id.String())
		}

		return watch, nil
	})
}

func (r *webWatchRepository) Update(
	ctx context.Context,
	watch *agentextension.WebWatch,
) (*agentextension.WebWatch, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentextension.WebWatch, error) {
		if err := agentextensiondb.WebWatches.Save(ctx, r.db.Keel(ctx), watch); err != nil {
			if dberror.IsVersionConflict(err) {
				return nil, webWatchErrors.Map(err, watch.ID.String())
			}
			r.l.Error("failed to update web watch", zap.Error(err))
			return nil, fmt.Errorf("update web watch: %w", err)
		}

		return watch, nil
	})
}

func (r *webWatchRepository) List(
	ctx context.Context,
	req *agentextensionports.ListWebWatchesRequest,
) ([]*agentextension.WebWatch, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultWebWatches
	}
	limit = min(limit, maxWebWatches)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentextension.WebWatch, error) {
		w := agentextensiondb.WebWatches
		watches, err := w.Where(
			keel.When(req.Status != "", w.Status.Eq(req.Status)),
			keel.When(req.ThreadID.IsNotNil(), w.ThreadID.Eq(req.ThreadID)),
			keel.When(req.RequestedBy.IsNotNil(), w.RequestedByID.Eq(req.RequestedBy)),
			keel.When(req.DefinitionID.IsNotNil(), w.AgentDefinitionID.Eq(req.DefinitionID)),
			keel.When(req.Before > 0, w.CreatedAt.Lt(req.Before)),
		).
			Order(w.CreatedAt.Desc(), w.ID.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list web watches", zap.Error(err))
			return nil, fmt.Errorf("list web watches: %w", err)
		}

		return watches, nil
	})
}

func (r *webWatchRepository) CountActive(
	ctx context.Context,
	_ pagination.TenantInfo,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		w := agentextensiondb.WebWatches
		count, err := w.Where(w.Status.Eq(agentextension.WebWatchStatusActive)).
			Count(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to count active web watches", zap.Error(err))
			return 0, fmt.Errorf("count active web watches: %w", err)
		}

		return int(count), nil
	})
}

type webWatchFindingRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewWebWatchFindingRepository(p Params) agentextensionports.WebWatchFindingRepository {
	return &webWatchFindingRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agent-web-watch-finding-repository"),
	}
}

func (r *webWatchFindingRepository) Create(
	ctx context.Context,
	finding *agentextension.WebWatchFinding,
) (*agentextension.WebWatchFinding, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*agentextension.WebWatchFinding, error) {
			err := agentextensiondb.WebWatchFindings.Create(ctx, r.db.Keel(ctx), finding)
			if err != nil {
				if dberror.IsUniqueConstraintViolation(err) {
					return nil, agentextensionports.ErrWebWatchFindingExists
				}
				r.l.Error("failed to record a web watch finding", zap.Error(err))
				return nil, fmt.Errorf("create web watch finding: %w", err)
			}

			return finding, nil
		},
	)
}

func (r *webWatchFindingRepository) ListByWatch(
	ctx context.Context,
	_ pagination.TenantInfo,
	watchID pulid.ID,
	limit int,
) ([]*agentextension.WebWatchFinding, error) {
	if limit <= 0 {
		limit = defaultWatchFindings
	}
	limit = min(limit, maxWatchFindings)

	return r.list(ctx, &findingQuery{
		watchID: watchID,
		limit:   limit,
		newest:  true,
	})
}

func (r *webWatchFindingRepository) ListUndelivered(
	ctx context.Context,
	_ pagination.TenantInfo,
	watchID pulid.ID,
) ([]*agentextension.WebWatchFinding, error) {
	return r.list(ctx, &findingQuery{
		watchID:     watchID,
		limit:       maxUndelivered,
		undelivered: true,
	})
}

type findingQuery struct {
	watchID     pulid.ID
	limit       int
	undelivered bool
	newest      bool
}

func (r *webWatchFindingRepository) list(
	ctx context.Context,
	q *findingQuery,
) ([]*agentextension.WebWatchFinding, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*agentextension.WebWatchFinding, error) {
			f := agentextensiondb.WebWatchFindings
			order := f.FoundAt.Asc()
			if q.newest {
				order = f.FoundAt.Desc()
			}

			findings, err := f.Where(
				f.WatchID.Eq(q.watchID),
				keel.When(q.undelivered, f.DeliveredAt.IsNull()),
			).
				Order(order, f.ID.Asc()).
				Limit(q.limit).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list web watch findings", zap.Error(err))
				return nil, fmt.Errorf("list web watch findings: %w", err)
			}

			return findings, nil
		},
	)
}

func (r *webWatchFindingRepository) MarkDelivered(
	ctx context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
	at int64,
) error {
	if len(ids) == 0 {
		return nil
	}

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		f := agentextensiondb.WebWatchFindings
		_, err := f.Where(f.ID.In(ids...), f.DeliveredAt.IsNull()).
			Update(ctx, r.db.Keel(ctx), f.DeliveredAt.To(at))
		if err != nil {
			r.l.Error("failed to mark web watch findings delivered", zap.Error(err))
			return fmt.Errorf("mark web watch findings delivered: %w", err)
		}

		return nil
	})
}
