import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { setLocale } from "@trenova/shared/i18n/runtime";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import type {
  BillingTransferCandidate,
  BillingTransferCandidateConnection,
  BillingTransferRun,
  BillingTransferRunItem,
  BillingTransferRunItemConnection,
} from "@/lib/graphql/billing-transfer";
import { BulkBillingTransferDialog } from "../bulk-billing-transfer-dialog";

const mocks = vi.hoisted(() => ({
  candidates: vi.fn(),
  startRun: vi.fn(),
  getRun: vi.fn(),
  activeRun: vi.fn(),
  cancelRun: vi.fn(),
  retryRun: vi.fn(),
  listItems: vi.fn(),
}));

vi.mock("@/lib/graphql/billing-transfer", () => ({
  listBillingTransferCandidatesGraphQL: mocks.candidates,
  startBillingTransferRunGraphQL: mocks.startRun,
  getBillingTransferRunGraphQL: mocks.getRun,
  getMyActiveBillingTransferRunGraphQL: mocks.activeRun,
  cancelBillingTransferRunGraphQL: mocks.cancelRun,
  retryBillingTransferRunGraphQL: mocks.retryRun,
  listBillingTransferRunItemsGraphQL: mocks.listItems,
}));

function candidate(
  id: string,
  overrides: Partial<BillingTransferCandidate> = {},
): BillingTransferCandidate {
  return {
    id,
    proNumber: `PRO-${id}`,
    bol: null,
    status: "ReadyToInvoice",
    billingTransferStatus: null,
    totalChargeAmount: "1250.00",
    actualDeliveryDate: 1_788_000_000,
    customer: { id: `cus-${id}`, code: "ACME", name: "Acme Freight" },
    ...overrides,
  };
}

function connection(nodes: BillingTransferCandidate[]): BillingTransferCandidateConnection {
  return {
    edges: nodes.map((node) => ({ node })),
    totalCount: nodes.length,
    pageInfo: { hasNextPage: false, endCursor: null },
  };
}

function run(overrides: Partial<BillingTransferRun> = {}): BillingTransferRun {
  return {
    id: "btr_1",
    status: "Running",
    scope: "Selected",
    billType: "Invoice",
    searchQuery: null,
    shipmentStatus: null,
    sourceRunId: null,
    totalCount: 10,
    processedCount: 4,
    transferredCount: 3,
    notTransferredCount: 1,
    skippedCount: 0,
    markedReadyToInvoiceCount: 0,
    retryableCount: 1,
    unmatchedCount: 0,
    failureMessage: null,
    cancelRequestedAt: null,
    queuedAt: 1_788_000_000,
    startedAt: 1_788_000_010,
    completedAt: null,
    ...overrides,
  };
}

function item(overrides: Partial<BillingTransferRunItem> = {}): BillingTransferRunItem {
  return {
    id: "btri_1",
    shipmentId: "shp_1",
    sequence: 0,
    proNumber: "PRO-1",
    status: "NotTransferred",
    failureCode: "RequirementsUnmet",
    errorMessage: "Billing requirements must be resolved first",
    markedReadyToInvoice: false,
    billingQueueNumber: null,
    billingQueueStatus: null,
    missingRequirements: [
      { documentTypeId: "dt_pod", documentTypeCode: "POD", documentTypeName: "Proof of delivery" },
    ],
    validationFailures: [],
    ...overrides,
  };
}

function itemConnection(nodes: BillingTransferRunItem[]): BillingTransferRunItemConnection {
  return {
    edges: nodes.map((node) => ({ node })),
    totalCount: nodes.length,
    pageInfo: { hasNextPage: false, endCursor: null },
  };
}

function renderDialog(runId: string | null = null) {
  const onRunIdChange = vi.fn();
  const onOpenChange = vi.fn();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <BulkBillingTransferDialog
          open
          onOpenChange={onOpenChange}
          runId={runId}
          onRunIdChange={onRunIdChange}
        />
      </QueryClientProvider>
    </MemoryRouter>,
  );

  return { onRunIdChange, onOpenChange };
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  setLocale("en");
});

