package masterdatatools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreset"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const customerUpdatePreference = "statusUpdatePreference"

const customerUpdateRecipients = "statusUpdateRecipients"

const customerMaxRecipients = 2000

var customerUpdatePreferences = agenttoolschema.Source(
	"customer.statusUpdatePreference",
	customer.AllStatusUpdatePreferences(),
)

type CustomerKeeper interface {
	Get(ctx context.Context, req customerports.GetCustomerByIDRequest) (*customer.Customer, error)
	PlanCreate(ctx context.Context, entity *customer.Customer) (*customer.Customer, error)
	Create(
		ctx context.Context,
		entity *customer.Customer,
		actor *permissionports.RequestActor,
	) (*customer.Customer, error)
	PlanUpdate(
		ctx context.Context,
		entity *customer.Customer,
	) (*kernelports.RecordChange[customer.Customer], error)
	Update(
		ctx context.Context,
		entity *customer.Customer,
		actor *permissionports.RequestActor,
	) (*customer.Customer, error)
	PlanBulkUpdateStatus(
		ctx context.Context,
		req *customerports.BulkUpdateCustomerStatusRequest,
	) ([]kernelports.RecordChange[customer.Customer], error)
	BulkUpdateStatus(
		ctx context.Context,
		req *customerports.BulkUpdateCustomerStatusRequest,
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
	entity:   changepreview.CustomerRecordEntity,
	idParam:  toolparams.ParamCustomerID,
	idsParam: "customerIds",
	lister:   "customer_list",
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
		masterText(mdCode, "The customer's short code, unique in this organization. When "+
			"the person and the document give none, make one from the name: up to ten "+
			"letters, such as HARBORLN for Harbor Line Foods.",
			mdMaxCode, true, func(c *customer.Customer) *string { return &c.Code }),
		masterText(MdName, "The customer's name.", mdMaxName, true,
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

func NewCreateCustomerTool(customers CustomerKeeper, states StateLookup) agenttoolports.AgentTool {
	return newMasterCreateTool(&masterCreateSpec[customer.Customer]{
		record: customerRecord,
		name:   agentpreset.ToolCreateCustomer,
		description: "Add a customer the organization ships for: its code, name, billing " +
			"address and DOT and MC numbers. Check customer_list first. Billing terms " +
			"start from the organization's defaults; who is emailed status updates is " +
			"only proposed.",
		rationale: "Adds a customer inside Trenova that shipments can be booked for; nothing " +
			"is sent. Setting who receives status emails decides where later emails go, so " +
			"that call is held for a person.",
		fields:      customerFields(),
		required:    []string{mdCode, MdName, mdAddressLine1, mdCity, mdState, mdPostalCode},
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

func NewUpdateCustomerTool(customers CustomerKeeper, states StateLookup) agenttoolports.AgentTool {
	return newMasterUpdateTool(&masterUpdateSpec[customer.Customer]{
		record: customerRecord,
		name:   "customer_update",
		description: "Change a customer's details: name, billing address, DOT and MC numbers, " +
			"who is emailed status updates, and consolidation. Fields left out keep their " +
			"value. Use customer_set_status to activate or retire it.",
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
			return customers.Get(ctx, customerports.GetCustomerByIDRequest{
				ID:         id,
				TenantInfo: tenant,
			})
		},
		plan:   customers.PlanUpdate,
		update: customers.Update,
	})
}

func NewUpdateCustomerStatusTool(
	customers CustomerKeeper,
	states StateLookup,
) agenttoolports.AgentTool {
	return newMasterStatusTool(&masterStatusSpec[customer.Customer, domaintypes.Status]{
		record: customerRecord,
		name:   "customer_set_status",
		description: "Set one or more customers Active or Inactive, which decides whether " +
			"shipments can be booked for them. Get the ids from customer_list.",
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
		) ([]kernelports.RecordChange[customer.Customer], error) {
			return customers.PlanBulkUpdateStatus(ctx,
				&customerports.BulkUpdateCustomerStatusRequest{
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
			return customers.BulkUpdateStatus(ctx, &customerports.BulkUpdateCustomerStatusRequest{
				TenantInfo:  tenant,
				CustomerIDs: ids,
				Status:      status,
			})
		},
	})
}
