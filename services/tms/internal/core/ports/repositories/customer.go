package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type CustomerFilterOptions struct {
	IncludeState          bool `form:"includeState"`
	IncludeBillingProfile bool `form:"includeBillingProfile"`
	IncludeEmailProfile   bool `form:"includeEmailProfile"`
}

type ListCustomerRequest struct {
	Filter                *pagination.QueryOptions `json:"filter" form:"filter"`
	CustomerFilterOptions ` form:"customerFilterOptions"`
}

type ListCustomerConnectionRequest struct {
	Filter                *pagination.QueryOptions `json:"filter"`
	Cursor                pagination.CursorInfo    `json:"-"`
	CustomerColumns       []string                 `json:"-"`
	CustomerFilterOptions `json:"-"`
}

type GetCustomerByIDRequest struct {
	ID                    pulid.ID              `json:"id"         form:"id"`
	TenantInfo            pagination.TenantInfo `json:"tenantInfo" form:"tenantInfo"`
	CustomerFilterOptions ` form:"customerFilterOptions"`
}

type BulkUpdateCustomerStatusRequest struct {
	TenantInfo  pagination.TenantInfo `json:"-"`
	CustomerIDs []pulid.ID            `json:"customerIds"`
	Status      domaintypes.Status    `json:"status"`
}

type GetCustomersByIDsRequest struct {
	TenantInfo            pagination.TenantInfo `json:"-"`
	CustomerIDs           []pulid.ID            `json:"customerIds"`
	CustomerFilterOptions `json:"-"`
}

type CustomerSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
}

// AdvanceBilledPeriodRequest moves a customer's billing watermark past the
// period a run just billed.
//
// It only ever moves forward: a retry or an out-of-order run must not drag the
// watermark backwards and re-bill a period that already produced invoices.
type AdvanceBilledPeriodRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	CustomerID pulid.ID              `json:"-"`
	PeriodEnd  int64                 `json:"-"`
}

// ListBillingSchedulesRequest narrows which statement-billed customers come back.
//
// A zero TenantInfo means every tenant, which is what the deployment-wide sweep
// needs; the statements view passes its own so a biller only ever sees their own
// organization's open statements.
type ListBillingSchedulesRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
}

// DueBillingSchedule is one statement-billed customer and the settings the
// scheduler needs to decide whether a period has closed for them.
//
// Cross-tenant by design: the sweep runs once for the whole deployment rather
// than once per organization, so the tenant travels with the row.
type DueBillingSchedule struct {
	TenantInfo            pagination.TenantInfo    `bun:"-"`
	OrganizationID        pulid.ID                 `bun:"organization_id"`
	BusinessUnitID        pulid.ID                 `bun:"business_unit_id"`
	CustomerID            pulid.ID                 `bun:"customer_id"`
	CustomerName          string                   `bun:"customer_name"`
	CustomerCode          string                   `bun:"customer_code"`
	CustomerStatus        string                   `bun:"customer_status"`
	BillingCycle          customer.BillingCycle    `bun:"billing_cycle"`
	BillingCycleAnchorDay int16                    `bun:"billing_cycle_anchor_day"`
	BillingCycleTimezone  string                   `bun:"billing_cycle_timezone"`
	InvoiceDelivery       customer.InvoiceDelivery `bun:"invoice_delivery"`
	LastBilledPeriodEnd   *int64                   `bun:"last_billed_period_end"`

	// The billing and presentation rules ride along so describing an open
	// statement needs no second lookup per customer.
	SplitBy                customer.InvoiceSplitKey   `bun:"split_by"`
	SectionBy              customer.InvoiceSectionKey `bun:"section_by"`
	InvoiceDetail          customer.InvoiceDetail     `bun:"invoice_detail"`
	MinConsolidatedAmount  decimal.NullDecimal        `bun:"min_consolidated_amount"`
	MaxShipmentsPerInvoice int16                      `bun:"max_shipments_per_invoice"`
	AutoBill               bool                       `bun:"auto_bill"`
	BillingCurrency        string                     `bun:"billing_currency"`
}

// Profile rebuilds just enough of the billing profile for the period maths and
// the grouping rules.
func (d *DueBillingSchedule) Profile() *customer.CustomerBillingProfile {
	return &customer.CustomerBillingProfile{
		OrganizationID:         d.OrganizationID,
		BusinessUnitID:         d.BusinessUnitID,
		CustomerID:             d.CustomerID,
		InvoiceDelivery:        d.InvoiceDelivery,
		BillingCycle:           d.BillingCycle,
		BillingCycleAnchorDay:  d.BillingCycleAnchorDay,
		BillingCycleTimezone:   d.BillingCycleTimezone,
		LastBilledPeriodEnd:    d.LastBilledPeriodEnd,
		SplitBy:                d.SplitBy,
		SectionBy:              d.SectionBy,
		InvoiceDetail:          d.InvoiceDetail,
		MinConsolidatedAmount:  d.MinConsolidatedAmount,
		MaxShipmentsPerInvoice: d.MaxShipmentsPerInvoice,
		AutoBill:               d.AutoBill,
		BillingCurrency:        d.BillingCurrency,
	}
}

type CustomerRepository interface {
	AdvanceBilledPeriod(ctx context.Context, req *AdvanceBilledPeriodRequest) error
	ListDueBillingSchedules(
		ctx context.Context,
		req *ListBillingSchedulesRequest,
	) ([]*DueBillingSchedule, error)
	List(
		ctx context.Context,
		req *ListCustomerRequest,
	) (*pagination.ListResult[*customer.Customer], error)
	ListConnection(
		ctx context.Context,
		req *ListCustomerConnectionRequest,
	) (*pagination.CursorListResult[*customer.Customer], error)
	GetByID(
		ctx context.Context,
		req GetCustomerByIDRequest,
	) (*customer.Customer, error)
	GetByIDs(
		ctx context.Context,
		req GetCustomersByIDsRequest,
	) ([]*customer.Customer, error)
	GetBillingProfile(
		ctx context.Context,
		cusID pulid.ID,
	) (*customer.CustomerBillingProfile, error)
	Create(
		ctx context.Context,
		entity *customer.Customer,
	) (*customer.Customer, error)
	Update(
		ctx context.Context,
		entity *customer.Customer,
	) (*customer.Customer, error)
	BulkUpdateStatus(
		ctx context.Context,
		req *BulkUpdateCustomerStatusRequest,
	) ([]*customer.Customer, error)
	SelectOptions(
		ctx context.Context,
		req *CustomerSelectOptionsRequest,
	) (*pagination.ListResult[*customer.Customer], error)
}
