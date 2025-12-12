package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/uptrace/bun"
)

type DocumentTemplateSeed struct {
	seedhelpers.BaseSeed
}

func NewDocumentTemplateSeed() *DocumentTemplateSeed {
	seed := &DocumentTemplateSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"DocumentTemplates",
		"1.0.0",
		"Creates document template data",
		[]common.Environment{
			common.EnvDevelopment,
		},
	)

	seed.SetDependencies("USStates", "AdminAccount", "Permissions", "DocumentTypes")

	return seed
}

func (s *DocumentTemplateSeed) Run(ctx context.Context, db *bun.DB) error {
	return seedhelpers.RunInTransaction(
		ctx,
		db,
		s.Name(),
		func(ctx context.Context, tx bun.Tx, seedCtx *seedhelpers.SeedContext) error {
			var count int
			err := db.NewSelect().
				Model((*documenttemplate.DocumentTemplate)(nil)).
				ColumnExpr("count(*)").
				Scan(ctx, &count)
			if err != nil {
				return err
			}

			if count > 0 {
				seedhelpers.LogSuccess("Document templates already exist, skipping")
				return nil
			}

			buId, err := seedCtx.GetDefaultBusinessUnit()
			if err != nil {
				return err
			}

			orgId, err := seedCtx.GetDefaultOrganization()
			if err != nil {
				return err
			}

			var invoiceType documenttype.DocumentType
			err = db.NewSelect().
				Model(&invoiceType).
				Where("code = ? AND organization_id = ?", "INVOICE", orgId.ID).
				Scan(ctx)
			if err != nil {
				return fmt.Errorf("failed to find invoice document type: %w", err)
			}

			var creditMemoType documenttype.DocumentType
			err = db.NewSelect().
				Model(&creditMemoType).
				Where("code = ? AND organization_id = ?", "CM", orgId.ID).
				Scan(ctx)
			if err != nil {
				return fmt.Errorf("failed to find credit memo document type: %w", err)
			}

			var bolType documenttype.DocumentType
			err = db.NewSelect().
				Model(&bolType).
				Where("code = ? AND organization_id = ?", "BOL", orgId.ID).
				Scan(ctx)
			if err != nil {
				return fmt.Errorf("failed to find BOL document type: %w", err)
			}

			templates := []documenttemplate.DocumentTemplate{
				{
					BusinessUnitID: buId.ID,
					OrganizationID: orgId.ID,
					Code:           "INV-STD-001",
					Name:           "Standard Invoice",
					Description:    "Standard invoice template for billing customers",
					DocumentTypeID: invoiceType.ID,
					HTMLContent:    invoiceHTMLTemplate,
					CSSContent:     invoiceCSSTemplate,
					HeaderHTML:     invoiceHeaderTemplate,
					FooterHTML:     invoiceFooterTemplate,
					PageSize:       documenttemplate.PageSizeLetter,
					Orientation:    documenttemplate.OrientationPortrait,
					MarginTop:      20,
					MarginBottom:   20,
					MarginLeft:     20,
					MarginRight:    20,
					Status:         documenttemplate.TemplateStatusActive,
					IsDefault:      true,
					IsSystem:       false,
				},
				{
					BusinessUnitID: buId.ID,
					OrganizationID: orgId.ID,
					Code:           "CM-STD-001",
					Name:           "Standard Credit Memo",
					Description:    "Standard credit memo template for customer credits",
					DocumentTypeID: creditMemoType.ID,
					HTMLContent:    creditMemoHTMLTemplate,
					CSSContent:     creditMemoCSSTemplate,
					HeaderHTML:     "",
					FooterHTML:     creditMemoFooterTemplate,
					PageSize:       documenttemplate.PageSizeLetter,
					Orientation:    documenttemplate.OrientationPortrait,
					MarginTop:      20,
					MarginBottom:   20,
					MarginLeft:     20,
					MarginRight:    20,
					Status:         documenttemplate.TemplateStatusActive,
					IsDefault:      true,
					IsSystem:       false,
				},
				{
					BusinessUnitID: buId.ID,
					OrganizationID: orgId.ID,
					Code:           "BOL-STD-001",
					Name:           "Standard Bill of Lading",
					Description:    "Standard bill of lading template for shipments",
					DocumentTypeID: bolType.ID,
					HTMLContent:    bolHTMLTemplate,
					CSSContent:     bolCSSTemplate,
					HeaderHTML:     "",
					FooterHTML:     bolFooterTemplate,
					PageSize:       documenttemplate.PageSizeLetter,
					Orientation:    documenttemplate.OrientationPortrait,
					MarginTop:      15,
					MarginBottom:   15,
					MarginLeft:     15,
					MarginRight:    15,
					Status:         documenttemplate.TemplateStatusActive,
					IsDefault:      true,
					IsSystem:       false,
				},
			}

			if _, err := tx.NewInsert().Model(&templates).Exec(ctx); err != nil {
				return fmt.Errorf("failed to insert document templates: %w", err)
			}

			seedhelpers.LogSuccess("Created document template fixtures",
				"- 3 document templates created (Invoice, Credit Memo, BOL)",
			)

			return nil
		},
	)
}

