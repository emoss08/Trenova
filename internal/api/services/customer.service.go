package services

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/customer"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/google/uuid"
)

type CustomerService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

type CustomerRequest struct {
	BusinessUnitID      uuid.UUID                       `json:"businessUnitId"`
	OrganizationID      uuid.UUID                       `json:"organizationId"`
	CreatedAt           time.Time                       `json:"createdAt" validate:"omitempty"`
	UpdatedAt           time.Time                       `json:"updatedAt" validate:"omitempty"`
	Version             int                             `json:"version" validate:"omitempty"`
	Status              customer.Status                 `json:"status" validate:"required,oneof=A I"`
	Code                string                          `json:"code" validate:"required,max=10"`
	Name                string                          `json:"name" validate:"required,max=150"`
	AddressLine1        string                          `json:"addressLine1" validate:"required,max=150"`
	AddressLine2        string                          `json:"addressLine2" validate:"omitempty,max=150"`
	City                string                          `json:"city" validate:"required,max=150"`
	StateID             uuid.UUID                       `json:"stateId" validate:"omitempty,uuid"`
	PostalCode          string                          `json:"postalCode" validate:"required,max=10"`
	HasCustomerPortal   bool                            `json:"hasCustomerPortal" validate:"omitempty"`
	AutoMarkReadyToBill bool                            `json:"autoMarkReadyToBill" validate:"omitempty"`
	EmailProfile        ent.CustomerEmailProfile        `json:"emailProfile" validate:"omitempty"`
	RuleProfile         ent.CustomerRuleProfile         `json:"ruleProfile" validate:"omitempty"`
	DeliverySlots       []ent.DeliverySlot              `json:"deliverySlots" validate:"omitempty"`
	DetentionPolicies   []ent.CustomerDetentionPolicies `json:"detentionPolicies" validate:"omitempty"`
	Contacts            []ent.CustomerContact           `json:"contacts" validate:"omitempty"`
	Edges               ent.CustomerEdges               `json:"edges" validate:"omitempty"`
}

type CustomerUpdateRequest struct {
	CustomerRequest
	ID uuid.UUID `json:"id,omitempty"`
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
		WithContacts().
		WithDeliverySlots().
		WithDetentionPolicies().
		WithEmailProfile().
		WithRuleProfile().
		WithState().
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
	ctx context.Context, entity *CustomerRequest,
) (*ent.Customer, error) {
	createdEntity := new(ent.Customer)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		createdEntity, err = r.createCustomerEntity(ctx, tx, entity)
		if err != nil {
			return err
		}

		// If comments are provided, create them and associate them with the customer
		if len(entity.Contacts) > 0 {
			if err = r.createCustomerContact(ctx, tx, createdEntity.ID, entity); err != nil {
				r.Logger.Err(err).Msg("Error creating customer contact")
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

func (r *CustomerService) createCustomerEntity(
	ctx context.Context, tx *ent.Tx, entity *CustomerRequest,
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
		return nil, err
	}

	return createdEntity, nil
}

func (r *CustomerService) createCustomerContact(
	ctx context.Context, tx *ent.Tx, customerID uuid.UUID, entity *CustomerRequest,
) error {
	for _, contact := range entity.Contacts {
		_, err := tx.CustomerContact.Create().
			SetBusinessUnitID(entity.BusinessUnitID).
			SetOrganizationID(entity.OrganizationID).
			SetCustomerID(customerID).
			SetName(contact.Name).
			SetEmail(contact.Email).
			SetTitle(contact.Title).
			SetPhoneNumber(contact.PhoneNumber).
			SetIsPayableContact(contact.IsPayableContact).
			Save(ctx)
		if err != nil {
			r.Logger.Error().Err(err).Msg("Error creating customer contact")
			return err
		}
	}

	return nil
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
		return nil, err
	}

	// Check if the version matches.
	if current.Version != entity.Version {
		return nil, util.NewValidationError("This record has been updated by another user. Please refresh and try again",
			"syncError",
			"code")
	}

	// Start building the update operation
	updateOp := tx.Customer.UpdateOneID(entity.ID).
		SetOrganizationID(entity.OrganizationID).
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
		return nil, err
	}

	return updatedEntity, nil
}
