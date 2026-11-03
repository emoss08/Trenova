import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { setLocale } from "@trenova/shared/i18n/runtime";
import { LazyMotion, MotionConfig, domMax } from "motion/react";
import { MemoryRouter } from "react-router";
import { useState } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type {
  BillingTransferCandidate,
  BillingTransferCandidateConnection,
  BillingTransferRun,
  BillingTransferRunItem,
  BillingTransferRunItemConnection,
} from "@/lib/graphql/billing-transfer";
import { BILLING_TRANSFER_RUN_KEY } from "@/routes/billing-queue/billing-queue-queries";
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
    failureCounts: [],
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
      <LazyMotion features={domMax}>
        <MotionConfig reducedMotion="always">
          <QueryClientProvider client={client}>
            <BulkBillingTransferDialog
              open
              onOpenChange={onOpenChange}
              runId={runId}
              onRunIdChange={onRunIdChange}
            />
          </QueryClientProvider>
        </MotionConfig>
      </LazyMotion>
    </MemoryRouter>,
  );

  return { onRunIdChange, onOpenChange, client };
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

  it("draws the run's own counts on the stage", async () => {
    mocks.getRun.mockResolvedValue(
      run({
        processedCount: 4,
        totalCount: 10,
        transferredCount: 3,
        notTransferredCount: 1,
        failureCounts: [{ code: "RateValidation", count: 1, retryable: true }],
      }),
    );

    renderDialog("btr_1");

    expect(await screen.findByText("4 of 10 shipments checked")).toBeInTheDocument();
    expect(screen.getByText(/^Delivered/)).toHaveTextContent("Delivered 6 left");
    expect(screen.getByText(/^Billing queue/)).toHaveTextContent("Billing queue 3");
    expect(screen.getByText(/^Needs attention/)).toHaveTextContent("Needs attention 1");
    expect(
      within(screen.getByRole("list", { name: "Readiness check" }))
        .getAllByRole("listitem")
        .map((step) => step.textContent),
    ).toEqual(["Status", "Documents", "Rate", "Not yet billed"]);
  });

  // The stage keeps the dialog until the run's last answer has been drawn, then gives way
  // to the results, with no step in between where neither is on screen.
  it("hands the stage over to the results when the run finishes while open", async () => {
    mocks.getRun.mockResolvedValue(run({ processedCount: 4, totalCount: 10 }));
    mocks.listItems.mockResolvedValue(itemConnection([item()]));

    const { client } = renderDialog("btr_1");

    await screen.findByText("4 of 10 shipments checked");
    client.setQueryData(
      [BILLING_TRANSFER_RUN_KEY, "btr_1"],
      run({
        status: "Completed",
        processedCount: 10,
        transferredCount: 9,
        notTransferredCount: 1,
        failureCounts: [{ code: "RequirementsUnmet", count: 1, retryable: true }],
        completedAt: 1_788_000_100,
      }),
    );

    expect(
      await screen.findByRole(
        "heading",
        { name: "9 of 10 shipments moved to billing" },
        { timeout: 3000 },
      ),
    ).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.queryByRole("list", { name: "Readiness check" })).not.toBeInTheDocument(),
    );
    expect(screen.getByRole("button", { name: "Done" })).toBeInTheDocument();
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
    await user.click(screen.getByRole("button", { name: "Stop transfer" }));

    await waitFor(() => expect(mocks.cancelRun).toHaveBeenCalledWith("btr_1"));
    expect(
      await screen.findByRole("button", { name: /Stopping after this batch/i }),
    ).toBeDisabled();
  });

  it("leads the report with what the run amounted to", async () => {
    mocks.getRun.mockResolvedValue(
      run({
        status: "Completed",
        processedCount: 10,
        transferredCount: 9,
        notTransferredCount: 1,
        markedReadyToInvoiceCount: 2,
        failureCounts: [{ code: "RequirementsUnmet", count: 1, retryable: true }],
        completedAt: 1_788_000_100,
      }),
    );
    mocks.listItems.mockResolvedValue(itemConnection([item()]));

    renderDialog("btr_1");

    expect(
      await screen.findByRole("heading", { name: "9 of 10 shipments moved to billing" }),
    ).toBeInTheDocument();
    expect(screen.getByText("1 needs attention")).toBeInTheDocument();
    expect(screen.getByText("2 marked Ready to Invoice first")).toBeInTheDocument();
    expect(screen.getByText("Took 1 min, 30 sec")).toBeInTheDocument();
  });

  // The shipments that did not transfer are the biller's remaining work, so the
  // report opens on them, grouped by what is holding each one back.
  it("opens on the held-back shipments, filterable by reason", async () => {
    mocks.getRun.mockResolvedValue(
      run({
        status: "Completed",
        processedCount: 10,
        transferredCount: 6,
        notTransferredCount: 4,
        failureCounts: [
          { code: "RequirementsUnmet", count: 3, retryable: true },
          { code: "AlreadyTransferred", count: 1, retryable: false },
        ],
        completedAt: 1_788_000_100,
      }),
    );
    mocks.listItems.mockResolvedValue(itemConnection([item()]));

    renderDialog("btr_1");

    const outcome = await screen.findByRole("group", { name: "Transfer outcome" });
    expect(within(outcome).getByRole("button", { name: /Needs attention/ })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    const reasons = screen.getByRole("radiogroup", { name: "Reasons" });
    expect(
      within(reasons)
        .getAllByRole("radio")
        .map((reason) => reason.textContent),
    ).toEqual(["All reasons4", "Missing billing requirements3", "Already in billing1"]);
    expect(await screen.findByText("Proof of delivery")).toBeInTheDocument();
    expect(mocks.listItems).toHaveBeenCalledWith(
      expect.objectContaining({
        runId: "btr_1",
        statuses: ["NotTransferred"],
        failureCodes: undefined,
      }),
      expect.anything(),
    );
  });

  it("narrows the report to the shipments one reason held back", async () => {
    const user = userEvent.setup();
    mocks.getRun.mockResolvedValue(
      run({
        status: "Completed",
        processedCount: 10,
        transferredCount: 6,
        notTransferredCount: 4,
        failureCounts: [
          { code: "RequirementsUnmet", count: 3, retryable: true },
          { code: "AlreadyTransferred", count: 1, retryable: false },
        ],
        completedAt: 1_788_000_100,
      }),
    );
    mocks.listItems.mockResolvedValue(itemConnection([item()]));

    renderDialog("btr_1");

    const reasons = await screen.findByRole("radiogroup", { name: "Reasons" });
    await user.click(within(reasons).getByRole("radio", { name: /Already in billing/ }));

    await waitFor(() =>
      expect(mocks.listItems).toHaveBeenCalledWith(
        expect.objectContaining({
          statuses: ["NotTransferred"],
          failureCodes: ["AlreadyTransferred"],
        }),
        expect.anything(),
      ),
    );
    expect(screen.getByRole("heading", { name: "Already in billing" })).toBeInTheDocument();
    expect(
      screen.getByText("The shipment is already in the billing queue. Retrying won't move these."),
    ).toBeInTheDocument();
  });

  it("opens on the transferred shipments when nothing was held back", async () => {
    mocks.getRun.mockResolvedValue(
      run({
        status: "Completed",
        processedCount: 10,
        transferredCount: 10,
        notTransferredCount: 0,
        retryableCount: 0,
        completedAt: 1_788_000_100,
      }),
    );
    mocks.listItems.mockResolvedValue(
      itemConnection([
        item({
          status: "Transferred",
          failureCode: null,
          errorMessage: null,
          billingQueueNumber: "BQ-1",
          missingRequirements: [],
        }),
      ]),
    );

    renderDialog("btr_1");

    expect(
      await screen.findByRole("heading", { name: "10 shipments moved to billing" }),
    ).toBeInTheDocument();
    expect(await screen.findByText("BQ-1")).toBeInTheDocument();
    expect(mocks.listItems).toHaveBeenCalledWith(
      expect.objectContaining({ statuses: ["Transferred"] }),
      expect.anything(),
    );
  });

  // A retry that fails to start must say so on the report it was pressed from,
  // not only on the picker the biller is no longer looking at.
  it("says so when a retry cannot be started", async () => {
    const user = userEvent.setup();
    mocks.getRun.mockResolvedValue(
      run({ status: "Completed", processedCount: 10, retryableCount: 2 }),
    );
    mocks.listItems.mockResolvedValue(itemConnection([item()]));
    mocks.retryRun.mockRejectedValue(new Error("You already have a transfer running"));

    renderDialog("btr_1");

    await user.click(await screen.findByRole("button", { name: /Retry 2/i }));

    expect(await screen.findByText("The retry could not be started")).toBeInTheDocument();
    expect(screen.getByText("You already have a transfer running")).toBeInTheDocument();
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

  // The active-run lookup last saw this run while it was still going. Once it has finished,
  // starting a fresh transfer must open the picker, not reattach to the run just closed.
  it("opens the picker on the first press of Transfer more after a run finishes", async () => {
    const user = userEvent.setup();
    mocks.activeRun.mockResolvedValueOnce(run()).mockResolvedValue(null);
    mocks.getRun.mockResolvedValue(
      run({ status: "Completed", processedCount: 10, transferredCount: 10 }),
    );
    mocks.listItems.mockResolvedValue(itemConnection([]));
    mocks.candidates.mockResolvedValue(connection([candidate("shp_1")]));

    function StatefulDialog() {
      const [runId, setRunId] = useState<string | null>(null);
      return (
        <BulkBillingTransferDialog
          open
          onOpenChange={vi.fn()}
          runId={runId}
          onRunIdChange={setRunId}
        />
      );
    }
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <MemoryRouter>
        <LazyMotion features={domMax}>
          <MotionConfig reducedMotion="always">
            <QueryClientProvider client={client}>
              <StatefulDialog />
            </QueryClientProvider>
          </MotionConfig>
        </LazyMotion>
      </MemoryRouter>,
    );

    await user.click(await screen.findByRole("button", { name: "Transfer more" }));

    expect(await screen.findByText("PRO-shp_1")).toBeInTheDocument();
    await waitFor(() => expect(mocks.activeRun).toHaveBeenCalledTimes(2));
    expect(screen.getByText("PRO-shp_1")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Transfer more" })).not.toBeInTheDocument();
  });

  // A run still working out which shipments it has says so, rather than drawing an
  // empty stack that reads as "nothing to do".
  it("says it is still finding shipments before the run knows its size", async () => {
    mocks.getRun.mockResolvedValue(
      run({ status: "Running", scope: "AllMatching", totalCount: 0, processedCount: 0 }),
    );

    renderDialog("btr_1");

    expect(await screen.findByText("Finding eligible shipments")).toBeInTheDocument();
    expect(screen.getByText("Finding shipments")).toBeInTheDocument();
    expect(screen.queryByText(/^Delivered/)).not.toBeInTheDocument();
  });
});
