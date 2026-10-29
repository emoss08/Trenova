package partytools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatakit"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const carrierPaymentMethod = "paymentMethod"

const carrierPaymentTerm = "paymentTermDays"

const carrierRemitName = "remitToName"

const carrierRemitLine1 = "remitAddressLine1"

const carrierRemitLine2 = "remitAddressLine2"

const carrierRemitCity = "remitCity"

const carrierRemitState = "remitState"

const carrierRemitPostal = "remitPostalCode"

const carrierTaxID = "taxId"

const carrierTaxIDType = "taxIdType"

const carrierW9OnFile = "w9OnFile"

const carrier1099 = "is1099Eligible"

const carrierMaxTaxID = 20

const carrierMaxSCAC = 4

const carrierStatusNote = "Active carriers can be tendered freight; Inactive ones are kept " +
	"on file; DoNotUse marks a carrier nobody may book."

var carrierStatuses = agenttoolschema.Source("carrier.status", carrier.StatusValues())

var carrierTypes = agenttoolschema.Source("carrier.type", carrier.TypeValues())

var carrierPayMethods = agenttoolschema.Source(
	"carrier.paymentMethod",
	carrier.PaymentMethodValues(),
)

var carrierTaxIDTypes = agenttoolschema.Source("carrier.taxIdType", carrier.TaxIDTypeValues())

var carrierMoneyKeys = []string{
	carrierPaymentMethod, carrierPaymentTerm, carrierRemitName, carrierRemitLine1,
	carrierRemitLine2, carrierRemitCity, carrierRemitState, carrierRemitPostal,
	carrierTaxID, carrierTaxIDType, carrierW9OnFile, carrier1099,
}

type CarrierKeeper interface {
	Get(ctx context.Context, req carrierports.GetCarrierByIDRequest) (*carrier.Carrier, error)
	PlanCreate(ctx context.Context, entity *carrier.Carrier) (*carrier.Carrier, error)
	Create(
		ctx context.Context,
		entity *carrier.Carrier,
		actor *permissionports.RequestActor,
	) (*carrier.Carrier, error)
	PlanUpdate(
		ctx context.Context,
		entity *carrier.Carrier,
	) (*kernelports.RecordChange[carrier.Carrier], error)
	Update(
		ctx context.Context,
		entity *carrier.Carrier,
		actor *permissionports.RequestActor,
	) (*carrier.Carrier, error)
	PlanBulkUpdateStatus(
		ctx context.Context,
		req *carrierports.BulkUpdateCarrierStatusRequest,
	) ([]kernelports.RecordChange[carrier.Carrier], error)
	BulkUpdateStatus(
		ctx context.Context,
		req *carrierports.BulkUpdateCarrierStatusRequest,
	) ([]*carrier.Carrier, error)
}

type carrierView struct {
	Code              string `json:"code"`
	Name              string `json:"name"`
	DBAName           string `json:"dbaName,omitempty"`
	CarrierType       string `json:"carrierType"`
	Status            string `json:"status"`
	DOTNumber         string `json:"dotNumber,omitempty"`
	MCNumber          string `json:"mcNumber,omitempty"`
	SCAC              string `json:"scac,omitempty"`
	AddressLine1      string `json:"addressLine1,omitempty"`
	AddressLine2      string `json:"addressLine2,omitempty"`
	City              string `json:"city,omitempty"`
	State             string `json:"state,omitempty"`
	PostalCode        string `json:"postalCode,omitempty"`
	Phone             string `json:"phone,omitempty"`
	Email             string `json:"email,omitempty"`
	PaymentMethod     string `json:"paymentMethod"`
	PaymentTermDays   int    `json:"paymentTermDays"`
	RemitToName       string `json:"remitToName,omitempty"`
	RemitAddressLine1 string `json:"remitAddressLine1,omitempty"`
	RemitAddressLine2 string `json:"remitAddressLine2,omitempty"`
	RemitCity         string `json:"remitCity,omitempty"`
	RemitState        string `json:"remitState,omitempty"`
	RemitPostalCode   string `json:"remitPostalCode,omitempty"`
	TaxIDType         string `json:"taxIdType,omitempty"`
	W9OnFile          bool   `json:"w9OnFile"`
	Is1099Eligible    bool   `json:"is1099Eligible"`
	Notes             string `json:"notes,omitempty"`
}

