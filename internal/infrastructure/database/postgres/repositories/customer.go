package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/postgressearch"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils/queryfilters"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// CustomerRepositoryParams defines dependencies required for initializing the CustomerRepository.
// This includes database connection, document type repository, and logger.
type CustomerRepositoryParams struct {
	fx.In

	DB      db.Connection
	DocRepo repositories.DocumentTypeRepository
	Logger  *logger.Logger
}

// customerRepository implements the CustomerRepository interface
// and provides methods to manage customer data, including CRUD operations.
type customerRepository struct {
	db      db.Connection
	docRepo repositories.DocumentTypeRepository
	l       *zerolog.Logger
}

// NewCustomerRepository initializes a new instance of customerRepository with its dependencies.
//
// Parameters:
//   - p: CustomerRepositoryParams containing dependencies.
//
// Returns:
//   - repositories.CustomerRepository: A ready-to-use customer repository instance.
func NewCustomerRepository(p CustomerRepositoryParams) repositories.CustomerRepository {
	log := p.Logger.With().
		Str("repository", "customer").
		Logger()

	return &customerRepository{
		db:      p.DB,
		docRepo: p.DocRepo,
		l:       &log,
	}
}

// filterQuery applies filters and pagination to the customer query.
// It includes tenant-based filtering and full-text search when provided.
//
// Parameters:
//   - q: The base select query.
//   - opts: ListCustomerOptions containing filter and pagination details.
//
// Returns:
//   - *bun.SelectQuery: The filtered and paginated query.
func (cr *customerRepository) filterQuery(q *bun.SelectQuery, opts *repositories.ListCustomerOptions) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "cus",
		Filter:     opts.Filter,
	})

	if opts.IncludeState {
		q = q.Relation("State")
	}

	if opts.IncludeBillingProfile {
		q = q.Relation("BillingProfile")
	}

	if opts.IncludeEmailProfile {
		q = q.Relation("EmailProfile")
	}

	if opts.Filter.Query != "" {
		q = postgressearch.BuildSearchQuery(
			q,
			opts.Filter.Query,
			(*customer.Customer)(nil),
		)
	}

	return q.Limit(opts.Filter.Limit).Offset(opts.Filter.Offset)
}

// List retrieves a list of customers based on the provided options.
//
// Parameters:
//   - ctx: The context for the operation.
//   - opts: ListCustomerOptions containing filter and pagination details.
//
// Returns:
//   - *ports.ListResult[*customer.Customer]: A list of customers.
//   - error: An error if the operation fails.
func (cr *customerRepository) List(ctx context.Context, opts *repositories.ListCustomerOptions) (*ports.ListResult[*customer.Customer], error) {
	dba, err := cr.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	log := cr.l.With().
		Str("operation", "List").
		Str("buID", opts.Filter.TenantOpts.BuID.String()).
		Str("userID", opts.Filter.TenantOpts.UserID.String()).
		Logger()

	entities := make([]*customer.Customer, 0)

	q := dba.NewSelect().Model(&entities)
	q = cr.filterQuery(q, opts)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan customers")
		return nil, err
	}

	return &ports.ListResult[*customer.Customer]{
		Items: entities,
		Total: total,
	}, nil
}

// GetByID retrieves a customer by their ID.
//
// Parameters:
//   - ctx: The context for the operation.
//   - opts: GetCustomerByIDOptions containing customer ID and tenant options.
//
// Returns:
//   - *customer.Customer: The customer entity.
//   - error: An error if the operation fails.
func (cr *customerRepository) GetByID(ctx context.Context, opts repositories.GetCustomerByIDOptions) (*customer.Customer, error) {
	dba, err := cr.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	log := cr.l.With().
		Str("operation", "GetByID").
		Str("customerID", opts.ID.String()).
		Logger()

	entity := new(customer.Customer)

	query := dba.NewSelect().Model(entity).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.
				Where("cus.id = ?", opts.ID).
				Where("cus.organization_id = ?", opts.OrgID).
				Where("cus.business_unit_id = ?", opts.BuID)
		})

	// * Include the state if requested
	if opts.IncludeState {
		query = query.Relation("State")
	}

	// * Include the billing profile if requested
	if opts.IncludeBillingProfile {
		query = query.Relation("BillingProfile")
	}

	// * Include the email profile if requested
	if opts.IncludeEmailProfile {
		query = query.Relation("EmailProfile")
	}

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("Customer not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get customer")
		return nil, err
	}

	return entity, nil
}

