package customerrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/quotatx"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	customerErrors = dberror.Mapping{
		Entity: "Customer",
		Unique: []dberror.Constraint{
			{
				Name:    "idx_customers_code",
				Field:   "code",
				Message: "Customer with this code already exists in your organization",
			},
		},
	}
	billingProfileErrors = dberror.Mapping{Entity: "Customer billing profile"}
	emailProfileErrors   = dberror.Mapping{Entity: "Customer email profile"}
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
	Quota  platformplanports.QuotaGuard `optional:"true"`
}

type repository struct {
	db    *postgres.Connection
	l     *zap.Logger
	quota platformplanports.QuotaGuard
}

func New(p Params) customerports.CustomerRepository {
	return &repository{
		db:    p.DB,
		l:     p.Logger.Named("postgres.customer-repository"),
		quota: quotaservice.OrUnlimited(p.Quota),
	}
}

func customerLoads(opts customerports.CustomerFilterOptions) []keel.Loader {
	c := customerdb.Customers
	bp := c.BillingProfile()

	loads := make([]keel.Loader, 0, 3)
	if opts.IncludeState {
		loads = append(loads, c.State())
	}
	if opts.IncludeBillingProfile {
		loads = append(loads, bp.Load(bp.DocumentTypes()))
	}
	if opts.IncludeEmailProfile {
		loads = append(loads, c.EmailProfile())
	}

	return loads
}

func (r *repository) List(
	ctx context.Context,
	req *customerports.ListCustomerRequest,
) (*pagination.ListResult[*customer.Customer], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*customer.Customer], error) {
			c := customerdb.Customers
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				c.Query(),
				&listquery.Request[customer.Customer]{
					Filter: req.Filter,
					Shape: func(q *keel.Query[customer.Customer]) *keel.Query[customer.Customer] {
						return q.Load(customerLoads(req.CustomerFilterOptions)...)
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list customers", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *customerports.ListCustomerConnectionRequest,
) (*pagination.CursorListResult[*customer.Customer], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*customer.Customer], error) {
			c := customerdb.Customers
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				c.Query(),
				&listquery.Request[customer.Customer]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.CustomerColumns,
					Shape: func(q *keel.Query[customer.Customer]) *keel.Query[customer.Customer] {
						return q.Load(customerLoads(req.CustomerFilterOptions)...)
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list customer connection", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req customerports.GetCustomerByIDRequest,
) (*customer.Customer, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*customer.Customer, error) {
		c := customerdb.Customers
		opts := req.CustomerFilterOptions
		opts.IncludeState = true

		entity, err := c.Where(c.ID.Eq(req.ID)).
			Load(customerLoads(opts)...).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get customer", zap.String("id", req.ID.String()), zap.Error(err))
			return nil, customerErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetBillingProfile(
	ctx context.Context,
	req customerports.GetCustomerBillingProfileRequest,
) (*customer.CustomerBillingProfile, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*customer.CustomerBillingProfile, error) {
			cbp := customerdb.CustomerBillingProfiles
			entity, err := cbp.Where(cbp.CustomerID.Eq(req.CustomerID)).
				Load(cbp.DocumentTypes()).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error(
					"failed to get billing profile",
					zap.String("customerID", req.CustomerID.String()),
					zap.Error(err),
				)
				return nil, billingProfileErrors.Map(err, req.CustomerID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req customerports.GetCustomersByIDsRequest,
) ([]*customer.Customer, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*customer.Customer, error) {
		if len(req.CustomerIDs) == 0 {
			return []*customer.Customer{}, nil
		}

		c := customerdb.Customers
		entities, err := c.Where(c.ID.In(req.CustomerIDs...)).
			Load(customerLoads(req.CustomerFilterOptions)...).
			Limit(len(req.CustomerIDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get customers", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *customerports.CustomerSelectOptionsRequest,
) (*pagination.ListResult[*customer.Customer], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*customer.Customer], error) {
			c := customerdb.Customers
			q := c.Pick(c.ID, c.Code, c.Name).
				Where(c.Status.Eq(domaintypes.StatusActive))

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(c.Code.ContainsFold(s), c.Name.ContainsFold(s))
				},
			)
		},
	)
}

func (r *repository) geocodeIfApplicable(entity *customer.Customer) *customer.Customer {
	if !entity.MeetGeocodingRequirements() {
		entity.ResetGeocoding()
		return entity
	}

	entity.SetGeocoding(true, entity.Longitude, entity.Latitude, entity.PlaceID)
	return entity
}

func (r *repository) Create(
	ctx context.Context,
	entity *customer.Customer,
) (*customer.Customer, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*customer.Customer, error) {
		err := quotatx.Run(ctx, r.db, r.quota, func(c context.Context) error {
			entity = r.geocodeIfApplicable(entity)

			if err := customerdb.Customers.Create(c, r.db.Keel(c), entity); err != nil {
				return customerErrors.Map(err, entity.ID.String())
			}

			if !entity.HasBillingProfile() {
				entity.BillingProfile = customer.NewDefaultBillingProfile(
					entity.OrganizationID,
					entity.BusinessUnitID,
					entity.ID,
				)
			}

			if err := r.saveBillingProfile(c, entity); err != nil {
				return err
			}

			return r.saveEmailProfile(c, entity)
		}, platformplanports.QuotaRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
			Meter:    platformcatalog.MeterCustomersTotal,
			Quantity: 1,
		})
		if err != nil {
			r.l.Error("failed to create customer", zap.String("code", entity.Code), zap.Error(err))
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *customer.Customer,
) (*customer.Customer, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*customer.Customer, error) {
		entity = r.geocodeIfApplicable(entity)

		err := customerdb.Customers.Save(ctx, r.db.Keel(ctx), entity)
		if err == nil {
			err = r.saveBillingProfile(ctx, entity)
		}
		if err == nil {
			err = r.saveEmailProfile(ctx, entity)
		}
		if err != nil {
			r.l.Error(
				"failed to update customer",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, dberror.MapRetryableTransactionError(
				customerErrors.Map(err, entity.ID.String()),
				"Customer is busy. Retry the request.",
			)
		}

		return entity, nil
	})
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *customerports.BulkUpdateCustomerStatusRequest,
) ([]*customer.Customer, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*customer.Customer, error) {
		c := customerdb.Customers
		entities, err := c.Where(c.ID.In(req.CustomerIDs...)).UpdateReturning(ctx, r.db.Keel(ctx), c.Status.To(req.Status))
		if err != nil {
			r.l.Error("failed to bulk update customer status", zap.Error(err))
			return nil, err
		}

		if err = dberror.CheckBulkAffected(
			int64(len(entities)),
			"Customer",
			req.CustomerIDs,
		); err != nil {
			return nil, err
		}

		return entities, nil
	})
}

