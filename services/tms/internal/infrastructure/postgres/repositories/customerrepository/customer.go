package customerrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/m2msync"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/querybuilder"
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

func NewRepository(p Params) repositories.CustomerRepository {
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
	opts *repositories.ListCustomerRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"cus",
		opts.Filter,
		(*customer.Customer)(nil),
	)

	q = q.Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.addOptions(sq, opts.CustomerFilterOptions)
	})

	return q.Limit(opts.Filter.Limit).Offset(opts.Filter.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListCustomerRequest,
) (*pagination.ListResult[*customer.Customer], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.String("orgId", req.Filter.TenantOpts.OrgID.String()),
		zap.String("buId", req.Filter.TenantOpts.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*customer.Customer, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&entities).Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.filterQuery(sq, req)
	}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan customers", zap.Error(err))
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
		zap.String("entityID", req.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(customer.Customer)
	err = db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("cus.id = ?", req.ID).
				Where("cus.organization_id = ?", req.OrgID).
				Where("cus.business_unit_id = ?", req.BuID)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.addOptions(sq, req.CustomerFilterOptions)
		}).
		Scan(ctx)
	if err != nil {
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

	billingProfile, err := r.getBillingProfile(ctx, cusID)
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

func (r *repository) getBillingProfile(
	ctx context.Context,
	cusID pulid.ID,
) (*customer.CustomerBillingProfile, error) {
	log := r.l.With(
		zap.String("operation", "getBillingProfile"),
		zap.String("customerID", cusID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	profile := new(customer.CustomerBillingProfile)
	query := db.NewSelect().Model(profile).
		Where("cbr.customer_id = ?", cusID).
		Relation("DocumentTypes")

	if err = query.Scan(ctx); err != nil {
		log.Error("failed to get billing profile", zap.Error(err))
		return nil, err
	}

	return profile, nil
}

func (r *repository) Create(
	ctx context.Context,
	cus *customer.Customer,
) (*customer.Customer, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("orgID", cus.OrganizationID.String()),
		zap.String("buID", cus.BusinessUnitID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	err = db.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		cus = r.geocodeIfApplicable(cus)

		if _, err = tx.NewInsert().Model(cus).Returning("*").Exec(c); err != nil {
			log.Error("failed to insert customer", zap.Error(err))
			return err
		}

		if err = r.createOrUpdateBillingProfile(c, tx, cus); err != nil {
			return err
		}

		if err = r.createOrUpdateEmailProfile(c, tx, cus); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		log.Error("failed to create customer", zap.Error(err))
		return nil, err
	}

	return cus, nil
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

func (r *repository) createOrUpdateBillingProfile(
	ctx context.Context,
	tx bun.Tx,
	cus *customer.Customer,
) error {
	log := r.l.With(
		zap.String("operation", "createOrUpdateBillingProfile"),
		zap.String("customerID", cus.ID.String()),
	)

	var billingProfile *customer.CustomerBillingProfile

	if cus.HasBillingProfile() {
		billingProfile = cus.BillingProfile
	} else {
		billingProfile = new(customer.CustomerBillingProfile)
	}

	billingProfile.CustomerID = cus.ID
	billingProfile.OrganizationID = cus.OrganizationID
	billingProfile.BusinessUnitID = cus.BusinessUnitID

	if _, err := tx.NewInsert().Model(billingProfile).
		Returning("*").
		Exec(ctx); err != nil {
		log.Error("failed to insert billing profile", zap.Error(err))
		return err
	}

	if cus.HasBillingProfile() && len(cus.BillingProfile.DocumentTypes) > 0 {
		if err := r.syncBillingProfileDocumentTypes(ctx, tx, billingProfile); err != nil {
			return err
		}
	}

	return nil
}

func (r *repository) createOrUpdateEmailProfile(
	ctx context.Context,
	tx bun.Tx,
	cus *customer.Customer,
) error {
	log := r.l.With(
		zap.String("operation", "createOrUpdateEmailProfile"),
		zap.String("customerID", cus.ID.String()),
	)

	var emailProfile *customer.CustomerEmailProfile

	if cus.HasEmailProfile() {
		emailProfile = cus.EmailProfile
	} else {
		emailProfile = new(customer.CustomerEmailProfile)
	}

	emailProfile.CustomerID = cus.ID
	emailProfile.OrganizationID = cus.OrganizationID
	emailProfile.BusinessUnitID = cus.BusinessUnitID

	if _, err := tx.NewInsert().Model(emailProfile).
		Returning("*").
		Exec(ctx); err != nil {
		log.Error("failed to insert email profile", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) Update(
	ctx context.Context,
	cus *customer.Customer,
) (*customer.Customer, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", cus.GetID()),
		zap.Int64("version", cus.Version),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	err = db.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := cus.Version
		cus.Version++

		cus = r.geocodeIfApplicable(cus)

		results, rErr := tx.NewUpdate().
			Model(cus).
			Where("cus.version = ?", ov).
			WherePK().
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error("failed to update customer", zap.Error(rErr))
			return rErr
		}

		if err = dberror.CheckRowsAffected(results, "Customer", cus.GetID()); err != nil {
			return err
		}

		if cus.HasBillingProfile() {
			if err = r.updateBillingProfile(c, tx, cus.BillingProfile); err != nil {
				return err
			}
		}

		if cus.HasEmailProfile() {
			if err = r.updateEmailProfile(c, tx, cus.EmailProfile); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		log.Error("failed to update customer", zap.Error(err))
		return nil, err
	}

	return cus, nil
}

func (r *repository) geocodeIfApplicable(entity *customer.Customer) *customer.Customer {
	if entity.PlaceID == "" || entity.Latitude == nil || entity.Longitude == nil {
		entity.IsGeocoded = false
		return entity
	}

	entity.IsGeocoded = true
	return entity
}

func (r *repository) updateBillingProfile(
	ctx context.Context,
	tx bun.IDB,
	profile *customer.CustomerBillingProfile,
) error {
	log := r.l.With(
		zap.String("operation", "updateBillingProfile"),
		zap.String("id", profile.GetID()),
		zap.Int64("version", profile.Version),
	)

	_, rErr := tx.NewUpdate().
		Model(profile).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.
				Where("cbr.id = ?", profile.GetID()).
				Where("cbr.organization_id = ?", profile.OrganizationID).
				Where("cbr.business_unit_id = ?", profile.BusinessUnitID).
				Where("cbr.customer_id = ?", profile.CustomerID)
		}).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update billing profile", zap.Error(rErr))
		return rErr
	}

	if err := r.syncBillingProfileDocumentTypes(ctx, tx, profile); err != nil {
		return err
	}

	return nil
}

func (r *repository) updateEmailProfile(
	ctx context.Context,
	tx bun.IDB,
	profile *customer.CustomerEmailProfile,
) error {
	log := r.l.With(
		zap.String("operation", "updateEmailProfile"),
		zap.String("id", profile.GetID()),
		zap.Int64("version", profile.Version),
	)

	_, err := tx.NewUpdate().
		Model(profile).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.
				Where("cem.id = ?", profile.GetID()).
				Where("cem.customer_id = ?", profile.CustomerID).
				Where("cem.organization_id = ?", profile.OrganizationID).
				Where("cem.business_unit_id = ?", profile.BusinessUnitID)
		}).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update email profile", zap.Error(err))
		return err
	}

	return nil
}
