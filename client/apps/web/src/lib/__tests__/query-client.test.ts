import { QueryObserver } from "@tanstack/react-query";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { queryClient } from "@/lib/query-client";
import { PREFETCH_STALE_TIME_MS } from "@/lib/route-prefetch";

const key = ["query-client-test", "shipments"] as const;

function mountPage(queryFn: () => Promise<string>) {
  const observer = new QueryObserver(queryClient, { queryKey: key, queryFn });
  const unsubscribe = observer.subscribe(() => {});
  return unsubscribe;
}

describe("queryClient on mount", () => {
  let unmount: () => void = () => {};

  beforeEach(() => {
    vi.useFakeTimers({ toFake: ["Date"] });
  });

  afterEach(() => {
    unmount();
    unmount = () => {};
    queryClient.removeQueries({ queryKey: key });
    vi.useRealTimers();
  });

  it("does not fetch again data a prefetch has just warmed", async () => {
    const queryFn = vi.fn(() => Promise.resolve("warmed"));
    await queryClient.prefetchQuery({ queryKey: key, queryFn });

    vi.advanceTimersByTime(PREFETCH_STALE_TIME_MS - 1);
    unmount = mountPage(queryFn);

    expect(queryFn).toHaveBeenCalledTimes(1);
    expect(queryClient.getQueryData(key)).toBe("warmed");
  });

  it("fetches again once the data is older than the prefetch window", async () => {
    const queryFn = vi.fn(() => Promise.resolve("warmed"));
    await queryClient.prefetchQuery({ queryKey: key, queryFn });

    vi.advanceTimersByTime(PREFETCH_STALE_TIME_MS);
    unmount = mountPage(queryFn);

    expect(queryFn).toHaveBeenCalledTimes(2);
  });

  it("fetches again data that was invalidated, however young", async () => {
    const queryFn = vi.fn(() => Promise.resolve("warmed"));
    await queryClient.prefetchQuery({ queryKey: key, queryFn });
    await queryClient.invalidateQueries({ queryKey: key });

    unmount = mountPage(queryFn);

    expect(queryFn).toHaveBeenCalledTimes(2);
  });
});