func billingProfileFormColumns() []keel.Col {
	cbp := customerdb.CustomerBillingProfiles

	return []keel.Col{
		cbp.InvoiceDelivery.Col,
		cbp.BillingCycle.Col,
		cbp.BillingCycleAnchorDay.Col,
		cbp.BillingCycleTimezone.Col,
		cbp.PaymentTerm.Col,
		cbp.HasBillingControlOverrides.Col,
		cbp.CreditLimit.Col,
		cbp.CreditStatus.Col,
		cbp.EnforceCreditLimit.Col,
		cbp.AutoCreditHold.Col,
		cbp.CreditHoldReason.Col,
		cbp.AutoSendInvoiceOnGeneration.Col,
		cbp.SendPaymentReminders.Col,
		cbp.SendStatements.Col,
		cbp.FactoringCompanyID.Col,
		cbp.EmailInvoiceEnabled.Col,
		cbp.EDIInvoiceEnabled.Col,
		cbp.SplitBy.Col,
		cbp.SectionBy.Col,
		cbp.InvoiceDetail.Col,
		cbp.MinConsolidatedAmount.Col,
		cbp.MinConsolidatedAmountMinor.Col,
		cbp.MaxShipmentsPerInvoice.Col,
		cbp.InvoiceNumberFormat.Col,
		cbp.CustomerInvoicePrefix.Col,
		cbp.InvoiceCopies.Col,
		cbp.RevenueAccountID.Col,
		cbp.ARAccountID.Col,
		cbp.ApplyLateCharges.Col,
		cbp.LateChargeRate.Col,
		cbp.GracePeriodDays.Col,
		cbp.TaxExempt.Col,
		cbp.TaxExemptNumber.Col,
		cbp.EnforceCustomerBillingReq.Col,
		cbp.ValidateCustomerRates.Col,
		cbp.AutoTransfer.Col,
		cbp.AutoMarkReadyToBill.Col,
		cbp.AutoApprove.Col,
		cbp.AutoBill.Col,
		cbp.CountLateOnlyOnAppointmentStops.Col,
		cbp.AutoApplyAccessorials.Col,
		cbp.BillingCurrency.Col,
		cbp.RequirePONumber.Col,
		cbp.RequireBOLNumber.Col,
		cbp.RequireDeliveryNumber.Col,
		cbp.DefaultBillerID.Col,
		cbp.BillingNotes.Col,
		cbp.FuelSurchargeMode.Col,
		cbp.FuelSurchargeProgramID.Col,
	}
}

