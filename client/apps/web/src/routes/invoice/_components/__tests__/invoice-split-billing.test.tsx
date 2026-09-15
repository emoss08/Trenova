import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import type { Invoice, InvoiceLine } from "@trenova/shared/types/invoice";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { InvoiceChargesTab } from "../invoice-charges-tab";
import { InvoiceOverviewTab } from "../invoice-overview-tab";

const mocks = vi.hoisted(() => ({
  fetchInvoiceArContext: vi.fn(),
  fetchInvoicesByShipment: vi.fn(),
}));

vi.mock("@/lib/graphql/invoice", () => ({
  fetchInvoiceArContext: mocks.fetchInvoiceArContext,
  fetchInvoicesByShipment: mocks.fetchInvoicesByShipment,
}));
vi.mock("../invoice-adjustment-runtime-section", () => ({
  InvoiceAdjustmentRuntimeSection: () => null,
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

const NOW = 1_789_325_147;

function invoice(overrides: Partial<Invoice> = {}): Invoice {
  return {
    id: "inv_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    billingQueueItemId: "bqi_1",
    shipmentId: "shp_1",
    scope: "Shipment",
    invoiceRunId: null,
    periodStart: null,
    periodEnd: null,
    shipmentCount: 1,
    detail: "Detailed",
    sectionBy: "Shipment",
    offCycleReason: null,
    orderId: null,
    customerId: "cus_amd",
    shipperCustomerId: "cus_intel",
    isSplitBill: true,
    number: "INV-1",
    billType: "Invoice",
    status: "Draft",
    paymentTerm: "Net30",
    currencyCode: "USD",
    invoiceDate: NOW,
    dueDate: null,
    postedAt: null,
    shipmentProNumber: "PRO-1",
    shipmentBol: "BOL-1",
    serviceDate: NOW,
    billToName: "AMD",
    subtotalAmount: "0",
    otherAmount: "400",
    totalAmount: "400",
    appliedAmount: "0",
    settlementStatus: "Unpaid",
    disputeStatus: "None",
    sendStatus: "NotSent",
    isAdjustmentArtifact: false,
    version: 1,
    createdAt: NOW,
    updatedAt: NOW,
    lines: [],
    attachments: [],
    emailAttempts: [],
    ...overrides,
  } as Invoice;
}

function line(overrides: Partial<InvoiceLine> & Pick<InvoiceLine, "id" | "lineNumber">) {
  return {
    organizationId: "org_1",
    businessUnitId: "bu_1",
    invoiceId: "inv_1",
    shipmentId: null,
    shipmentProNumber: null,
    shipmentBol: null,
    type: "Accessorial",
    description: "Detention",
    quantity: 1,
    unitPrice: 100,
    amount: 100,
    accessorialChargeId: null,
    chargeCode: null,
    chargeMethod: null,
    rateUnit: null,
    rate: null,
    rateBasisAmount: null,
    formulaTemplateName: null,
    allocationPercent: null,
    chargeAllocationId: null,
    ...overrides,
  } as InvoiceLine;
}

function renderWithin(ui: React.ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>{ui}</MemoryRouter>
    </QueryClientProvider>,
  );
}

function renderOverview(value: Invoice) {
  return renderWithin(
    <InvoiceOverviewTab
      invoice={value}
      isCurrentVersion
      correctionSummary={undefined}
      latestAdjustment={null}
      latestAdjustmentDetail={undefined}
    />,
  );
}

beforeEach(() => {
  mocks.fetchInvoiceArContext.mockResolvedValue({
    id: "inv_1",
    customerId: "cus_amd",
    shipperCustomerId: "cus_intel",
    isSplitBill: true,
    billToCode: "AMD",
    billToAddressLine1: null,
    billToAddressLine2: null,
    billToCity: null,
    billToState: null,
    billToPostalCode: null,
    billToCountry: null,
    shipperCustomer: { id: "cus_intel", name: "Intel Corporation", code: "INTEL" },
    relatedInvoices: [
      {
        id: "inv_2",
        number: "INV-2",
        billType: "Invoice",
        status: "Draft",
        scope: "Shipment",
        currencyCode: "USD",
        totalAmount: "1000.00",
        settlementStatus: "Unpaid",
        billToName: "Intel",
        customerId: "cus_intel",
        isSplitBill: true,
      },
    ],
  });
});

describe("invoice overview for a split bill", () => {
  // The bill-to is who pays; the shipper is whose freight it was. A payer's
  // AR clerk needs the second name to recognise the invoice at all.
  it("says whose shipment the payer is being billed for", async () => {
    renderOverview(invoice());

    expect(await screen.findByText("Intel Corporation")).toBeInTheDocument();
    expect(screen.getByText(/on behalf of/i)).toBeInTheDocument();
    expect(mocks.fetchInvoiceArContext).toHaveBeenCalledWith("inv_1", expect.anything());
  });

  it("lists the sibling invoices that bill the rest of the shipment", async () => {
    renderOverview(invoice());

    const link = await screen.findByRole("link", { name: /INV-2/ });
    expect(link).toHaveAttribute("href", "/billing/invoices?item=inv_2");
    expect(screen.getByText("$1,000.00")).toBeInTheDocument();
  });

  it("asks nothing of the server for an ordinary single-payer invoice", () => {
    renderOverview(invoice({ shipperCustomerId: "cus_amd", isSplitBill: false }));

    expect(screen.queryByText(/on behalf of/i)).not.toBeInTheDocument();
    expect(mocks.fetchInvoiceArContext).not.toHaveBeenCalled();
  });
});

describe("invoice charges for a split bill", () => {
  it("shows the payer's share on a partially allocated line", () => {
    renderWithin(
      <InvoiceChargesTab
        invoice={invoice({
          lines: [line({ id: "line_1", lineNumber: 1, amount: 40, allocationPercent: 40 })],
        })}
      />,
    );

    expect(screen.getByText(/40% share/)).toBeInTheDocument();
  });

  it("says nothing about a share on a fully billed line", () => {
    renderWithin(
      <InvoiceChargesTab
        invoice={invoice({
          lines: [line({ id: "line_1", lineNumber: 1, amount: 100, allocationPercent: 100 })],
        })}
      />,
    );

    expect(screen.queryByText(/share/)).not.toBeInTheDocument();
  });
});
