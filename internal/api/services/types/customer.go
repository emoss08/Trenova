package types

import (
	"time"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/customer"
	"github.com/emoss08/trenova/internal/ent/customerruleprofile"
	"github.com/google/uuid"
)

// CustomerRuleProfileRequest represents a request to create or update a customer rule profile.
type CustomerRuleProfileRequest struct {
	ID             uuid.UUID                        `json:"id,omitempty"`
	BusinessUnitID uuid.UUID                        `json:"businessUnitId"`
	OrganizationID uuid.UUID                        `json:"organizationId"`
	CreatedAt      time.Time                        `json:"createdAt" validate:"omitempty"`
	UpdatedAt      time.Time                        `json:"updatedAt" validate:"omitempty"`
	Version        int                              `json:"version" validate:"omitempty"`
	CustomerID     uuid.UUID                        `json:"customer_id,omitempty"`
	BillingCycle   customerruleprofile.BillingCycle `json:"billingCycle" validate:"required,oneof=PER_JOB QUARTERLY MONTHLY ANNUALLY"`
	DocClassIDs    []uuid.UUID                      `json:"docClassIds,omitempty"`
}

// CustomerRequest represents a request to create or update a customer.
type CustomerRequest struct {
	BusinessUnitID      uuid.UUID                       `json:"businessUnitId"`
	OrganizationID      uuid.UUID                       `json:"organizationId"`
	CreatedAt           time.Time                       `json:"createdAt" validate:"omitempty"`
	UpdatedAt           time.Time                       `json:"updatedAt" validate:"omitempty"`
	Version             int                             `json:"version" validate:"omitempty"`
	Status              customer.Status                 `json:"status" validate:"required,oneof=A I"`
	Code                string                          `json:"code" validate:"omitempty"` // Auto generated in the hooks.
	Name                string                          `json:"name" validate:"required,max=150"`
	AddressLine1        string                          `json:"addressLine1" validate:"required,max=150"`
	AddressLine2        string                          `json:"addressLine2" validate:"omitempty,max=150"`
	City                string                          `json:"city" validate:"required,max=150"`
	StateID             uuid.UUID                       `json:"stateId" validate:"omitempty,uuid"`
	PostalCode          string                          `json:"postalCode" validate:"required,max=10"`
	HasCustomerPortal   bool                            `json:"hasCustomerPortal" validate:"omitempty"`
	AutoMarkReadyToBill bool                            `json:"autoMarkReadyToBill" validate:"omitempty"`
	EmailProfile        ent.CustomerEmailProfile        `json:"emailProfile" validate:"omitempty"`
	RuleProfile         CustomerRuleProfileRequest      `json:"ruleProfile" validate:"omitempty"`
	DeliverySlots       []ent.DeliverySlot              `json:"deliverySlots" validate:"omitempty"`
	DetentionPolicies   []ent.CustomerDetentionPolicies `json:"detentionPolicies" validate:"omitempty"`
	Contacts            []ent.CustomerContact           `json:"contacts" validate:"omitempty"`
	Edges               ent.CustomerEdges               `json:"edges" validate:"omitempty"`
}

// CustomerUpdateRequest represents a request to update a customer.
type CustomerUpdateRequest struct {
	CustomerRequest           // Embedding CustomerRequest for reuse of its fields.
	ID              uuid.UUID `json:"id,omitempty"` // ID is the unique identifier for the customer being updated.
}
