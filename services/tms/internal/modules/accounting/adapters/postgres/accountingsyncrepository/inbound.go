package accountingsyncrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync/accountingsyncdb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	inboundChangeEntity       = "Accounting inbound change"
	inboundChangeDetectedSort = "detectedAt"
	defaultInboundBatch       = 100
	maxInboundBatch           = 500
)

var inboundChangeErrors = dberror.Mapping{Entity: inboundChangeEntity}

type InboundChangeParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type inboundChangeRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewInboundChangeRepository(
	p InboundChangeParams,
) accountingsyncports.AccountingInboundChangeRepository {
	return &inboundChangeRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-inbound-change-repository"),
	}
}

func (r *inboundChangeRepository) Create(
	ctx context.Context,
	entity *accountingsync.AccountingInboundChange,
) (*accountingsync.AccountingInboundChange, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingInboundChange, error) {
			err := accountingsyncdb.AccountingInboundChanges.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				return nil, fmt.Errorf("create accounting inbound change: %w", err)
			}

			return entity, nil
		},
	)
}

func (r *inboundChangeRepository) Update(
	ctx context.Context,
	entity *accountingsync.AccountingInboundChange,
) (*accountingsync.AccountingInboundChange, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingInboundChange, error) {
			ov := entity.Version
			err := accountingsyncdb.AccountingInboundChanges.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				entity.Version = ov
				return nil, inboundChangeErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *inboundChangeRepository) GetByID(
	ctx context.Context,
	req accountingsyncports.GetAccountingInboundChangeQuery,
) (*accountingsync.AccountingInboundChange, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingInboundChange, error) {
			ic := accountingsyncdb.AccountingInboundChanges
			q := ic.Where(ic.ID.Eq(req.ID))
			if req.ForUpdate {
				q = q.Lock(keel.ForUpdate)
			}

			entity, err := q.One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, inboundChangeErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *inboundChangeRepository) ListByExternalIDs(
	ctx context.Context,
	req *accountingsyncports.ListAccountingInboundChangesByExternalIDsRequest,
) ([]*accountingsync.AccountingInboundChange, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingsync.AccountingInboundChange, error) {
			if len(req.ExternalIDs) == 0 {
				return []*accountingsync.AccountingInboundChange{}, nil
			}

			ic := accountingsyncdb.AccountingInboundChanges
			return ic.Where(
				ic.ConnectionID.Eq(req.ConnectionID),
				ic.Kind.Eq(req.Kind),
				ic.ExternalID.In(req.ExternalIDs...),
			).All(ctx, r.db.Keel(ctx))
		},
	)
}

func (r *inboundChangeRepository) ListDetected(
	ctx context.Context,
	req *accountingsyncports.ListDetectedAccountingInboundChangesRequest,
) ([]*accountingsync.AccountingInboundChange, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingsync.AccountingInboundChange, error) {
			limit := req.Limit
			if limit <= 0 {
				limit = defaultInboundBatch
			}
			limit = intutils.Clamp(limit, 1, maxInboundBatch)

			ic := accountingsyncdb.AccountingInboundChanges
			return ic.Where(
				ic.ConnectionID.Eq(req.ConnectionID),
				ic.Status.Eq(accountingsync.InboundStatusDetected),
				ic.DetectedAt.Lte(req.DetectedBefore),
			).
				Order(ic.DetectedAt.Asc(), ic.ID.Asc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
		},
	)
}

func inboundListFilters(
	req *accountingsyncports.ListAccountingInboundChangesConnectionRequest,
) []keel.Pred {
	ic := accountingsyncdb.AccountingInboundChanges
	term := strings.TrimSpace(req.Search)

	return []keel.Pred{
		ic.ConnectionID.Eq(req.ConnectionID),
		keel.When(len(req.Statuses) > 0, ic.Status.In(req.Statuses...)),
		keel.When(len(req.Kinds) > 0, ic.Kind.In(req.Kinds...)),
		keel.When(len(req.Reasons) > 0, ic.Reason.In(req.Reasons...)),
		keel.When(term != "", keel.Or(
			ic.ExternalNumber.ContainsFold(term),
			ic.PartyName.ContainsFold(term),
		)),
	}
}

func (r *inboundChangeRepository) ListConnection(
	ctx context.Context,
	req *accountingsyncports.ListAccountingInboundChangesConnectionRequest,
) (*pagination.CursorListResult[*accountingsync.AccountingInboundChange], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*accountingsync.AccountingInboundChange], error) {
			if req.Filter != nil && len(req.Filter.Sort) == 0 {
				req.Filter.Sort = []domaintypes.SortField{
					{Field: inboundChangeDetectedSort, Direction: dbtype.SortDirectionDesc},
				}
			}

			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				accountingsyncdb.AccountingInboundChanges.Where(inboundListFilters(req)...),
				&listquery.Request[accountingsync.AccountingInboundChange]{
					Filter: req.Filter,
					Cursor: req.Cursor,
				},
			)
			if err != nil {
				r.l.Error("failed to list accounting inbound changes", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *inboundChangeRepository) Summarize(
	ctx context.Context,
	req *accountingsyncports.SummarizeAccountingInboundChangesRequest,
) (*accountingsyncports.AccountingInboundSummary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsyncports.AccountingInboundSummary, error) {
			ic := accountingsyncdb.AccountingInboundChanges
			summary, err := keel.IntoOne[accountingsyncports.AccountingInboundSummary](
				ctx,
				r.db.Keel(ctx),
				ic.Pick(
					keel.Count().Filter(ic.Status.Eq(accountingsync.InboundStatusDetected)),
					keel.Count().Filter(ic.Status.Eq(accountingsync.InboundStatusProposed)),
					keel.Count().Filter(keel.And(
						ic.Status.Eq(accountingsync.InboundStatusApplied),
						ic.DecidedAt.Gte(req.DecidedSince),
					)),
					keel.Count().Filter(keel.And(
						ic.Status.Eq(accountingsync.InboundStatusIgnored),
						ic.DecidedAt.Gte(req.DecidedSince),
					)),
				).Where(ic.ConnectionID.Eq(req.ConnectionID)),
			)
			if err != nil {
				return nil, err
			}

			return &summary, nil
		},
	)
}

func (r *inboundChangeRepository) ListAttention(
	ctx context.Context,
	req *accountingsyncports.ListAccountingInboundAttentionRequest,
) ([]accountingsyncports.AccountingInboundAttentionGroup, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]accountingsyncports.AccountingInboundAttentionGroup, error) {
			ic := accountingsyncdb.AccountingInboundChanges
			groups, err := keel.Into[accountingsyncports.AccountingInboundAttentionGroup](
				ctx,
				r.db.Keel(ctx),
				ic.Group(ic.Reason).
					Pick(
						keel.CastText(ic.Reason).Coalesce(""),
						keel.Count(),
						ic.DetectedAt.Min(),
						ic.ID.Min(),
					).
					Where(
						ic.ConnectionID.Eq(req.ConnectionID),
						ic.Status.Eq(accountingsync.InboundStatusProposed),
						ic.DetectedAt.Lte(req.DetectedBefore),
					).
					Order(keel.Count().Desc(), ic.DetectedAt.Min().Asc()),
			)
			if err != nil {
				return nil, err
			}
			if groups == nil {
				groups = []accountingsyncports.AccountingInboundAttentionGroup{}
			}

			return groups, nil
		},
	)
}
