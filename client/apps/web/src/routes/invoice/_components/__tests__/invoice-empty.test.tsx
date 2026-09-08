import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter, type OnUrlUpdateFunction } from "nuqs/adapters/testing";
import type { ReactElement } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import InvoiceDetailPane from "../invoice-detail-pane";
import { InvoiceSidebar } from "../invoice-sidebar";

const mocks = vi.hoisted(() => ({ requestGraphQL: vi.fn() }));

vi.mock("@trenova/shared/lib/graphql", () => ({ requestGraphQL: mocks.requestGraphQL }));
vi.mock("@/hooks/use-post-invoice", () => ({
  usePostInvoice: () => ({ mutate: vi.fn(), isPending: false }),
}));
vi.mock("@/services/api", () => ({ apiService: { invoiceService: {} } }));
vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }));
vi.mock("@/components/audit-tab", () => ({ default: () => null }));
vi.mock("../../../billing-queue/_components/billing-queue-documents-tab", () => ({
  BillingQueueDocumentsTab: () => null,
}));
vi.mock("../invoice-adjustment-panel", () => ({ InvoiceAdjustmentPanel: () => null }));
vi.mock("../invoice-overview-tab", () => ({ InvoiceOverviewTab: () => null }));

function renderWithin(ui: ReactElement, searchParams = "", onUrlUpdate?: OnUrlUpdateFunction) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <NuqsTestingAdapter searchParams={searchParams} onUrlUpdate={onUrlUpdate}>
          {ui}
        </NuqsTestingAdapter>
      </QueryClientProvider>
    </MemoryRouter>,
  );
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

describe("invoice empty states", () => {
  it("says where invoices come from when there are none", async () => {
    renderWithin(<InvoiceSidebar selectedInvoiceId={null} onSelectInvoice={vi.fn()} />);

    expect(await screen.findByText("No invoices yet")).toBeInTheDocument();
    expect(screen.getByText(/billing queue/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Clear filters" })).not.toBeInTheDocument();
  });

  it("offers to clear the filters when they are what emptied the list", async () => {
    const onUrlUpdate = vi.fn();
    const user = userEvent.setup();
    renderWithin(
      <InvoiceSidebar selectedInvoiceId={null} onSelectInvoice={vi.fn()} />,
      "?status=Draft&query=INV-1",
      onUrlUpdate,
    );

    expect(await screen.findByText("Nothing matches")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Clear filters" }));

    const last = onUrlUpdate.mock.calls.at(-1)?.[0] as { searchParams: URLSearchParams };
    expect(last.searchParams.get("status")).toBeNull();
    expect(last.searchParams.get("query")).toBeNull();
  });

  it("draws the invoice it is waiting to show when nothing is open", () => {
    renderWithin(
      <InvoiceDetailPane
        selectedInvoiceId={null}
        selectedDocumentId={null}
        onDocumentSelect={vi.fn()}
      />,
    );

    expect(screen.getByText("Nothing open")).toBeInTheDocument();
    expect(screen.getByText(/Pick an invoice/)).toBeInTheDocument();
  });
});