const invoiceHTMLTemplate = `<div class="invoice">
  <div class="invoice-header">
    <div class="company-info">
      <h1 class="company-name">{{ .Organization.Name }}</h1>
      {{ if .Organization.Address }}
      <p class="company-address">
        {{ .Organization.Address.Line1 }}<br>
        {{ if .Organization.Address.Line2 }}{{ .Organization.Address.Line2 }}<br>{{ end }}
        {{ .Organization.Address.City }}, {{ .Organization.Address.State }} {{ .Organization.Address.PostalCode }}
      </p>
      {{ end }}
    </div>
    <div class="invoice-info">
      <h2 class="invoice-title">INVOICE</h2>
      <table class="invoice-meta">
        <tr>
          <td class="label">Invoice #:</td>
          <td class="value">{{ .InvoiceNumber }}</td>
        </tr>
        <tr>
          <td class="label">Date:</td>
          <td class="value">{{ formatDate .InvoiceDate "Jan 02, 2006" }}</td>
        </tr>
        <tr>
          <td class="label">Due Date:</td>
          <td class="value">{{ formatDate .DueDate "Jan 02, 2006" }}</td>
        </tr>
        {{ if .PurchaseOrder }}
        <tr>
          <td class="label">PO #:</td>
          <td class="value">{{ .PurchaseOrder }}</td>
        </tr>
        {{ end }}
      </table>
    </div>
  </div>

  <div class="bill-to">
    <h3>Bill To:</h3>
    <p class="customer-name">{{ .Customer.Name }}</p>
    {{ if .Customer.BillingAddress }}
    <p class="customer-address">
      {{ .Customer.BillingAddress.Line1 }}<br>
      {{ if .Customer.BillingAddress.Line2 }}{{ .Customer.BillingAddress.Line2 }}<br>{{ end }}
      {{ .Customer.BillingAddress.City }}, {{ .Customer.BillingAddress.State }} {{ .Customer.BillingAddress.PostalCode }}
    </p>
    {{ end }}
  </div>

  <table class="line-items">
    <thead>
      <tr>
        <th class="description">Description</th>
        <th class="quantity">Qty</th>
        <th class="rate">Rate</th>
        <th class="amount">Amount</th>
      </tr>
    </thead>
    <tbody>
      {{ range .LineItems }}
      <tr>
        <td class="description">{{ .Description }}</td>
        <td class="quantity">{{ .Quantity }}</td>
        <td class="rate">{{ formatCurrency .UnitPrice }}</td>
        <td class="amount">{{ formatCurrency .TotalAmount }}</td>
      </tr>
      {{ end }}
    </tbody>
  </table>

  <div class="totals">
    <table class="totals-table">
      <tr>
        <td class="label">Subtotal:</td>
        <td class="value">{{ formatCurrency .Subtotal }}</td>
      </tr>
      {{ if gt .TaxAmount 0 }}
      <tr>
        <td class="label">Tax ({{ .TaxRate }}%):</td>
        <td class="value">{{ formatCurrency .TaxAmount }}</td>
      </tr>
      {{ end }}
      {{ if gt .DiscountAmount 0 }}
      <tr>
        <td class="label">Discount:</td>
        <td class="value">-{{ formatCurrency .DiscountAmount }}</td>
      </tr>
      {{ end }}
      <tr class="total-row">
        <td class="label">Total:</td>
        <td class="value">{{ formatCurrency .TotalAmount }}</td>
      </tr>
      {{ if gt .AmountPaid 0 }}
      <tr>
        <td class="label">Amount Paid:</td>
        <td class="value">{{ formatCurrency .AmountPaid }}</td>
      </tr>
      <tr class="balance-due">
        <td class="label">Balance Due:</td>
        <td class="value">{{ formatCurrency .BalanceDue }}</td>
      </tr>
      {{ end }}
    </table>
  </div>

  {{ if .Notes }}
  <div class="notes">
    <h4>Notes:</h4>
    <p>{{ .Notes }}</p>
  </div>
  {{ end }}

  <div class="payment-terms">
    <p>Payment Terms: {{ .PaymentTerms }}</p>
  </div>
</div>`

