import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useDataTableLiveRefresh } from "../use-data-table-live-refresh";

type Row = { id: string; status: string };

const a: Row = { id: "a", status: "New" };
const b: Row = { id: "b", status: "New" };
const c: Row = { id: "c", status: "New" };

type Props = {
  scopeKey: string;
  results: Row[] | undefined;
  isPlaceholderData?: boolean;
  intervalMs?: number;
};

function setup(initial: Props, queryClient = new QueryClient()) {
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  const hook = renderHook(
    ({ scopeKey, results, isPlaceholderData = false, intervalMs }: Props) =>
      useDataTableLiveRefresh<Row>({
        intervalMs,
        enabled: true,
        queryKey: ["shipment-list", scopeKey],
        scopeKey,
        results,
        isPlaceholderData,
      }),
    { wrapper, initialProps: initial },
  );
  return { ...hook, queryClient };
}

describe("useDataTableLiveRefresh", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("shows the first answer for a page as it is", () => {
    const { result } = setup({ scopeKey: "p1", results: [a, b] });

    expect(result.current.results).toEqual([a, b]);
    expect(result.current.hasPendingUpdate).toBe(false);
    expect(result.current.changes.ids.size).toBe(0);
  });

  it("shows a newer answer with the same rows at once and marks only the rows that changed", () => {
    const { result, rerender } = setup({ scopeKey: "p1", results: [a, b, c] });
    const bChanged = { ...b, status: "InTransit" };

    rerender({ scopeKey: "p1", results: [a, bChanged, c] });

    expect(result.current.results).toEqual([a, bChanged, c]);
    expect(result.current.hasPendingUpdate).toBe(false);
    expect([...result.current.changes.ids]).toEqual(["b"]);
    expect(result.current.changes.version).toBe(1);
  });

  it("marks nothing when a refetch brings back the very same rows", () => {
    const { result, rerender } = setup({ scopeKey: "p1", results: [a, b] });

    rerender({ scopeKey: "p1", results: [a, b] });

    expect(result.current.changes.ids.size).toBe(0);
  });

  it("holds an answer that adds a row and keeps the page on screen", () => {
    const { result, rerender } = setup({ scopeKey: "p1", results: [a, b] });

    rerender({ scopeKey: "p1", results: [c, a, b] });

    expect(result.current.results).toEqual([a, b]);
    expect(result.current.hasPendingUpdate).toBe(true);
  });

  it("holds an answer that only reorders the rows", () => {
    const { result, rerender } = setup({ scopeKey: "p1", results: [a, b] });

    rerender({ scopeKey: "p1", results: [b, a] });

    expect(result.current.results).toEqual([a, b]);
    expect(result.current.hasPendingUpdate).toBe(true);
  });

  it("shows the held answer when asked and marks the rows that are new or changed", () => {
    const { result, rerender } = setup({ scopeKey: "p1", results: [a, b] });
    const aChanged = { ...a, status: "Completed" };
    rerender({ scopeKey: "p1", results: [c, aChanged, b] });

    act(() => result.current.applyStaged());

    expect(result.current.results).toEqual([c, aChanged, b]);
    expect(result.current.hasPendingUpdate).toBe(false);
    expect([...result.current.changes.ids].sort()).toEqual(["a", "c"]);
  });

  it("drops a held answer when dismissed and keeps the page as it was", () => {
    const { result, rerender } = setup({ scopeKey: "p1", results: [a, b] });
    rerender({ scopeKey: "p1", results: [c, a, b] });

    act(() => result.current.dismissStaged());

    expect(result.current.results).toEqual([a, b]);
    expect(result.current.hasPendingUpdate).toBe(false);
  });

  it("shows a page the person asked for straight away, whatever rows it holds", () => {
    const { result, rerender } = setup({ scopeKey: "p1", results: [a, b] });

    rerender({ scopeKey: "p2", results: [c] });

    expect(result.current.results).toEqual([c]);
    expect(result.current.hasPendingUpdate).toBe(false);
    expect(result.current.changes.ids.size).toBe(0);
  });

  it("does not hold a page's first real answer behind the page that stood in for it", () => {
    const { result, rerender } = setup({ scopeKey: "p1", results: [a, b] });

    rerender({ scopeKey: "p2", results: [a, b], isPlaceholderData: true });
    rerender({ scopeKey: "p2", results: [c] });

    expect(result.current.results).toEqual([c]);
    expect(result.current.hasPendingUpdate).toBe(false);
  });

  it("replays the glow with the other animation when the same row changes again", () => {
    const { result, rerender } = setup({ scopeKey: "p1", results: [a] });
    const once = { ...a, status: "One" };
    const twice = { ...a, status: "Two" };

    rerender({ scopeKey: "p1", results: [once] });
    const first = result.current.changes.version;
    rerender({ scopeKey: "p1", results: [twice] });

    expect(result.current.changes.version).toBe(first + 1);
    expect(result.current.changes.ids.has("a")).toBe(true);
  });

  it("asks for the page again on the interval only while the tab is visible", () => {
    vi.useFakeTimers();
    const queryClient = new QueryClient();
    const invalidate = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();
    setup({ scopeKey: "p1", results: [a], intervalMs: 1000 }, queryClient);

    vi.advanceTimersByTime(1000);
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["shipment-list", "p1"], exact: true });

    invalidate.mockClear();
    const visibility = vi.spyOn(document, "visibilityState", "get").mockReturnValue("hidden");
    vi.advanceTimersByTime(1000);
    expect(invalidate).not.toHaveBeenCalled();
    visibility.mockRestore();
  });
});
