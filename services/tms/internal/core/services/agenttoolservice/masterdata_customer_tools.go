package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	customerUpdatePreference = "statusUpdatePreference"
	customerUpdateRecipients = "statusUpdateRecipients"
	customerMaxRecipients    = 2000
)

var customerUpdatePreferences = agenttoolschema.Source(
	"customer.statusUpdatePreference",
	customer.AllStatusUpdatePreferences(),
)

type customerKeeper interface {
	Get(ctx context.Context, req repositories.GetCustomerByIDRequest) (*customer.Customer, error)
	PlanCreate(ctx context.Context, entity *customer.Customer) (*customer.Customer, error)
	Create(
		ctx context.Context,
		entity *customer.Customer,
		actor *serviceports.RequestActor,
	) (*customer.Customer, error)
	PlanUpdate(
		ctx context.Context,
		entity *customer.Customer,
	) (*serviceports.RecordChange[customer.Customer], error)
	Update(
		ctx context.Context,
		entity *customer.Customer,
		actor *serviceports.RequestActor,
	) (*customer.Customer, error)
	PlanBulkUpdateStatus(
		ctx context.Context,
		req *repositories.BulkUpdateCustomerStatusRequest,
	) ([]serviceports.RecordChange[customer.Customer], error)
	BulkUpdateStatus(
		ctx context.Context,
		req *repositories.BulkUpdateCustomerStatusRequest,
	) ([]*customer.Customer, error)
}

type customerView struct {
	Code                   string `json:"code"`
	Name                   string `json:"name"`
	Status                 string `json:"status"`
	AddressLine1           string `json:"addressLine1"`
	AddressLine2           string `json:"addressLine2,omitempty"`
	City                   string `json:"city"`
	State                  string `json:"state"`
	PostalCode             string `json:"postalCode"`
	DOTNumber              string `json:"dotNumber,omitempty"`
	MCNumber               string `json:"mcNumber,omitempty"`
	StatusUpdatePreference string `json:"statusUpdatePreference"`
	StatusUpdateRecipients string `json:"statusUpdateRecipients,omitempty"`
	AllowConsolidation     bool   `json:"allowConsolidation"`
	ExclusiveConsolidation bool   `json:"exclusiveConsolidation"`
	ConsolidationPriority  int    `json:"consolidationPriority"`
}

func customerViewOf(entity *customer.Customer, states map[pulid.ID]string) any {
	return &customerView{
		Code:                   entity.Code,
		Name:                   entity.Name,
		Status:                 string(entity.Status),
		AddressLine1:           entity.AddressLine1,
		AddressLine2:           entity.AddressLine2,
		City:                   entity.City,
		State:                  stateName(states, entity.StateID),
		PostalCode:             entity.PostalCode,
		DOTNumber:              entity.DOTNumber,
		MCNumber:               entity.MCNumber,
		StatusUpdatePreference: string(entity.StatusUpdatePreference),
		StatusUpdateRecipients: entity.StatusUpdateRecipients,
		AllowConsolidation:     entity.AllowConsolidation,
		ExclusiveConsolidation: entity.ExclusiveConsolidation,
		ConsolidationPriority:  entity.ConsolidationPriority,
	}
}

var customerRecord = &masterRecord[customer.Customer]{
	kind:     "customer",
	resource: permission.ResourceCustomer,
	entity:   customerRecordEntity,
	idParam:  paramCustomerID,
	idsParam: "customerIds",
	supplier: "from list_customers",
	label:    func(entity *customer.Customer) string { return entity.Name },
	id:       func(entity *customer.Customer) pulid.ID { return entity.ID },
	version:  func(entity *customer.Customer) int64 { return entity.Version },
	detach: func(entity *customer.Customer) {
		entity.BillingProfile = nil
		entity.EmailProfile = nil
		entity.State = nil
		entity.BusinessUnit = nil
		entity.Organization = nil
	},
	stateIDs: func(entity *customer.Customer) []pulid.ID { return []pulid.ID{entity.StateID} },
	view:     customerViewOf,
}

func customerFields() []masterField[customer.Customer] {
	return []masterField[customer.Customer]{
		masterText(mdCode, "The customer's short code, unique in this organization.",
			mdMaxCode, true, func(c *customer.Customer) *string { return &c.Code }),
		masterText(mdName, "The customer's name.", mdMaxName, true,
			func(c *customer.Customer) *string { return &c.Name }),
		masterText(mdAddressLine1, "The billing street address.", mdMaxAddress, true,
			func(c *customer.Customer) *string { return &c.AddressLine1 }),
		masterText(mdAddressLine2, "Suite, building or unit.", mdMaxAddress, false,
			func(c *customer.Customer) *string { return &c.AddressLine2 }),
		masterText(mdCity, "The city.", mdMaxCity, true,
			func(c *customer.Customer) *string { return &c.City }),
		masterState(mdState, "The state.", true,
			func(c *customer.Customer) *pulid.ID { return &c.StateID }),
		masterText(mdPostalCode, "The ZIP code.", mdMaxPostalCode, true,
			func(c *customer.Customer) *string { return &c.PostalCode }),
		masterText(mdDOTNumber, "The USDOT number, digits only, for a broker customer.",
			mdMaxNumber, false, func(c *customer.Customer) *string { return &c.DOTNumber }),
		masterText(mdMCNumber, "The MC docket number, digits only.", mdMaxNumber, false,
			func(c *customer.Customer) *string { return &c.MCNumber }),
		masterEnum(customerUpdatePreference, "Which stop events the customer is emailed "+
			"about. None stops the emails and clears the recipients. Changing who is "+
			"emailed is only ever proposed.", customerUpdatePreferences,
			func(c *customer.Customer) *customer.StatusUpdatePreference {
				return &c.StatusUpdatePreference
			}),
		masterText(customerUpdateRecipients, "The addresses status updates go to, separated "+
			"by commas.", customerMaxRecipients, false,
			func(c *customer.Customer) *string { return &c.StatusUpdateRecipients }),
		masterBool("allowConsolidation", "Whether its shipments may be consolidated.",
			func(c *customer.Customer) *bool { return &c.AllowConsolidation }),
		masterBool("exclusiveConsolidation", "Whether its shipments are consolidated only "+
			"with each other; needs allowConsolidation.",
			func(c *customer.Customer) *bool { return &c.ExclusiveConsolidation }),
		masterInt("consolidationPriority", "Its priority when shipments are consolidated, "+
			"1 first.", 1, mdMaxConsolidPrio,
			func(c *customer.Customer) *int { return &c.ConsolidationPriority }),
	}
}

