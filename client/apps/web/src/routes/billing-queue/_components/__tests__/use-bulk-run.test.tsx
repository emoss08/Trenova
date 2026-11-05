import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { BillingQueueApprovalRun } from "@trenova/shared/types/billing-queue";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useBulkRun } from "../use-bulk-run";

const mocks = vi.hoisted(() => ({
  startBulk: vi.fn(),
  getBulk: vi.fn(),
  undoBulk: vi.fn(),
  toast: Object.assign(vi.fn(), {
    success: vi.fn(),
    warning: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
  }),
}));

vi.mock("@/services/billing-queue", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/services/billing-queue")>()),
  billingQueueService: {
    startBulk: mocks.startBulk,
    getBulk: mocks.getBulk,
    undoBulk: mocks.undoBulk,
  },
}));
vi.mock("@trenova/shared/components/ui/toast", () => ({ toast: mocks.toast }));

afterEach(() => vi.clearAllMocks());

function run(overrides: Partial<BillingQueueApprovalRun> = {}): BillingQueueApprovalRun {
  return {
    id: "bar_1",
    action: "Approve",
    status: "Scheduled",
    totalCount: 3,
    succeededCount: 0,
    failedCount: 0,
    skippedCount: 0,
    commitAt: Math.floor(Date.now() / 1000) + 10,
    items: [],
    ...overrides,
  };
}

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

describe("useBulkRun", () => {
  it("starts the action with its params and offers Undo until the run commits", async () => {
    mocks.startBulk.mockResolvedValue(run());
    mocks.getBulk.mockReturnValue(new Promise(() => undefined));
    const { result } = renderHook(() => useBulkRun(), { wrapper });

    await act(() =>
      result.current
        .start("Hold", ["bqi_1", "bqi_2", "bqi_3"], {
          holdReasonCode: "RateQuestion",
        })
        .then(() => undefined),
    );

    expect(mocks.startBulk).toHaveBeenCalledWith(
      "Hold",
      ["bqi_1", "bqi_2", "bqi_3"],
      expect.any(String),
      { holdReasonCode: "RateQuestion" },
    );
    const [, options] = mocks.toast.mock.calls[0] as [string, { action?: { label: string } }];
    expect(options.action?.label).toBe("Undo");
  });

  it("offers no Undo for posting, which writes at once", async () => {
    mocks.startBulk.mockResolvedValue(run({ action: "Post" }));
    mocks.getBulk.mockReturnValue(new Promise(() => undefined));
    const { result } = renderHook(() => useBulkRun(), { wrapper });

    await act(() => result.current.start("Post", ["bqi_1"]).then(() => undefined));

    const [message, options] = mocks.toast.mock.calls[0] as [string, { action?: unknown }];
    expect(message).toBe("Posting 3 invoices");
    expect(options.action).toBeUndefined();
  });

  it("reports what the finished run did, read from the run", async () => {
    mocks.startBulk.mockResolvedValue(run());
    mocks.getBulk.mockResolvedValue(
      run({ status: "Completed", succeededCount: 2, skippedCount: 1 }),
    );
    const { result } = renderHook(() => useBulkRun(), { wrapper });

    await act(() =>
      result.current.start("Approve", ["bqi_1", "bqi_2", "bqi_3"]).then(() => undefined),
    );

    await waitFor(() =>
      expect(mocks.toast.warning).toHaveBeenCalledWith("Approved 2 of 3", expect.anything()),
    );
    expect(mocks.toast.success).not.toHaveBeenCalled();
  });
});