const invoiceCSSTemplate = `* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: 'Helvetica Neue', Arial, sans-serif;
  font-size: 12px;
  line-height: 1.5;
  color: #333;
}

.invoice {
  max-width: 800px;
  margin: 0 auto;
  padding: 40px;
}

.invoice-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 40px;
  padding-bottom: 20px;
  border-bottom: 2px solid #2563eb;
}

.company-name {
  font-size: 24px;
  font-weight: 700;
  color: #1e40af;
  margin-bottom: 8px;
}

.company-address {
  color: #6b7280;
  font-size: 11px;
}

.invoice-title {
  font-size: 28px;
  font-weight: 700;
  color: #1e40af;
  text-align: right;
  margin-bottom: 16px;
}

.invoice-meta {
  text-align: right;
}

.invoice-meta td {
  padding: 2px 0;
}

.invoice-meta .label {
  color: #6b7280;
  padding-right: 12px;
}

.invoice-meta .value {
  font-weight: 600;
}

.bill-to {
  margin-bottom: 30px;
}

.bill-to h3 {
  font-size: 11px;
  text-transform: uppercase;
  color: #6b7280;
  margin-bottom: 8px;
}

.customer-name {
  font-weight: 600;
  font-size: 14px;
  margin-bottom: 4px;
}

.customer-address {
  color: #6b7280;
}

.line-items {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 30px;
}

.line-items th {
  background-color: #f3f4f6;
  padding: 12px;
  text-align: left;
  font-size: 11px;
  text-transform: uppercase;
  color: #6b7280;
  border-bottom: 2px solid #e5e7eb;
}

.line-items td {
  padding: 12px;
  border-bottom: 1px solid #e5e7eb;
}

.line-items .quantity,
.line-items .rate,
.line-items .amount {
  text-align: right;
  width: 100px;
}

.totals {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 30px;
}

.totals-table {
  width: 250px;
}

.totals-table td {
  padding: 8px 0;
}

.totals-table .label {
  text-align: right;
  padding-right: 16px;
  color: #6b7280;
}

.totals-table .value {
  text-align: right;
  font-weight: 500;
}

.total-row td {
  border-top: 2px solid #1e40af;
  padding-top: 12px;
  font-size: 16px;
  font-weight: 700;
}

.balance-due td {
  color: #dc2626;
  font-weight: 700;
}

.notes {
  background-color: #f9fafb;
  padding: 16px;
  border-radius: 8px;
  margin-bottom: 20px;
}

.notes h4 {
  font-size: 11px;
  text-transform: uppercase;
  color: #6b7280;
  margin-bottom: 8px;
}

.payment-terms {
  text-align: center;
  color: #6b7280;
  font-size: 11px;
  padding-top: 20px;
  border-top: 1px solid #e5e7eb;
}`

const invoiceHeaderTemplate = `<div style="display: flex; justify-content: space-between; padding: 10px 20px; border-bottom: 1px solid #e5e7eb; font-size: 10px; color: #6b7280;">
  <span>{{ .Organization.Name }}</span>
  <span>Invoice #{{ .InvoiceNumber }}</span>
</div>`

const invoiceFooterTemplate = `<div style="display: flex; justify-content: space-between; padding: 10px 20px; border-top: 1px solid #e5e7eb; font-size: 10px; color: #6b7280;">
  <span>Generated on {{ formatDate .GeneratedAt "Jan 02, 2006 3:04 PM" }}</span>
  <span>Page {{ .PageNumber }} of {{ .TotalPages }}</span>
</div>`

