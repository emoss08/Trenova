import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import {
  ClipboardList,
  FileText,
  Package,
  Receipt,
  Sparkles,
  Truck,
} from "lucide-react";
import { useState } from "react";

interface TemplatePreset {
  id: string;
  name: string;
  description: string;
  icon: React.ElementType;
  gradient: string;
  htmlContent: string;
  cssContent: string;
  headerHtml?: string;
  footerHtml?: string;
}

const templatePresets: TemplatePreset[] = [
  {
    id: "invoice",
    name: "Professional Invoice",
    description: "Clean, modern invoice with line items table",
    icon: Receipt,
    gradient: "from-blue-500 to-indigo-600",
    htmlContent: `<div class="invoice">
  <header class="invoice-header">
    <div class="company-info">
      <img src="{{ .CompanyLogo }}" alt="Logo" class="logo" />
      <div>
        <h1>{{ .CompanyName }}</h1>
        <p>{{ .CompanyAddress }}</p>
        <p>{{ .CompanyPhone }} | {{ .CompanyEmail }}</p>
      </div>
    </div>
    <div class="invoice-meta">
      <h2>INVOICE</h2>
      <p><strong>Invoice #:</strong> {{ .DocumentNumber }}</p>
      <p><strong>Date:</strong> {{ formatDate .DocumentDate }}</p>
      <p><strong>Due Date:</strong> {{ formatDate .DueDate }}</p>
    </div>
  </header>

  <section class="bill-to">
    <h3>Bill To:</h3>
    <p><strong>{{ .CustomerName }}</strong></p>
    <p>{{ .CustomerAddress }}</p>
    <p>{{ .CustomerEmail }}</p>
  </section>

  <table class="items-table">
    <thead>
      <tr>
        <th>Description</th>
        <th>Qty</th>
        <th>Unit Price</th>
        <th>Amount</th>
      </tr>
    </thead>
    <tbody>
      {{ range .LineItems }}
      <tr>
        <td>{{ .Description }}</td>
        <td>{{ .Quantity }}</td>
        <td>{{ formatCurrency .UnitPrice }}</td>
        <td>{{ formatCurrency .Total }}</td>
      </tr>
      {{ end }}
    </tbody>
  </table>

  <section class="totals">
    <div class="totals-row">
      <span>Subtotal</span>
      <span>{{ formatCurrency .Subtotal }}</span>
    </div>
    <div class="totals-row">
      <span>Tax</span>
      <span>{{ formatCurrency .TaxAmount }}</span>
    </div>
    <div class="totals-row total">
      <span>Total Due</span>
      <span>{{ formatCurrency .TotalAmount }}</span>
    </div>
  </section>

  <footer class="invoice-footer">
    <p>Thank you for your business!</p>
    <p class="terms">Payment terms: Net 30 days</p>
  </footer>
</div>`,
    cssContent: `.invoice {
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
  max-width: 800px;
  margin: 0 auto;
  padding: 40px;
  color: #1f2937;
}

.invoice-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 40px;
  padding-bottom: 20px;
  border-bottom: 2px solid #e5e7eb;
}

.company-info {
  display: flex;
  gap: 16px;
  align-items: center;
}

.logo {
  width: 60px;
  height: 60px;
  object-fit: contain;
}

.company-info h1 {
  font-size: 24px;
  font-weight: 700;
  margin: 0 0 4px;
  color: #111827;
}

.company-info p {
  margin: 0;
  font-size: 13px;
  color: #6b7280;
}

.invoice-meta {
  text-align: right;
}

.invoice-meta h2 {
  font-size: 32px;
  font-weight: 800;
  color: #3b82f6;
  margin: 0 0 12px;
  letter-spacing: -0.5px;
}

.invoice-meta p {
  margin: 4px 0;
  font-size: 14px;
}

.bill-to {
  background: #f9fafb;
  padding: 20px;
  border-radius: 8px;
  margin-bottom: 30px;
}

.bill-to h3 {
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: #6b7280;
  margin: 0 0 8px;
}

.bill-to p {
  margin: 4px 0;
  font-size: 14px;
}

.items-table {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 30px;
}

.items-table th {
  background: #f3f4f6;
  padding: 12px 16px;
  text-align: left;
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: #6b7280;
  font-weight: 600;
}

.items-table th:last-child,
.items-table td:last-child {
  text-align: right;
}

.items-table td {
  padding: 16px;
  border-bottom: 1px solid #e5e7eb;
  font-size: 14px;
}

.totals {
  margin-left: auto;
  width: 280px;
}

.totals-row {
  display: flex;
  justify-content: space-between;
  padding: 8px 0;
  font-size: 14px;
}

.totals-row.total {
  border-top: 2px solid #111827;
  margin-top: 8px;
  padding-top: 12px;
  font-size: 18px;
  font-weight: 700;
}

.invoice-footer {
  margin-top: 60px;
  text-align: center;
  color: #6b7280;
}

.invoice-footer p {
  margin: 4px 0;
}

.terms {
  font-size: 12px;
}`,
    headerHtml: `<div class="page-header">
  <span>Invoice #{{ .DocumentNumber }}</span>
  <span>Page {{ .PageNumber }} of {{ .TotalPages }}</span>
</div>`,
    footerHtml: `<div class="page-footer">
  <p>{{ .CompanyName }} | {{ .CompanyPhone }} | {{ .CompanyEmail }}</p>
</div>`,
  },
  {
    id: "bol",
    name: "Bill of Lading",
    description: "Standard BOL with shipper/consignee details",
    icon: Truck,
    gradient: "from-orange-500 to-red-600",
    htmlContent: `<div class="bol">
  <header class="bol-header">
    <div class="carrier-info">
      <img src="{{ .CompanyLogo }}" alt="Logo" class="logo" />
      <div>
        <h1>{{ .CompanyName }}</h1>
        <p>{{ .CompanyAddress }}</p>
      </div>
    </div>
    <div class="bol-title">
      <h2>BILL OF LADING</h2>
      <p class="bol-number">{{ .BOLNumber }}</p>
      <p class="date">Date: {{ formatDate .DocumentDate }}</p>
    </div>
  </header>

  <div class="parties-grid">
    <section class="party shipper">
      <h3>Shipper</h3>
      <p class="name">{{ .ShipperName }}</p>
      <p>{{ .ShipperAddress }}</p>
      <p>{{ .ShipperPhone }}</p>
    </section>
    <section class="party consignee">
      <h3>Consignee</h3>
      <p class="name">{{ .ConsigneeName }}</p>
      <p>{{ .ConsigneeAddress }}</p>
      <p>{{ .ConsigneePhone }}</p>
    </section>
  </div>

  <section class="shipment-details">
    <div class="detail-row">
      <div class="detail">
        <label>PRO Number</label>
        <span>{{ .ProNumber }}</span>
      </div>
      <div class="detail">
        <label>PO Number</label>
        <span>{{ .PONumber }}</span>
      </div>
      <div class="detail">
        <label>Ship Date</label>
        <span>{{ formatDate .PickupDate }}</span>
      </div>
    </div>
  </section>

  <table class="freight-table">
    <thead>
      <tr>
        <th>Pieces</th>
        <th>Description</th>
        <th>Weight</th>
        <th>Class</th>
        <th>NMFC</th>
      </tr>
    </thead>
    <tbody>
      {{ range .Commodities }}
      <tr>
        <td>{{ .Pieces }}</td>
        <td>{{ .Description }}</td>
        <td>{{ formatWeight .Weight }}</td>
        <td>{{ .FreightClass }}</td>
        <td>{{ .NMFC }}</td>
      </tr>
      {{ end }}
    </tbody>
    <tfoot>
      <tr>
        <td><strong>{{ .TotalPieces }}</strong></td>
        <td></td>
        <td><strong>{{ formatWeight .TotalWeight }}</strong></td>
        <td></td>
        <td></td>
      </tr>
    </tfoot>
  </table>

  {{ if .IsHazmat }}
  <section class="hazmat-notice">
    <h4>HAZARDOUS MATERIALS</h4>
    <p>This shipment contains hazardous materials as defined by DOT regulations.</p>
    <p>UN#: {{ .HazmatUN }} | Class: {{ .HazmatClass }}</p>
  </section>
  {{ end }}

  <section class="signatures">
    <div class="signature-block">
      <label>Shipper Signature</label>
      <div class="signature-line"></div>
      <p class="date-line">Date: _____________</p>
    </div>
    <div class="signature-block">
      <label>Driver Signature</label>
      <div class="signature-line"></div>
      <p class="date-line">Date: _____________</p>
    </div>
    <div class="signature-block">
      <label>Consignee Signature</label>
      <div class="signature-line"></div>
      <p class="date-line">Date: _____________</p>
    </div>
  </section>
</div>`,
    cssContent: `.bol {
  font-family: 'Inter', -apple-system, sans-serif;
  max-width: 800px;
  margin: 0 auto;
  padding: 30px;
  color: #1f2937;
}

.bol-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 30px;
  padding-bottom: 20px;
  border-bottom: 3px solid #ea580c;
}

.carrier-info {
  display: flex;
  gap: 12px;
  align-items: center;
}

.logo {
  width: 50px;
  height: 50px;
}

.carrier-info h1 {
  font-size: 20px;
  font-weight: 700;
  margin: 0;
}

.carrier-info p {
  margin: 4px 0 0;
  font-size: 12px;
  color: #6b7280;
}

.bol-title {
  text-align: right;
}

.bol-title h2 {
  font-size: 24px;
  font-weight: 800;
  color: #ea580c;
  margin: 0;
}

.bol-number {
  font-size: 18px;
  font-weight: 600;
  margin: 4px 0;
}

.date {
  font-size: 13px;
  color: #6b7280;
  margin: 0;
}

.parties-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
  margin-bottom: 24px;
}

.party {
  background: #fef3e2;
  padding: 16px;
  border-radius: 8px;
  border-left: 4px solid #ea580c;
}

.party h3 {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: #9a3412;
  margin: 0 0 8px;
}

.party .name {
  font-weight: 600;
  font-size: 15px;
  margin: 0 0 4px;
}

.party p {
  margin: 2px 0;
  font-size: 13px;
  color: #6b7280;
}

.shipment-details {
  background: #f9fafb;
  padding: 16px;
  border-radius: 8px;
  margin-bottom: 24px;
}

.detail-row {
  display: flex;
  gap: 32px;
}

.detail {
  display: flex;
  flex-direction: column;
}

.detail label {
  font-size: 11px;
  text-transform: uppercase;
  color: #6b7280;
  margin-bottom: 4px;
}

.detail span {
  font-size: 15px;
  font-weight: 600;
}

.freight-table {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 24px;
}

.freight-table th {
  background: #fed7aa;
  padding: 10px 12px;
  text-align: left;
  font-size: 11px;
  text-transform: uppercase;
  font-weight: 600;
  color: #9a3412;
}

.freight-table td {
  padding: 12px;
  border-bottom: 1px solid #e5e7eb;
  font-size: 13px;
}

.freight-table tfoot td {
  background: #fef3e2;
  font-weight: 600;
}

.hazmat-notice {
  background: #fef2f2;
  border: 2px solid #ef4444;
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 24px;
}

.hazmat-notice h4 {
  color: #dc2626;
  margin: 0 0 8px;
  font-size: 14px;
}

.hazmat-notice p {
  margin: 4px 0;
  font-size: 13px;
}

.signatures {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 24px;
  margin-top: 40px;
}

.signature-block label {
  display: block;
  font-size: 12px;
  font-weight: 600;
  margin-bottom: 40px;
}

.signature-line {
  border-bottom: 1px solid #111827;
  margin-bottom: 8px;
}

.date-line {
  font-size: 12px;
  color: #6b7280;
  margin: 0;
}`,
  },
  {
    id: "delivery-receipt",
    name: "Delivery Receipt",
    description: "Proof of delivery with signature capture",
    icon: ClipboardList,
    gradient: "from-green-500 to-emerald-600",
    htmlContent: `<div class="receipt">
  <header class="receipt-header">
    <div class="company">
      <img src="{{ .CompanyLogo }}" alt="Logo" class="logo" />
      <h1>{{ .CompanyName }}</h1>
    </div>
    <div class="receipt-info">
      <h2>DELIVERY RECEIPT</h2>
      <p><strong>Receipt #:</strong> {{ .DocumentNumber }}</p>
      <p><strong>Date:</strong> {{ formatDate .DeliveryDate }}</p>
    </div>
  </header>

  <section class="delivery-details">
    <div class="detail-grid">
      <div class="detail-item">
        <label>PRO Number</label>
        <span>{{ .ProNumber }}</span>
      </div>
      <div class="detail-item">
        <label>BOL Number</label>
        <span>{{ .BOLNumber }}</span>
      </div>
      <div class="detail-item">
        <label>PO Number</label>
        <span>{{ .PONumber }}</span>
      </div>
      <div class="detail-item">
        <label>Delivery Time</label>
        <span>{{ .DeliveryTime }}</span>
      </div>
    </div>
  </section>

  <section class="consignee-info">
    <h3>Delivered To</h3>
    <p class="name">{{ .ConsigneeName }}</p>
    <p>{{ .ConsigneeAddress }}</p>
  </section>

  <table class="items-delivered">
    <thead>
      <tr>
        <th>Qty</th>
        <th>Description</th>
        <th>Weight</th>
        <th>Condition</th>
      </tr>
    </thead>
    <tbody>
      {{ range .Commodities }}
      <tr>
        <td>{{ .Pieces }}</td>
        <td>{{ .Description }}</td>
        <td>{{ formatWeight .Weight }}</td>
        <td class="condition good">Good</td>
      </tr>
      {{ end }}
    </tbody>
  </table>

  <section class="confirmation">
    <div class="checkbox-group">
      <div class="checkbox">
        <span class="box"></span>
        <span>All items received in good condition</span>
      </div>
      <div class="checkbox">
        <span class="box"></span>
        <span>Items received with exceptions (see notes)</span>
      </div>
    </div>

    <div class="notes">
      <label>Notes / Exceptions:</label>
      <div class="notes-box"></div>
    </div>

    <div class="signature-section">
      <div class="signature-block">
        <label>Received By (Print Name)</label>
        <div class="line"></div>
      </div>
      <div class="signature-block">
        <label>Signature</label>
        <div class="line"></div>
      </div>
      <div class="signature-block small">
        <label>Date / Time</label>
        <div class="line"></div>
      </div>
    </div>
  </section>
</div>`,
    cssContent: `.receipt {
  font-family: 'Inter', -apple-system, sans-serif;
  max-width: 800px;
  margin: 0 auto;
  padding: 30px;
  color: #1f2937;
}

.receipt-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 30px;
  padding-bottom: 16px;
  border-bottom: 3px solid #10b981;
}

.company {
  display: flex;
  align-items: center;
  gap: 12px;
}

.logo {
  width: 48px;
  height: 48px;
}

.company h1 {
  font-size: 20px;
  font-weight: 700;
  margin: 0;
}

.receipt-info {
  text-align: right;
}

.receipt-info h2 {
  font-size: 22px;
  font-weight: 800;
  color: #10b981;
  margin: 0 0 8px;
}

.receipt-info p {
  margin: 4px 0;
  font-size: 13px;
}

.delivery-details {
  background: #ecfdf5;
  padding: 16px;
  border-radius: 8px;
  margin-bottom: 20px;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.detail-item {
  display: flex;
  flex-direction: column;
}

.detail-item label {
  font-size: 10px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: #6b7280;
  margin-bottom: 4px;
}

.detail-item span {
  font-size: 14px;
  font-weight: 600;
}

.consignee-info {
  margin-bottom: 20px;
}

.consignee-info h3 {
  font-size: 11px;
  text-transform: uppercase;
  color: #6b7280;
  margin: 0 0 8px;
}

.consignee-info .name {
  font-size: 16px;
  font-weight: 600;
  margin: 0 0 4px;
}

.consignee-info p {
  margin: 2px 0;
  font-size: 13px;
  color: #6b7280;
}

.items-delivered {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 24px;
}

.items-delivered th {
  background: #d1fae5;
  padding: 10px 12px;
  text-align: left;
  font-size: 11px;
  text-transform: uppercase;
  font-weight: 600;
  color: #065f46;
}

.items-delivered td {
  padding: 12px;
  border-bottom: 1px solid #e5e7eb;
  font-size: 13px;
}

.condition.good {
  color: #10b981;
  font-weight: 600;
}

.confirmation {
  background: #f9fafb;
  padding: 20px;
  border-radius: 8px;
}

.checkbox-group {
  margin-bottom: 20px;
}

.checkbox {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  font-size: 14px;
}

.box {
  width: 16px;
  height: 16px;
  border: 2px solid #9ca3af;
  border-radius: 3px;
}

.notes {
  margin-bottom: 24px;
}

.notes label {
  display: block;
  font-size: 12px;
  font-weight: 600;
  margin-bottom: 8px;
}

.notes-box {
  height: 60px;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  background: white;
}

.signature-section {
  display: grid;
  grid-template-columns: 2fr 2fr 1fr;
  gap: 20px;
}

.signature-block label {
  display: block;
  font-size: 11px;
  font-weight: 600;
  margin-bottom: 30px;
}

.signature-block .line {
  border-bottom: 1px solid #111827;
}

.signature-block.small {
  min-width: 100px;
}`,
  },
  {
    id: "packing-list",
    name: "Packing List",
    description: "Detailed list of package contents",
    icon: Package,
    gradient: "from-purple-500 to-violet-600",
    htmlContent: `<div class="packing-list">
  <header class="list-header">
    <div class="company">
      <h1>{{ .CompanyName }}</h1>
      <p>{{ .CompanyAddress }}</p>
    </div>
    <div class="list-meta">
      <h2>PACKING LIST</h2>
      <p>PO #: {{ .PONumber }}</p>
      <p>Date: {{ formatDate .DocumentDate }}</p>
    </div>
  </header>

  <div class="address-grid">
    <section class="address ship-from">
      <h3>Ship From</h3>
      <p class="name">{{ .ShipperName }}</p>
      <p>{{ .ShipperAddress }}</p>
    </section>
    <section class="address ship-to">
      <h3>Ship To</h3>
      <p class="name">{{ .ConsigneeName }}</p>
      <p>{{ .ConsigneeAddress }}</p>
    </section>
  </div>

  <table class="items-table">
    <thead>
      <tr>
        <th>Item #</th>
        <th>Description</th>
        <th>Qty Ordered</th>
        <th>Qty Shipped</th>
        <th>Unit</th>
      </tr>
    </thead>
    <tbody>
      {{ range .LineItems }}
      <tr>
        <td>{{ .ItemNumber }}</td>
        <td>{{ .Description }}</td>
        <td>{{ .QuantityOrdered }}</td>
        <td>{{ .QuantityShipped }}</td>
        <td>{{ .Unit }}</td>
      </tr>
      {{ end }}
    </tbody>
  </table>

  <section class="package-summary">
    <h3>Package Summary</h3>
    <div class="summary-grid">
      <div class="summary-item">
        <span class="value">{{ .TotalPackages }}</span>
        <span class="label">Packages</span>
      </div>
      <div class="summary-item">
        <span class="value">{{ formatWeight .TotalWeight }}</span>
        <span class="label">Total Weight</span>
      </div>
      <div class="summary-item">
        <span class="value">{{ .TotalPieces }}</span>
        <span class="label">Total Items</span>
      </div>
    </div>
  </section>
</div>`,
    cssContent: `.packing-list {
  font-family: 'Inter', -apple-system, sans-serif;
  max-width: 800px;
  margin: 0 auto;
  padding: 30px;
  color: #1f2937;
}

.list-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 30px;
  padding-bottom: 16px;
  border-bottom: 3px solid #8b5cf6;
}

.company h1 {
  font-size: 22px;
  font-weight: 700;
  margin: 0 0 4px;
}

.company p {
  margin: 0;
  font-size: 13px;
  color: #6b7280;
}

.list-meta {
  text-align: right;
}

.list-meta h2 {
  font-size: 24px;
  font-weight: 800;
  color: #8b5cf6;
  margin: 0 0 8px;
}

.list-meta p {
  margin: 4px 0;
  font-size: 13px;
}

.address-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
  margin-bottom: 24px;
}

.address {
  background: #f5f3ff;
  padding: 16px;
  border-radius: 8px;
  border-left: 4px solid #8b5cf6;
}

.address h3 {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: #6d28d9;
  margin: 0 0 8px;
}

.address .name {
  font-weight: 600;
  font-size: 15px;
  margin: 0 0 4px;
}

.address p {
  margin: 2px 0;
  font-size: 13px;
  color: #6b7280;
}

.items-table {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 30px;
}

.items-table th {
  background: #ede9fe;
  padding: 10px 12px;
  text-align: left;
  font-size: 11px;
  text-transform: uppercase;
  font-weight: 600;
  color: #6d28d9;
}

.items-table td {
  padding: 12px;
  border-bottom: 1px solid #e5e7eb;
  font-size: 13px;
}

.package-summary {
  background: linear-gradient(135deg, #8b5cf6 0%, #6d28d9 100%);
  padding: 24px;
  border-radius: 12px;
  color: white;
}

.package-summary h3 {
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  opacity: 0.8;
  margin: 0 0 16px;
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 20px;
}

.summary-item {
  text-align: center;
}

.summary-item .value {
  display: block;
  font-size: 28px;
  font-weight: 700;
}

.summary-item .label {
  font-size: 12px;
  opacity: 0.8;
}`,
  },
  {
    id: "blank",
    name: "Blank Template",
    description: "Start from scratch with a clean slate",
    icon: FileText,
    gradient: "from-gray-400 to-gray-600",
    htmlContent: `<div class="document">
  <!-- Start building your template here -->
  <h1>Document Title</h1>
  <p>Your content goes here...</p>
</div>`,
    cssContent: `.document {
  font-family: 'Inter', -apple-system, sans-serif;
  max-width: 800px;
  margin: 0 auto;
  padding: 40px;
  color: #1f2937;
}

h1 {
  font-size: 28px;
  font-weight: 700;
  margin: 0 0 16px;
}

p {
  font-size: 14px;
  line-height: 1.6;
  color: #4b5563;
}`,
  },
];

