import type { Invoice } from "@trenova/shared/types/invoice";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { InvoiceOverviewTab } from "../invoice-overview-tab";

vi.mock("../invoice-adjustment-runtime-section", () => ({
  InvoiceAdjustmentRuntimeSection: () => null,
}));
vi.mock("../invoice-ar-context-section", () => ({ InvoiceArContextSection: () => null }));

afterEach(cleanup);

const NOW = 1_789_325_147;

function invoice(overrides: Partial<Invoice> = {}): Invoice {
  return {
    id: "inv_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    billingQueueItemId: "bqi_1",
    shipmentId: "shp_1",
    scope: "Shipment",
    shipmentCount: 1,
    detail: "Detailed",
    sectionBy: "Shipment",
    customerId: "cus_1",
    number: "INV-1",
    billType: "Invoice",
    status: "Posted",
    paymentTerm: "Net30",
    currencyCode: "USD",
    invoiceDate: NOW,
    postedAt: NOW,
    billToName: "AMD",
    subtotalAmount: "100",
    otherAmount: "0",
    totalAmount: "100",
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

function renderOverview(value: Invoice) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <InvoiceOverviewTab
          invoice={value}
          isCurrentVersion
          correctionSummary={undefined}
          latestAdjustment={null}
          latestAdjustmentDetail={undefined}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("invoice overview for voided invoices and memos", () => {
  it("ends the lifecycle at the void, with its reason and disposition", () => {
    renderOverview(
      invoice({
        status: "Voided",
        voidedAt: NOW + 3600,
        voidReason: "Customer never received the freight",
        voidDisposition: "DoNotRebill",
      }),
    );

    expect(screen.getByText("Voided")).toBeInTheDocument();
    expect(screen.getByText("Customer never received the freight")).toBeInTheDocument();
    expect(screen.getByText("Freight retired, not rebilled")).toBeInTheDocument();
  });

  it("does not draw a void step on an invoice that stands", () => {
    renderOverview(invoice());

    expect(screen.queryByText("Voided")).toBeNull();
  });

  it("explains a standalone memo: what it references and why it was raised", () => {
    renderOverview(
      invoice({
        scope: "Memo",
        billType: "CreditMemo",
        shipmentId: null,
        billingQueueItemId: "bqi_memo",
        referenceInvoiceId: "inv_9",
        memoReason: "Goodwill credit for the late delivery",
        memoKind: "Manual",
        totalAmount: "-50",
      }),
    );

    expect(screen.getByText("Goodwill credit for the late delivery")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Referenced invoice/ })).toHaveAttribute(
      "href",
      "/billing/invoices?item=inv_9",
    );
    expect(screen.getByText("Generated as a memo")).toBeInTheDocument();
  });

  it("names a late-charge memo as raised by the nightly run", () => {
    renderOverview(
      invoice({
        scope: "Memo",
        billType: "DebitMemo",
        shipmentId: null,
        memoReason: "Late charges assessed as of 2026-09-01",
        memoKind: "LateCharge",
      }),
    );

    expect(screen.getByText("Late charge")).toBeInTheDocument();
  });
});
