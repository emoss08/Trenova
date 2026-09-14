import type { Invoice } from "@trenova/shared/types/invoice";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import { periodRange } from "@/lib/billing-schedule";
import { InvoiceOverviewTab } from "../invoice-overview-tab";

vi.mock("../invoice-adjustment-runtime-section", () => ({
  InvoiceAdjustmentRuntimeSection: () => null,
}));

const PERIOD_START = 1_788_235_200;
const PERIOD_END = 1_789_325_147;
const SERVICE_DATE = 1_787_914_097;

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
    customerId: "cus_1",
    number: "INV-1",
    billType: "Invoice",
    status: "Draft",
    paymentTerm: "Net30",
    currencyCode: "USD",
    invoiceDate: PERIOD_END,
    dueDate: null,
    postedAt: null,
    shipmentProNumber: "SEED-DET-011",
    shipmentBol: "BOL-2026-0211",
    serviceDate: SERVICE_DATE,
    billToName: "GlobalTrade Imports",
    subtotalAmount: "5000",
    otherAmount: "0",
    totalAmount: "5000",
    appliedAmount: "0",
    settlementStatus: "Unpaid",
    disputeStatus: "None",
    sendStatus: "NotSent",
    isAdjustmentArtifact: false,
    version: 1,
    createdAt: PERIOD_END,
    updatedAt: PERIOD_END,
    lines: [],
    attachments: [],
    emailAttempts: [],
    ...overrides,
  } as Invoice;
}

function renderOverview(value: Invoice) {
  render(
    <MemoryRouter>
      <InvoiceOverviewTab
        invoice={value}
        isCurrentVersion
        correctionSummary={undefined}
        latestAdjustment={null}
        latestAdjustmentDetail={undefined}
      />
    </MemoryRouter>,
  );
}

describe("invoice overview scope", () => {
  it("states a consolidated invoice's period and shipments instead of one leg's references", () => {
    renderOverview(
      invoice({
        id: "inv_2",
        number: "INV-2",
        scope: "Consolidated",
        shipmentId: null,
        invoiceRunId: "invrun_1",
        periodStart: PERIOD_START,
        periodEnd: PERIOD_END,
        shipmentCount: 12,
      }),
    );

    expect(screen.getByText("Billing Period")).toBeInTheDocument();
    expect(screen.getByText(periodRange(PERIOD_START, PERIOD_END))).toBeInTheDocument();
    expect(screen.queryByText("Service Date")).toBeNull();
    expect(screen.getByText("12 shipments")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /Queue Item/ })).toBeNull();
    expect(screen.queryByText("BOL-2026-0211")).toBeNull();
    expect(screen.getByText("Generated from Statement")).toBeInTheDocument();
  });

  it("does not point an order invoice at its anchor queue item", () => {
    renderOverview(
      invoice({
        scope: "Order",
        shipmentId: null,
        orderId: "ord_1",
        orderNumber: "ORD-1",
        shipmentCount: 3,
      }),
    );

    expect(screen.getByRole("link", { name: /ORD-1/ })).toBeInTheDocument();
    expect(screen.getByText("3 shipments")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /Queue Item/ })).toBeNull();
    expect(screen.getByText("Service Date")).toBeInTheDocument();
    expect(screen.getByText("Generated from Billing Queue")).toBeInTheDocument();
  });

  it("keeps a single-shipment invoice's shipment, queue item, BOL and service date", () => {
    renderOverview(invoice());

    expect(screen.getByRole("link", { name: /SEED-DET-011/ })).toHaveAttribute(
      "href",
      "/shipment-management/shipments?expanded=shp_1&panelType=edit&panelEntityId=shp_1",
    );
    expect(screen.getByRole("link", { name: /Queue Item/ })).toBeInTheDocument();
    expect(screen.getByText("BOL-2026-0211")).toBeInTheDocument();
    expect(screen.getByText("Service Date")).toBeInTheDocument();
    expect(screen.queryByText("Billing Period")).toBeNull();
  });
});