interface TemplatePresetsProps {
  onSelect: (preset: {
    htmlContent: string;
    cssContent: string;
    headerHtml?: string;
    footerHtml?: string;
  }) => void;
}

export function TemplatePresets({ onSelect }: TemplatePresetsProps) {
  const [selectedPreset, setSelectedPreset] = useState<TemplatePreset | null>(
    null,
  );
  const [open, setOpen] = useState(false);

  const handleSelect = () => {
    if (selectedPreset) {
      onSelect({
        htmlContent: selectedPreset.htmlContent,
        cssContent: selectedPreset.cssContent,
        headerHtml: selectedPreset.headerHtml,
        footerHtml: selectedPreset.footerHtml,
      });
      setOpen(false);
      setSelectedPreset(null);
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" className="size-full">
          <Sparkles className="size-4" />
          Start from Template
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Sparkles className="size-5 text-primary" />
            Choose a Template
          </DialogTitle>
          <DialogDescription>
            Select a pre-built template to get started quickly
          </DialogDescription>
        </DialogHeader>
        <ScrollArea className="h-[400px] pr-4">
          <div className="grid grid-cols-2 gap-3">
            {templatePresets.map((preset) => {
              const Icon = preset.icon;
              const isSelected = selectedPreset?.id === preset.id;
              return (
                <button
                  key={preset.id}
                  type="button"
                  onClick={() => setSelectedPreset(preset)}
                  className={cn(
                    "group relative flex flex-col items-start rounded-xl border-2 p-4 text-left transition-all",
                    isSelected
                      ? "border-primary bg-primary/5 shadow-md"
                      : "border-border hover:border-primary/50 hover:bg-muted/50",
                  )}
                >
                  <div
                    className={cn(
                      "mb-3 flex size-10 items-center justify-center rounded-lg bg-gradient-to-br text-white transition-transform group-hover:scale-110",
                      preset.gradient,
                    )}
                  >
                    <Icon className="size-5" />
                  </div>
                  <h4 className="font-semibold">{preset.name}</h4>
                  <p className="mt-1 text-sm text-muted-foreground">
                    {preset.description}
                  </p>
                  {isSelected && (
                    <div className="absolute top-3 right-3 size-5 rounded-full bg-primary text-white">
                      <svg
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth={3}
                        className="size-5"
                      >
                        <path d="M5 13l4 4L19 7" />
                      </svg>
                    </div>
                  )}
                </button>
              );
            })}
          </div>
        </ScrollArea>

        <div className="flex justify-end gap-2 border-t border-border pt-4">
          <Button variant="outline" onClick={() => setOpen(false)}>
            Cancel
          </Button>
          <Button onClick={handleSelect} disabled={!selectedPreset}>
            Use Template
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
