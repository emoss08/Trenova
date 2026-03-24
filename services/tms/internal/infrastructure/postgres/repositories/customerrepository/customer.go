package customerrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/m2msync"
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
	DocRepo repositories.DocumentTypeRepository
	Logger  *zap.Logger
	M2MSync *m2msync.Syncer
}

type repository struct {
	db      *postgres.Connection
	docRepo repositories.DocumentTypeRepository
	l       *zap.Logger
	m2mSync *m2msync.Syncer
}

func New(p Params) repositories.CustomerRepository {
	return &repository{
		db:      p.DB,
		docRepo: p.DocRepo,
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

	return q.Limit(req.Filter.Pagination.Limit).Offset(req.Filter.Pagination.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListCustomerRequest,
) (*pagination.ListResult[*customer.Customer], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*customer.Customer, 0, req.Filter.Pagination.Limit)
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

func (r *repository) GetDocumentRequirements(
	ctx context.Context,
	cusID pulid.ID,
) ([]*repositories.CustomerDocRequirementResponse, error) {
	log := r.l.With(
		zap.String("operation", "GetDocumentRequirements"),
		zap.String("customerID", cusID.String()),
	)

	billingProfile, err := r.GetBillingProfile(ctx, cusID)
	if err != nil {
		log.Error("failed to get customer billing profile", zap.Error(err))
		return nil, err
	}

	response := make(
		[]*repositories.CustomerDocRequirementResponse,
		0,
		len(billingProfile.DocumentTypes),
	)

	for _, docType := range billingProfile.DocumentTypes {
		response = append(response, &repositories.CustomerDocRequirementResponse{
			Name:        docType.Name,
			DocID:       docType.ID.String(),
			Description: docType.Description,
			Color:       docType.Color,
		})
	}

	return response, nil
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
			return sq.Where("cus.organization_id = ?", req.TenantInfo.OrgID).
				Where("cus.business_unit_id = ?", req.TenantInfo.BuID).
				Where("cus.id IN (?)", bun.List(req.CustomerIDs))
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

	if _, err := tx.NewInsert().
		Model(billingProfile).
		On("CONFLICT (customer_id, organization_id, business_unit_id) DO UPDATE").
		Set("billing_cycle_type = EXCLUDED.billing_cycle_type").
		Set("billing_cycle_day_of_week = EXCLUDED.billing_cycle_day_of_week").
		Set("payment_term = EXCLUDED.payment_term").
		Set("has_billing_control_overrides = EXCLUDED.has_billing_control_overrides").
		Set("credit_limit = EXCLUDED.credit_limit").
		Set("credit_balance = EXCLUDED.credit_balance").
		Set("credit_status = EXCLUDED.credit_status").
		Set("enforce_credit_limit = EXCLUDED.enforce_credit_limit").
		Set("auto_credit_hold = EXCLUDED.auto_credit_hold").
		Set("credit_hold_reason = EXCLUDED.credit_hold_reason").
		Set("invoice_method = EXCLUDED.invoice_method").
		Set("summary_transmit_on_generation = EXCLUDED.summary_transmit_on_generation").
		Set("allow_invoice_consolidation = EXCLUDED.allow_invoice_consolidation").
		Set("consolidation_period_days = EXCLUDED.consolidation_period_days").
		Set("consolidation_group_by = EXCLUDED.consolidation_group_by").
		Set("invoice_number_format = EXCLUDED.invoice_number_format").
		Set("customer_invoice_prefix = EXCLUDED.customer_invoice_prefix").
		Set("invoice_copies = EXCLUDED.invoice_copies").
		Set("revenue_account_id = EXCLUDED.revenue_account_id").
		Set("ar_account_id = EXCLUDED.ar_account_id").
		Set("apply_late_charges = EXCLUDED.apply_late_charges").
		Set("late_charge_rate = EXCLUDED.late_charge_rate").
		Set("grace_period_days = EXCLUDED.grace_period_days").
		Set("tax_exempt = EXCLUDED.tax_exempt").
		Set("tax_exempt_number = EXCLUDED.tax_exempt_number").
		Set("enforce_customer_billing_req = EXCLUDED.enforce_customer_billing_req").
		Set("validate_customer_rates = EXCLUDED.validate_customer_rates").
		Set("auto_transfer = EXCLUDED.auto_transfer").
		Set("auto_mark_ready_to_bill = EXCLUDED.auto_mark_ready_to_bill").
		Set("auto_bill = EXCLUDED.auto_bill").
		Set("detention_billing_enabled = EXCLUDED.detention_billing_enabled").
		Set("detention_free_minutes = EXCLUDED.detention_free_minutes").
		Set("detention_rate_per_hour = EXCLUDED.detention_rate_per_hour").
		Set("auto_apply_accessorials = EXCLUDED.auto_apply_accessorials").
		Set("billing_currency = EXCLUDED.billing_currency").
		Set("require_po_number = EXCLUDED.require_po_number").
		Set("require_bol_number = EXCLUDED.require_bol_number").
		Set("require_delivery_number = EXCLUDED.require_delivery_number").
		Set("billing_notes = EXCLUDED.billing_notes").
		Set("version = cbp.version + 1").
		Set("updated_at = EXCLUDED.updated_at").
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
		Set("send_invoice_on_generation = EXCLUDED.send_invoice_on_generation").
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
