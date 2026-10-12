package tableinsight

import (
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/carriercol"
	"github.com/emoss08/trenova/pkg/buncolgen/customercol"
	"github.com/emoss08/trenova/pkg/buncolgen/invoicecol"
	"github.com/emoss08/trenova/pkg/buncolgen/locationcol"
	"github.com/emoss08/trenova/pkg/buncolgen/ordercol"
	"github.com/emoss08/trenova/pkg/buncolgen/servicefailurecol"
	"github.com/emoss08/trenova/pkg/dbhelper"
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

func facet(column *buncolgen.Column) dbhelper.InsightColumn {
	return dbhelper.InsightColumn{Column: *column, Facetable: true}
}

func sum(column *buncolgen.Column) dbhelper.InsightColumn {
	return dbhelper.InsightColumn{Column: *column, Summable: true}
}

func timeline(column *buncolgen.Column) dbhelper.InsightColumn {
	return dbhelper.InsightColumn{Column: *column, Timeline: true}
}

func NewCustomerSource(p Params) Result {
	cols := customercol.CustomerColumns
	return New(&Config{
		Resource: permission.ResourceCustomer,
		DB:       p.DB,
		IDColumn: cols.ID,
		Base:     Filtered((*customer.Customer)(nil), customercol.CustomerTable.Alias),
		Columns: dbhelper.InsightColumns{
			fieldStatus:  facet(&cols.Status),
			fieldStateID: facet(&cols.StateID),
		},
	})
}

func NewInvoiceSource(p Params) Result {
	cols := invoicecol.InvoiceColumns
	return New(&Config{
		Resource: permission.ResourceInvoice,
		DB:       p.DB,
		IDColumn: cols.ID,
		Base:     Filtered((*invoice.Invoice)(nil), invoicecol.InvoiceTable.Alias),
		Columns: dbhelper.InsightColumns{
			fieldStatus:         facet(&cols.Status),
			"settlementStatus":  facet(&cols.SettlementStatus),
			"disputeStatus":     facet(&cols.DisputeStatus),
			"sendStatus":        facet(&cols.SendStatus),
			"billType":          facet(&cols.BillType),
			"customerId":        facet(&cols.CustomerID),
			"shipperCustomerId": facet(&cols.ShipperCustomerID),
			"totalAmount":       sum(&cols.TotalAmount),
			"subtotalAmount":    sum(&cols.SubtotalAmount),
			"otherAmount":       sum(&cols.OtherAmount),
			"appliedAmount":     sum(&cols.AppliedAmount),
			"invoiceDate":       timeline(&cols.InvoiceDate),
			"postedAt":          timeline(&cols.PostedAt),
		},
	})
}

func NewLocationSource(p Params) Result {
	cols := locationcol.LocationColumns
	return New(&Config{
		Resource: permission.ResourceLocation,
		DB:       p.DB,
		IDColumn: cols.ID,
		Base:     Filtered((*location.Location)(nil), locationcol.LocationTable.Alias),
		Columns: dbhelper.InsightColumns{
			fieldStatus:          facet(&cols.Status),
			"locationCategoryId": facet(&cols.LocationCategoryID),
			fieldStateID:         facet(&cols.StateID),
		},
	})
}

func NewCarrierSource(p Params) Result {
	cols := carriercol.CarrierColumns
	return New(&Config{
		Resource: permission.ResourceCarrier,
		DB:       p.DB,
		IDColumn: cols.ID,
		Base:     Filtered((*carrier.Carrier)(nil), carriercol.CarrierTable.Alias),
		Columns: dbhelper.InsightColumns{
			fieldStatus:        facet(&cols.Status),
			"carrierType":      facet(&cols.CarrierType),
			"complianceStatus": facet(&cols.ComplianceStatus),
			fieldStateID:       facet(&cols.StateID),
		},
	})
}

func NewOrderSource(p Params) Result {
	cols := ordercol.OrderColumns
	return New(&Config{
		Resource: permission.ResourceOrder,
		DB:       p.DB,
		IDColumn: cols.ID,
		Base:     Filtered((*order.Order)(nil), ordercol.OrderTable.Alias),
		Columns: dbhelper.InsightColumns{
			fieldStatus:    facet(&cols.Status),
			"customerId":   facet(&cols.CustomerID),
			"ownerId":      facet(&cols.OwnerID),
			"enteredById":  facet(&cols.EnteredByID),
			"quotedAmount": sum(&cols.QuotedAmount),
			"baseAmount":   sum(&cols.BaseAmount),
			"totalAmount":  sum(&cols.TotalAmount),
			"createdAt":    timeline(&cols.CreatedAt),
		},
	})
}

func NewServiceFailureSource(p Params) Result {
	cols := servicefailurecol.ServiceFailureColumns
	return New(&Config{
		Resource: permission.ResourceServiceFailure,
		DB:       p.DB,
		IDColumn: cols.ID,
		Base: Filtered(
			(*servicefailure.ServiceFailure)(nil),
			servicefailurecol.ServiceFailureTable.Alias,
		),
		Columns: dbhelper.InsightColumns{
			fieldStatus:    facet(&cols.Status),
			"type":         facet(&cols.Type),
			"stopType":     facet(&cols.StopType),
			"reasonCodeId": facet(&cols.ReasonCodeID),
		},
	})
}
