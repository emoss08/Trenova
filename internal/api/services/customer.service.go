package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/services/types"
	"github.com/emoss08/trenova/internal/queries"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/google/uuid"
)

// CustomerService provides methods for managing customers.
type CustomerService struct {
	Client       *ent.Client           // Client is the database client used for querying and mutating customer records.
	Logger       *zerolog.Logger       // Logger is used for logging messages.
	QueryService *queries.QueryService // QueryService provides methods for querying the database.
}

// NewCustomerService creates a new CustomerService.
// s is the server instance containing necessary dependencies.
//
// Parameters:
//   - s *api.Server: A pointer to an instance of api.Server which contains configuration and state needed by
//     CustomerService.
//
// Returns:
//   - *CustomerService: A pointer to the newly created CustomerService instance.
func NewCustomerService(s *api.Server) *CustomerService {
	return &CustomerService{
		Client: s.Client,
		Logger: s.Logger,
		QueryService: &queries.QueryService{
			Client: s.Client,
			Logger: s.Logger,
		},
	}
}

// GetCustomers retrieves a list of customers for a given organization and business unit.
// It returns a slice of Customer entities, the total number of customer records, and an error object.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - limit int: The maximum number of records to return.
//   - offset int: The number of records to skip before starting to return records.
//   - orgID uuid.UUID: The identifier of the organization.
//   - buID uuid.UUID: The identifier of the business unit.
//
// Returns:
//   - []*ent.Customer: A slice of Customer entities.
//   - int: The total number of customer records.
//   - error: An error object that indicates why the retrieval failed, nil if no error occurred.
func (r *CustomerService) GetCustomers(ctx context.Context, limit, offset int, orgID, buID uuid.UUID) ([]*ent.Customer, int, error) {
	return r.QueryService.GetCustomers(ctx, limit, offset, orgID, buID)
}

// CreateCustomer creates a new customer. It returns a pointer to the newly created Customer entity and an error object.
//
// Parameters:
//
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - entity *CustomerRequest: The customer request containing the details of the customer to be created.
//
// Returns:
//   - *ent.Customer: A pointer to the newly created Customer entity.
//
// Possible errors:
//   - Error creating customer entity
func (r *CustomerService) CreateCustomer(
	ctx context.Context, entity *types.CustomerRequest,
) (*ent.Customer, error) {
	createdEntity := new(ent.Customer)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error

		createdEntity, err = r.QueryService.CreateCustomerEntity(ctx, tx, entity)
		if err != nil {
			return err
		}

		// Create the customer rule profile
		if err = r.QueryService.CreateCustomerRuleProfileEntity(ctx, tx, createdEntity.ID, entity); err != nil {
			r.Logger.Err(err).Msg("Error creating customer rule profile")
			return err
		}

		// Create the email profile
		if err = r.QueryService.CreateCustomerEmailProfileEntity(ctx, tx, createdEntity.ID, entity); err != nil {
			r.Logger.Err(err).Msg("Error creating customer email profile")
			return err
		}

		// If comments are provided, create them and associate them with the customer
		if len(entity.Contacts) > 0 {
			if err = r.QueryService.CreateCustomerContacts(ctx, tx, createdEntity.ID, entity); err != nil {
				r.Logger.Err(err).Msg("Error creating customer contact")
				return err
			}
		}

		// if delivery slots are provided, create them and associate them with the customer
		if len(entity.DeliverySlots) > 0 {
			if err = r.QueryService.CreateDeliverySlots(ctx, tx, createdEntity.ID, entity); err != nil {
				r.Logger.Err(err).Msg("Error creating delivery slots")
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

// UpdateCustomer updates a customer.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - entity *CustomerUpdateRequest: The customer update request containing the details of the customer to be updated.
//
// Returns:
//   - *ent.Customer: A pointer to the updated Customer entity.
//   - error: An error object that indicates why the update failed, nil if no error occurred.
func (r *CustomerService) UpdateCustomer(ctx context.Context, entity *types.CustomerUpdateRequest) (*ent.Customer, error) {
	updatedEntity := new(ent.Customer)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.QueryService.UpdateCustomerEntity(ctx, tx, entity)
		if err != nil {
			return err
		}

		// Update the email profile
		if err = r.QueryService.UpdateCustomerEmailProfileEntity(ctx, tx, entity); err != nil {
			return err
		}

		// Update the rule profile
		if err = r.QueryService.UpdateCustomerRuleProfileEntity(ctx, tx, entity); err != nil {
			return err
		}

		// Sync delivery slots
		if err = r.QueryService.SyncDeliverySlots(ctx, tx, entity, updatedEntity); err != nil {
			return err
		}

		return r.QueryService.SyncCustomerContacts(ctx, tx, entity, updatedEntity)
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
