import type { InvoiceTableRowFieldsFragment } from "@trenova/graphql/generated/graphql";
import type { Invoice } from "@trenova/shared/types/invoice";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter, type OnUrlUpdateFunction } from "nuqs/adapters/testing";
import type { ReactElement } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { periodRange } from "@/lib/billing-schedule";
import InvoiceDetailPane from "../invoice-detail-pane";
import { InvoiceItemCard } from "../invoice-item-card";
import { InvoiceSidebar } from "../invoice-sidebar";

const mocks = vi.hoisted(() => ({ requestGraphQL: vi.fn(), getById: vi.fn() }));

vi.mock("@trenova/shared/lib/graphql", () => ({ requestGraphQL: mocks.requestGraphQL }));
vi.mock("@/hooks/use-post-invoice", () => ({
  usePostInvoice: () => ({ mutate: vi.fn(), isPending: false }),
}));
vi.mock("@/services/api", () => ({ apiService: { invoiceService: { getById: mocks.getById } } }));
vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }));
vi.mock("@/components/audit-tab", () => ({ default: () => null }));
vi.mock("../../../billing-queue/_components/billing-queue-documents-tab", () => ({
  BillingQueueDocumentsTab: () => null,
}));
vi.mock("../invoice-adjustment-panel", () => ({ InvoiceAdjustmentPanel: () => null }));
vi.mock("../invoice-overview-tab", () => ({ InvoiceOverviewTab: () => null }));

const PERIOD_START = 1_788_235_200;
const PERIOD_END = 1_789_325_147;

