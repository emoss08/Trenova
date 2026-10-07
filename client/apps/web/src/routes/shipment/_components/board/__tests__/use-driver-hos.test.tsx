import { act, renderHook } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { queries } from "@/lib/queries";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useDriverHos } from "../use-driver-hos";

const { capabilities } = vi.hoisted(() => ({ capabilities: { hos: true } }));

vi.mock("@/lib/shipment-board/capabilities", () => ({
  useShipmentCapabilities: () => capabilities,
}));

const HOS_KEY = queries.telematics.workerHosStates().queryKey;

function seededClient(states: { workerId: string; driveRemainingMs: number }[]) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity, refetchInterval: false } },
  });
  queryClient.setQueryData(HOS_KEY, states);
  return queryClient;
}

function wrapperFor(queryClient: QueryClient) {
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

describe("useDriverHos", () => {
  beforeEach(() => {
    capabilities.hos = true;
  });

  it("reads each driver's drive time from the page's one HOS list", () => {
    const queryClient = seededClient([
      { workerId: "wrk_a", driveRemainingMs: 3_600_000 },
      { workerId: "wrk_b", driveRemainingMs: 0 },
    ]);
    const wrapper = wrapperFor(queryClient);

    expect(renderHook(() => useDriverHos("wrk_a"), { wrapper }).result.current).toBe(3_600_000);
    expect(renderHook(() => useDriverHos("wrk_b"), { wrapper }).result.current).toBe(0);
  });

  it("has nothing for a driver the ELD does not report, or no driver at all", () => {
    const wrapper = wrapperFor(seededClient([{ workerId: "wrk_a", driveRemainingMs: 1 }]));

    expect(renderHook(() => useDriverHos("wrk_z"), { wrapper }).result.current).toBeNull();
    expect(renderHook(() => useDriverHos(null), { wrapper }).result.current).toBeNull();
    expect(renderHook(() => useDriverHos(undefined), { wrapper }).result.current).toBeNull();
  });

  it("has nothing when no ELD is connected", () => {
    capabilities.hos = false;
    const wrapper = wrapperFor(seededClient([{ workerId: "wrk_a", driveRemainingMs: 1 }]));

    expect(renderHook(() => useDriverHos("wrk_a"), { wrapper }).result.current).toBeNull();
  });

  it("redraws only the cells whose driver's number changed on a refetch", async () => {
    const queryClient = seededClient([
      { workerId: "wrk_a", driveRemainingMs: 1_000 },
      { workerId: "wrk_b", driveRemainingMs: 2_000 },
    ]);
    const wrapper = wrapperFor(queryClient);
    const renders = { a: 0, b: 0 };
    const a = renderHook(
      () => {
        renders.a += 1;
        return useDriverHos("wrk_a");
      },
      { wrapper },
    );
    const b = renderHook(
      () => {
        renders.b += 1;
        return useDriverHos("wrk_b");
      },
      { wrapper },
    );
    renders.a = 0;
    renders.b = 0;

    await act(async () => {
      queryClient.setQueryData(HOS_KEY, [
        { workerId: "wrk_a", driveRemainingMs: 500 },
        { workerId: "wrk_b", driveRemainingMs: 2_000 },
      ]);
      await new Promise((resolve) => setTimeout(resolve, 20));
    });

    expect(a.result.current).toBe(500);
    expect(b.result.current).toBe(2_000);
    expect(renders.a).toBeGreaterThan(0);
    expect(renders.b).toBe(0);
  });
});
