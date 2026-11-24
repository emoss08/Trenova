import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { DriverSettlementRow } from "@/lib/graphql/driver-settlement";
import { SettlementQueue } from "../settlement-queue";

const mocks = vi.hoisted(() => ({
  bulkDriverSettlementAction: vi.fn(),
  toast: { error: vi.fn(), success: vi.fn(), info: vi.fn(), warning: vi.fn() },
}));

vi.mock("@/lib/graphql/driver-settlement", () => ({
  bulkDriverSettlementAction: mocks.bulkDriverSettlementAction,
}));
vi.mock("@trenova/shared/components/ui/toast", () => ({ toast: mocks.toast }));
vi.mock("@/components/settlements/bulk-mark-paid-dialog", () => ({
  BulkMarkPaidDialog: () => null,
}));

function row(over: Partial<DriverSettlementRow>): DriverSettlementRow {
  return {
    id: "dstl_1",
    settlementNumber: "DS-0001",
    status: "Approved",
    workerId: "wrk_1",
    worker: { id: "wrk_1", firstName: "Ada", lastName: "Byrne" },
    netPayMinor: 182_500,
    currencyCode: "USD",
    hasExceptions: false,
    holdReason: null,
    holdNote: "",
    heldAt: null,
    ...over,
  } as DriverSettlementRow;
}

const ready = row({ id: "dstl_ready", settlementNumber: "DS-0001" });
const held = row({
  id: "dstl_held",
  settlementNumber: "DS-0002",
  worker: { id: "wrk_2", firstName: "Ray", lastName: "Donnelly" },
  holdReason: "RateDispute",
  holdNote: "Waiting on the corrected rate",
  heldAt: 1_790_000_000,
});

function renderQueue(checked: string[], onFilterChange = vi.fn()) {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <SettlementQueue
        settlements={[ready, held]}
        allSettlements={[ready, held]}
        loading={false}
        filter="all"
        onFilterChange={onFilterChange}
        selectedId={null}
        onSelect={() => {}}
        checkedIds={new Set(checked)}
        onCheckedChange={() => {}}
        onActionComplete={() => {}}
      />
    </QueryClientProvider>,
  );
  return onFilterChange;
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("the settlement queue and holds", () => {
  it("marks a held settlement On hold and counts it under its own chip", () => {
    const onFilterChange = renderQueue([]);

    expect(screen.getAllByText("On hold")).toHaveLength(2);
    fireEvent.click(screen.getByRole("button", { name: /^On hold 1$/ }));
    expect(onFilterChange).toHaveBeenCalledWith("held");
  });

  it("posts only the settlements that are not held and says the held one was left out", async () => {
    mocks.bulkDriverSettlementAction.mockResolvedValue({
      results: [{ settlementId: "dstl_ready", success: true, error: "" }],
      successCount: 1,
      failureCount: 0,
    });
    renderQueue(["dstl_ready", "dstl_held"]);

    expect(
      screen.getByText("1 on hold, left out of approving, posting and paying"),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Post \(1\)/ }));

    await waitFor(() =>
      expect(mocks.bulkDriverSettlementAction).toHaveBeenCalledWith({
        settlementIds: ["dstl_ready"],
        action: "Post",
        paymentMethod: undefined,
        paymentReference: undefined,
      }),
    );
    await waitFor(() =>
      expect(mocks.toast.warning).toHaveBeenCalledWith(
        "1 settlement posted — 1 settlement on hold was left out",
      ),
    );
  });

  it("offers no post at all when every approved settlement chosen is held", () => {
    renderQueue(["dstl_held"]);

    expect(screen.queryByRole("button", { name: /Post \(/ })).not.toBeInTheDocument();
    expect(
      screen.getByText("1 on hold, left out of approving, posting and paying"),
    ).toBeInTheDocument();
  });
});
