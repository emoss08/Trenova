package customerrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/m2msync"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB      *postgres.Connection
	Logger  *zap.Logger
	M2MSync *m2msync.Syncer
}

type repository struct {
	db      *postgres.Connection
	l       *zap.Logger
	m2mSync *m2msync.Syncer
}

func New(p Params) repositories.CustomerRepository {
	return &repository{
		db:      p.DB,
		l:       p.Logger.Named("postgres.customer-repository"),
		m2mSync: p.M2MSync,
	}
}

func (r *repository) addOptions(
	q *bun.SelectQuery,
	opts repositories.CustomerFilterOptions,
) *bun.SelectQuery {
	if opts.IncludeState {
		q = q.Relation("State")
	}

	if opts.IncludeBillingProfile {
		q = q.Relation("BillingProfile")
		q = q.Relation("BillingProfile.DocumentTypes")
	}

	if opts.IncludeEmailProfile {
		q = q.Relation("EmailProfile")
	}

	return q
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListCustomerRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"cus",
		req.Filter,
		(*customer.Customer)(nil),
	)

	q = q.Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.addOptions(sq, req.CustomerFilterOptions)
	})

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListCustomerRequest,
) (*pagination.ListResult[*customer.Customer], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*customer.Customer, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count customers", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*customer.Customer]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *repositories.ListCustomerConnectionRequest,
) (*bun.SelectQuery, error) {
	return querybuilder.ApplyCursorFilters(
		q,
		buncolgen.CustomerTable.Alias,
		req.Filter,
		req.Cursor,
		(*customer.Customer)(nil),
	)
}

func (r *repository) applyTotalCountFilters(
	q *bun.SelectQuery,
	req *repositories.ListCustomerConnectionRequest,
) *bun.SelectQuery {
	return querybuilder.ApplyFiltersWithoutSort(
		q,
		buncolgen.CustomerTable.Alias,
		req.Filter,
		(*customer.Customer)(nil),
	)
}

func applyCustomerColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(buncolgen.CustomerTable.All())
	}

	return q.Column(columns...)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListCustomerConnectionRequest,
) (*pagination.CursorListResult[*customer.Customer], error) {
	log := r.l.With(
		zap.String("operation", "ListConnection"),
		zap.Any("request", req),
	)

	dba := r.db.DBForContext(ctx)
	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*customer.Customer)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				return r.applyTotalCountFilters(sq, req)
			}).
			Count(ctx)
		if err != nil {
			log.Error("failed to count customers", zap.Error(err))
			return nil, err
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*customer.Customer]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: totalCount,
			Query: func(entities *[]*customer.Customer) *bun.SelectQuery {
				return dba.
					NewSelect().
					Model(entities).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return applyCustomerColumns(sq, req.CustomerColumns)
					}).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return r.addOptions(sq, req.CustomerFilterOptions)
					})
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return r.applyCursorPageFilters(sq, req)
			},
		},
	)
	if err != nil {
		log.Error("failed to scan customers", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetCustomerByIDRequest,
) (*customer.Customer, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(customer.Customer)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		Relation("State").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("cus.id = ?", req.ID).
				Where("cus.organization_id = ?", req.TenantInfo.OrgID).
				Where("cus.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.addOptions(sq, req.CustomerFilterOptions)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get customer", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Customer")
	}

	return entity, nil
}

func (r *repository) GetBillingProfile(
	ctx context.Context,
	cusID pulid.ID,
) (*customer.CustomerBillingProfile, error) {
	log := r.l.With(
		zap.String("operation", "getBillingProfile"),
		zap.String("customerID", cusID.String()),
	)

	entity := new(customer.CustomerBillingProfile)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		Where("cbp.customer_id = ?", cusID).
		Relation("DocumentTypes").
		Scan(ctx)
	if err != nil {
		log.Error("failed to get billing profile", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req repositories.GetCustomersByIDsRequest,
) ([]*customer.Customer, error) {
	log := r.l.With(
		zap.String("operation", "GetByIDs"),
		zap.Any("request", req),
	)

	entities := make([]*customer.Customer, 0, len(req.CustomerIDs))
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CustomerScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.CustomerColumns.ID.In(), bun.List(req.CustomerIDs))
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.addOptions(sq, req.CustomerFilterOptions)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get customers", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Customer")
	}

	return entities, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *repositories.CustomerSelectOptionsRequest,
) (*pagination.ListResult[*customer.Customer], error) {
	return dbhelper.SelectOptions[*customer.Customer](
		ctx,
		r.db.DB(),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			Columns: []string{
				"id",
				"code",
				"name",
			},
			OrgColumn: "cus.organization_id",
			BuColumn:  "cus.business_unit_id",
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where("cus.status = ?", domaintypes.StatusActive)
			},
			EntityName: "Customer",
			SearchColumns: []string{
				"cus.code",
				"cus.name",
			},
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
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("code", entity.Code),
	)

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		entity = r.geocodeIfApplicable(entity)

		if _, err := r.db.DBForContext(c).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(c); err != nil {
			log.Error("failed to create customer", zap.Error(err))
			return err
		}

		if !entity.HasBillingProfile() {
			entity.BillingProfile = customer.NewDefaultBillingProfile(
				entity.OrganizationID,
				entity.BusinessUnitID,
				entity.ID,
			)
		}

		if err := r.saveBillingProfile(c, tx, entity); err != nil {
			return err
		}

		if err := r.saveEmailProfile(c, tx, entity); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		log.Error("failed to create customer", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) syncBillingProfileDocumentTypes(
	ctx context.Context,
	tx bun.IDB,
	bp *customer.CustomerBillingProfile,
) error {
	log := r.l.With(
		zap.String("operation", "syncBillingProfileDocumentTypes"),
		zap.String("billingProfileID", bp.GetID()),
		zap.Int("documentTypeCount", len(bp.DocumentTypes)),
	)

	config := m2msync.Config{
		Table:       "customer_billing_profile_document_types",
		SourceField: "billing_profile_id",
		TargetField: "document_type_id",
		AdditionalFields: map[string]any{
			"organization_id":  bp.OrganizationID,
			"business_unit_id": bp.BusinessUnitID,
		},
	}

	if err := r.m2mSync.SyncEntities(ctx, tx, config, bp.ID, bp.DocumentTypes); err != nil {
		log.Error("failed to sync billing profile document types", zap.Error(err))
		return err
	}

	log.Debug("successfully synced billing profile document types")
	return nil
}

