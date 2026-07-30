package invoiceservice

import (
	"context"
	"html/template"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
)

// ContextBuilder turns an invoice into the variables its templates declare.
//
// It deliberately depends on repositories rather than on *Service. The invoice
// service depends on the template resolver to render, and the template service
// depends on the builder group; routing the builder through *Service would close
// that loop and fx would refuse to start.
type ContextBuilder struct {
	invoiceRepo  repositories.InvoiceRepository
	profileRepos deliveryProfileRepos
	inliner      services.AssetInliner
}

type ContextBuilderParams struct {
	fx.In

	InvoiceRepo      repositories.InvoiceRepository
	CustomerRepo     repositories.CustomerRepository
	ShipmentRepo     repositories.ShipmentRepository
	BillingRepo      repositories.BillingControlRepository
	OrganizationRepo repositories.OrganizationRepository
	Inliner          services.AssetInliner
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func NewContextBuilder(p ContextBuilderParams) *ContextBuilder {
	return &ContextBuilder{
		invoiceRepo: p.InvoiceRepo,
		profileRepos: deliveryProfileRepos{
			customerRepo:     p.CustomerRepo,
			shipmentRepo:     p.ShipmentRepo,
			billingRepo:      p.BillingRepo,
			organizationRepo: p.OrganizationRepo,
		},
		inliner: p.Inliner,
	}
}

var _ services.ContextBuilder = (*ContextBuilder)(nil)

func (b *ContextBuilder) Kind() documenttemplate.Kind {
	return documenttemplate.KindInvoicePDF
}

// Build loads the invoice and everything its document names.
func (b *ContextBuilder) Build(
	ctx context.Context,
	req *services.BuildContextRequest,
) (any, error) {
	entity, err := b.invoiceRepo.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         req.ReferenceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	deliveryProfile, err := resolveDeliveryProfileWith(
		ctx,
		b.profileRepos,
		resolveDeliveryProfileParams{
			Entity:                 entity,
			TenantInfo:             req.TenantInfo,
			IncludeShipmentDetails: true,
			IncludeCustomer:        true,
			IncludeCustomerState:   true,
			IncludeBillingControl:  true,
		},
	)
	if err != nil {
		return nil, err
	}

	if err = resolveDeliveryOrganizationWith(
		ctx, b.profileRepos.organizationRepo, deliveryProfile, req.TenantInfo,
	); err != nil {
		return nil, err
	}

	return b.BuildFrom(ctx, entity, deliveryProfile)
}

// BuildFrom converts an already-loaded invoice and profile.
//
// The delivery path has both in hand, and re-fetching them to render would
// double the query cost of sending an invoice.
func (b *ContextBuilder) BuildFrom(
	ctx context.Context,
	entity *invoice.Invoice,
	deliveryProfile *invoiceDeliveryProfile,
) (*documenttemplate.InvoiceContext, error) {
	data := buildInvoicePDFData(entity, deliveryProfile)

	logoDataURI, err := b.resolveLogo(ctx, deliveryProfile)
	if err != nil {
		return nil, err
	}

	return invoiceContextFromPDFData(&data, logoDataURI), nil
}

// resolveLogo turns the organization's configured logo into a data: URI.
//
// The old renderer fetched this URL with http.DefaultClient, which made an
// organization-controlled string into an SSRF primitive. The inliner routes it
// through the safe client and re-checks every resolved address at dial time.
func (b *ContextBuilder) resolveLogo(
	ctx context.Context,
	deliveryProfile *invoiceDeliveryProfile,
) (template.URL, error) {
	if deliveryProfile == nil || deliveryProfile.Organization == nil {
		return "", nil
	}

	return services.ResolveLogoDataURI(ctx, b.inliner, deliveryProfile.Organization.LogoURL)
}

// invoiceContextFromPDFData is the whole mapping between the surviving data
// builder and the published variable catalog.
//
// It is a straight copy because the catalog was defined from this struct: the
// registry test asserts the two match by reflection in both directions, so a
// field added to one without the other fails the build rather than silently
// rendering blank.
func invoiceContextFromPDFData(
	data *invoicePDFData,
	logoDataURI template.URL,
) *documenttemplate.InvoiceContext {
	return &documenttemplate.InvoiceContext{
		InvoiceNumber: data.InvoiceNumber,
		InvoiceDate:   data.InvoiceDate,
		DueDate:       data.DueDate,
		PaymentTerm:   data.PaymentTerm,
		CurrencyCode:  data.CurrencyCode,

		Organization: invoiceAddressBlock(data.Organization),
		BillTo:       invoiceAddressBlock(data.BillTo),
		RemitTo:      invoiceAddressBlock(data.RemitTo),
		Shipper:      invoiceAddressBlock(data.Shipper),
		Consignee:    invoiceAddressBlock(data.Consignee),

		LogoDataURI: logoDataURI,

		HeaderRows:    invoiceKeyValues(data.HeaderRows),
		CommodityRows: invoiceCommodityRows(data.CommodityRows),
		ChargeRows:    invoiceChargeRows(data.ChargeRows),

		Subtotal:   data.Subtotal,
		Other:      data.Other,
		Total:      data.Total,
		BalanceDue: data.BalanceDue,

		Terms:         data.Terms,
		InvoiceTerms:  data.InvoiceTerms,
		InvoiceFooter: data.InvoiceFooter,
		Notes:         data.Notes,
		Attachments:   data.Attachments,
	}
}

func invoiceAddressBlock(block invoicePDFAddressBlock) documenttemplate.AddressBlock {
	return documenttemplate.AddressBlock{
		Name:    block.Name,
		Lines:   block.Lines,
		Details: invoiceKeyValues(block.Details),
	}
}

func invoiceKeyValues(rows []invoicePDFKeyValue) []documenttemplate.KeyValue {
	if len(rows) == 0 {
		return nil
	}

	out := make([]documenttemplate.KeyValue, 0, len(rows))
	for _, row := range rows {
		out = append(out, documenttemplate.KeyValue{Label: row.Label, Value: row.Value})
	}

	return out
}

func invoiceChargeRows(rows []invoicePDFChargeRow) []documenttemplate.ChargeRow {
	if len(rows) == 0 {
		return nil
	}

	out := make([]documenttemplate.ChargeRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, documenttemplate.ChargeRow{
			Line:        row.Line,
			Description: row.Description,
			Quantity:    row.Quantity,
			UnitPrice:   row.UnitPrice,
			Amount:      row.Amount,
		})
	}

	return out
}

// invoiceCommodityRows drops PiecesValue and WeightValue.
//
// Those existed so the old renderer could sum a totals row at fixed
// coordinates. A template sums with its own markup, and exposing the raw
// integers would put two representations of the same number in the catalog.
func invoiceCommodityRows(rows []invoicePDFCommodityRow) []documenttemplate.CommodityRow {
	if len(rows) == 0 {
		return nil
	}

	out := make([]documenttemplate.CommodityRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, documenttemplate.CommodityRow{
			Quantity:         row.Quantity,
			Type:             row.Type,
			DescriptionLines: row.DescriptionLines,
			Weight:           row.Weight,
			NMFC:             row.NMFC,
			Class:            row.Class,
		})
	}

	return out
}
