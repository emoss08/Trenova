package factoringrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring/factoringdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission/factoringsubmissiondb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type factorCustomerRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewFactorCustomerRepository(p Params) factoringports.FactorCustomerRepository {
	return &factorCustomerRepository{
		db: p.DB,
		l:  p.Logger.Named("repository.factoring-customers"),
	}
}

type releasedByCustomer struct {
	CustomerID pulid.ID
	ReleasedAt *int64
}

type fundedByCustomer struct {
	CustomerID  pulid.ID
	Count       int
	AmountMinor int64
}

func (r *factorCustomerRepository) List(
	ctx context.Context,
	req *factoringports.ListFactorCustomersRequest,
) (*factoringports.FactorCustomerPage, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*factoringports.FactorCustomerPage, error) {
			db := r.db.Keel(ctx)
			result, err := listquery.Connection(
				ctx,
				db,
				factorCustomers(req),
				&listquery.Request[customer.Customer]{
					Filter: byNameWhenUnsorted(req.Filter),
					Cursor: req.Cursor,
				},
			)
			if err != nil {
				r.l.Error("failed to list the factor's customers", zap.Error(err))
				return nil, err
			}

			out := &factoringports.FactorCustomerPage{
				Items:           make([]*factoringports.FactorCustomer, 0, len(result.Items)),
				HasNextPage:     result.HasNextPage,
				TotalCount:      result.TotalCount,
				TotalCountExact: result.TotalCountExact,
				EdgeCursors:     result.EdgeCursors,
			}
			if len(result.Items) == 0 {
				return out, nil
			}

			ids := make([]pulid.ID, len(result.Items))
			for i, c := range result.Items {
				ids[i] = c.ID
			}
			var (
				notices  []*factoring.Noa
				releases []releasedByCustomer
				funded   []fundedByCustomer
			)
			if err = keel.Batch(ctx, db,
				keel.ListInto(openNotices(req.FactoringCompanyID, ids), &notices),
				keel.IntoSlice(lastReleases(req.FactoringCompanyID, ids), &releases),
				keel.IntoSlice(fundedForCustomers(req.FactoringCompanyID, ids), &funded),
			); err != nil {
				r.l.Error("failed to read the factor's customer notices", zap.Error(err))
				return nil, err
			}

			open := make(map[pulid.ID]*factoring.Noa, len(notices))
			for _, noa := range notices {
				open[noa.CustomerID] = noa
			}
			released := make(map[pulid.ID]*int64, len(releases))
			for _, row := range releases {
				released[row.CustomerID] = row.ReleasedAt
			}
			byCustomer := make(map[pulid.ID]fundedByCustomer, len(funded))
			for _, row := range funded {
				byCustomer[row.CustomerID] = row
			}

			for _, c := range result.Items {
				row := &factoringports.FactorCustomer{
					CustomerID:  c.ID,
					Code:        c.Code,
					Name:        c.Name,
					Status:      c.Status,
					Notice:      open[c.ID],
					FundedCount: byCustomer[c.ID].Count,
					FundedMinor: byCustomer[c.ID].AmountMinor,
				}
				if row.Notice == nil {
					row.LastReleasedAt = released[c.ID]
				}
				out.Items = append(out.Items, row)
			}

			return out, nil
		},
	)
}

func byNameWhenUnsorted(filter *pagination.QueryOptions) *pagination.QueryOptions {
	if filter == nil || len(filter.Sort) > 0 {
		return filter
	}
	sorted := *filter
	sorted.Sort = []domaintypes.SortField{{Field: "name", Direction: dbtype.SortDirectionAsc}}
	return &sorted
}

func factorCustomers(
	req *factoringports.ListFactorCustomersRequest,
) keel.Query[*customer.Customer] {
	cus := customerdb.Customers
	noa := factoringdb.Noas
	pair := keel.And(
		noa.CustomerID.EqCol(cus.ID),
		noa.FactoringCompanyID.Eq(req.FactoringCompanyID),
	)
	standing := keel.Or(
		noa.Status.In(factoring.NoaStatusDraft, factoring.NoaStatusSent),
		noa.ExpiresAt.IsNull(),
		noa.ExpiresAt.Gt(req.At),
	)

	var stage keel.Pred
	switch req.Stage {
	case factoringports.CustomerNoticeNeeded:
		stage = keel.NotExists(noa.Where(
			pair,
			noa.Status.Ne(factoring.NoaStatusReleased),
			standing,
		))
	case factoringports.CustomerNoticeAwaiting:
		stage = keel.Exists(noa.Where(
			pair,
			noa.Status.In(factoring.NoaStatusDraft, factoring.NoaStatusSent),
		))
	case factoringports.CustomerNoticeAcknowledged:
		stage = keel.Exists(noa.Where(
			pair,
			noa.Status.Eq(factoring.NoaStatusAcknowledged),
			keel.Or(noa.ExpiresAt.IsNull(), noa.ExpiresAt.Gt(req.At)),
		))
	case factoringports.CustomerNoticeAll:
	}

	return cus.Where(
		cus.BillingProfile().FactoringCompanyID.Eq(req.FactoringCompanyID),
		stage,
	)
}

func openNotices(factorID pulid.ID, customerIDs []pulid.ID) keel.Query[*factoring.Noa] {
	noa := factoringdb.Noas
	return noa.Where(
		noa.FactoringCompanyID.Eq(factorID),
		noa.CustomerID.In(customerIDs...),
		noa.Status.Ne(factoring.NoaStatusReleased),
	).Limit(len(customerIDs))
}

func lastReleases(factorID pulid.ID, customerIDs []pulid.ID) keel.Projection {
	noa := factoringdb.Noas
	return noa.Group(noa.CustomerID).
		Pick(noa.CustomerID, noa.ReleasedAt.Max()).
		Where(
			noa.FactoringCompanyID.Eq(factorID),
			noa.CustomerID.In(customerIDs...),
			noa.Status.Eq(factoring.NoaStatusReleased),
		)
}

func fundedForCustomers(factorID pulid.ID, customerIDs []pulid.ID) keel.Projection {
	item := factoringsubmissiondb.SubmissionItems
	return item.Group(item.CustomerID).
		Pick(
			item.CustomerID,
			keel.Count(),
			keel.NumericOf(item.FaceMinor.Sum()).Coalesce(0),
		).
		Where(
			item.Status.Eq(factoringsubmission.SubmissionItemStatusFunded),
			item.CustomerID.In(customerIDs...),
			item.Submission().FactoringCompanyID.Eq(factorID),
		)
}