func (r *repository) saveBillingProfile(
	ctx context.Context,
	tx bun.IDB,
	cus *customer.Customer,
) error {
	log := r.l.With(
		zap.String("operation", "saveBillingProfile"),
		zap.String("customerID", cus.ID.String()),
	)

	if !cus.HasBillingProfile() {
		return nil
	}

	billingProfile := cus.BillingProfile
	billingProfile.CustomerID = cus.ID
	billingProfile.OrganizationID = cus.OrganizationID
	billingProfile.BusinessUnitID = cus.BusinessUnitID

	cbp := buncolgen.CustomerBillingProfileColumns

	// Every column the profile owns is listed below except last_billed_period_end,
	// and that omission is deliberate: the billing watermark is advanced only by
	// AdvanceBilledPeriod inside the commit transaction. Taking it from the
	// incoming row would let an ordinary customer edit drag it backwards and
	// re-open a period whose invoices already exist.
	if _, err := tx.NewInsert().
		Model(billingProfile).
		On("CONFLICT (customer_id, organization_id, business_unit_id) DO UPDATE").
		Set(cbp.InvoiceDelivery.SetExcluded()).
		Set(cbp.BillingCycle.SetExcluded()).
		Set(cbp.BillingCycleAnchorDay.SetExcluded()).
		Set(cbp.BillingCycleTimezone.SetExcluded()).
		Set(cbp.PaymentTerm.SetExcluded()).
		Set(cbp.HasBillingControlOverrides.SetExcluded()).
		Set(cbp.CreditLimit.SetExcluded()).
		Set(cbp.CreditBalance.SetExcluded()).
		Set(cbp.CreditStatus.SetExcluded()).
		Set(cbp.EnforceCreditLimit.SetExcluded()).
		Set(cbp.AutoCreditHold.SetExcluded()).
		Set(cbp.CreditHoldReason.SetExcluded()).
		Set(cbp.AutoSendInvoiceOnGeneration.SetExcluded()).
		Set(cbp.SplitBy.SetExcluded()).
		Set(cbp.SectionBy.SetExcluded()).
		Set(cbp.InvoiceDetail.SetExcluded()).
		Set(cbp.ConsolidationLookbackDays.SetExcluded()).
		Set(cbp.MinConsolidatedAmount.SetExcluded()).
		Set(cbp.MinConsolidatedAmountMinor.SetExcluded()).
		Set(cbp.MaxShipmentsPerInvoice.SetExcluded()).
		Set(cbp.InvoiceNumberFormat.SetExcluded()).
		Set(cbp.CustomerInvoicePrefix.SetExcluded()).
		Set(cbp.InvoiceCopies.SetExcluded()).
		Set(cbp.RevenueAccountID.SetExcluded()).
		Set(cbp.ARAccountID.SetExcluded()).
		Set(cbp.ApplyLateCharges.SetExcluded()).
		Set(cbp.LateChargeRate.SetExcluded()).
		Set(cbp.GracePeriodDays.SetExcluded()).
		Set(cbp.TaxExempt.SetExcluded()).
		Set(cbp.TaxExemptNumber.SetExcluded()).
		Set(cbp.EnforceCustomerBillingReq.SetExcluded()).
		Set(cbp.ValidateCustomerRates.SetExcluded()).
		Set(cbp.AutoTransfer.SetExcluded()).
		Set(cbp.AutoMarkReadyToBill.SetExcluded()).
		Set(cbp.AutoBill.SetExcluded()).
		Set(cbp.CountLateOnlyOnAppointmentStops.SetExcluded()).
		Set(cbp.AutoApplyAccessorials.SetExcluded()).
		Set(cbp.BillingCurrency.SetExcluded()).
		Set(cbp.RequirePONumber.SetExcluded()).
		Set(cbp.RequireBOLNumber.SetExcluded()).
		Set(cbp.RequireDeliveryNumber.SetExcluded()).
		Set(cbp.InvoiceAdjustmentSupportingDocumentPolicy.SetExcluded()).
		Set(cbp.DefaultBillerID.SetExcluded()).
		Set(cbp.BillingNotes.SetExcluded()).
		Set(cbp.FuelSurchargeMode.SetExcluded()).
		Set(cbp.FuelSurchargeProgramID.SetExcluded()).
		Set(cbp.Version.SetExpr("cbp.version + 1")).
		Set(cbp.UpdatedAt.SetExcluded()).
		Returning("*").
		Exec(ctx); err != nil {
		log.Error("failed to save billing profile", zap.Error(err))
		return err
	}

	if err := r.syncBillingProfileDocumentTypes(ctx, tx, billingProfile); err != nil {
		return err
	}

	return nil
}

