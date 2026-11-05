import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { MemoryRouter } from "react-router";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { afterEach, describe, expect, it, vi } from "vitest";
import BillingQueueDetailPane from "../billing-queue-detail-pane";
import { queueItem } from "./split-shipment-fixture";

const { getItem } = vi.hoisted(() => ({ getItem: vi.fn() }));

vi.mock("@/lib/queries/billing-queue", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/queries/billing-queue")>();
  return {
    ...actual,
    billingQueue: {
      ...actual.billingQueue,
      get: (id: string) => ({ queryKey: ["billingQueue", "get", id], queryFn: () => getItem(id) }),
    },
  };
});
vi.mock("@/services/billing-queue", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/services/billing-queue")>()),
  billingQueueService: { updateCharges: vi.fn(), reassignCharge: vi.fn() },
}));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));
vi.mock("../billing-queue-action-bar", () => ({ BillingQueueActionBar: () => null }));
vi.mock("../billing-queue-documents-tab", () => ({ BillingQueueDocumentsTab: () => null }));
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
    const lines = screen.getByRole("table", { name: "Charge lines" });
    const outsideLines = (pattern: RegExp) =>
      screen.getAllByText(pattern).filter((element) => !lines.contains(element));
    expect(outsideLines(/Peak Distributing/).map((element) => element.textContent)).toEqual([
      "Bills Peak Distributing on behalf of Acme Manufacturing",
    ]);
    expect(within(lines).getByTestId("payer-line-freight")).toHaveTextContent("Peak Distributing");
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

describe("billing queue review pane links", () => {
  afterEach(clearPermissions);

  it("offers the shipment's page to someone who can read shipments", async () => {
    seedPermissions({ [Resource.Shipment]: Operation.Read });
    getItem.mockResolvedValue(queueItem({ payer: "peak" }));
    renderPane();

    expect(await screen.findByRole("link", { name: "Open shipment" })).toBeInTheDocument();
  });

  it("keeps the shipment's page from someone who cannot read shipments", async () => {
    seedPermissions({});
    getItem.mockResolvedValue(queueItem({ payer: "peak" }));
    renderPane();

    expect(await screen.findByTestId("payer-bill-total")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Open shipment" })).not.toBeInTheDocument();
  });
});

describe("billing queue review pane entrance", () => {
  it("rises in again for a different item, and stays still while the same item refreshes", async () => {
    getItem.mockImplementation(async (id: string) =>
      id === "bqi_peak" ? queueItem({ payer: "peak" }) : queueItem({ payer: "acme" }),
    );
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const view = (selectedItemId: string) => (
      <MemoryRouter>
        <QueryClientProvider client={client}>
          <NuqsTestingAdapter>
            <BillingQueueDetailPane selectedItemId={selectedItemId} onDocumentSelect={vi.fn()} />
          </NuqsTestingAdapter>
        </QueryClientProvider>
      </MemoryRouter>
    );
    const { rerender } = render(view("bqi_peak"));

    const first = await screen.findByTestId("billing-queue-detail-enter");
    expect(first).toHaveClass("animate-reveal");

    await client.invalidateQueries();
    expect(await screen.findByTestId("billing-queue-detail-enter")).toBe(first);

    rerender(view("bqi_acme"));
    await waitFor(() => expect(screen.getByTestId("billing-queue-detail-enter")).not.toBe(first));
    expect(screen.getByTestId("billing-queue-detail-enter")).toHaveClass("animate-reveal");
  });
});
