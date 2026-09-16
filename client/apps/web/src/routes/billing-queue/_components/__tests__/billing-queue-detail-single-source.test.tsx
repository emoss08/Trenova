import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import BillingQueueDetailPane from "../billing-queue-detail-pane";
import { queueItem } from "./split-shipment-fixture";

const { getItem } = vi.hoisted(() => ({ getItem: vi.fn() }));

vi.mock("@/lib/queries", () => ({
  queries: {
    billingQueue: {
      get: (id: string) => ({ queryKey: ["billingQueue", "get", id], queryFn: () => getItem(id) }),
    },
  },
}));
vi.mock("@/services/api", () => ({
  apiService: { billingQueueService: { updateCharges: vi.fn(), reassignCharge: vi.fn() } },
}));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));
vi.mock("../billing-queue-action-bar", () => ({ BillingQueueActionBar: () => null }));
vi.mock("../billing-queue-documents-tab", () => ({ BillingQueueDocumentsTab: () => null }));
vi.mock("../billing-queue-charge-dialog", () => ({ BillingQueueChargeDialog: () => null }));
vi.mock("../billing-queue-rerate-dialog", () => ({ BillingQueueRerateDialog: () => null }));
vi.mock("../billing-queue-reassign-charge-dialog", () => ({
  BillingQueueReassignChargeDialog: () => null,
}));
vi.mock("@/routes/shipment/_components/comments", () => ({ default: () => null }));
vi.mock("@/components/audit-tab", () => ({ default: () => null }));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function renderPane() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <NuqsTestingAdapter>
          <BillingQueueDetailPane selectedItemId="bqi_peak" onDocumentSelect={vi.fn()} />
        </NuqsTestingAdapter>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

// The review pane is one page: the header and the open Charges tab are read
// together, so each fact about the split belongs in exactly one place. The
// header owns who pays and what share of the shipment that is; each charge line
// owns its own share of that charge.
describe("billing queue review pane shows each split fact once", () => {
  it("states the payer, the share of the shipment and each charge's share in one place", async () => {
    getItem.mockResolvedValue(queueItem({ payer: "peak" }));
    renderPane();

    expect(await screen.findByTestId("payer-bill-total")).toHaveTextContent("$1,500.00");

    expect(screen.getAllByText(/of \$4,004\.38 shipment total/)).toHaveLength(1);
    expect(screen.getAllByText(/Peak Distributing/)).toHaveLength(1);
    expect(screen.getAllByText(/Acme Manufacturing/)).toHaveLength(1);
    expect(screen.getAllByText(/of \$2,850\.00/)).toHaveLength(1);
    expect(screen.queryByText(/Amount split/)).not.toBeInTheDocument();
  });

  it("still tells the reviewer that charges are shared, without repeating the payers", async () => {
    getItem.mockResolvedValue(queueItem({ payer: "peak" }));
    renderPane();

    expect(
      await screen.findByText(
        "Charges belong to the shipment, so changing one here also changes the other payers' bills.",
      ),
    ).toBeInTheDocument();
  });
});