// GetDocumentRequirements retrieves the document requirements for a customer.
//
// Parameters:
//   - ctx: The context for the operation.
//   - cusID: The ID of the customer.
//
// Returns:
//   - []*repositories.CustomerDocRequirementResponse: A list of document requirements.
//   - error: An error if the operation fails.
func (cr *customerRepository) GetDocumentRequirements(ctx context.Context, cusID pulid.ID) ([]*repositories.CustomerDocRequirementResponse, error) {
	log := cr.l.With().
		Str("operation", "GetDocumentRequirements").
		Str("customerID", cusID.String()).
		Logger()

	// * Get the customer billing profile
	billingProfile, err := cr.getBillingProfile(ctx, cusID, "document_type_ids")
	if err != nil {
		log.Error().Err(err).Msg("failed to get customer billing profile")
		return nil, err
	}

	// * Get the document types
	docTypes, err := cr.docRepo.GetByIDs(ctx, billingProfile.DocumentTypeIDs)
	if err != nil {
		log.Error().Err(err).Msg("failed to get document types")
		return nil, err
	}

	// * Create the response with the exact capacity needed
	response := make([]*repositories.CustomerDocRequirementResponse, 0, len(docTypes))

	// * Iterate over the document types and create the response
	for _, docType := range docTypes {
		response = append(response, &repositories.CustomerDocRequirementResponse{
			Name:        docType.Name,
			DocID:       docType.ID.String(),
			Description: docType.Description,
			Color:       docType.Color,
		})
	}

	return response, nil
}

// getBillingProfile gets and returns a billing profile for a customer.
// If fields are provided, only the specified fields are retrieved.
//
// Parameters:
//   - ctx: The context for the operation.
//   - cusID: The ID of the customer.
//   - fields: Optional fields to retrieve from the billing profile.
//
// Returns:
//   - *customer.BillingProfile: The billing profile entity.
//   - error: An error if the operation fails.
func (cr *customerRepository) getBillingProfile(ctx context.Context, cusID pulid.ID, fields ...string) (*customer.BillingProfile, error) {
	dba, err := cr.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	log := cr.l.With().
		Str("operation", "getBillingProfile").
		Str("customerID", cusID.String()).
		Logger()

	profile := new(customer.BillingProfile)
	query := dba.NewSelect().Model(profile).
		Where("cbr.customer_id = ?", cusID)

	// If specific fields are requested, only select those
	if len(fields) > 0 {
		query = query.Column(fields...)
	}

	if err = query.Scan(ctx); err != nil {
		log.Error().Err(err).Msg("failed to get billing profile")
		return nil, err
	}

	return profile, nil
}