func customerPolicy() masterPolicy {
	policy := mdInternalAsk
	policy.outsideKeys = []string{customerUpdatePreference, customerUpdateRecipients}

	return policy
}

func newCreateCustomerTool(customers customerKeeper, states stateLookup) serviceports.AgentTool {
	return newMasterCreateTool(&masterCreateSpec[customer.Customer]{
		record: customerRecord,
		name:   "create_customer",
		description: "Add a customer the organization ships for: its code, name, billing " +
			"address and DOT and MC numbers. Check list_customers first. Billing terms " +
			"start from the organization's defaults; who is emailed status updates is " +
			"only proposed.",
		rationale: "Adds a customer inside Trenova that shipments can be booked for; nothing " +
			"is sent. Setting who receives status emails decides where later emails go, so " +
			"that call is held for a person.",
		fields:      customerFields(),
		required:    []string{mdCode, mdName, mdAddressLine1, mdCity, mdState, mdPostalCode},
		searchTerms: []string{"new customer", "add customer", "shipper", "bill to account"},
		policy:      customerPolicy(),
		states:      states,
		fresh: func(tenant pagination.TenantInfo) *customer.Customer {
			return &customer.Customer{
				OrganizationID:         tenant.OrgID,
				BusinessUnitID:         tenant.BuID,
				Status:                 domaintypes.StatusActive,
				StatusUpdatePreference: customer.StatusUpdateNone,
				ConsolidationPriority:  1,
			}
		},
		plan:   customers.PlanCreate,
		create: customers.Create,
	})
}

func newUpdateCustomerTool(customers customerKeeper, states stateLookup) serviceports.AgentTool {
	return newMasterUpdateTool(&masterUpdateSpec[customer.Customer]{
		record: customerRecord,
		name:   "update_customer",
		description: "Change a customer's details: name, billing address, DOT and MC numbers, " +
			"who is emailed status updates, and consolidation. Fields left out keep their " +
			"value. Use update_customer_status to activate or retire it.",
		rationale: "Changes a customer inside Trenova and leaves its billing and email " +
			"profiles alone; nothing is sent. Changing who receives status emails is held " +
			"for a person.",
		fields:      customerFields(),
		searchTerms: []string{"edit customer", "customer address", "status update emails"},
		policy:      customerPolicy(),
		states:      states,
		get: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			id pulid.ID,
		) (*customer.Customer, error) {
			return customers.Get(ctx, repositories.GetCustomerByIDRequest{
				ID:         id,
				TenantInfo: tenant,
			})
		},
		plan:   customers.PlanUpdate,
		update: customers.Update,
	})
}

func newUpdateCustomerStatusTool(
	customers customerKeeper,
	states stateLookup,
) serviceports.AgentTool {
	return newMasterStatusTool(&masterStatusSpec[customer.Customer, domaintypes.Status]{
		record: customerRecord,
		name:   "update_customer_status",
		description: "Set one or more customers Active or Inactive, which decides whether " +
			"shipments can be booked for them. Get the ids from list_customers.",
		rationale:   "Changes whether customers are offered for booking inside Trenova; nothing is sent.",
		searchTerms: []string{"deactivate customer", "reactivate customer", "inactive customer"},
		statuses:    activeStatuses,
		statusNote:  activeStatusNote,
		policy:      mdInternalStatus,
		states:      states,
		plan: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			ids []pulid.ID,
			status domaintypes.Status,
		) ([]serviceports.RecordChange[customer.Customer], error) {
			return customers.PlanBulkUpdateStatus(ctx,
				&repositories.BulkUpdateCustomerStatusRequest{
					TenantInfo:  tenant,
					CustomerIDs: ids,
					Status:      status,
				})
		},
		run: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			ids []pulid.ID,
			status domaintypes.Status,
		) ([]*customer.Customer, error) {
			return customers.BulkUpdateStatus(ctx, &repositories.BulkUpdateCustomerStatusRequest{
				TenantInfo:  tenant,
				CustomerIDs: ids,
				Status:      status,
			})
		},
	})
}
