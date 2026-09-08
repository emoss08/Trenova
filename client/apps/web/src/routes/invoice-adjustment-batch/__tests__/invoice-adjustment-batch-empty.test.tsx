import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter, type OnUrlUpdateFunction } from "nuqs/adapters/testing";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { InvoiceAdjustmentBatchPage } from "../page";

const mocks = vi.hoisted(() => ({
  listBatches: vi.fn(),
  getSummary: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiService: {
    invoiceAdjustmentService: {
      listBatches: mocks.listBatches,
      getSummary: mocks.getSummary,
      getBatch: vi.fn(),
    },
  },
}));
vi.mock("@/components/billing/billing-workspace-layout", () => ({
  BillingWorkspaceLayout: ({
    toolbar,
    sidebar,
    detail,
  }: {
    toolbar?: ReactNode;
    sidebar: ReactNode;
    detail: ReactNode;
  }) => (
    <div>
      {toolbar}
      <aside>{sidebar}</aside>
      <main>{detail}</main>
    </div>
  ),
}));

function renderPage(searchParams = "", onUrlUpdate?: OnUrlUpdateFunction) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <NuqsTestingAdapter searchParams={searchParams} onUrlUpdate={onUrlUpdate}>
          <InvoiceAdjustmentBatchPage />
        </NuqsTestingAdapter>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  mocks.listBatches.mockResolvedValue({ results: [], count: 0 });
  mocks.getSummary.mockResolvedValue({
    approvalsPending: 0,
    reconciliationPending: 0,
    writeOffPending: 0,
    failedBatchItems: 0,
    batchesInFlight: 0,
  });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("batch monitor empty states", () => {
  it("says what a batch is when there are none, and draws the review pane", async () => {
    renderPage();

    expect(await screen.findByText("No batches yet")).toBeInTheDocument();
    expect(screen.getByText(/several invoices/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Clear filters" })).not.toBeInTheDocument();
    expect(screen.getByText("Nothing open")).toBeInTheDocument();
  });

  it("offers to clear the filters when they are what emptied the list", async () => {
    const onUrlUpdate = vi.fn();
    const user = userEvent.setup();
    renderPage("?status=Failed&query=batch-1", onUrlUpdate);

    expect(await screen.findByText("Nothing matches")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Clear filters" }));

    const last = onUrlUpdate.mock.calls.at(-1)?.[0] as { searchParams: URLSearchParams };
    expect(last.searchParams.get("status")).toBeNull();
    expect(last.searchParams.get("query")).toBeNull();
  });
});
