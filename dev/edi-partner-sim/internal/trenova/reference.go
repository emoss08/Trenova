package trenova

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

const (
	ResourceCustomer          = "CUSTOMER"
	ResourceServiceType       = "SERVICE_TYPE"
	ResourceShipmentType      = "SHIPMENT_TYPE"
	ResourceFormulaTemplate   = "FORMULA_TEMPLATE"
	ResourceLocation          = "LOCATION"
	ResourceLocationCategory  = "LOCATION_CATEGORY"
	ResourceCommodity         = "COMMODITY"
	ResourceAccessorialCharge = "ACCESSORIAL_CHARGE"
	ResourceUSState           = "US_STATE"
)

type SelectOption struct {
	ID          string         `json:"id"`
	Label       string         `json:"label"`
	Description string         `json:"description"`
	Meta        map[string]any `json:"meta"`
}

func (o *SelectOption) MetaString(key string) string {
	value, _ := o.Meta[key].(string)
	return value
}

func (o *SelectOption) Display() string {
	if o.Description == "" || strings.EqualFold(o.Description, o.Label) {
		return o.Label
	}
	return o.Label + " - " + o.Description
}

func (c *Client) SelectOptions(
	ctx context.Context,
	resource, query string,
	first int,
) ([]SelectOption, error) {
	input := map[string]any{"resource": resource, "first": first}
	if query != "" {
		input["query"] = query
	}
	var out struct {
		Options struct {
			Edges []struct {
				Node SelectOption `json:"node"`
			} `json:"edges"`
		} `json:"selectOptions"`
	}
	err := c.GraphQL(ctx, `query EdiSimSelectOptions($input: SelectOptionsInput!) {
  selectOptions(input: $input) { edges { node { id label description meta } } }
}`, map[string]any{"input": input}, &out)
	if err != nil {
		return nil, err
	}
	options := make([]SelectOption, 0, len(out.Options.Edges))
	for _, edge := range out.Options.Edges {
		options = append(options, edge.Node)
	}
	return options, nil
}

type NewCustomer struct {
	Code         string
	Name         string
	AddressLine1 string
	City         string
	StateID      string
	PostalCode   string
}

func (c *Client) CreateCustomer(ctx context.Context, req *NewCustomer) (*SelectOption, error) {
	var out struct {
		Customer struct {
			ID   string `json:"id"`
			Code string `json:"code"`
			Name string `json:"name"`
		} `json:"createCustomer"`
	}
	err := c.GraphQL(ctx, `mutation EdiSimCreateCustomer($input: CustomerInput!) {
  createCustomer(input: $input) { id code name }
}`, map[string]any{"input": map[string]any{
		"status":                 "Active",
		"code":                   req.Code,
		"name":                   req.Name,
		"addressLine1":           req.AddressLine1,
		"city":                   req.City,
		"stateId":                req.StateID,
		"postalCode":             req.PostalCode,
		"brokerVettingEnabled":   false,
		"allowConsolidation":     false,
		"exclusiveConsolidation": false,
		"consolidationPriority":  1,
		"statusUpdatePreference": "None",
	}}, &out)
	if err != nil {
		return nil, err
	}
	return &SelectOption{
		ID:          out.Customer.ID,
		Label:       out.Customer.Name,
		Description: out.Customer.Code,
	}, nil
}