// Create a customer and ensure it has a billing profile and email profile
//
// Parameters:
//   - ctx: The context for the operation.
//   - cus: The customer entity to create.
//
// Returns:
//   - *customer.Customer: The created customer entity.
//   - error: An error if the operation fails.
func (cr *customerRepository) Create(ctx context.Context, cus *customer.Customer) (*customer.Customer, error) {
	dba, err := cr.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	log := cr.l.With().
		Str("operation", "Create").
		Str("orgID", cus.OrganizationID.String()).
		Str("buID", cus.BusinessUnitID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		// Insert the customer first
		if _, iErr := tx.NewInsert().Model(cus).Returning("*").Exec(c); iErr != nil {
			log.Error().
				Err(iErr).
				Interface("customer", cus).
				Msg("failed to insert customer")
			return iErr
		}

		// Create or update the billing profile
		if iErr := cr.createOrUpdateBillingProfile(c, tx, cus); iErr != nil {
			return iErr
		}

		// Create or update the email profile
		if iErr := cr.createOrUpdateEmailProfile(c, tx, cus); iErr != nil {
			return iErr
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to create customer")
		return nil, err
	}

	return cus, nil
}

// createOrUpdateBillingProfile ensures a customer has a billing profile
// If the customer already has a billing profile, it's used; otherwise a default one is created
//
// Parameters:
//   - ctx: The context for the operation.
//   - tx: The database transaction.
//   - cus: The customer entity to create.
//
// Returns:
//   - error: An error if the operation fails.
func (cr *customerRepository) createOrUpdateBillingProfile(ctx context.Context, tx bun.Tx, cus *customer.Customer) error {
	log := cr.l.With().
		Str("operation", "createOrUpdateBillingProfile").
		Str("customerID", cus.ID.String()).
		Logger()

	// Check if the customer already has a billing profile
	if cus.HasBillingProfile() {
		// Update the billing profile with the new customer ID
		cus.BillingProfile.CustomerID = cus.ID
		cus.BillingProfile.OrganizationID = cus.OrganizationID
		cus.BillingProfile.BusinessUnitID = cus.BusinessUnitID

		// Insert the existing billing profile
		if _, err := tx.NewInsert().Model(cus.BillingProfile).
			Returning("*").
			Exec(ctx); err != nil {
			log.Error().
				Err(err).
				Interface("billingProfile", cus.BillingProfile).
				Msg("failed to insert billing profile")
			return eris.Wrap(err, "insert billing profile")
		}

		return nil
	}

	// Create default billing profile
	billingProfile := new(customer.BillingProfile)
	billingProfile.CustomerID = cus.ID
	billingProfile.OrganizationID = cus.OrganizationID
	billingProfile.BusinessUnitID = cus.BusinessUnitID

	// Insert the default billing profile
	if _, err := tx.NewInsert().Model(billingProfile).
		Returning("*").
		Exec(ctx); err != nil {
		log.Error().
			Err(err).
			Interface("billingProfile", billingProfile).
			Msg("failed to insert billing profile")
		return eris.Wrap(err, "insert billing profile")
	}

	return nil
}

// createOrUpdateEmailProfile ensures a customer has an email profile
// If the customer already has an email profile, it's used; otherwise a default one is created
//
// Parameters:
//   - ctx: The context for the operation.
//   - tx: The database transaction.
//   - cus: The customer entity to create.
//
// Returns:
//   - error: An error if the operation fails.
func (cr *customerRepository) createOrUpdateEmailProfile(ctx context.Context, tx bun.Tx, cus *customer.Customer) error {
	log := cr.l.With().
		Str("operation", "createOrUpdateEmailProfile").
		Str("customerID", cus.ID.String()).
		Logger()

	// Check if the customer already has an email profile
	if cus.HasEmailProfile() {
		// Update the email profile with the new customer ID
		cus.EmailProfile.CustomerID = cus.ID
		cus.EmailProfile.OrganizationID = cus.OrganizationID
		cus.EmailProfile.BusinessUnitID = cus.BusinessUnitID

		// Insert the existing email profile
		if _, err := tx.NewInsert().Model(cus.EmailProfile).
			Returning("*").
			Exec(ctx); err != nil {
			log.Error().
				Err(err).
				Interface("emailProfile", cus.EmailProfile).
				Msg("failed to insert email profile")
			return eris.Wrap(err, "insert email profile")
		}

		return nil
	}

	// Create default email profile
	emailProfile := new(customer.CustomerEmailProfile)
	emailProfile.CustomerID = cus.ID
	emailProfile.OrganizationID = cus.OrganizationID
	emailProfile.BusinessUnitID = cus.BusinessUnitID

	// Insert the default email profile
	if _, err := tx.NewInsert().Model(emailProfile).
		Returning("*").
		Exec(ctx); err != nil {
		log.Error().
			Err(err).
			Interface("emailProfile", emailProfile).
			Msg("failed to insert email profile")
		return eris.Wrap(err, "insert email profile")
	}

	return nil
}

// Update updates a customer and ensures it has a billing profile and email profile
//
// Parameters:
//   - ctx: The context for the operation.
//   - cus: The customer entity to update.
//
// Returns:
//   - *customer.Customer: The updated customer entity.
//   - error: An error if the operation fails.
func (cr *customerRepository) Update(ctx context.Context, cus *customer.Customer) (*customer.Customer, error) {
	dba, err := cr.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	log := cr.l.With().
		Str("operation", "Update").
		Str("id", cus.GetID()).
		Int64("version", cus.Version).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := cus.Version

		cus.Version++

		results, rErr := tx.NewUpdate().
			Model(cus).
			Where("cus.version = ?", ov).
			WherePK().
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("customer", cus).
				Msg("failed to update customer")
			return rErr
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("customer", cus).
				Msg("failed to get rows affected")
			return roErr
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf("Version mismatch. The Customer (%s) has either been updated or deleted since the last request.", cus.GetID()),
			)
		}

		if cus.HasBillingProfile() {
			if err = cr.updateBillingProfile(c, cus.BillingProfile); err != nil {
				return err
			}
		}

		if cus.HasEmailProfile() {
			if err = cr.updateEmailProfile(c, cus.EmailProfile); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update customer")
		return nil, err
	}

	return cus, nil
}

