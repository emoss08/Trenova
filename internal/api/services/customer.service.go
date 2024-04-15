package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/customer"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/google/uuid"
	"github.com/rotisserie/eris"
)

type CustomerService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewCustomerService creates a new customer service.
func NewCustomerService(s *api.Server) *CustomerService {
	return &CustomerService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetCustomers gets the customer for an organization.
func (r *CustomerService) GetCustomers(ctx context.Context, limit, offset int, orgID, buID uuid.UUID) ([]*ent.Customer, int, error) {
	entityCount, countErr := r.Client.Customer.Query().Where(
		customer.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.Customer.Query().
		Limit(limit).
		Offset(offset).
		Where(
			customer.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateCustomer creates a new customer.
func (r *CustomerService) CreateCustomer(
	ctx context.Context, entity *ent.Customer,
) (*ent.Customer, error) {
	updatedEntity := new(ent.Customer)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.createCustomerEntity(ctx, tx, entity)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *CustomerService) createCustomerEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.Customer,
) (*ent.Customer, error) {
	createdEntity, err := tx.Customer.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetName(entity.Name).
		SetAddressLine1(entity.AddressLine1).
		SetAddressLine2(entity.AddressLine2).
		SetCity(entity.City).
		SetStateID(entity.StateID).
		SetPostalCode(entity.PostalCode).
		SetHasCustomerPortal(entity.HasCustomerPortal).
		SetAutoMarkReadyToBill(entity.AutoMarkReadyToBill).
		Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to create customer")
	}

	return createdEntity, nil
}

// UpdateCustomer updates a customer.
func (r *CustomerService) UpdateCustomer(ctx context.Context, entity *ent.Customer) (*ent.Customer, error) {
	updatedEntity := new(ent.Customer)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateCustomerEntity(ctx, tx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *CustomerService) updateCustomerEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.Customer,
) (*ent.Customer, error) {
	current, err := tx.Customer.Get(ctx, entity.ID)
	if err != nil {
		return nil, eris.Wrap(err, "failed to retrieve requested entity")
	}

	// Check if the version matches.
	if current.Version != entity.Version {
		return nil, util.NewValidationError("This record has been updated by another user. Please refresh and try again",
			"syncError",
			"code")
	}

	// Start building the update operation
	updateOp := tx.Customer.UpdateOneID(entity.ID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetName(entity.Name).
		SetAddressLine1(entity.AddressLine1).
		SetAddressLine2(entity.AddressLine2).
		SetCity(entity.City).
		SetStateID(entity.StateID).
		SetPostalCode(entity.PostalCode).
		SetHasCustomerPortal(entity.HasCustomerPortal).
		SetAutoMarkReadyToBill(entity.AutoMarkReadyToBill).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to update entity")
	}

	return updatedEntity, nil
}