describe("BulkBillingTransferDialog", () => {
  it("starts a run for the picked shipments and hands back its id", async () => {
    const user = userEvent.setup();
    mocks.candidates.mockResolvedValue(connection([candidate("shp_1"), candidate("shp_2")]));
    mocks.activeRun.mockResolvedValue(null);
    mocks.startRun.mockResolvedValue(run({ status: "Queued", processedCount: 0 }));
    mocks.getRun.mockResolvedValue(run({ status: "Queued", processedCount: 0 }));

    const { onRunIdChange } = renderDialog();

    await screen.findByText("PRO-shp_1");
    await user.click(screen.getByRole("checkbox", { name: /PRO-shp_1/i }));
    await user.click(screen.getByRole("button", { name: /Transfer 1 selected/i }));

    await waitFor(() => expect(mocks.startRun).toHaveBeenCalledTimes(1));
    expect(mocks.startRun).toHaveBeenCalledWith(
      expect.objectContaining({ scope: "Selected", shipmentIds: ["shp_1"] }),
    );
    await waitFor(() => expect(onRunIdChange).toHaveBeenCalledWith("btr_1"));
  });

  // The whole point of moving the run server-side: the biller can walk away.
  it("reattaches to a run already in flight instead of offering to start another", async () => {
    mocks.candidates.mockResolvedValue(connection([candidate("shp_1")]));
    mocks.activeRun.mockResolvedValue(run());

    const { onRunIdChange } = renderDialog(null);

    await waitFor(() => expect(onRunIdChange).toHaveBeenCalledWith("btr_1"));
  });

  it("shows progress from the run's own counters", async () => {
    mocks.getRun.mockResolvedValue(run({ processedCount: 4, totalCount: 10 }));

    renderDialog("btr_1");

    expect(await screen.findByText("4 of 10 shipments checked")).toBeInTheDocument();
    const bar = screen.getByRole("progressbar", { name: "Transfer progress" });
    expect(bar).toHaveAttribute("aria-valuenow", "4");
    expect(bar).toHaveAttribute("aria-valuemax", "10");
  });

  // A running transfer must not trap the dialog open the way the old one did.
  it("can be closed while the transfer is still running", async () => {
    const user = userEvent.setup();
    mocks.getRun.mockResolvedValue(run());

    const { onOpenChange } = renderDialog("btr_1");

    await screen.findByText("4 of 10 shipments checked");
    await user.click(screen.getByRole("button", { name: "Run in background" }));

    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("asks the run to stop and says so until it does", async () => {
    const user = userEvent.setup();
    mocks.getRun.mockResolvedValue(run());
    mocks.cancelRun.mockResolvedValue(run({ cancelRequestedAt: 1_788_000_020 }));

    renderDialog("btr_1");

    await screen.findByText("4 of 10 shipments checked");
    await user.click(screen.getByRole("button", { name: "Stop" }));

    await waitFor(() => expect(mocks.cancelRun).toHaveBeenCalledWith("btr_1"));
    expect(
      await screen.findByRole("button", { name: /Stopping after this batch/i }),
    ).toBeDisabled();
  });

  it("reports what happened once the run finishes", async () => {
    mocks.getRun.mockResolvedValue(
      run({
        status: "Completed",
        processedCount: 10,
        transferredCount: 9,
        notTransferredCount: 1,
        completedAt: 1_788_000_100,
      }),
    );
    mocks.listItems.mockResolvedValue(itemConnection([item()]));

    renderDialog("btr_1");

    const summary = await screen.findByRole("group", { name: "Transfer summary" });
    expect(summary).toHaveTextContent("9");
    expect(await screen.findByText("Missing billing requirements")).toBeInTheDocument();
    expect(screen.getByText("Proof of delivery")).toBeInTheDocument();
  });

  it("starts a second run over what a retry could still move", async () => {
    const user = userEvent.setup();
    mocks.getRun.mockResolvedValue(
      run({ status: "Completed", processedCount: 10, retryableCount: 2 }),
    );
    mocks.listItems.mockResolvedValue(itemConnection([item()]));
    mocks.retryRun.mockResolvedValue(run({ id: "btr_2", status: "Queued" }));

    const { onRunIdChange } = renderDialog("btr_1");

    await user.click(await screen.findByRole("button", { name: /Retry 2/i }));

    await waitFor(() => expect(mocks.retryRun).toHaveBeenCalledWith("btr_1"));
    await waitFor(() => expect(onRunIdChange).toHaveBeenCalledWith("btr_2"));
  });

  // A run that has not been told its size yet must not show an empty bar, which
  // reads as "nothing has happened" rather than "we are still working it out".
  it("does not draw an empty bar before the run knows its size", async () => {
    mocks.getRun.mockResolvedValue(
      run({ status: "Running", scope: "AllMatching", totalCount: 0, processedCount: 0 }),
    );

    renderDialog("btr_1");

    expect(await screen.findByText("Finding eligible shipments...")).toBeInTheDocument();
    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
  });
});
