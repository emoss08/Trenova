import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { worker } from "@/lib/queries/worker";
import {
  WORKER_WRITE_REACH,
  useWorkerInvalidation,
  workerWriteKeys,
} from "../use-worker-invalidation";

describe("workerWriteKeys", () => {
  it("refreshes a credential write's own reads, the overview, the record and the roster", () => {
    expect(workerWriteKeys("wrk_1", "credentials")).toEqual([
      ["worker-credential-summary", "wrk_1"],
      ["worker-credentials", "wrk_1"],
      ["credential-expiry-forecast"],
      worker.overview("wrk_1").queryKey,
      worker.record("wrk_1").queryKey,
      ["worker-list"],
    ]);
  });

  it("leaves the record and the roster alone for a review, which only the overview shows", () => {
    expect(workerWriteKeys("wrk_1", "reviews")).toEqual([
      ["worker-reviews", "wrk_1"],
      worker.overview("wrk_1").queryKey,
    ]);
  });

  it("refreshes the time off book with an employment event, under the requests' own key", () => {
    const keys = workerWriteKeys("wrk_1", "employment");
    expect(keys).toContainEqual(["worker-pto-requests", "wrk_1"]);
    expect(keys).toContainEqual(["worker-pto-list"]);
    expect(keys).not.toContainEqual(["worker"]);
  });

  it("refreshes the worker's time off book, the organisation's summaries and the roster for a time off write", () => {
    expect(workerWriteKeys("wrk_1", "time-off")).toEqual([
      ["worker-pto-balances", "wrk_1"],
      ["worker-pto-ledger", "wrk_1"],
      ["worker-pto-assignments", "wrk_1"],
      ["worker-pto-requests", "wrk_1"],
      ["pto-balance-summary"],
      ["worker-pto-list"],
      worker.overview("wrk_1").queryKey,
      ["worker-list"],
    ]);
  });

  it("refreshes the worker's shifts and availability, the rota and the overview for a schedule write", () => {
    expect(workerWriteKeys("wrk_1", "schedule")).toEqual([
      ["shift-assignments", "wrk_1"],
      ["availability-preferences", "wrk_1"],
      ["rota"],
      worker.overview("wrk_1").queryKey,
    ]);
  });

  it("never invalidates every worker query wholesale", () => {
    for (const area of Object.keys(WORKER_WRITE_REACH) as (keyof typeof WORKER_WRITE_REACH)[]) {
      expect(workerWriteKeys("wrk_1", area)).not.toContainEqual(["worker"]);
    }
  });
});

describe("useWorkerInvalidation", () => {
  it("invalidates exactly the area's keys and settles once they have", async () => {
    const client = new QueryClient();
    const invalidate = vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    );
    const { result } = renderHook(() => useWorkerInvalidation("wrk_1", "leave"), { wrapper });

    await result.current();

    expect(invalidate.mock.calls.map(([filters]) => filters?.queryKey)).toEqual([
      ["worker-leave", "wrk_1"],
      ["worker-employment-events", "wrk_1"],
      worker.overview("wrk_1").queryKey,
      worker.record("wrk_1").queryKey,
      ["worker-list"],
    ]);
  });
});
