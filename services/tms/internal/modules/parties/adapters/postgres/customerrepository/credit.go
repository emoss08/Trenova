package customerrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/buncolgen/customercol"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

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
			db := r.db.DBForContext(ctx)
			cbp := customercol.CustomerBillingProfileColumns
			cus := customercol.CustomerColumns

			result := new(customerports.ReviewCustomerCreditResult)
			if err := db.NewSelect().
				Model((*customer.Customer)(nil)).
				Column(cus.Name.Bare(), cus.Code.Bare()).
				Where(cus.ID.Eq(), req.CustomerID).
				Where(cus.OrganizationID.Eq(), req.TenantInfo.OrgID).
				Where(cus.BusinessUnitID.Eq(), req.TenantInfo.BuID).
				Scan(ctx, &result.CustomerName, &result.CustomerCode); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return result, nil
				}
				return nil, fmt.Errorf("read customer for credit review: %w", err)
			}

			profile := new(customer.CustomerBillingProfile)
			if err := db.NewSelect().
				Model(profile).
				Where(cbp.CustomerID.Eq(), req.CustomerID).
				Where(cbp.OrganizationID.Eq(), req.TenantInfo.OrgID).
				Where(cbp.BusinessUnitID.Eq(), req.TenantInfo.BuID).
				For("UPDATE").
				Scan(ctx); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return result, nil
				}
				return nil, fmt.Errorf("lock billing profile for credit review: %w", err)
			}

			result.Outcome = profile.ReviewCredit(req.ExposureMinor, req.Now)
			result.Profile = profile

			query := db.NewUpdate().
				Model(profile).
				Column(
					cbp.CreditBalance.Bare(),
					cbp.CreditStatus.Bare(),
					cbp.CreditHoldReason.Bare(),
					cbp.CreditHoldAutomatic.Bare(),
					cbp.CreditReviewedAt.Bare(),
				).
				WherePK()
			if result.Outcome.ChangesStanding() {
				profile.Version++
				profile.UpdatedAt = req.Now
				query = query.Column(cbp.Version.Bare(), cbp.UpdatedAt.Bare())
			}
			if _, err := query.Exec(ctx); err != nil {
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
			cbp := customercol.CustomerBillingProfileColumns
			type row struct {
				OrganizationID pulid.ID `bun:"organization_id"`
				BusinessUnitID pulid.ID `bun:"business_unit_id"`
				CustomerID     pulid.ID `bun:"customer_id"`
			}
			rows := make([]row, 0, req.Limit)
			query := r.db.DBForContext(ctx).NewSelect().
				Model((*customer.CustomerBillingProfile)(nil)).
				Column(
					cbp.OrganizationID.Bare(),
					cbp.BusinessUnitID.Bare(),
					cbp.CustomerID.Bare(),
				).
				OrderExpr(cbp.OrganizationID.OrderAsc()).
				OrderExpr(cbp.BusinessUnitID.OrderAsc()).
				OrderExpr(cbp.CustomerID.OrderAsc()).
				Limit(int64(req.Limit))
			if req.After != nil {
				query = query.Where(
					"("+cbp.OrganizationID.Qualified()+", "+cbp.BusinessUnitID.Qualified()+", "+
						cbp.CustomerID.Qualified()+") > (?, ?, ?)",
					req.After.TenantInfo.OrgID,
					req.After.TenantInfo.BuID,
					req.After.CustomerID,
				)
			}
			if err := query.Scan(ctx, &rows); err != nil {
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
