package factoringrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring/factoringdb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

const reserveReversalIndex = "uq_factoring_reserve_entries_reversal"

var reserveErrors = dberror.Mapping{Entity: "FactoringReserveEntry"}

type reserveEntryRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewReserveEntryRepository(p Params) factoringports.ReserveEntryRepository {
	return &reserveEntryRepository{
		db: p.DB,
		l:  p.Logger.Named("repository.factoring-reserve-entry"),
	}
}

func (r *reserveEntryRepository) Create(
	ctx context.Context,
	entity *factoring.ReserveEntry,
) (*factoring.ReserveEntry, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*factoring.ReserveEntry, error) {
		if err := factoringdb.ReserveEntries.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			if dberror.IsUniqueConstraintViolation(err) &&
				dberror.ExtractConstraintName(err) == reserveReversalIndex {
				return nil, errortypes.NewConflictError(
					"That chargeback has already been reversed; reload to see the reserve",
				)
			}
			return nil, reserveErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *reserveEntryRepository) ListConnection(
	ctx context.Context,
	req *factoringports.ListReserveEntryConnectionRequest,
) (*pagination.CursorListResult[*factoring.ReserveEntry], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*factoring.ReserveEntry], error) {
			filter, term := withoutQuery(req.Filter)
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				reserveScope(req, term),
				&listquery.Request[factoring.ReserveEntry]{Filter: filter, Cursor: req.Cursor},
			)
			if err != nil {
				r.l.Error("failed to list factoring reserve entries", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func reserveScope(
	req *factoringports.ListReserveEntryConnectionRequest,
	term string,
) *keel.Query[factoring.ReserveEntry] {
	re := factoringdb.ReserveEntries
	pattern := "%" + stringutils.EscapeLikePattern(term) + "%"
	return re.Where(
		keel.When(term != "", keel.Or(
			re.Memo.ContainsFold(term),
			keel.Raw("?::text ILIKE ?", re.Kind, pattern),
		)),
		keel.When(
			req.FactoringCompanyID.IsNotNil(),
			re.FactoringCompanyID.Eq(req.FactoringCompanyID),
		),
		keel.When(req.SubmissionID.IsNotNil(), re.SubmissionID.Eq(req.SubmissionID)),
		keel.When(req.InvoiceID.IsNotNil(), re.InvoiceID.Eq(req.InvoiceID)),
		keel.When(len(req.Kinds) > 0, re.Kind.In(req.Kinds...)),
	)
}

func (r *reserveEntryRepository) ListForItem(
	ctx context.Context,
	req factoringports.ListReserveEntriesForItemRequest,
) ([]*factoring.ReserveEntry, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*factoring.ReserveEntry, error) {
		re := factoringdb.ReserveEntries
		rows, err := re.Where(re.SubmissionItemID.Eq(req.SubmissionItemID)).
			Order(re.CreatedAt.Asc(), re.ID.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}
		if rows == nil {
			return []*factoring.ReserveEntry{}, nil
		}

		return rows, nil
	})
}

type reserveTotals struct {
	ReserveMinor     int64
	LiabilityMinor   int64
	AdvancedMinor    int64
	ReleasedMinor    int64
	ChargedBackMinor int64
	FeeMinor         int64
	EntryCount       int
	LastOccurredOn   *int64
}

func (r *reserveEntryRepository) GetBalance(
	ctx context.Context,
	req factoringports.GetReserveBalanceRequest,
) (*factoringports.ReserveBalance, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*factoringports.ReserveBalance, error) {
		totals, err := keel.IntoOne[reserveTotals](
			ctx,
			r.db.Keel(ctx),
			reserveBalance(&req),
		)
		if err != nil {
			return nil, err
		}

		return &factoringports.ReserveBalance{
			FactoringCompanyID: req.FactoringCompanyID,
			ReserveMinor:       totals.ReserveMinor,
			LiabilityMinor:     totals.LiabilityMinor,
			AdvancedMinor:      totals.AdvancedMinor,
			ReleasedMinor:      totals.ReleasedMinor,
			ChargedBackMinor:   totals.ChargedBackMinor,
			FeeMinor:           totals.FeeMinor,
			EntryCount:         totals.EntryCount,
			LastOccurredOn:     totals.LastOccurredOn,
		}, nil
	})
}

func reserveBalance(
	req *factoringports.GetReserveBalanceRequest,
) *keel.Query[factoring.ReserveEntry] {
	re := factoringdb.ReserveEntries
	cash := func(kinds ...factoring.ReserveEntryKind) keel.Agg {
		return re.CashMinor.Sum().Filter(re.Kind.In(kinds...))
	}

	return re.Pick(
		keel.DynOf(re.ReserveMinor.Sum()).Coalesce(0),
		keel.DynOf(re.LiabilityMinor.Sum()).Coalesce(0),
		keel.DynOf(cash(factoring.ReserveEntryFunding)).Coalesce(0),
		keel.DynOf(cash(factoring.ReserveEntryRelease)).Coalesce(0),
		keel.Raw("COALESCE(-?, 0)", cash(
			factoring.ReserveEntryChargeback,
			factoring.ReserveEntryChargebackReversal,
		)),
		keel.DynOf(re.FeeMinor.Sum()).Coalesce(0),
		keel.Count(),
		re.OccurredOn.Max(),
	).Where(re.FactoringCompanyID.Eq(req.FactoringCompanyID))
}
