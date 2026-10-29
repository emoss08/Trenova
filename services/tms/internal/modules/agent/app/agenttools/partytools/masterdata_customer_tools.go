package partytools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatakit"
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
		State:                  masterdatakit.StateName(states, entity.StateID),
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

var customerRecord = &masterdatakit.Record[customer.Customer]{
	Kind:     "customer",
	Resource: permission.ResourceCustomer,
	Entity:   changepreview.CustomerRecordEntity,
	IDParam:  toolparams.ParamCustomerID,
	IDsParam: "customerIds",
	Lister:   "customer_list",
	Label:    func(entity *customer.Customer) string { return entity.Name },
	ID:       func(entity *customer.Customer) pulid.ID { return entity.ID },
	Version:  func(entity *customer.Customer) int64 { return entity.Version },
	Detach: func(entity *customer.Customer) {
		entity.BillingProfile = nil
		entity.EmailProfile = nil
		entity.State = nil
		entity.BusinessUnit = nil
		entity.Organization = nil
	},
	StateIDs: func(entity *customer.Customer) []pulid.ID { return []pulid.ID{entity.StateID} },
	View:     customerViewOf,
}

func customerFields() []masterdatakit.Field[customer.Customer] {
	return []masterdatakit.Field[customer.Customer]{
		masterdatakit.Text(
			masterdatakit.MdCode,
			"The customer's short code, unique in this organization. When "+
				"the person and the document give none, make one from the name: up to ten "+
				"letters, such as HARBORLN for Harbor Line Foods.",
			masterdatakit.MdMaxCode,
			true,
			func(c *customer.Customer) *string { return &c.Code },
		),
		masterdatakit.Text(
			masterdatakit.MdName,
			"The customer's name.",
			masterdatakit.MdMaxName,
			true,
			func(c *customer.Customer) *string { return &c.Name },
		),
		masterdatakit.Text(
			masterdatakit.MdAddressLine1,
			"The billing street address.",
			masterdatakit.MdMaxAddress,
			true,
			func(c *customer.Customer) *string { return &c.AddressLine1 },
		),
		masterdatakit.Text(
			masterdatakit.MdAddressLine2,
			"Suite, building or unit.",
			masterdatakit.MdMaxAddress,
			false,
			func(c *customer.Customer) *string { return &c.AddressLine2 },
		),
		masterdatakit.Text(masterdatakit.MdCity, "The city.", masterdatakit.MdMaxCity, true,
			func(c *customer.Customer) *string { return &c.City }),
		masterdatakit.State(masterdatakit.MdState, "The state.", true,
			func(c *customer.Customer) *pulid.ID { return &c.StateID }),
		masterdatakit.Text(
			masterdatakit.MdPostalCode,
			"The ZIP code.",
			masterdatakit.MdMaxPostalCode,
			true,
			func(c *customer.Customer) *string { return &c.PostalCode },
		),
		masterdatakit.Text(
			masterdatakit.MdDOTNumber,
			"The USDOT number, digits only, for a broker customer.",
			masterdatakit.MdMaxNumber,
			false,
			func(c *customer.Customer) *string { return &c.DOTNumber },
		),
		masterdatakit.Text(
			masterdatakit.MdMCNumber,
			"The MC docket number, digits only.",
			masterdatakit.MdMaxNumber,
			false,
			func(c *customer.Customer) *string { return &c.MCNumber },
		),
		masterdatakit.Enum(customerUpdatePreference, "Which stop events the customer is emailed "+
			"about. None stops the emails and clears the recipients. Changing who is "+
			"emailed is only ever proposed.", customerUpdatePreferences,
			func(c *customer.Customer) *customer.StatusUpdatePreference {
				return &c.StatusUpdatePreference
			}),
		masterdatakit.Text(
			customerUpdateRecipients,
			"The addresses status updates go to, separated "+
				"by commas.",
			customerMaxRecipients,
			false,
			func(c *customer.Customer) *string { return &c.StatusUpdateRecipients },
		),
		masterdatakit.Bool("allowConsolidation", "Whether its shipments may be consolidated.",
			func(c *customer.Customer) *bool { return &c.AllowConsolidation }),
		masterdatakit.Bool("exclusiveConsolidation", "Whether its shipments are consolidated only "+
			"with each other; needs allowConsolidation.",
			func(c *customer.Customer) *bool { return &c.ExclusiveConsolidation }),
		masterdatakit.Int("consolidationPriority", "Its priority when shipments are consolidated, "+
			"1 first.", 1, masterdatakit.MdMaxConsolidPrio,
			func(c *customer.Customer) *int { return &c.ConsolidationPriority }),
	}
}