func (c *Client) CreateCommodity(
	ctx context.Context,
	name, description string,
) (*SelectOption, error) {
	var out struct {
		Commodity struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"createCommodity"`
	}
	err := c.GraphQL(ctx, `mutation EdiSimCreateCommodity($input: CommodityInput!) {
  createCommodity(input: $input) { id name description }
}`, map[string]any{"input": map[string]any{
		"status":      "Active",
		"name":        name,
		"description": description,
		"stackable":   true,
		"fragile":     false,
	}}, &out)
	if err != nil {
		return nil, err
	}
	return &SelectOption{
		ID:          out.Commodity.ID,
		Label:       out.Commodity.Name,
		Description: out.Commodity.Description,
	}, nil
}

type NewLocation struct {
	Name         string
	Description  string
	AddressLine1 string
	AddressLine2 string
	City         string
	StateID      string
	PostalCode   string
	CategoryID   string
}

func (c *Client) CreateLocation(ctx context.Context, req *NewLocation) (*SelectOption, error) {
	var created struct {
		ID   string `json:"id"`
		Code string `json:"code"`
		Name string `json:"name"`
	}
	err := c.post(ctx, "/locations/", map[string]any{
		"status":             "Active",
		"name":               req.Name,
		"description":        req.Description,
		"addressLine1":       req.AddressLine1,
		"addressLine2":       req.AddressLine2,
		"city":               req.City,
		"stateId":            req.StateID,
		"postalCode":         req.PostalCode,
		"locationCategoryId": req.CategoryID,
	}, &created)
	if err != nil {
		return nil, err
	}
	return &SelectOption{ID: created.ID, Label: created.Name, Description: created.Code}, nil
}

type MappingResolution struct {
	EntityType  string `json:"entityType"`
	SourceID    string `json:"sourceId"`
	SourceLabel string `json:"sourceLabel"`
	TargetID    string `json:"targetId"`
	TargetLabel string `json:"targetLabel"`
	Resolved    bool   `json:"resolved"`
}

type MappingPreview struct {
	Resolved   []MappingResolution `json:"resolved"`
	Unresolved []MappingResolution `json:"unresolved"`
	All        []MappingResolution `json:"all"`
}

func (c *Client) MappingPreview(ctx context.Context, transferID string) (*MappingPreview, error) {
	preview := new(MappingPreview)
	err := c.get(ctx, "/edi/transfers/"+url.PathEscape(transferID)+"/mapping-preview/", preview)
	return preview, err
}

type MappingItem struct {
	EntityType  string `json:"entityType"`
	SourceID    string `json:"sourceId"`
	SourceLabel string `json:"sourceLabel"`
	TargetID    string `json:"targetId"`
	TargetLabel string `json:"targetLabel"`
}

type TenderStop struct {
	LocationID           string `json:"locationId"`
	LocationName         string `json:"locationName"`
	LocationCode         string `json:"locationCode"`
	LocationAddressLine1 string `json:"locationAddressLine1"`
	LocationAddressLine2 string `json:"locationAddressLine2"`
	LocationCity         string `json:"locationCity"`
	LocationStateCode    string `json:"locationStateCode"`
	LocationPostalCode   string `json:"locationPostalCode"`
}

type TenderCommodity struct {
	CommodityID          string `json:"commodityId"`
	CommodityLabel       string `json:"commodityLabel"`
	CommodityName        string `json:"commodityName"`
	CommodityDescription string `json:"commodityDescription"`
}

type TenderCharge struct {
	AccessorialChargeID    string `json:"accessorialChargeId"`
	AccessorialLabel       string `json:"accessorialLabel"`
	AccessorialCode        string `json:"accessorialCode"`
	AccessorialDescription string `json:"accessorialDescription"`
}

type TenderPayload struct {
	Moves []struct {
		Stops []TenderStop `json:"stops"`
	} `json:"moves"`
	Commodities       []TenderCommodity `json:"commodities"`
	AdditionalCharges []TenderCharge    `json:"additionalCharges"`
}

func (p *TenderPayload) Stop(locationID string) (*TenderStop, bool) {
	for _, move := range p.Moves {
		for index := range move.Stops {
			if strings.EqualFold(move.Stops[index].LocationID, locationID) {
				return &move.Stops[index], true
			}
		}
	}
	return nil, false
}

func (p *TenderPayload) Commodity(commodityID string) (*TenderCommodity, bool) {
	for index := range p.Commodities {
		if strings.EqualFold(p.Commodities[index].CommodityID, commodityID) {
			return &p.Commodities[index], true
		}
	}
	return nil, false
}

func (p *TenderPayload) Charge(chargeID string) (*TenderCharge, bool) {
	for index := range p.AdditionalCharges {
		if strings.EqualFold(p.AdditionalCharges[index].AccessorialChargeID, chargeID) {
			return &p.AdditionalCharges[index], true
		}
	}
	return nil, false
}

func (c *Client) TransferPayload(ctx context.Context, transferID string) (*TenderPayload, error) {
	var out struct {
		TenderPayload *TenderPayload `json:"tenderPayload"`
	}
	if err := c.get(ctx, "/edi/transfers/"+url.PathEscape(transferID)+"/", &out); err != nil {
		return nil, err
	}
	if out.TenderPayload == nil {
		return nil, errors.New("the transfer carries no tender payload")
	}
	return out.TenderPayload, nil
}

const customerFields = `status code name addressLine1 addressLine2 city stateId postalCode
	brokerVettingEnabled allowConsolidation exclusiveConsolidation consolidationPriority
	statusUpdatePreference version`

const billingProfileFields = `invoiceDelivery billingCycle billingCycleAnchorDay
	billingCycleTimezone splitBy sectionBy invoiceDetail minConsolidatedAmount
	maxShipmentsPerInvoice paymentTerm hasBillingControlOverrides creditLimit creditStatus
	enforceCreditLimit autoCreditHold creditHoldReason autoSendInvoiceOnGeneration
	sendPaymentReminders sendStatements emailInvoiceEnabled ediInvoiceEnabled
	invoiceNumberFormat customerInvoicePrefix invoiceCopies revenueAccountId arAccountId
	applyLateCharges lateChargeRate gracePeriodDays taxExempt taxExemptNumber
	enforceCustomerBillingReq validateCustomerRates autoTransfer autoMarkReadyToBill
	autoApprove autoBill countLateOnlyOnAppointmentStops autoApplyAccessorials billingCurrency
	requirePONumber requireBOLNumber requireDeliveryNumber defaultBillerId billingNotes
	fuelSurchargeMode fuelSurchargeProgramId factoringCompanyId`

func (c *Client) EnableEDIInvoicing(ctx context.Context, customerID string) (bool, error) {
	var out struct {
		Customer map[string]any `json:"customer"`
	}
	err := c.GraphQL(ctx, `query EdiSimCustomerBilling($id: ID!) {
  customer(id: $id) { `+customerFields+`
    billingProfile { `+billingProfileFields+` documentTypes { id } } }
}`, map[string]any{"id": customerID}, &out)
	if err != nil {
		return false, err
	}
	if out.Customer == nil {
		return false, fmt.Errorf("the customer: %w", ErrNotFound)
	}
	profile, _ := out.Customer["billingProfile"].(map[string]any)
	if profile == nil {
		return false, errors.New("the customer has no billing profile")
	}
	if profile["ediInvoiceEnabled"] == true && profile["autoSendInvoiceOnGeneration"] == true &&
		profile["emailInvoiceEnabled"] == false {
		return false, nil
	}
	documentTypes, _ := profile["documentTypes"].([]any)
	documentTypeIDs := make([]any, 0, len(documentTypes))
	for _, raw := range documentTypes {
		if documentType, ok := raw.(map[string]any); ok {
			documentTypeIDs = append(documentTypeIDs, documentType["id"])
		}
	}
	delete(profile, "documentTypes")
	profile["documentTypeIds"] = documentTypeIDs
	profile["ediInvoiceEnabled"] = true
	profile["autoSendInvoiceOnGeneration"] = true
	profile["emailInvoiceEnabled"] = false
	input := out.Customer
	input["billingProfile"] = profile
	err = c.GraphQL(ctx, `mutation EdiSimUpdateCustomer($id: ID!, $input: CustomerInput!) {
  updateCustomer(id: $id, input: $input) { id }
}`, map[string]any{"id": customerID, "input": input}, nil)
	return err == nil, err
}
