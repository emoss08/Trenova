import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { setLocale } from "@trenova/shared/i18n/runtime";
import type { ReactElement } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import type {
  BillingTransferCandidate,
  BillingTransferCandidateConnection,
  BulkBillingTransferResponse,
  BulkBillingTransferResult,
} from "@/lib/graphql/billing-transfer";
import { BulkBillingTransferDialog } from "../bulk-billing-transfer-dialog";

const mocks = vi.hoisted(() => ({
  candidates: vi.fn(),
  candidateIds: vi.fn(),
  transfer: vi.fn(),
}));

vi.mock("@/lib/graphql/billing-transfer", () => ({
  listBillingTransferCandidatesGraphQL: mocks.candidates,
  listBillingTransferCandidateIdsGraphQL: mocks.candidateIds,
  bulkTransferShipmentsToBillingGraphQL: mocks.transfer,
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

function transferred(shipmentId: string, overrides: Partial<BulkBillingTransferResult> = {}) {
  return {
    shipmentId,
    proNumber: `PRO-${shipmentId}`,
    success: true,
    markedReadyToInvoice: false,
    failureCode: null,
    error: null,
    billingQueueItem: {
      id: `bqi-${shipmentId}`,
      number: `INV-${shipmentId}`,
      status: "ReadyForReview",
    },
    missingRequirements: [],
    validationFailures: [],
    ...overrides,
  } satisfies BulkBillingTransferResult;
}

function notTransferred(
  shipmentId: string,
  overrides: Partial<BulkBillingTransferResult> = {},
): BulkBillingTransferResult {
  return {
    shipmentId,
    proNumber: `PRO-${shipmentId}`,
    success: false,
    markedReadyToInvoice: false,
    failureCode: "RequirementsUnmet",
    error: "Shipment billing requirements must be resolved before transfer to billing",
    billingQueueItem: null,
    missingRequirements: [
      { documentTypeId: "dt_pod", documentTypeCode: "POD", documentTypeName: "Proof of Delivery" },
    ],
    validationFailures: [],
    ...overrides,
  };
}

function respond(results: BulkBillingTransferResult[]): BulkBillingTransferResponse {
  const successCount = results.filter((r) => r.success).length;
  return {
    results,
    totalCount: results.length,
    successCount,
    errorCount: results.length - successCount,
  };
}

function renderDialog(
  ui: ReactElement = <BulkBillingTransferDialog open onOpenChange={vi.fn()} />,
) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const invalidate = vi.spyOn(client, "invalidateQueries");
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>{ui}</QueryClientProvider>
    </MemoryRouter>,
  );
  return { client, invalidate };
}

afterEach(async () => {
  cleanup();
  vi.clearAllMocks();
  await act(() => setLocale("en"));
});

describe("BulkBillingTransferDialog", () => {
  it("transfers the picked shipments and says why the rest did not transfer", async () => {
    mocks.candidates.mockResolvedValue(
      connection([
        candidate("shp_1"),
        candidate("shp_2", { status: "Completed", customer: null }),
        candidate("shp_3"),
      ]),
    );
    mocks.transfer.mockResolvedValue(
      respond([transferred("shp_1", { markedReadyToInvoice: false }), notTransferred("shp_2")]),
    );
    const user = userEvent.setup();
    const { invalidate } = renderDialog();

    await user.click(await screen.findByRole("checkbox", { name: "Select PRO-shp_1" }));
    await user.click(screen.getByRole("checkbox", { name: "Select PRO-shp_2" }));
    await user.click(screen.getByRole("button", { name: "Transfer 2 selected" }));

    expect(mocks.transfer).toHaveBeenCalledTimes(1);
    expect(mocks.transfer).toHaveBeenCalledWith(["shp_1", "shp_2"]);

    const failures = await screen.findByRole("list", { name: "Not transferred" });
    const failure = within(failures).getByRole("listitem");
    expect(within(failure).getByText("PRO-shp_2")).toBeInTheDocument();
    expect(within(failure).getByText("Missing billing requirements")).toBeInTheDocument();
    expect(
      within(failure).getByText(
        "Shipment billing requirements must be resolved before transfer to billing",
      ),
    ).toBeInTheDocument();
    expect(within(failure).getByText("Proof of Delivery")).toBeInTheDocument();
    expect(screen.queryByText("PRO-shp_1")).not.toBeInTheDocument();

    const summary = screen.getByRole("group", { name: "Transfer summary" });
    expect(within(summary).getByText("Transferred").nextSibling).toHaveTextContent("1");
    expect(within(summary).getByText("Not transferred").nextSibling).toHaveTextContent("1");

    await user.click(screen.getByRole("radio", { name: /Transferred/ }));
    const successes = screen.getByRole("list", { name: "Transferred" });
    expect(within(successes).getByText("PRO-shp_1")).toBeInTheDocument();
    expect(within(successes).getByText("INV-shp_1")).toBeInTheDocument();

    await waitFor(() =>
      expect(invalidate).toHaveBeenCalledWith({ queryKey: ["billing-queue-list"] }),
    );
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["billing-transfer-candidates"] });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["shipment-list"] });
  });

  it("confirms before transferring everything that matches the current filters", async () => {
    mocks.candidates.mockImplementation(async ({ query, status }) =>
      connection(
        query === "ACME" && status === "Completed"
          ? [candidate("shp_7", { status: "Completed" })]
          : [candidate("shp_1"), candidate("shp_2")],
      ),
    );
    mocks.candidateIds.mockResolvedValue({
      ids: ["shp_7", "shp_8"],
      totalCount: 2,
      truncated: false,
    });
    mocks.transfer.mockImplementation(async (ids: string[]) =>
      respond(ids.map((id) => transferred(id))),
    );
    const user = userEvent.setup();
    renderDialog();

    await screen.findByText("PRO-shp_1");
    await user.type(screen.getByPlaceholderText("Search PRO, BOL..."), "ACME");
    await user.click(screen.getByRole("radio", { name: "Completed" }));
    await screen.findByText("PRO-shp_7");

    await user.click(screen.getByRole("button", { name: "Transfer all 1" }));
    expect(mocks.transfer).not.toHaveBeenCalled();
    expect(
      screen.getByText("Transfer all 1 shipment that matches the current search and status?"),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Start transfer" }));

    await waitFor(() => expect(mocks.transfer).toHaveBeenCalledWith(["shp_7", "shp_8"]));
    expect(mocks.candidateIds).toHaveBeenCalledWith(
      { query: "ACME", status: "Completed" },
      expect.anything(),
    );
    const summary = await screen.findByRole("group", { name: "Transfer summary" });
    expect(within(summary).getByText("Transferred").nextSibling).toHaveTextContent("2");
  });

  it("warns when more shipments matched than one run transfers", async () => {
    mocks.candidates.mockResolvedValue(connection([candidate("shp_1")]));
    mocks.candidateIds.mockResolvedValue({ ids: ["shp_1"], totalCount: 5200, truncated: true });
    mocks.transfer.mockResolvedValue(respond([transferred("shp_1")]));
    const user = userEvent.setup();
    renderDialog();

    await user.click(await screen.findByRole("button", { name: "Transfer all 1" }));
    await user.click(screen.getByRole("button", { name: "Start transfer" }));

    expect(
      await screen.findByText(
        "5,199 more shipments matched than one run transfers. Run Transfer all again for the rest.",
      ),
    ).toBeInTheDocument();
  });

  it("stops after the batch in flight and lists the rest as not processed", async () => {
    const ids = Array.from({ length: 30 }, (_, i) => `shp_${i + 1}`);
    mocks.candidates.mockResolvedValue(connection([candidate("shp_1")]));
    mocks.candidateIds.mockResolvedValue({ ids, totalCount: 30, truncated: false });
    let releaseFirstBatch: (value: BulkBillingTransferResponse) => void = () => undefined;
    mocks.transfer.mockImplementationOnce(
      (batch: string[]) =>
        new Promise<BulkBillingTransferResponse>((resolve) => {
          releaseFirstBatch = () => resolve(respond(batch.map((id) => transferred(id))));
        }),
    );
    const user = userEvent.setup();
    renderDialog();

    await user.click(await screen.findByRole("button", { name: "Transfer all 1" }));
    await user.click(screen.getByRole("button", { name: "Start transfer" }));
    await waitFor(() => expect(mocks.transfer).toHaveBeenCalledTimes(1));

    await user.click(screen.getByRole("button", { name: "Stop" }));
    expect(screen.getByRole("button", { name: "Stopping after this batch..." })).toBeDisabled();
    await act(async () => releaseFirstBatch(respond([])));

    const summary = await screen.findByRole("group", { name: "Transfer summary" });
    expect(mocks.transfer).toHaveBeenCalledTimes(1);
    expect(within(summary).getByText("Transferred").nextSibling).toHaveTextContent("25");
    expect(within(summary).getByText("Not processed").nextSibling).toHaveTextContent("5");
    expect(
      screen.getByText("You stopped the transfer. The remaining shipments were not sent."),
    ).toBeInTheDocument();
  });

  it("explains a run that ended because a request failed", async () => {
    mocks.candidates.mockResolvedValue(connection([candidate("shp_1"), candidate("shp_2")]));
    mocks.transfer.mockRejectedValue(new Error("Network request failed"));
    const user = userEvent.setup();
    renderDialog();

    await user.click(await screen.findByRole("checkbox", { name: "Select all shown shipments" }));
    await user.click(screen.getByRole("button", { name: "Transfer 2 selected" }));

    expect(await screen.findByText("The transfer stopped early")).toBeInTheDocument();
    expect(screen.getByText(/Network request failed/)).toBeInTheDocument();
    const summary = screen.getByRole("group", { name: "Transfer summary" });
    expect(within(summary).getByText("Not processed").nextSibling).toHaveTextContent("2");
  });

  it("retries only the shipments that could transfer now", async () => {
    mocks.candidates.mockResolvedValue(
      connection([candidate("shp_1"), candidate("shp_2"), candidate("shp_3")]),
    );
    mocks.transfer
      .mockResolvedValueOnce(
        respond([
          notTransferred("shp_1"),
          notTransferred("shp_2", { failureCode: "AlreadyTransferred", missingRequirements: [] }),
          transferred("shp_3"),
        ]),
      )
      .mockResolvedValueOnce(respond([transferred("shp_1")]));
    const user = userEvent.setup();
    renderDialog();

    await user.click(await screen.findByRole("checkbox", { name: "Select all shown shipments" }));
    await user.click(screen.getByRole("button", { name: "Transfer 3 selected" }));
    await user.click(await screen.findByRole("button", { name: "Retry 1" }));

    await waitFor(() => expect(mocks.transfer).toHaveBeenCalledTimes(2));
    expect(mocks.transfer).toHaveBeenLastCalledWith(["shp_1"]);
    const summary = await screen.findByRole("group", { name: "Transfer summary" });
    await waitFor(() =>
      expect(within(summary).getByText("Transferred").nextSibling).toHaveTextContent("2"),
    );
    expect(within(summary).getByText("Not transferred").nextSibling).toHaveTextContent("1");
    const failures = screen.getByRole("list", { name: "Not transferred" });
    expect(within(failures).getByText("PRO-shp_2")).toBeInTheDocument();
    expect(within(failures).getByText("Already in billing")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^Retry/ })).not.toBeInTheDocument();
  });

  it("keeps the dialog open while a transfer is running", async () => {
    mocks.candidates.mockResolvedValue(connection([candidate("shp_1")]));
    mocks.transfer.mockReturnValue(new Promise(() => undefined));
    const onOpenChange = vi.fn();
    const user = userEvent.setup();
    renderDialog(<BulkBillingTransferDialog open onOpenChange={onOpenChange} />);

    await user.click(await screen.findByRole("checkbox", { name: "Select PRO-shp_1" }));
    await user.click(screen.getByRole("button", { name: "Transfer 1 selected" }));
    await screen.findByRole("button", { name: "Stop" });

    await user.keyboard("{Escape}");
    expect(onOpenChange).not.toHaveBeenCalledWith(false, expect.anything());
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("loads the next page as the list scrolls to its end", async () => {
    const observed: Element[] = [];
    vi.stubGlobal(
      "IntersectionObserver",
      class {
        constructor(private readonly callback: IntersectionObserverCallback) {}
        observe(element: Element) {
          observed.push(element);
          queueMicrotask(() =>
            this.callback(
              [{ isIntersecting: true, target: element } as unknown as IntersectionObserverEntry],
              this as unknown as IntersectionObserver,
            ),
          );
        }
        unobserve() {}
        disconnect() {}
        takeRecords() {
          return [];
        }
      },
    );
    const firstPage = Array.from({ length: 50 }, (_, i) => candidate(`shp_${i + 1}`));
    const secondPage = Array.from({ length: 12 }, (_, i) => candidate(`shp_${i + 51}`));
    mocks.candidates.mockImplementation(async ({ after }: { after?: string | null }) =>
      after === "cursor-50"
        ? {
            edges: secondPage.map((node) => ({ node })),
            totalCount: null,
            pageInfo: { hasNextPage: false, endCursor: "cursor-62" },
          }
        : {
            edges: firstPage.map((node) => ({ node })),
            totalCount: 62,
            pageInfo: { hasNextPage: true, endCursor: "cursor-50" },
          },
    );
    mocks.transfer.mockImplementation(async (ids: string[]) =>
      respond(ids.map((id) => transferred(id))),
    );
    const user = userEvent.setup();

    try {
      renderDialog();

      expect(await screen.findByText("PRO-shp_62")).toBeInTheDocument();
      expect(observed.length).toBeGreaterThan(0);
      expect(mocks.candidates).toHaveBeenCalledTimes(2);
      expect(mocks.candidates).toHaveBeenLastCalledWith(
        expect.objectContaining({ after: "cursor-50" }),
        expect.anything(),
      );
      expect(screen.getByText("62 shipments can transfer")).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Transfer all 62" })).toBeInTheDocument();

      await user.click(screen.getByRole("checkbox", { name: "Select all shown shipments" }));
      await user.click(screen.getByRole("button", { name: "Transfer 62 selected" }));

      await waitFor(() => expect(mocks.transfer).toHaveBeenCalledTimes(3));
      expect(mocks.transfer.mock.calls.flatMap(([ids]) => ids)).toHaveLength(62);
    } finally {
      vi.unstubAllGlobals();
    }
  });

  it("says there is nothing to transfer when every shipment is already in billing", async () => {
    mocks.candidates.mockResolvedValue(connection([]));
    renderDialog();

    expect(await screen.findByText("Nothing to transfer")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Transfer 0 selected" })).toBeDisabled();
    expect(screen.queryByRole("button", { name: /Transfer all/ })).not.toBeInTheDocument();
  });
});