// updateBillingProfile updates a billing profile
//
// Parameters:
//   - ctx: The context for the operation.
//   - profile: The billing profile entity to update.
//
// Returns:
//   - error: An error if the operation fails.
func (cr *customerRepository) updateBillingProfile(ctx context.Context, profile *customer.BillingProfile) error {
	dba, err := cr.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	log := cr.l.With().
		Str("operation", "UpdateBillingProfile").
		Str("id", profile.GetID()).
		Int64("version", profile.Version).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := profile.Version

		profile.Version++

		results, rErr := tx.NewUpdate().
			Model(profile).
			Where("cbr.version = ?", ov).
			WherePK().
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("billingProfile", profile).
				Msg("failed to update billing profile")
			return rErr
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("billingProfile", profile).
				Msg("failed to get rows affected")
			return roErr
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf("Version mismatch. The Billing Profile (%s) has either been updated or deleted since the last request.", profile.GetID()),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update billing profile")
		return err
	}

	return nil
}

// updateEmailProfile updates an email profile
//
// Parameters:
//   - ctx: The context for the operation.
//   - profile: The email profile entity to update.
//
// Returns:
//   - error: An error if the operation fails.
func (cr *customerRepository) updateEmailProfile(ctx context.Context, profile *customer.CustomerEmailProfile) error {
	dba, err := cr.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	log := cr.l.With().
		Str("operation", "UpdateEmailProfile").
		Str("id", profile.GetID()).
		Int64("version", profile.Version).
		Logger()

	log.Info().
		Interface("emailProfile", profile).
		Msg("updating email profile")

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := profile.Version

		profile.Version++

		results, rErr := tx.NewUpdate().
			Model(profile).
			Where("cem.version = ?", ov).
			WherePK().
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("emailProfile", profile).
				Msg("failed to update email profile")
			return rErr
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("emailProfile", profile).
				Msg("failed to get rows affected")
			return roErr
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf("Version mismatch. The Email Profile (%s) has either been updated or deleted since the last request.", profile.GetID()),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update email profile")
		return err
	}

	return nil
}