func carrierViewOf(entity *carrier.Carrier, states map[pulid.ID]string) any {
	view := &carrierView{
		Code:              entity.Code,
		Name:              entity.Name,
		DBAName:           entity.DBAName,
		CarrierType:       string(entity.CarrierType),
		Status:            string(entity.Status),
		DOTNumber:         entity.DOTNumber,
		MCNumber:          entity.MCNumber,
		SCAC:              entity.SCAC,
		AddressLine1:      entity.AddressLine1,
		AddressLine2:      entity.AddressLine2,
		City:              entity.City,
		PostalCode:        entity.PostalCode,
		Phone:             entity.Phone,
		Email:             entity.Email,
		PaymentMethod:     string(entity.PaymentMethod),
		PaymentTermDays:   entity.PaymentTermDays,
		RemitToName:       entity.RemitToName,
		RemitAddressLine1: entity.RemitAddressLine1,
		RemitAddressLine2: entity.RemitAddressLine2,
		RemitCity:         entity.RemitCity,
		RemitPostalCode:   entity.RemitPostalCode,
		W9OnFile:          entity.W9OnFile,
		Is1099Eligible:    entity.Is1099Eligible,
		Notes:             entity.Notes,
		State:             masterdatakit.StateNamePointer(states, entity.StateID),
		RemitState:        masterdatakit.StateNamePointer(states, entity.RemitStateID),
	}
	if entity.TaxIDType != nil {
		view.TaxIDType = string(*entity.TaxIDType)
	}

	return view
}

var carrierRecord = &masterdatakit.Record[carrier.Carrier]{
	Kind:     "carrier",
	Resource: permission.ResourceCarrier,
	Entity:   changepreview.CarrierRecordEntity,
	IDParam:  toolparams.ParamCarrierID,
	IDsParam: "carrierIds",
	Lister:   "carrier_list",
	Label:    func(entity *carrier.Carrier) string { return entity.Name },
	ID:       func(entity *carrier.Carrier) pulid.ID { return entity.ID },
	Version:  func(entity *carrier.Carrier) int64 { return entity.Version },
	Detach: func(entity *carrier.Carrier) {
		entity.State = nil
		entity.RemitState = nil
		entity.BusinessUnit = nil
		entity.Organization = nil
	},
	StateIDs: func(entity *carrier.Carrier) []pulid.ID {
		ids := make([]pulid.ID, 0, 2)
		for _, id := range []*pulid.ID{entity.StateID, entity.RemitStateID} {
			if id != nil {
				ids = append(ids, *id)
			}
		}

		return ids
	},
	View: carrierViewOf,
}

