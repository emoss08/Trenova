package customerrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type customerIdentity struct {
	Name string
	Code string
}

type creditReviewRow struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	CustomerID     pulid.ID
}

// ReviewCredit locks the customer's billing profile, lets its credit standing
// follow the open receivables, and saves what changed. The profile's version
// moves only when the standing does: the balance alone changes with every
// posting, and bumping the version for it would refuse the next edit anyone
// made to the customer.
func (r *repository) ReviewCredit(
	ctx context.Context,
	req *customerports.ReviewCustomerCreditRequest,
) (*customerports.ReviewCustomerCreditResult, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*customerports.ReviewCustomerCreditResult, error) {
			db := r.db.Keel(ctx)
			c := customerdb.Customers
			cbp := customerdb.CustomerBillingProfiles

			result := new(customerports.ReviewCustomerCreditResult)
			identity, err := keel.IntoOne[customerIdentity](
				ctx,
				db,
				c.Pick(keel.DynOf(c.Name).Coalesce(""), c.Code).Where(c.ID.Eq(req.CustomerID)),
			)
			if dberror.IsNotFoundError(err) {
				return result, nil
			}
			if err != nil {
				return nil, fmt.Errorf("read customer for credit review: %w", err)
			}
			result.CustomerName = identity.Name
			result.CustomerCode = identity.Code

			profile, err := cbp.Where(cbp.CustomerID.Eq(req.CustomerID)).
				Lock(keel.ForUpdate).
				One(ctx, db)
			if dberror.IsNotFoundError(err) {
				return result, nil
			}
			if err != nil {
				return nil, fmt.Errorf("lock billing profile for credit review: %w", err)
			}

			result.Outcome = profile.ReviewCredit(req.ExposureMinor, req.Now)
			result.Profile = profile

			if result.Outcome.ChangesStanding() {
				profile.Version++
				profile.UpdatedAt = req.Now
			}
			if _, err = cbp.Where(cbp.ID.Eq(profile.ID), cbp.CustomerID.Eq(profile.CustomerID)).
				Update(
					ctx,
					db,
					cbp.CreditBalance.To(profile.CreditBalance),
					cbp.CreditStatus.To(profile.CreditStatus),
					cbp.CreditHoldReason.To(profile.CreditHoldReason),
					cbp.CreditHoldAutomatic.To(profile.CreditHoldAutomatic),
					cbp.CreditReviewedAt.ToExpr(keel.Val(profile.CreditReviewedAt)),
					cbp.Version.To(profile.Version),
					cbp.UpdatedAt.To(profile.UpdatedAt),
				); err != nil {
				return nil, fmt.Errorf("save credit review: %w", err)
			}

			return result, nil
		},
	)
}

// ListCreditReviewCustomers pages through every customer with a billing
// profile across organizations, for the nightly credit sweep.
func (r *repository) ListCreditReviewCustomers(
	ctx context.Context,
	req *customerports.ListCreditReviewCustomersRequest,
) ([]customerports.CreditReviewCustomer, error) {
	ctx = dbscope.WithSystem(ctx, "sweep every customer's credit standing across organizations")
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]customerports.CreditReviewCustomer, error) {
			cbp := customerdb.CustomerBillingProfiles
			q := cbp.AllTenants().
				Pick(cbp.OrganizationID, cbp.BusinessUnitID, cbp.CustomerID).
				Order(cbp.OrganizationID.Asc(), cbp.BusinessUnitID.Asc(), cbp.CustomerID.Asc()).
				Limit(req.Limit)
			if req.After != nil {
				q = q.Where(keel.DynRow(cbp.OrganizationID, cbp.BusinessUnitID, cbp.CustomerID).Gt(
					keel.DynRow(
						req.After.TenantInfo.OrgID.String(),
						req.After.TenantInfo.BuID.String(),
						req.After.CustomerID.String(),
					),
				))
			}

			rows, err := keel.Into[creditReviewRow](ctx, r.db.Keel(ctx), q)
			if err != nil {
				return nil, fmt.Errorf("list credit review customers: %w", err)
			}

			customers := make([]customerports.CreditReviewCustomer, 0, len(rows))
			for _, rec := range rows {
				customers = append(customers, customerports.CreditReviewCustomer{
					TenantInfo: pagination.TenantInfo{
						OrgID: rec.OrganizationID,
						BuID:  rec.BusinessUnitID,
					},
					CustomerID: rec.CustomerID,
				})
			}
			return customers, nil
		},
	)
}