const creditMemoHTMLTemplate = `<div class="credit-memo">
  <div class="memo-header">
    <div class="company-info">
      <h1 class="company-name">{{ .Organization.Name }}</h1>
      {{ if .Organization.Address }}
      <p class="company-address">
        {{ .Organization.Address.Line1 }}<br>
        {{ .Organization.Address.City }}, {{ .Organization.Address.State }} {{ .Organization.Address.PostalCode }}
      </p>
      {{ end }}
    </div>
    <div class="memo-info">
      <h2 class="memo-title">CREDIT MEMO</h2>
      <table class="memo-meta">
        <tr>
          <td class="label">Credit Memo #:</td>
          <td class="value">{{ .CreditMemoNumber }}</td>
        </tr>
        <tr>
          <td class="label">Date:</td>
          <td class="value">{{ formatDate .IssueDate "Jan 02, 2006" }}</td>
        </tr>
        {{ if .OriginalInvoiceNumber }}
        <tr>
          <td class="label">Original Invoice #:</td>
          <td class="value">{{ .OriginalInvoiceNumber }}</td>
        </tr>
        {{ end }}
      </table>
    </div>
  </div>

  <div class="customer-section">
    <h3>Credit To:</h3>
    <p class="customer-name">{{ .Customer.Name }}</p>
    {{ if .Customer.BillingAddress }}
    <p class="customer-address">
      {{ .Customer.BillingAddress.Line1 }}<br>
      {{ .Customer.BillingAddress.City }}, {{ .Customer.BillingAddress.State }} {{ .Customer.BillingAddress.PostalCode }}
    </p>
    {{ end }}
  </div>

  <div class="reason-section">
    <h3>Reason for Credit:</h3>
    <p>{{ .Reason }}</p>
  </div>

  <table class="credit-items">
    <thead>
      <tr>
        <th class="description">Description</th>
        <th class="quantity">Qty</th>
        <th class="rate">Rate</th>
        <th class="amount">Credit Amount</th>
      </tr>
    </thead>
    <tbody>
      {{ range .LineItems }}
      <tr>
        <td class="description">{{ .Description }}</td>
        <td class="quantity">{{ .Quantity }}</td>
        <td class="rate">{{ formatCurrency .UnitPrice }}</td>
        <td class="amount">{{ formatCurrency .TotalAmount }}</td>
      </tr>
      {{ end }}
    </tbody>
  </table>

  <div class="totals">
    <table class="totals-table">
      <tr class="total-row">
        <td class="label">Total Credit:</td>
        <td class="value">{{ formatCurrency .TotalCredit }}</td>
      </tr>
    </table>
  </div>

  <div class="application-notice">
    <p>This credit will be applied to your account and can be used against future invoices.</p>
  </div>
</div>`

const creditMemoCSSTemplate = `* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: 'Helvetica Neue', Arial, sans-serif;
  font-size: 12px;
  line-height: 1.5;
  color: #333;
}

.credit-memo {
  max-width: 800px;
  margin: 0 auto;
  padding: 40px;
}

.memo-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 40px;
  padding-bottom: 20px;
  border-bottom: 2px solid #059669;
}

.company-name {
  font-size: 24px;
  font-weight: 700;
  color: #047857;
  margin-bottom: 8px;
}

.company-address {
  color: #6b7280;
  font-size: 11px;
}

.memo-title {
  font-size: 28px;
  font-weight: 700;
  color: #059669;
  text-align: right;
  margin-bottom: 16px;
}

.memo-meta {
  text-align: right;
}

.memo-meta td {
  padding: 2px 0;
}

.memo-meta .label {
  color: #6b7280;
  padding-right: 12px;
}

.memo-meta .value {
  font-weight: 600;
}

.customer-section,
.reason-section {
  margin-bottom: 24px;
}

.customer-section h3,
.reason-section h3 {
  font-size: 11px;
  text-transform: uppercase;
  color: #6b7280;
  margin-bottom: 8px;
}

.customer-name {
  font-weight: 600;
  font-size: 14px;
  margin-bottom: 4px;
}

.customer-address {
  color: #6b7280;
}

.credit-items {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 30px;
}

.credit-items th {
  background-color: #ecfdf5;
  padding: 12px;
  text-align: left;
  font-size: 11px;
  text-transform: uppercase;
  color: #047857;
  border-bottom: 2px solid #059669;
}

.credit-items td {
  padding: 12px;
  border-bottom: 1px solid #e5e7eb;
}

.credit-items .quantity,
.credit-items .rate,
.credit-items .amount {
  text-align: right;
  width: 100px;
}

.totals {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 30px;
}

.totals-table {
  width: 250px;
}

.total-row td {
  padding: 12px 0;
  border-top: 2px solid #059669;
  font-size: 18px;
  font-weight: 700;
  color: #059669;
}

.total-row .label {
  text-align: right;
  padding-right: 16px;
}

.total-row .value {
  text-align: right;
}

.application-notice {
  background-color: #ecfdf5;
  padding: 16px;
  border-radius: 8px;
  text-align: center;
  color: #047857;
  font-size: 11px;
}`