func carrierFields() []masterdatakit.Field[carrier.Carrier] {
	return []masterdatakit.Field[carrier.Carrier]{
		masterdatakit.Text(
			masterdatakit.MdCode,
			"The carrier's short code, unique in this organization.",
			masterdatakit.MdMaxCode,
			true,
			func(c *carrier.Carrier) *string { return &c.Code },
		),
		masterdatakit.Text(
			masterdatakit.MdName,
			"The carrier's legal name.",
			masterdatakit.MdMaxName,
			true,
			func(c *carrier.Carrier) *string { return &c.Name },
		),
		masterdatakit.Text(
			"dbaName",
			"The name it does business as, when that differs.",
			masterdatakit.MdMaxName,
			false,
			func(c *carrier.Carrier) *string { return &c.DBAName },
		),
		masterdatakit.Enum("carrierType", "What kind of carrier it is. Defaults to Common.",
			carrierTypes, func(c *carrier.Carrier) *carrier.Type { return &c.CarrierType }),
		masterdatakit.Text(
			masterdatakit.MdDOTNumber,
			"The USDOT number, digits only.",
			masterdatakit.MdMaxNumber,
			false,
			func(c *carrier.Carrier) *string { return &c.DOTNumber },
		),
		masterdatakit.Text(
			masterdatakit.MdMCNumber,
			"The MC docket number, digits only.",
			masterdatakit.MdMaxNumber,
			false,
			func(c *carrier.Carrier) *string { return &c.MCNumber },
		),
		masterdatakit.UpperText("scac", "The two to four letter SCAC.", carrierMaxSCAC,
			func(c *carrier.Carrier) *string { return &c.SCAC }),
		masterdatakit.Text(
			masterdatakit.MdAddressLine1,
			"The street address.",
			masterdatakit.MdMaxAddress,
			false,
			func(c *carrier.Carrier) *string { return &c.AddressLine1 },
		),
		masterdatakit.Text(
			masterdatakit.MdAddressLine2,
			"Suite, building or unit.",
			masterdatakit.MdMaxAddress,
			false,
			func(c *carrier.Carrier) *string { return &c.AddressLine2 },
		),
		masterdatakit.Text(masterdatakit.MdCity, "The city.", masterdatakit.MdMaxCity, false,
			func(c *carrier.Carrier) *string { return &c.City }),
		masterdatakit.StatePointer(masterdatakit.MdState, "The state of the street address.",
			func(c *carrier.Carrier) **pulid.ID { return &c.StateID }),
		masterdatakit.Text(
			masterdatakit.MdPostalCode,
			"The ZIP code.",
			masterdatakit.MdMaxPostalCode,
			false,
			func(c *carrier.Carrier) *string { return &c.PostalCode },
		),
		masterdatakit.Text(
			masterdatakit.MdPhone,
			"The dispatch phone number.",
			masterdatakit.MdMaxPhone,
			false,
			func(c *carrier.Carrier) *string { return &c.Phone },
		),
		masterdatakit.Text(
			masterdatakit.MdEmail,
			"The dispatch email address.",
			masterdatakit.MdMaxName,
			false,
			func(c *carrier.Carrier) *string { return &c.Email },
		),
		masterdatakit.Text(
			masterdatakit.MdNote,
			"Anything colleagues booking this carrier should know.",
			masterdatakit.MdMaxNotes,
			false,
			func(c *carrier.Carrier) *string { return &c.Notes },
		),
		masterdatakit.Enum(carrierPaymentMethod, "How the carrier is paid. Defaults to Check. "+
			"Payment details are only ever proposed and change on a person's approval.",
			carrierPayMethods,
			func(c *carrier.Carrier) *carrier.PaymentMethod { return &c.PaymentMethod }),
		masterdatakit.Int(carrierPaymentTerm, "Days after the invoice the carrier is paid. "+
			"Defaults to 30.", 0, masterdatakit.MdMaxPaymentTerm,
			func(c *carrier.Carrier) *int { return &c.PaymentTermDays }),
		masterdatakit.Text(
			carrierRemitName,
			"Who payment is made out to.",
			masterdatakit.MdMaxName,
			false,
			func(c *carrier.Carrier) *string { return &c.RemitToName },
		),
		masterdatakit.Text(
			carrierRemitLine1,
			"The remit-to street address.",
			masterdatakit.MdMaxAddress,
			false,
			func(c *carrier.Carrier) *string { return &c.RemitAddressLine1 },
		),
		masterdatakit.Text(
			carrierRemitLine2,
			"The remit-to suite or unit.",
			masterdatakit.MdMaxAddress,
			false,
			func(c *carrier.Carrier) *string { return &c.RemitAddressLine2 },
		),
		masterdatakit.Text(carrierRemitCity, "The remit-to city.", masterdatakit.MdMaxCity, false,
			func(c *carrier.Carrier) *string { return &c.RemitCity }),
		masterdatakit.StatePointer(carrierRemitState, "The remit-to state.",
			func(c *carrier.Carrier) **pulid.ID { return &c.RemitStateID }),
		masterdatakit.Text(
			carrierRemitPostal,
			"The remit-to ZIP code.",
			masterdatakit.MdMaxPostalCode,
			false,
			func(c *carrier.Carrier) *string { return &c.RemitPostalCode },
		),
		masterdatakit.Text(carrierTaxID, "The tax id payments are reported under.", carrierMaxTaxID,
			false, func(c *carrier.Carrier) *string { return &c.TaxID }),
		masterdatakit.EnumPointer(carrierTaxIDType, "Whether the tax id is an EIN or an SSN; "+
			"required with a tax id.", carrierTaxIDTypes,
			func(c *carrier.Carrier) **carrier.TaxIDType { return &c.TaxIDType }),
		masterdatakit.Bool(carrierW9OnFile, "Whether a signed W-9 is on file.",
			func(c *carrier.Carrier) *bool { return &c.W9OnFile }),
		masterdatakit.Bool(carrier1099, "Whether payments are reported on a 1099.",
			func(c *carrier.Carrier) *bool { return &c.Is1099Eligible }),
	}
}

func carrierPolicy() masterdatakit.Policy {
	policy := masterdatakit.MdInternalAsk
	policy.MoneyKeys = carrierMoneyKeys

	return policy
}

func carrierGetter(carriers CarrierKeeper) func(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*carrier.Carrier, error) {
	return func(
		ctx context.Context,
		tenant pagination.TenantInfo,
		id pulid.ID,
	) (*carrier.Carrier, error) {
		return carriers.Get(ctx, carrierports.GetCarrierByIDRequest{
			ID:         id,
			TenantInfo: tenant,
			CarrierFilterOptions: carrierports.CarrierFilterOptions{
				IncludeContacts:          true,
				IncludeInsurancePolicies: true,
				IncludeEDIChannels:       true,
			},
		})
	}
}

