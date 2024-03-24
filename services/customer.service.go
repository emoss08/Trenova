package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/customer"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type CustomerOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewCustomerOps creates a new customer service.
func NewCustomerOps(ctx context.Context) *CustomerOps {
	return &CustomerOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetCustomer gets the customer for an organization.
func (r *CustomerOps) GetCustomers(limit, offset int, orgID, buID uuid.UUID) ([]*ent.Customer, int, error) {
	customerCount, countErr := r.client.Customer.Query().Where(
		customer.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(r.ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	customers, err := r.client.Customer.Query().
		Limit(limit).
		Offset(offset).
		Where(
			customer.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return customers, customerCount, nil
}

// CreateCustomer creates a new customer.
func (r *CustomerOps) CreateCustomer(newCustomer ent.Customer) (*ent.Customer, error) {
	customer, err := r.client.Customer.Create().
		SetOrganizationID(newCustomer.OrganizationID).
		SetBusinessUnitID(newCustomer.BusinessUnitID).
		SetStatus(newCustomer.Status).
		SetCode(newCustomer.Code).
		SetName(newCustomer.Name).
		SetAddressLine1(newCustomer.AddressLine1).
		SetAddressLine2(newCustomer.AddressLine2).
		SetCity(newCustomer.City).
		SetState(newCustomer.State).
		SetPostalCode(newCustomer.PostalCode).
		SetHasCustomerPortal(newCustomer.HasCustomerPortal).
		SetAutoMarkReadyToBill(newCustomer.AutoMarkReadyToBill).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return customer, nil
}

// UpdateCustomer updates a customer.
func (r *CustomerOps) UpdateCustomer(customer ent.Customer) (*ent.Customer, error) {
	// Start building the update operation
	updateOp := r.client.Customer.UpdateOneID(customer.ID).
		SetStatus(customer.Status).
		SetCode(customer.Code).
		SetName(customer.Name).
		SetAddressLine1(customer.AddressLine1).
		SetAddressLine2(customer.AddressLine2).
		SetCity(customer.City).
		SetState(customer.State).
		SetPostalCode(customer.PostalCode).
		SetHasCustomerPortal(customer.HasCustomerPortal).
		SetAutoMarkReadyToBill(customer.AutoMarkReadyToBill)

	// Execute the update operation
	updatedCustomer, err := updateOp.Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updatedCustomer, nil
}