const creditMemoFooterTemplate = `<div style="display: flex; justify-content: space-between; padding: 10px 20px; border-top: 1px solid #e5e7eb; font-size: 10px; color: #6b7280;">
  <span>Credit Memo #{{ .CreditMemoNumber }}</span>
  <span>Page {{ .PageNumber }} of {{ .TotalPages }}</span>
</div>`

const bolHTMLTemplate = `<div class="bol">
  <div class="bol-header">
    <div class="title-section">
      <h1>BILL OF LADING</h1>
      <p class="subtitle">Straight Bill of Lading - Short Form</p>
    </div>
    <div class="bol-number">
      <table>
        <tr>
          <td class="label">BOL #:</td>
          <td class="value">{{ .BolNumber }}</td>
        </tr>
        <tr>
          <td class="label">Date:</td>
          <td class="value">{{ formatDate .ShipDate "Jan 02, 2006" }}</td>
        </tr>
        <tr>
          <td class="label">Pro #:</td>
          <td class="value">{{ .ProNumber }}</td>
        </tr>
      </table>
    </div>
  </div>

  <div class="parties-section">
    <div class="party shipper">
      <h3>SHIPPER</h3>
      <p class="name">{{ .Shipper.Name }}</p>
      <p class="address">
        {{ .Shipper.Address.Line1 }}<br>
        {{ .Shipper.Address.City }}, {{ .Shipper.Address.State }} {{ .Shipper.Address.PostalCode }}
      </p>
      {{ if .Shipper.Phone }}
      <p class="phone">Phone: {{ .Shipper.Phone }}</p>
      {{ end }}
    </div>
    <div class="party consignee">
      <h3>CONSIGNEE</h3>
      <p class="name">{{ .Consignee.Name }}</p>
      <p class="address">
        {{ .Consignee.Address.Line1 }}<br>
        {{ .Consignee.Address.City }}, {{ .Consignee.Address.State }} {{ .Consignee.Address.PostalCode }}
      </p>
      {{ if .Consignee.Phone }}
      <p class="phone">Phone: {{ .Consignee.Phone }}</p>
      {{ end }}
    </div>
  </div>

  <div class="carrier-section">
    <div class="carrier">
      <h3>CARRIER</h3>
      <p class="name">{{ .Organization.Name }}</p>
      {{ if .Organization.MCNumber }}
      <p class="mc">MC #: {{ .Organization.MCNumber }}</p>
      {{ end }}
    </div>
    <div class="equipment">
      <h3>EQUIPMENT</h3>
      {{ if .Tractor }}
      <p>Tractor: {{ .Tractor.UnitNumber }}</p>
      {{ end }}
      {{ if .Trailer }}
      <p>Trailer: {{ .Trailer.UnitNumber }}</p>
      {{ end }}
      {{ if .SealNumber }}
      <p>Seal #: {{ .SealNumber }}</p>
      {{ end }}
    </div>
  </div>

  <table class="commodity-table">
    <thead>
      <tr>
        <th>Pieces</th>
        <th>HM</th>
        <th>Description</th>
        <th>Weight</th>
        <th>Class</th>
      </tr>
    </thead>
    <tbody>
      {{ range .Commodities }}
      <tr>
        <td class="center">{{ .Pieces }}</td>
        <td class="center">{{ if .IsHazmat }}X{{ end }}</td>
        <td>{{ .Description }}</td>
        <td class="right">{{ formatWeight .Weight }}</td>
        <td class="center">{{ .FreightClass }}</td>
      </tr>
      {{ end }}
    </tbody>
    <tfoot>
      <tr>
        <td colspan="3" class="right"><strong>Total Weight:</strong></td>
        <td class="right"><strong>{{ formatWeight .TotalWeight }}</strong></td>
        <td></td>
      </tr>
    </tfoot>
  </table>

  {{ if .SpecialInstructions }}
  <div class="special-instructions">
    <h3>SPECIAL INSTRUCTIONS</h3>
    <p>{{ .SpecialInstructions }}</p>
  </div>
  {{ end }}

  <div class="signature-section">
    <div class="signature shipper-sig">
      <p class="sig-label">Shipper Signature</p>
      <div class="sig-line"></div>
      <p class="sig-date">Date: _______________</p>
    </div>
    <div class="signature carrier-sig">
      <p class="sig-label">Carrier Signature</p>
      <div class="sig-line"></div>
      <p class="sig-date">Date: _______________</p>
    </div>
    <div class="signature consignee-sig">
      <p class="sig-label">Consignee Signature</p>
      <div class="sig-line"></div>
      <p class="sig-date">Date: _______________</p>
    </div>
  </div>

  <div class="terms">
    <p class="terms-text">
      Received, subject to individually determined rates or contracts that have been agreed upon in writing between the carrier and shipper,
      if applicable, otherwise to the rates, classifications and rules that have been established by the carrier and are available to the
      shipper, on request, the property described above in apparent good order, except as noted.
    </p>
  </div>
</div>`