func NewCreateCarrierTool(
	carriers CarrierKeeper,
	states masterdatakit.StateLookup,
) agenttoolports.AgentTool {
	return masterdatakit.NewCreateTool(&masterdatakit.CreateSpec[carrier.Carrier]{
		Record: carrierRecord,
		Name:   "carrier_create",
		Description: "Add a carrier the organization can tender freight to: its code, name, " +
			"USDOT and MC numbers, SCAC, address and dispatch contact. Check carrier_list " +
			"first. Vetting and compliance are carrier_vet's; payment and remit-to details " +
			"are only proposed.",
		Rationale: "Adds a carrier inside Trenova that colleagues can then book; nothing is " +
			"sent. Payment and remit-to details decide where money goes, so a call that " +
			"sets them is money and runs only on a person's approval.",
		Fields:      carrierFields(),
		Required:    []string{masterdatakit.MdCode, masterdatakit.MdName},
		SearchTerms: []string{"new carrier", "add carrier", "onboard carrier", "trucking company"},
		Policy:      carrierPolicy(),
		States:      states,
		Fresh: func(tenant pagination.TenantInfo) *carrier.Carrier {
			return &carrier.Carrier{
				OrganizationID:   tenant.OrgID,
				BusinessUnitID:   tenant.BuID,
				Status:           carrier.StatusActive,
				CarrierType:      carrier.TypeCommon,
				ComplianceStatus: carrier.ComplianceStatusPending,
				SafetyRating:     carrier.SafetyRatingNotRated,
				PaymentMethod:    carrier.PaymentMethodCheck,
				PaymentTermDays:  30,
			}
		},
		Plan:   carriers.PlanCreate,
		Create: carriers.Create,
	})
}

func NewUpdateCarrierTool(
	carriers CarrierKeeper,
	states masterdatakit.StateLookup,
) agenttoolports.AgentTool {
	return masterdatakit.NewUpdateTool(&masterdatakit.UpdateSpec[carrier.Carrier]{
		Record: carrierRecord,
		Name:   "carrier_update",
		Description: "Change a carrier's details: name, numbers, SCAC, address, dispatch " +
			"contact, notes, or payment and remit-to details. Fields left out keep their " +
			"value. Use carrier_set_status to activate or retire it and carrier_vet for " +
			"compliance.",
		Rationale: "Changes a carrier inside Trenova and keeps its contacts, insurance and EDI " +
			"channels; nothing is sent. A call that changes payment or remit-to details is " +
			"money and runs only on a person's approval.",
		Fields:      carrierFields(),
		SearchTerms: []string{"edit carrier", "carrier address", "remit to", "carrier details"},
		Policy:      carrierPolicy(),
		States:      states,
		Get:         carrierGetter(carriers),
		Plan:        carriers.PlanUpdate,
		Update:      carriers.Update,
	})
}

func NewUpdateCarrierStatusTool(
	carriers CarrierKeeper,
	states masterdatakit.StateLookup,
) agenttoolports.AgentTool {
	return masterdatakit.NewStatusTool(&masterdatakit.StatusSpec[carrier.Carrier, carrier.Status]{
		Record: carrierRecord,
		Name:   "carrier_set_status",
		Description: "Set one or more carriers Active, Inactive or DoNotUse, which decides " +
			"whether they can be tendered freight. Get the ids from carrier_list.",
		Rationale:   "Changes whether carriers are offered for booking inside Trenova; nothing is sent.",
		SearchTerms: []string{"deactivate carrier", "do not use", "reactivate carrier"},
		Statuses:    carrierStatuses,
		StatusNote:  carrierStatusNote,
		Policy:      masterdatakit.MdInternalStatus,
		States:      states,
		Plan: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			ids []pulid.ID,
			status carrier.Status,
		) ([]kernelports.RecordChange[carrier.Carrier], error) {
			return carriers.PlanBulkUpdateStatus(ctx, &carrierports.BulkUpdateCarrierStatusRequest{
				TenantInfo: tenant,
				CarrierIDs: ids,
				Status:     status,
			})
		},
		Run: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			ids []pulid.ID,
			status carrier.Status,
		) ([]*carrier.Carrier, error) {
			return carriers.BulkUpdateStatus(ctx, &carrierports.BulkUpdateCarrierStatusRequest{
				TenantInfo: tenant,
				CarrierIDs: ids,
				Status:     status,
			})
		},
	})
}
