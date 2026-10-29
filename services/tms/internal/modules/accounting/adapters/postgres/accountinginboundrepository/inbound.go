package accountinginboundrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountinginbound"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountinginbound/accountinginbounddb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountinginboundports"
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
) accountinginboundports.AccountingInboundChangeRepository {
	return &inboundChangeRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-inbound-change-repository"),
	}
}

func (r *inboundChangeRepository) Create(
	ctx context.Context,
	entity *accountinginbound.AccountingInboundChange,
) (*accountinginbound.AccountingInboundChange, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountinginbound.AccountingInboundChange, error) {
			err := accountinginbounddb.AccountingInboundChanges.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				return nil, fmt.Errorf("create accounting inbound change: %w", err)
			}

			return entity, nil
		},
	)
}

func (r *inboundChangeRepository) Update(
	ctx context.Context,
	entity *accountinginbound.AccountingInboundChange,
) (*accountinginbound.AccountingInboundChange, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountinginbound.AccountingInboundChange, error) {
			ov := entity.Version
			err := accountinginbounddb.AccountingInboundChanges.Save(ctx, r.db.Keel(ctx), entity)
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
	req accountinginboundports.GetAccountingInboundChangeQuery,
) (*accountinginbound.AccountingInboundChange, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountinginbound.AccountingInboundChange, error) {
			ic := accountinginbounddb.AccountingInboundChanges
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
	req *accountinginboundports.ListAccountingInboundChangesByExternalIDsRequest,
) ([]*accountinginbound.AccountingInboundChange, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountinginbound.AccountingInboundChange, error) {
			if len(req.ExternalIDs) == 0 {
				return []*accountinginbound.AccountingInboundChange{}, nil
			}

			ic := accountinginbounddb.AccountingInboundChanges
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
	req *accountinginboundports.ListDetectedAccountingInboundChangesRequest,
) ([]*accountinginbound.AccountingInboundChange, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountinginbound.AccountingInboundChange, error) {
			limit := req.Limit
			if limit <= 0 {
				limit = defaultInboundBatch
			}
			limit = intutils.Clamp(limit, 1, maxInboundBatch)

			ic := accountinginbounddb.AccountingInboundChanges
			return ic.Where(
				ic.ConnectionID.Eq(req.ConnectionID),
				ic.Status.Eq(accountinginbound.InboundStatusDetected),
				ic.DetectedAt.Lte(req.DetectedBefore),
			).
				Order(ic.DetectedAt.Asc(), ic.ID.Asc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
		},
	)
}

func inboundListFilters(
	req *accountinginboundports.ListAccountingInboundChangesConnectionRequest,
) []keel.Pred {
	ic := accountinginbounddb.AccountingInboundChanges
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
	req *accountinginboundports.ListAccountingInboundChangesConnectionRequest,
) (*pagination.CursorListResult[*accountinginbound.AccountingInboundChange], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*accountinginbound.AccountingInboundChange], error) {
			if req.Filter != nil && len(req.Filter.Sort) == 0 {
				req.Filter.Sort = []domaintypes.SortField{
					{Field: inboundChangeDetectedSort, Direction: dbtype.SortDirectionDesc},
				}
			}

			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				accountinginbounddb.AccountingInboundChanges.Where(inboundListFilters(req)...),
				&listquery.Request[accountinginbound.AccountingInboundChange]{
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
	req *accountinginboundports.SummarizeAccountingInboundChangesRequest,
) (*accountinginboundports.AccountingInboundSummary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*accountinginboundports.AccountingInboundSummary, error) {
			ic := accountinginbounddb.AccountingInboundChanges
			summary, err := keel.IntoOne[accountinginboundports.AccountingInboundSummary](
				ctx,
				r.db.Keel(ctx),
				ic.Pick(
					keel.Count().Filter(ic.Status.Eq(accountinginbound.InboundStatusDetected)),
					keel.Count().Filter(ic.Status.Eq(accountinginbound.InboundStatusProposed)),
					keel.Count().Filter(keel.And(
						ic.Status.Eq(accountinginbound.InboundStatusApplied),
						ic.DecidedAt.Gte(req.DecidedSince),
					)),
					keel.Count().Filter(keel.And(
						ic.Status.Eq(accountinginbound.InboundStatusIgnored),
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
	req *accountinginboundports.ListAccountingInboundAttentionRequest,
) ([]accountinginboundports.AccountingInboundAttentionGroup, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]accountinginboundports.AccountingInboundAttentionGroup, error) {
			ic := accountinginbounddb.AccountingInboundChanges
			groups, err := keel.Into[accountinginboundports.AccountingInboundAttentionGroup](
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
						ic.Status.Eq(accountinginbound.InboundStatusProposed),
						ic.DetectedAt.Lte(req.DetectedBefore),
					).
					Order(keel.Count().Desc(), ic.DetectedAt.Min().Asc()),
			)
			if err != nil {
				return nil, err
			}
			if groups == nil {
				groups = []accountinginboundports.AccountingInboundAttentionGroup{}
			}

			return groups, nil
		},
	)
}
