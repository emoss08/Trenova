import type { Invoice } from "@trenova/shared/types/invoice";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter, type OnUrlUpdateFunction } from "nuqs/adapters/testing";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import InvoiceDetailPane from "../invoice-detail-pane";

const mocks = vi.hoisted(() => ({ getById: vi.fn(), fetchInvoiceArContext: vi.fn() }));

vi.mock("@/hooks/use-post-invoice", () => ({
  usePostInvoice: () => ({ mutate: vi.fn(), isPending: false }),
}));
vi.mock("@/services/api", () => ({ apiService: { invoiceService: { getById: mocks.getById } } }));
vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }));
vi.mock("@/components/audit-tab", () => ({ default: () => <p>Audit trail</p> }));
vi.mock("../../../billing-queue/_components/billing-queue-documents-tab", () => ({
  BillingQueueDocumentsTab: () => null,
}));
vi.mock("../invoice-adjustment-panel", () => ({ InvoiceAdjustmentPanel: () => null }));
vi.mock("../invoice-overview-tab", () => ({ InvoiceOverviewTab: () => <p>Overview content</p> }));
vi.mock("../invoice-charges-tab", () => ({ InvoiceChargesTab: () => <p>Charges content</p> }));
vi.mock("../invoice-payments-tab", () => ({
  InvoicePaymentsTab: () => <p>Payments content</p>,
}));
vi.mock("../invoice-disputes-tab", () => ({
  InvoiceDisputesTab: () => <p>Disputes content</p>,
}));
vi.mock("../invoice-actions-menu", () => ({ InvoiceActionsMenu: () => null }));
vi.mock("@/lib/graphql/invoice", () => ({
  fetchInvoiceArContext: mocks.fetchInvoiceArContext,
  fetchInvoicesByShipment: vi.fn(),
}));

const invoice = {
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
  invoiceDate: 1_789_325_147,
  dueDate: null,
  postedAt: null,
  shipmentProNumber: "SEED-DET-011",
  shipmentBol: "BOL-2026-0211",
  serviceDate: null,
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
  createdAt: 1_789_325_147,
  updatedAt: 1_789_325_147,
  lines: [],
  attachments: [],
  emailAttempts: [],
} as Invoice;

function renderPane(searchParams: string, onUrlUpdate?: OnUrlUpdateFunction) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <NuqsTestingAdapter
          hasMemory
          resetUrlUpdateQueueOnMount={false}
          searchParams={searchParams}
          onUrlUpdate={onUrlUpdate}
        >
          <InvoiceDetailPane
            selectedInvoiceId="inv_1"
            selectedDocumentId={null}
            onDocumentSelect={vi.fn()}
          />
        </NuqsTestingAdapter>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

function lastSearchParams(onUrlUpdate: ReturnType<typeof vi.fn>) {
  return (onUrlUpdate.mock.calls.at(-1)?.[0] as { searchParams: URLSearchParams }).searchParams;
}

beforeEach(() => {
  mocks.getById.mockResolvedValue(invoice);
  mocks.fetchInvoiceArContext.mockResolvedValue({
    id: "inv_1",
    openBalance: "5000.00",
    daysPastDue: 12,
    paymentApplications: [],
    creditApplications: [],
    disputes: [],
    openDispute: null,
    lateChargeAssessments: [],
    relatedInvoices: [],
    referenceInvoice: null,
    ediSendPlan: {
      invoiceId: "inv_1",
      enabled: false,
      autoSend: false,
      partnerId: null,
      partnerName: "",
      documentProfileId: null,
      communicationMethod: "",
      status: "NotConfigured",
      lastMessageId: null,
      lastError: "",
      sentAt: null,
      blockers: [],
    },
  });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("invoice detail tabs in the URL", () => {
  it("opens the tab the URL names", async () => {
    renderPane("?item=inv_1&tab=charges");

    expect(await screen.findByRole("tab", { name: "Charges" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    expect(screen.getByText("Charges content")).toBeInTheDocument();
    expect(screen.queryByText("Overview content")).toBeNull();
  });

  it("opens the overview when the URL names no tab", async () => {
    renderPane("?item=inv_1");

    expect(await screen.findByRole("tab", { name: "Overview" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    expect(screen.getByText("Overview content")).toBeInTheDocument();
  });

  it("falls back to the overview for a tab that does not exist", async () => {
    renderPane("?item=inv_1&tab=nope");

    expect(await screen.findByRole("tab", { name: "Overview" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    expect(screen.getByText("Overview content")).toBeInTheDocument();
  });

  it("writes the picked tab to the URL and keeps the selected invoice", async () => {
    const onUrlUpdate = vi.fn();
    const user = userEvent.setup();
    renderPane("?item=inv_1", onUrlUpdate);

    await user.click(await screen.findByRole("tab", { name: "Activity" }));

    const params = lastSearchParams(onUrlUpdate);
    expect(params.get("tab")).toBe("activity");
    expect(params.get("item")).toBe("inv_1");
    expect(screen.getByRole("tab", { name: "Activity" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByText("Audit trail")).toBeInTheDocument();
  });

  it("opens the payments and disputes tabs the URL names", async () => {
    renderPane("?item=inv_1&tab=payments");
    expect(await screen.findByRole("tab", { name: "Payments" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    expect(screen.getByText("Payments content")).toBeInTheDocument();
    cleanup();

    renderPane("?item=inv_1&tab=disputes");
    expect(await screen.findByRole("tab", { name: "Disputes" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    expect(screen.getByText("Disputes content")).toBeInTheDocument();
  });

  it("shows how many days past due an open invoice is", async () => {
    renderPane("?item=inv_1");

    expect(await screen.findByText("Days past due")).toBeInTheDocument();
    expect(screen.getByText("12")).toBeInTheDocument();
  });

  it("hides posting and explains the void on a voided invoice", async () => {
    mocks.getById.mockResolvedValue({
      ...invoice,
      status: "Voided",
      voidedAt: 1_789_400_000,
      voidReason: "Billed the wrong customer",
      voidDisposition: "Rebill",
    });
    renderPane("?item=inv_1");

    expect(await screen.findByText("Billed the wrong customer")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Post Invoice" })).toBeNull();
    expect(screen.getByText("Released for rebilling")).toBeInTheDocument();
  });

  it("offers to share the open invoice from its header", async () => {
    renderPane("?item=inv_1&tab=charges");

    expect(await screen.findByRole("button", { name: "Share" })).toBeVisible();
  });

  it("drops the tab from the URL when returning to the overview", async () => {
    const onUrlUpdate = vi.fn();
    const user = userEvent.setup();
    renderPane("?item=inv_1&tab=charges", onUrlUpdate);

    await user.click(await screen.findByRole("tab", { name: "Overview" }));

    const params = lastSearchParams(onUrlUpdate);
    expect(params.get("tab")).toBeNull();
    expect(params.get("item")).toBe("inv_1");
  });
});
