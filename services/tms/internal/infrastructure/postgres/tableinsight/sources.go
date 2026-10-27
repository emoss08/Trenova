package tableinsight

import (
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order/orderdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure/servicefailuredb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

const (
	fieldStatus  = "status"
	fieldStateID = "stateId"
)

func NewCustomerSource(p Params) Result {
	c := customerdb.Customers
	return New(&Config{
		Resource: permission.ResourceCustomer,
		DB:       p.DB,
		IDColumn: c.ID.Col,
		Base:     Filtered(c.Handle),
		Columns: Columns{
			fieldStatus:  Facet(c.Status.Col),
			fieldStateID: Facet(c.StateID.Col),
		},
	})
}

func NewInvoiceSource(p Params) Result {
	i := invoicedb.Invoices
	return New(&Config{
		Resource: permission.ResourceInvoice,
		DB:       p.DB,
		IDColumn: i.ID.Col,
		Base:     Filtered(i.Handle),
		Columns: Columns{
			fieldStatus:         Facet(i.Status.Col),
			"settlementStatus":  Facet(i.SettlementStatus.Col),
			"disputeStatus":     Facet(i.DisputeStatus.Col),
			"sendStatus":        Facet(i.SendStatus.Col),
			"billType":          Facet(i.BillType.Col),
			"customerId":        Facet(i.CustomerID.Col),
			"shipperCustomerId": Facet(i.ShipperCustomerID.Col),
			"totalAmount":       Sum(i.TotalAmount.Col),
			"subtotalAmount":    Sum(i.SubtotalAmount.Col),
			"otherAmount":       Sum(i.OtherAmount.Col),
			"appliedAmount":     Sum(i.AppliedAmount.Col),
			"invoiceDate":       Timeline(i.InvoiceDate.Col),
			"postedAt":          Timeline(i.PostedAt.Col),
		},
	})
}

func NewLocationSource(p Params) Result {
	l := locationdb.Locations
	return New(&Config{
		Resource: permission.ResourceLocation,
		DB:       p.DB,
		IDColumn: l.ID.Col,
		Base:     Filtered(l.Handle),
		Columns: Columns{
			fieldStatus:          Facet(l.Status.Col),
			"locationCategoryId": Facet(l.LocationCategoryID.Col),
			fieldStateID:         Facet(l.StateID.Col),
		},
	})
}

func NewCarrierSource(p Params) Result {
	c := carrierdb.Carriers
	return New(&Config{
		Resource: permission.ResourceCarrier,
		DB:       p.DB,
		IDColumn: c.ID.Col,
		Base:     Filtered(c.Handle),
		Columns: Columns{
			fieldStatus:        Facet(c.Status.Col),
			"carrierType":      Facet(c.CarrierType.Col),
			"complianceStatus": Facet(c.ComplianceStatus.Col),
			fieldStateID:       Facet(c.StateID.Col),
		},
	})
}

func NewOrderSource(p Params) Result {
	o := orderdb.Orders
	return New(&Config{
		Resource: permission.ResourceOrder,
		DB:       p.DB,
		IDColumn: o.ID.Col,
		Base:     Filtered(o.Handle),
		Columns: Columns{
			fieldStatus:    Facet(o.Status.Col),
			"customerId":   Facet(o.CustomerID.Col),
			"ownerId":      Facet(o.OwnerID.Col),
			"enteredById":  Facet(o.EnteredByID.Col),
			"quotedAmount": Sum(o.QuotedAmount.Col),
			"baseAmount":   Sum(o.BaseAmount.Col),
			"totalAmount":  Sum(o.TotalAmount.Col),
			"createdAt":    Timeline(o.CreatedAt.Col),
		},
	})
}

func NewServiceFailureSource(p Params) Result {
	sf := servicefailuredb.ServiceFailures
	return New(&Config{
		Resource: permission.ResourceServiceFailure,
		DB:       p.DB,
		IDColumn: sf.ID.Col,
		Base:     Filtered(sf.Handle),
		Columns: Columns{
			fieldStatus:    Facet(sf.Status.Col),
			"type":         Facet(sf.Type.Col),
			"stopType":     Facet(sf.StopType.Col),
			"reasonCodeId": Facet(sf.ReasonCodeID.Col),
		},
	})
}