func customerPolicy() masterdatakit.Policy {
	policy := masterdatakit.MdInternalAsk
	policy.OutsideKeys = []string{customerUpdatePreference, customerUpdateRecipients}

	return policy
}

func NewCreateCustomerTool(
	customers CustomerKeeper,
	states masterdatakit.StateLookup,
) agenttoolports.AgentTool {
	return masterdatakit.NewCreateTool(&masterdatakit.CreateSpec[customer.Customer]{
		Record: customerRecord,
		Name:   agentpreset.ToolCreateCustomer,
		Description: "Add a customer the organization ships for: its code, name, billing " +
			"address and DOT and MC numbers. Check customer_list first. Billing terms " +
			"start from the organization's defaults; who is emailed status updates is " +
			"only proposed.",
		Rationale: "Adds a customer inside Trenova that shipments can be booked for; nothing " +
			"is sent. Setting who receives status emails decides where later emails go, so " +
			"that call is held for a person.",
		Fields: customerFields(),
		Required: []string{
			masterdatakit.MdCode,
			masterdatakit.MdName,
			masterdatakit.MdAddressLine1,
			masterdatakit.MdCity,
			masterdatakit.MdState,
			masterdatakit.MdPostalCode,
		},
		SearchTerms: []string{"new customer", "add customer", "shipper", "bill to account"},
		Policy:      customerPolicy(),
		States:      states,
		Fresh: func(tenant pagination.TenantInfo) *customer.Customer {
			return &customer.Customer{
				OrganizationID:         tenant.OrgID,
				BusinessUnitID:         tenant.BuID,
				Status:                 domaintypes.StatusActive,
				StatusUpdatePreference: customer.StatusUpdateNone,
				ConsolidationPriority:  1,
			}
		},
		Plan:   customers.PlanCreate,
		Create: customers.Create,
	})
}

func NewUpdateCustomerTool(
	customers CustomerKeeper,
	states masterdatakit.StateLookup,
) agenttoolports.AgentTool {
	return masterdatakit.NewUpdateTool(&masterdatakit.UpdateSpec[customer.Customer]{
		Record: customerRecord,
		Name:   "customer_update",
		Description: "Change a customer's details: name, billing address, DOT and MC numbers, " +
			"who is emailed status updates, and consolidation. Fields left out keep their " +
			"value. Use customer_set_status to activate or retire it.",
		Rationale: "Changes a customer inside Trenova and leaves its billing and email " +
			"profiles alone; nothing is sent. Changing who receives status emails is held " +
			"for a person.",
		Fields:      customerFields(),
		SearchTerms: []string{"edit customer", "customer address", "status update emails"},
		Policy:      customerPolicy(),
		States:      states,
		Get: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			id pulid.ID,
		) (*customer.Customer, error) {
			return customers.Get(ctx, customerports.GetCustomerByIDRequest{
				ID:         id,
				TenantInfo: tenant,
			})
		},
		Plan:   customers.PlanUpdate,
		Update: customers.Update,
	})
}

func NewUpdateCustomerStatusTool(
	customers CustomerKeeper,
	states masterdatakit.StateLookup,
) agenttoolports.AgentTool {
	return masterdatakit.NewStatusTool(
		&masterdatakit.StatusSpec[customer.Customer, domaintypes.Status]{
			Record: customerRecord,
			Name:   "customer_set_status",
			Description: "Set one or more customers Active or Inactive, which decides whether " +
				"shipments can be booked for them. Get the ids from customer_list.",
			Rationale: "Changes whether customers are offered for booking inside Trenova; nothing is sent.",
			SearchTerms: []string{
				"deactivate customer",
				"reactivate customer",
				"inactive customer",
			},
			Statuses:   masterdatakit.ActiveStatuses,
			StatusNote: masterdatakit.ActiveStatusNote,
			Policy:     masterdatakit.MdInternalStatus,
			States:     states,
			Plan: func(
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
			Run: func(
				ctx context.Context,
				tenant pagination.TenantInfo,
				ids []pulid.ID,
				status domaintypes.Status,
			) ([]*customer.Customer, error) {
				return customers.BulkUpdateStatus(
					ctx,
					&customerports.BulkUpdateCustomerStatusRequest{
						TenantInfo:  tenant,
						CustomerIDs: ids,
						Status:      status,
					},
				)
			},
		},
	)
}