func (r *repository) saveEmailProfile(
	ctx context.Context,
	tx bun.IDB,
	cus *customer.Customer,
) error {
	log := r.l.With(
		zap.String("operation", "saveEmailProfile"),
		zap.String("customerID", cus.ID.String()),
	)

	if !cus.HasEmailProfile() {
		return nil
	}

	emailProfile := cus.EmailProfile
	emailProfile.CustomerID = cus.ID
	emailProfile.OrganizationID = cus.OrganizationID
	emailProfile.BusinessUnitID = cus.BusinessUnitID

	if _, err := tx.NewInsert().
		Model(emailProfile).
		On("CONFLICT (customer_id, organization_id, business_unit_id) DO UPDATE").
		Set("subject = EXCLUDED.subject").
		Set("comment = EXCLUDED.comment").
		Set("from_email = EXCLUDED.from_email").
		Set("to_recipients = EXCLUDED.to_recipients").
		Set("cc_recipients = EXCLUDED.cc_recipients").
		Set("bcc_recipients = EXCLUDED.bcc_recipients").
		Set("attachment_name = EXCLUDED.attachment_name").
		Set("read_receipt = EXCLUDED.read_receipt").
		Set("include_shipment_detail = EXCLUDED.include_shipment_detail").
		Set("version = cem.version + 1").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("*").
		Exec(ctx); err != nil {
		log.Error("failed to save email profile", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *customer.Customer,
) (*customer.Customer, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		ov := entity.Version
		entity.Version++

		entity = r.geocodeIfApplicable(entity)

		results, err := r.db.DBForContext(c).NewUpdate().
			Model(entity).
			WherePK().
			Where("version = ?", ov).
			Returning("*").
			Exec(c)
		if err != nil {
			log.Error("failed to update customer", zap.Error(err))
			return err
		}

		if err = dberror.CheckRowsAffected(results, "Customer", entity.ID.String()); err != nil {
			return err
		}

		if err = r.saveBillingProfile(c, tx, entity); err != nil {
			return err
		}

		if err = r.saveEmailProfile(c, tx, entity); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		log.Error("failed to update customer", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Customer is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateCustomerStatusRequest,
) ([]*customer.Customer, error) {
	log := r.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	entities := make([]*customer.Customer, 0, len(req.CustomerIDs))
	results, err := r.db.DB().
		NewUpdate().
		Model(&entities).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("cus.organization_id = ?", req.TenantInfo.OrgID).
				Where("cus.business_unit_id = ?", req.TenantInfo.BuID).
				Where("cus.id IN (?)", bun.List(req.CustomerIDs))
		}).
		Set("status = ?", req.Status).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to bulk update customer status", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckBulkRowsAffected(results, "Customer", req.CustomerIDs); err != nil {
		return nil, err
	}

	return entities, nil
}