func billingProfileConflict() keel.Conflict {
	cbp := customerdb.CustomerBillingProfiles
	conflict := keel.OnConflict(cbp.CustomerID, cbp.OrganizationID, cbp.BusinessUnitID)
	for _, col := range billingProfileFormColumns() {
		conflict = conflict.Update(col)
	}

	return conflict.Set(cbp.CreditHoldAutomatic.ToExpr(keel.Raw(
		"CASE WHEN ? IS DISTINCT FROM ? THEN FALSE ELSE ? END",
		cbp.CreditStatus,
		cbp.CreditStatus.Excluded(),
		cbp.CreditHoldAutomatic,
	)))
}

func (r *repository) saveBillingProfile(ctx context.Context, cus *customer.Customer) error {
	if !cus.HasBillingProfile() {
		return nil
	}

	profile := cus.BillingProfile
	profile.CustomerID = cus.ID
	profile.OrganizationID = cus.OrganizationID
	profile.BusinessUnitID = cus.BusinessUnitID

	cbp := customerdb.CustomerBillingProfiles
	db := r.db.Keel(ctx)
	inserted, err := cbp.Upsert(
		ctx,
		db,
		[]*customer.CustomerBillingProfile{profile},
		billingProfileConflict(),
	)
	if err != nil {
		r.l.Error("failed to save billing profile", zap.Error(err))
		return billingProfileErrors.Map(err, profile.ID.String())
	}

	if !inserted[0] {
		stored, readErr := cbp.Where(cbp.CustomerID.Eq(cus.ID)).One(ctx, db)
		if readErr != nil {
			return billingProfileErrors.Map(readErr, cus.ID.String())
		}
		stored.DocumentTypes = profile.DocumentTypes
		*profile = *stored
	}

	if err = keel.SetLinks(
		ctx,
		db,
		cbp.DocumentTypes(),
		profile.ID,
		documentTypeIDs(profile),
	); err != nil {
		r.l.Error("failed to sync billing profile document types", zap.Error(err))
		return err
	}

	return nil
}

func documentTypeIDs(profile *customer.CustomerBillingProfile) []pulid.ID {
	ids := make([]pulid.ID, 0, len(profile.DocumentTypes))
	for _, dt := range profile.DocumentTypes {
		if dt != nil {
			ids = append(ids, dt.ID)
		}
	}

	return ids
}

func (r *repository) saveEmailProfile(ctx context.Context, cus *customer.Customer) error {
	if !cus.HasEmailProfile() {
		return nil
	}

	profile := cus.EmailProfile
	profile.CustomerID = cus.ID
	profile.OrganizationID = cus.OrganizationID
	profile.BusinessUnitID = cus.BusinessUnitID

	cem := customerdb.CustomerEmailProfiles
	db := r.db.Keel(ctx)
	inserted, err := cem.Upsert(
		ctx,
		db,
		[]*customer.CustomerEmailProfile{profile},
		keel.OnConflict(cem.CustomerID, cem.OrganizationID, cem.BusinessUnitID).Update(
			cem.Subject,
			cem.Comment,
			cem.ToRecipients,
			cem.CCRecipients,
			cem.BCCRecipients,
			cem.AttachmentName,
			cem.ReadReceipt,
			cem.IncludeShipmentDetail,
		),
	)
	if err != nil {
		r.l.Error("failed to save email profile", zap.Error(err))
		return emailProfileErrors.Map(err, profile.ID.String())
	}

	if !inserted[0] {
		stored, readErr := cem.Where(cem.CustomerID.Eq(cus.ID)).One(ctx, db)
		if readErr != nil {
			return emailProfileErrors.Map(readErr, cus.ID.String())
		}
		*profile = *stored
	}

	return nil
}