function renderWithin(ui: ReactElement, searchParams = "", onUrlUpdate?: OnUrlUpdateFunction) {
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
          {ui}
        </NuqsTestingAdapter>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

function row(
  overrides: Partial<InvoiceTableRowFieldsFragment> = {},
): InvoiceTableRowFieldsFragment {
  return {
    id: "inv_1",
    billingQueueItemId: "bqi_1",
    shipmentId: "shp_1",
    orderId: null,
    customerId: "cus_1",
    number: "INV-1",
    billType: "Invoice",
    scope: "Shipment",
    periodStart: null,
    periodEnd: null,
    shipmentCount: 1,
    status: "Draft",
    paymentTerm: "Net30",
    currencyCode: "USD",
    invoiceDate: PERIOD_END,
    dueDate: null,
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
    customer: { id: "cus_1", name: "GlobalTrade Imports", code: "GLBL" },
    ...overrides,
  };
}

const consolidatedRow = row({
  id: "inv_2",
  number: "INV-2",
  shipmentId: null,
  scope: "Consolidated",
  periodStart: PERIOD_START,
  periodEnd: PERIOD_END,
  shipmentCount: 12,
});

async function openContextMenu(invoiceNumber: string) {
  fireEvent.contextMenu(screen.getByRole("button", { name: new RegExp(invoiceNumber) }));
  return screen.findByRole("menu");
}

beforeEach(() => {
  vi.stubGlobal(
    "IntersectionObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    },
  );
  mocks.requestGraphQL.mockResolvedValue({
    invoices: {
      edges: [],
      totalCount: 0,
      pageInfo: { hasNextPage: false, endCursor: null },
    },
  });
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  vi.clearAllMocks();
});

describe("invoice list card scope", () => {
  it("marks a consolidated invoice and states the period and shipments it covers", () => {
    renderWithin(
      <InvoiceItemCard
        invoice={consolidatedRow}
        isSelected={false}
        onClick={vi.fn()}
        onPost={vi.fn()}
      />,
    );

    expect(screen.getByText("Consolidated")).toBeInTheDocument();
    expect(
      screen.getByText(`${periodRange(PERIOD_START, PERIOD_END)} · 12 shipments`),
    ).toBeInTheDocument();
  });

  it("draws no scope badge or shipment count on a single-shipment invoice", () => {
    renderWithin(
      <InvoiceItemCard invoice={row()} isSelected={false} onClick={vi.fn()} onPost={vi.fn()} />,
    );

    for (const label of ["Consolidated", "Order", "Adjustment"]) {
      expect(screen.queryByText(label)).not.toBeInTheDocument();
    }
    expect(screen.queryByText(/shipments?$/)).not.toBeInTheDocument();
  });

  it("counts the shipments on an order invoice without inventing a period", () => {
    renderWithin(
      <InvoiceItemCard
        invoice={row({
          scope: "Order",
          shipmentId: null,
          orderId: "ord_1",
          shipmentCount: 1,
          periodStart: PERIOD_START,
          periodEnd: PERIOD_END,
        })}
        isSelected={false}
        onClick={vi.fn()}
        onPost={vi.fn()}
      />,
    );

    expect(screen.getByText("Order")).toBeInTheDocument();
    expect(screen.getByText("1 shipment")).toBeInTheDocument();
    expect(screen.queryByText(new RegExp(periodRange(PERIOD_START, PERIOD_END)))).toBeNull();
  });

  it("does not link a consolidated invoice to its anchor shipment or queue item", async () => {
    renderWithin(
      <InvoiceItemCard
        invoice={consolidatedRow}
        isSelected={false}
        onClick={vi.fn()}
        onPost={vi.fn()}
      />,
    );

    const menu = await openContextMenu("INV-2");
    expect(within(menu).queryByRole("menuitem", { name: "View shipment" })).toBeNull();
    expect(within(menu).queryByRole("menuitem", { name: "View billing queue item" })).toBeNull();
    expect(within(menu).getByRole("menuitem", { name: "Post invoice" })).toBeInTheDocument();
  });

  it("links an order invoice to its order rather than to one leg", async () => {
    renderWithin(
      <InvoiceItemCard
        invoice={row({ scope: "Order", shipmentId: null, orderId: "ord_1", shipmentCount: 3 })}
        isSelected={false}
        onClick={vi.fn()}
        onPost={vi.fn()}
      />,
    );

    const menu = await openContextMenu("INV-1");
    expect(within(menu).getByRole("menuitem", { name: "View order" })).toBeInTheDocument();
    expect(within(menu).queryByRole("menuitem", { name: "View shipment" })).toBeNull();
    expect(within(menu).queryByRole("menuitem", { name: "View billing queue item" })).toBeNull();
  });

  it("keeps the shipment and queue item links on a single-shipment invoice", async () => {
    renderWithin(
      <InvoiceItemCard invoice={row()} isSelected={false} onClick={vi.fn()} onPost={vi.fn()} />,
    );

    const menu = await openContextMenu("INV-1");
    expect(within(menu).getByRole("menuitem", { name: "View shipment" })).toBeInTheDocument();
    expect(
      within(menu).getByRole("menuitem", { name: "View billing queue item" }),
    ).toBeInTheDocument();
    expect(within(menu).queryByRole("menuitem", { name: "View order" })).toBeNull();
  });

  it("opens the shipment in its edit panel from the context menu", async () => {
    const open = vi.spyOn(window, "open").mockImplementation(() => null);
    const user = userEvent.setup();
    renderWithin(
      <InvoiceItemCard invoice={row()} isSelected={false} onClick={vi.fn()} onPost={vi.fn()} />,
    );

    const menu = await openContextMenu("INV-1");
    await user.click(within(menu).getByRole("menuitem", { name: "View shipment" }));

    expect(open).toHaveBeenCalledWith(
      "/shipment-management/shipments?expanded=shp_1&panelType=edit&panelEntityId=shp_1",
      "_blank",
    );
    open.mockRestore();
  });
});

describe("invoice sidebar scope filter", () => {
  function lastFieldFilters() {
    const call = mocks.requestGraphQL.mock.calls.at(-1)?.[0] as {
      variables: { input: { fieldFilters: Array<{ field: string; value: string }> } };
    };
    return call.variables.input.fieldFilters;
  }

  it("filters the list to the scope picked", async () => {
    const onUrlUpdate = vi.fn();
    const user = userEvent.setup();
    renderWithin(
      <InvoiceSidebar selectedInvoiceId={null} onSelectInvoice={vi.fn()} />,
      "",
      onUrlUpdate,
    );

    await user.click(screen.getByRole("combobox", { name: "Invoice scope" }));
    await user.click(await screen.findByRole("option", { name: "Consolidated" }));

    const last = onUrlUpdate.mock.calls.at(-1)?.[0] as { searchParams: URLSearchParams };
    expect(last.searchParams.get("scope")).toBe("Consolidated");
    await waitFor(() =>
      expect(lastFieldFilters()).toContainEqual({
        field: "scope",
        operator: "eq",
        value: "Consolidated",
      }),
    );
  });

  it("sends no scope filter until one is picked", async () => {
    renderWithin(<InvoiceSidebar selectedInvoiceId={null} onSelectInvoice={vi.fn()} />);

    await waitFor(() => expect(mocks.requestGraphQL).toHaveBeenCalled());
    expect(lastFieldFilters().some((filter) => filter.field === "scope")).toBe(false);
    expect(screen.getByRole("combobox", { name: "Invoice scope" })).toHaveTextContent("All scopes");
  });

  it("clears the scope along with the other filters", async () => {
    const onUrlUpdate = vi.fn();
    const user = userEvent.setup();
    renderWithin(
      <InvoiceSidebar selectedInvoiceId={null} onSelectInvoice={vi.fn()} />,
      "?scope=Consolidated",
      onUrlUpdate,
    );

    expect(screen.getByRole("combobox", { name: "Invoice scope" })).toHaveTextContent(
      "Consolidated",
    );
    expect(await screen.findByText("Nothing matches")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Clear filters" }));

    const last = onUrlUpdate.mock.calls.at(-1)?.[0] as { searchParams: URLSearchParams };
    expect(last.searchParams.get("scope")).toBeNull();
  });
});

describe("invoice detail pane scope", () => {
  function invoice(overrides: Partial<Invoice> = {}): Invoice {
    return {
      id: "inv_2",
      organizationId: "org_1",
      businessUnitId: "bu_1",
      billingQueueItemId: "bqi_1",
      shipmentId: null,
      scope: "Consolidated",
      invoiceRunId: "invrun_1",
      periodStart: PERIOD_START,
      periodEnd: PERIOD_END,
      shipmentCount: 12,
      detail: "Detailed",
      sectionBy: "Shipment",
      offCycleReason: null,
      orderId: null,
      customerId: "cus_1",
      number: "INV-2",
      billType: "Invoice",
      status: "Draft",
      paymentTerm: "Net30",
      currencyCode: "USD",
      invoiceDate: PERIOD_END,
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
      createdAt: PERIOD_END,
      updatedAt: PERIOD_END,
      lines: [],
      attachments: [],
      emailAttempts: [],
      ...overrides,
    } as Invoice;
  }

  it("names a consolidated invoice, its period, and every shipment it bills", async () => {
    mocks.getById.mockResolvedValue(invoice());
    renderWithin(
      <InvoiceDetailPane
        selectedInvoiceId="inv_2"
        selectedDocumentId={null}
        onDocumentSelect={vi.fn()}
      />,
    );

    expect(await screen.findByRole("heading", { name: "INV-2" })).toBeInTheDocument();
    expect(screen.getByText("Consolidated")).toBeInTheDocument();
    expect(screen.getByText("Billing period")).toBeInTheDocument();
    expect(screen.getByText(periodRange(PERIOD_START, PERIOD_END))).toBeInTheDocument();
    expect(screen.getByText("Shipments")).toBeInTheDocument();
    expect(screen.getByText("12")).toBeInTheDocument();
    expect(screen.queryByText("PRO number")).toBeNull();
    expect(screen.queryByText("SEED-DET-011")).toBeNull();
  });

  it("keeps the PRO and BOL of a single-shipment invoice and states no period", async () => {
    mocks.getById.mockResolvedValue(
      invoice({
        id: "inv_1",
        number: "INV-1",
        scope: "Shipment",
        shipmentId: "shp_1",
        invoiceRunId: null,
        periodStart: null,
        periodEnd: null,
        shipmentCount: 1,
      }),
    );
    renderWithin(
      <InvoiceDetailPane
        selectedInvoiceId="inv_1"
        selectedDocumentId={null}
        onDocumentSelect={vi.fn()}
      />,
    );

    expect(await screen.findByRole("heading", { name: "INV-1" })).toBeInTheDocument();
    expect(screen.getByText("SEED-DET-011")).toBeInTheDocument();
    expect(screen.queryByText("Billing period")).toBeNull();
    expect(screen.queryByText("Consolidated")).toBeNull();
  });
});
