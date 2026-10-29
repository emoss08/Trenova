package invoiceservice

import (
	"context"
	"html/template"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/invoicetemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templatecatalog"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/fx"
)

// ContextBuilder turns an invoice into the variables its templates declare.
//
// It deliberately depends on repositories rather than on *Service. The invoice
// service depends on the template resolver to render, and the template service
// depends on the builder group; routing the builder through *Service would close
// that loop and fx would refuse to start.
type ContextBuilder struct {
	invoiceRepo  invoiceports.InvoiceRepository
	profileRepos deliveryProfileRepos
	inliner      documenttemplateports.AssetInliner
}

type ContextBuilderParams struct {
	fx.In

	InvoiceRepo      invoiceports.InvoiceRepository
	CustomerRepo     customerports.CustomerRepository
	ShipmentRepo     shipmentports.ShipmentRepository
	BillingRepo      tenantports.BillingControlRepository
	OrganizationRepo tenantports.OrganizationRepository
	FactoringRepo    factoringcompanyports.FactoringCompanyRepository
	Inliner          documenttemplateports.AssetInliner
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
			factoringRepo:    p.FactoringRepo,
		},
		inliner: p.Inliner,
	}
}

var _ documenttemplateports.ContextBuilder = (*ContextBuilder)(nil)

func (b *ContextBuilder) Kind() documenttemplate.Kind {
	return documenttemplate.KindInvoicePDF
}

// Build loads the invoice and everything its document names.
func (b *ContextBuilder) Build(
	ctx context.Context,
	req *documenttemplateports.BuildContextRequest,
) (any, error) {
	entity, err := b.invoiceRepo.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
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
) (*invoicetemplate.InvoiceContext, error) {
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

	return documenttemplateports.ResolveLogoDataURI(
		ctx,
		b.inliner,
		deliveryProfile.Organization.LogoURL,
	)
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
) *invoicetemplate.InvoiceContext {
	return &invoicetemplate.InvoiceContext{
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
		ShipmentRows:  invoiceShipmentRows(data.ShipmentRows),
		ShipmentCount: data.ShipmentCount,
		Period:        data.Period,

		Subtotal:   data.Subtotal,
		Other:      data.Other,
		Total:      data.Total,
		BalanceDue: data.BalanceDue,

		Terms:         data.Terms,
		InvoiceTerms:  data.InvoiceTerms,
		InvoiceFooter: data.InvoiceFooter,
		Notes:         data.Notes,
		Attachments:   data.Attachments,

		NoticeOfAssignment: data.NoticeOfAssignment,
	}
}

func invoiceAddressBlock(block invoicePDFAddressBlock) templatecatalog.AddressBlock {
	return templatecatalog.AddressBlock{
		Name:    block.Name,
		Lines:   block.Lines,
		Details: invoiceKeyValues(block.Details),
	}
}

func invoiceKeyValues(rows []invoicePDFKeyValue) []templatecatalog.KeyValue {
	if len(rows) == 0 {
		return nil
	}

	out := make([]templatecatalog.KeyValue, 0, len(rows))
	for _, row := range rows {
		out = append(out, templatecatalog.KeyValue{Label: row.Label, Value: row.Value})
	}

	return out
}

func invoiceChargeRows(rows []invoicePDFChargeRow) []invoicetemplate.ChargeRow {
	if len(rows) == 0 {
		return nil
	}

	out := make([]invoicetemplate.ChargeRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, invoicetemplate.ChargeRow{
			Line:        row.Line,
			Description: row.Description,
			Quantity:    row.Quantity,
			UnitPrice:   row.UnitPrice,
			Amount:      row.Amount,
			ProNumber:   row.ProNumber,
		})
	}

	return out
}

func invoiceShipmentRows(rows []invoicePDFShipmentRow) []invoicetemplate.ShipmentRow {
	if len(rows) == 0 {
		return nil
	}

	out := make([]invoicetemplate.ShipmentRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, invoicetemplate.ShipmentRow{
			ProNumber:   row.ProNumber,
			BOL:         row.BOL,
			PONumber:    row.PONumber,
			ServiceDate: row.ServiceDate,
			Origin:      row.Origin,
			Destination: row.Destination,
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
func invoiceCommodityRows(rows []invoicePDFCommodityRow) []invoicetemplate.CommodityRow {
	if len(rows) == 0 {
		return nil
	}

	out := make([]invoicetemplate.CommodityRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, invoicetemplate.CommodityRow{
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

// EmailContextParams is one invoice email's data, assembled once and rendered
// once per message part.
type EmailContextParams struct {
	Entity  *invoice.Invoice
	Profile *invoiceDeliveryProfile

	// PartLabel is "2 of 3" when a large invoice is split, so the template can
	// say so rather than sending three identical-looking emails.
	PartLabel string

	// AttachmentName is the file the recipient will see, which a template may
	// want to name in the body so a stripped attachment is recoverable.
	AttachmentName string
}

// emailContext maps an invoice onto the variables the invoice email template
// declares.
//
// It reads the same fields the ad-hoc {number}/{customer} substitution reads, so
// an organization moving from a customer email profile to a template gets the
// same values under documented names. Where the two disagree the delivery profile
// wins, exactly as it does for the ad-hoc engine.
func emailContext(p *EmailContextParams) invoicetemplate.InvoiceEmailContext {
	entity := p.Entity

	out := invoicetemplate.InvoiceEmailContext{
		InvoiceNumber: entity.Number,
		InvoiceDate:   unixDate(entity.InvoiceDate),
		DueDate:       unixDatePtr(entity.DueDate),
		PaymentTerm:   string(entity.PaymentTerm),
		Total: moneyString(
			entity.CurrencyCode,
			entity.TotalAmount.StringFixed(2),
		),
		CurrencyCode:           entity.CurrencyCode,
		Memo:                   entity.Memo,
		CustomerName:           entity.BillToName,
		CustomerCode:           entity.BillToCode,
		ShipmentProNumber:      entity.ShipmentProNumber,
		ShipmentBOL:            entity.ShipmentBOL,
		ShipmentServiceDate:    unixDatePtr(entity.ServiceDate),
		RemittanceInstructions: entity.RemittanceInstructions,
		PartLabel:              p.PartLabel,
		AttachmentName:         p.AttachmentName,
	}

	if p.Profile == nil {
		return out
	}

	if factor := p.Profile.Factor; factor != nil {
		out.RemittanceInstructions = factor.RemittanceText()
		out.NoticeOfAssignment = factor.Notice()
	}

	if org := p.Profile.Organization; org != nil {
		out.CompanyName = org.Name
	}

	if cust := p.Profile.Customer; cust != nil {
		out.CustomerName = stringutils.FirstNonEmpty(cust.Name, out.CustomerName)
		out.CustomerCode = stringutils.FirstNonEmpty(cust.Code, out.CustomerCode)
		if out.CompanyName == "" && cust.Organization != nil {
			out.CompanyName = cust.Organization.Name
		}
	}

	if shp := p.Profile.Shipment; shp != nil {
		out.ShipmentProNumber = stringutils.FirstNonEmpty(shp.ProNumber, out.ShipmentProNumber)
		out.ShipmentBOL = stringutils.FirstNonEmpty(shp.BOL, out.ShipmentBOL)
		out.ShipmentOrigin = shipmentOrigin(shp)
		out.ShipmentDestination = shipmentDestination(shp)
		out.ShipmentServiceDate = stringutils.FirstNonEmpty(
			unixDatePtr(shp.ActualDeliveryDate),
			out.ShipmentServiceDate,
		)
	}

	return out
}