const bolCSSTemplate = `* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: 'Courier New', monospace;
  font-size: 11px;
  line-height: 1.4;
  color: #000;
}

.bol {
  max-width: 800px;
  margin: 0 auto;
  padding: 20px;
}

.bol-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 20px;
  padding-bottom: 15px;
  border-bottom: 3px double #000;
}

.title-section h1 {
  font-size: 24px;
  font-weight: bold;
  letter-spacing: 2px;
}

.subtitle {
  font-size: 10px;
  color: #666;
  margin-top: 4px;
}

.bol-number table {
  border: 2px solid #000;
  padding: 8px;
}

.bol-number td {
  padding: 2px 8px;
}

.bol-number .label {
  font-weight: bold;
}

.bol-number .value {
  font-size: 12px;
}

.parties-section {
  display: flex;
  gap: 20px;
  margin-bottom: 20px;
}

.party {
  flex: 1;
  border: 1px solid #000;
  padding: 12px;
}

.party h3 {
  font-size: 10px;
  background: #000;
  color: #fff;
  padding: 4px 8px;
  margin: -12px -12px 10px -12px;
}

.party .name {
  font-weight: bold;
  margin-bottom: 4px;
}

.party .address {
  margin-bottom: 4px;
}

.party .phone {
  font-size: 10px;
  color: #666;
}

.carrier-section {
  display: flex;
  gap: 20px;
  margin-bottom: 20px;
}

.carrier-section > div {
  flex: 1;
  border: 1px solid #000;
  padding: 12px;
}

.carrier-section h3 {
  font-size: 10px;
  background: #000;
  color: #fff;
  padding: 4px 8px;
  margin: -12px -12px 10px -12px;
}

.carrier-section .name {
  font-weight: bold;
}

.commodity-table {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 20px;
}

.commodity-table th,
.commodity-table td {
  border: 1px solid #000;
  padding: 8px;
}

.commodity-table th {
  background: #f0f0f0;
  font-weight: bold;
  text-transform: uppercase;
  font-size: 10px;
}

.commodity-table .center {
  text-align: center;
}

.commodity-table .right {
  text-align: right;
}

.commodity-table tfoot td {
  border-top: 2px solid #000;
}

.special-instructions {
  border: 1px solid #000;
  padding: 12px;
  margin-bottom: 20px;
}

.special-instructions h3 {
  font-size: 10px;
  margin-bottom: 8px;
  text-transform: uppercase;
}

.signature-section {
  display: flex;
  gap: 20px;
  margin-bottom: 20px;
}

.signature {
  flex: 1;
  text-align: center;
}

.sig-label {
  font-size: 10px;
  font-weight: bold;
  margin-bottom: 30px;
}

.sig-line {
  border-bottom: 1px solid #000;
  margin-bottom: 8px;
  height: 20px;
}

.sig-date {
  font-size: 10px;
}

.terms {
  border-top: 1px solid #000;
  padding-top: 12px;
}

.terms-text {
  font-size: 8px;
  color: #666;
  text-align: justify;
}`

const bolFooterTemplate = `<div style="display: flex; justify-content: space-between; padding: 8px 20px; border-top: 1px solid #000; font-size: 9px;">
  <span>BOL #{{ .BolNumber }}</span>
  <span>{{ .Organization.Name }}</span>
  <span>Page {{ .PageNumber }} of {{ .TotalPages }}</span>
</div>`
