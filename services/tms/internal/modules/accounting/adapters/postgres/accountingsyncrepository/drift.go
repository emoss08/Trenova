package accountingsyncrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingdrift"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingdrift/accountingdriftdb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingdriftports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	driftFindingEntity       = "Accounting drift finding"
	driftFindingDetectedSort = "detectedAt"
)

var driftFindingErrors = dberror.Mapping{Entity: driftFindingEntity}

type DriftFindingParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type driftFindingRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewDriftFindingRepository(
	p DriftFindingParams,
) accountingdriftports.AccountingDriftFindingRepository {
	return &driftFindingRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-drift-finding-repository"),
	}
}

func (r *driftFindingRepository) Create(
	ctx context.Context,
	entity *accountingdrift.AccountingDriftFinding,
) (*accountingdrift.AccountingDriftFinding, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingdrift.AccountingDriftFinding, error) {
			err := accountingdriftdb.AccountingDriftFindings.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				return nil, fmt.Errorf("create accounting drift finding: %w", err)
			}

			return entity, nil
		},
	)
}

func (r *driftFindingRepository) Update(
	ctx context.Context,
	entity *accountingdrift.AccountingDriftFinding,
) (*accountingdrift.AccountingDriftFinding, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingdrift.AccountingDriftFinding, error) {
			ov := entity.Version
			err := accountingdriftdb.AccountingDriftFindings.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				entity.Version = ov
				return nil, driftFindingErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *driftFindingRepository) GetByID(
	ctx context.Context,
	req accountingdriftports.GetAccountingDriftFindingQuery,
) (*accountingdrift.AccountingDriftFinding, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingdrift.AccountingDriftFinding, error) {
			df := accountingdriftdb.AccountingDriftFindings
			q := df.Where(df.ID.Eq(req.ID))
			if req.ForUpdate {
				q = q.Lock(keel.ForUpdate)
			}

			entity, err := q.One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, driftFindingErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *driftFindingRepository) ListOpen(
	ctx context.Context,
	req *accountingdriftports.ListOpenAccountingDriftFindingsRequest,
) ([]*accountingdrift.AccountingDriftFinding, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingdrift.AccountingDriftFinding, error) {
			if len(req.ObjectIDs) == 0 {
				return []*accountingdrift.AccountingDriftFinding{}, nil
			}

			df := accountingdriftdb.AccountingDriftFindings
			return df.Where(
				df.ConnectionID.Eq(req.ConnectionID),
				df.Status.Eq(accountingdrift.DriftStatusOpen),
				df.ObjectID.In(req.ObjectIDs...),
			).
				Lock(keel.ForUpdate).
				All(ctx, r.db.Keel(ctx))
		},
	)
}

func driftListFilters(
	req *accountingdriftports.ListAccountingDriftFindingsConnectionRequest,
) []keel.Pred {
	df := accountingdriftdb.AccountingDriftFindings
	term := strings.TrimSpace(req.Search)

	return []keel.Pred{
		df.ConnectionID.Eq(req.ConnectionID),
		keel.When(len(req.Statuses) > 0, df.Status.In(req.Statuses...)),
		keel.When(len(req.Kinds) > 0, df.Kind.In(req.Kinds...)),
		keel.When(len(req.ObjectTypes) > 0, df.ObjectType.In(req.ObjectTypes...)),
		keel.When(!req.ObjectID.IsNil(), df.ObjectID.Eq(req.ObjectID)),
		keel.When(term != "", keel.Or(
			df.ObjectNumber.ContainsFold(term),
			df.PartyName.ContainsFold(term),
		)),
	}
}

func (r *driftFindingRepository) ListConnection(
	ctx context.Context,
	req *accountingdriftports.ListAccountingDriftFindingsConnectionRequest,
) (*pagination.CursorListResult[*accountingdrift.AccountingDriftFinding], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*accountingdrift.AccountingDriftFinding], error) {
			if req.Filter != nil && len(req.Filter.Sort) == 0 {
				req.Filter.Sort = []domaintypes.SortField{
					{Field: driftFindingDetectedSort, Direction: dbtype.SortDirectionDesc},
				}
			}

			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				accountingdriftdb.AccountingDriftFindings.Where(driftListFilters(req)...),
				&listquery.Request[accountingdrift.AccountingDriftFinding]{
					Filter: req.Filter,
					Cursor: req.Cursor,
				},
			)
			if err != nil {
				r.l.Error("failed to list accounting drift findings", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *driftFindingRepository) Summarize(
	ctx context.Context,
	req *accountingdriftports.SummarizeAccountingDriftRequest,
) (*accountingdriftports.AccountingDriftSummary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingdriftports.AccountingDriftSummary, error) {
			df := accountingdriftdb.AccountingDriftFindings
			open := df.Status.Eq(accountingdrift.DriftStatusOpen)

			summary, err := keel.IntoOne[accountingdriftports.AccountingDriftSummary](
				ctx,
				r.db.Keel(ctx),
				df.Pick(
					keel.Count().Filter(open),
					keel.Count().
						Filter(keel.And(open, df.Kind.Eq(accountingdrift.DriftAmountMismatch))),
					keel.Count().Filter(keel.And(open, df.Kind.In(
						accountingdrift.DriftDeletedInProvider,
						accountingdrift.DriftVoidedInProvider,
					))),
					keel.Count().
						Filter(keel.And(open, df.Kind.Eq(accountingdrift.DriftStatusMismatch))),
					keel.Count().Filter(keel.And(
						open,
						df.Kind.Eq(accountingdrift.DriftCustomerBalanceMismatch),
					)),
					keel.Count().Filter(keel.And(
						df.Status.Eq(accountingdrift.DriftStatusResolved),
						df.ResolvedAt.Gte(req.ResolvedSince),
					)),
				).Where(df.ConnectionID.Eq(req.ConnectionID)),
			)
			if err != nil {
				return nil, err
			}

			return &summary, nil
		},
	)
}

func (r *driftFindingRepository) ListAttention(
	ctx context.Context,
	req *accountingdriftports.ListAccountingDriftAttentionRequest,
) ([]accountingdriftports.AccountingDriftAttentionGroup, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]accountingdriftports.AccountingDriftAttentionGroup, error) {
			df := accountingdriftdb.AccountingDriftFindings
			groups, err := keel.Into[accountingdriftports.AccountingDriftAttentionGroup](
				ctx,
				r.db.Keel(ctx),
				df.Group(df.Kind).
					Pick(df.Kind, keel.Count(), df.DetectedAt.Min(), df.ID.Min()).
					Where(
						df.ConnectionID.Eq(req.ConnectionID),
						df.Status.Eq(accountingdrift.DriftStatusOpen),
						df.DetectedAt.Lte(req.DetectedBefore),
					).
					Order(keel.Count().Desc(), df.DetectedAt.Min().Asc()),
			)
			if err != nil {
				return nil, err
			}
			if groups == nil {
				groups = []accountingdriftports.AccountingDriftAttentionGroup{}
			}

			return groups, nil
		},
	)
}
