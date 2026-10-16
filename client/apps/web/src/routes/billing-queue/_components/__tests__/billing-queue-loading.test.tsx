import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import type { ReactElement } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import BillingQueueDetailPane from "../billing-queue-detail-pane";
import BillingQueueDocumentPreview from "../billing-queue-document-preview";
import { StatementDetail } from "../statements/statement-detail";

const mocks = vi.hoisted(() => ({
  getById: vi.fn(),
  download: vi.fn(),
  getStatement: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiService: {
    billingQueueService: { getById: mocks.getById },
    invoiceRunService: { getStatement: mocks.getStatement },
  },
}));
vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { error: vi.fn(), success: vi.fn() },
}));
vi.mock("@trenova/shared/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/api")>();
  return { ...actual, api: { ...actual.api, download: mocks.download } };
});
vi.mock("@/routes/shipment/_components/comments", () => ({ default: () => null }));
vi.mock("@/components/audit-tab", () => ({ default: () => null }));

function renderWithin(ui: ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <NuqsTestingAdapter>{ui}</NuqsTestingAdapter>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

// A request that has not answered yet, so the component stays on its first-load state.
const pending = () => new Promise<never>(() => {});

beforeEach(() => {
  mocks.getById.mockImplementation(pending);
  mocks.download.mockImplementation(pending);
  mocks.getStatement.mockImplementation(pending);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("billing queue loading states", () => {
  // The page draws the same skeleton while the pane's code arrives, so the item
  // loading inside the pane must not swap it for a different shape.
  it("draws the review pane's outline while the chosen item loads", () => {
    renderWithin(
      <BillingQueueDetailPane
        selectedItemId="bqi_1"
        onDocumentSelect={vi.fn()}
        onAutoAdvance={vi.fn()}
      />,
    );

    expect(screen.getByLabelText("Loading the billing queue item")).toHaveAttribute(
      "aria-busy",
      "true",
    );
    expect(screen.queryByText("Nothing open")).not.toBeInTheDocument();
    expect(mocks.getById).toHaveBeenCalledWith("bqi_1", { expandShipmentDetails: "true" });
  });

  it("draws the statement's outline while the customer's statement loads", () => {
    renderWithin(<StatementDetail customerId="cus_1" nowSeconds={1_760_000_000} />);

    expect(screen.getByLabelText("Loading the statement")).toHaveAttribute("aria-busy", "true");
    expect(screen.queryByText("Pick a statement")).not.toBeInTheDocument();
    expect(mocks.getStatement).toHaveBeenCalledWith("cus_1");
  });

  it("asks for a statement to be picked when no customer is chosen", () => {
    renderWithin(<StatementDetail customerId={null} nowSeconds={1_760_000_000} />);

    expect(screen.getByText("Pick a statement")).toBeInTheDocument();
    expect(screen.queryByLabelText("Loading the statement")).not.toBeInTheDocument();
    expect(mocks.getStatement).not.toHaveBeenCalled();
  });

  it("draws the document viewer's outline while the document downloads", () => {
    renderWithin(<BillingQueueDocumentPreview documentId="doc_1" fileName="pod.pdf" />);

    expect(screen.getByLabelText("Loading the document")).toHaveAttribute("aria-busy", "true");
    expect(mocks.download).toHaveBeenCalledWith("/documents/doc_1/view/", "document.pdf", {
      signal: expect.any(AbortSignal),
    });
  });
});
