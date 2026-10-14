import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import InvoiceDetailPane from "../invoice-detail-pane";

const mocks = vi.hoisted(() => ({ getById: vi.fn(), requestGraphQL: vi.fn() }));

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

function renderPane(selectedInvoiceId: string | null) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <NuqsTestingAdapter>
          <InvoiceDetailPane
            selectedInvoiceId={selectedInvoiceId}
            selectedDocumentId={null}
            onDocumentSelect={vi.fn()}
          />
        </NuqsTestingAdapter>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  mocks.getById.mockImplementation(() => new Promise<never>(() => {}));
  mocks.requestGraphQL.mockImplementation(() => new Promise<never>(() => {}));
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("invoice detail loading state", () => {
  // The page draws the same skeleton while the pane's code arrives, so the
  // invoice loading inside the pane must not swap it for a different shape.
  it("draws the invoice's outline while the chosen invoice loads", () => {
    renderPane("inv_1");

    expect(screen.getByLabelText("Loading the invoice")).toHaveAttribute("aria-busy", "true");
    expect(screen.queryByText("Nothing open")).not.toBeInTheDocument();
    expect(mocks.getById).toHaveBeenCalledWith("inv_1");
  });

  it("asks for an invoice to be picked rather than loading one when none is chosen", () => {
    renderPane(null);

    expect(screen.getByText("Nothing open")).toBeInTheDocument();
    expect(screen.queryByLabelText("Loading the invoice")).not.toBeInTheDocument();
    expect(mocks.getById).not.toHaveBeenCalled();
  });
});
