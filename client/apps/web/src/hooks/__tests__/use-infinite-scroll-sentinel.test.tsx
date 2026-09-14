import { act, cleanup, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useInfiniteScrollSentinel } from "../use-infinite-scroll-sentinel";

type ObserverRecord = {
  callback: IntersectionObserverCallback;
  options: IntersectionObserverInit | undefined;
  observed: Element[];
  disconnected: boolean;
};

let observers: ObserverRecord[] = [];

class FakeIntersectionObserver {
  record: ObserverRecord;

  constructor(callback: IntersectionObserverCallback, options?: IntersectionObserverInit) {
    this.record = { callback, options, observed: [], disconnected: false };
    observers.push(this.record);
  }

  observe(element: Element) {
    this.record.observed.push(element);
  }

  unobserve() {}

  disconnect() {
    this.record.disconnected = true;
  }

  takeRecords() {
    return [];
  }
}

function intersect(record: ObserverRecord, isIntersecting: boolean) {
  act(() => {
    record.callback(
      [{ isIntersecting } as IntersectionObserverEntry],
      record as unknown as IntersectionObserver,
    );
  });
}

function activeObservers() {
  return observers.filter((record) => !record.disconnected && record.observed.length > 0);
}

function Harness(props: {
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  onLoadMore: () => void;
  scrollable?: boolean;
}) {
  const sentinelRef = useInfiniteScrollSentinel<HTMLDivElement>(props);
  return (
    <div data-testid="container" style={props.scrollable ? { overflowY: "auto" } : undefined}>
      <div data-testid="sentinel" ref={sentinelRef} />
    </div>
  );
}

beforeEach(() => {
  observers = [];
  vi.stubGlobal("IntersectionObserver", FakeIntersectionObserver);
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("useInfiniteScrollSentinel", () => {
  it("loads the next page when the sentinel scrolls into view", () => {
    const onLoadMore = vi.fn();
    const { getByTestId } = render(
      <Harness hasNextPage isFetchingNextPage={false} onLoadMore={onLoadMore} />,
    );

    const [observer] = activeObservers();
    expect(observer.observed).toEqual([getByTestId("sentinel")]);

    intersect(observer, false);
    expect(onLoadMore).not.toHaveBeenCalled();

    intersect(observer, true);
    expect(onLoadMore).toHaveBeenCalledTimes(1);
  });

  it("watches from the nearest scroll container so it can load ahead of the edge", () => {
    const { getByTestId } = render(
      <Harness hasNextPage isFetchingNextPage={false} onLoadMore={vi.fn()} scrollable />,
    );

    const [observer] = activeObservers();
    expect(observer.options?.root).toBe(getByTestId("container"));
    expect(observer.options?.rootMargin).toBeTruthy();
  });

  it("does not watch while a page is loading or when there is nothing left", () => {
    const onLoadMore = vi.fn();
    const { rerender } = render(<Harness hasNextPage isFetchingNextPage onLoadMore={onLoadMore} />);
    expect(activeObservers()).toHaveLength(0);

    rerender(<Harness hasNextPage={false} isFetchingNextPage={false} onLoadMore={onLoadMore} />);
    expect(activeObservers()).toHaveLength(0);
    expect(onLoadMore).not.toHaveBeenCalled();
  });

  it("watches again once the page arrives, so a list shorter than its box keeps filling", () => {
    const onLoadMore = vi.fn();
    const { rerender } = render(
      <Harness hasNextPage isFetchingNextPage={false} onLoadMore={onLoadMore} />,
    );
    const first = activeObservers()[0];

    rerender(<Harness hasNextPage isFetchingNextPage onLoadMore={onLoadMore} />);
    expect(first.disconnected).toBe(true);

    rerender(<Harness hasNextPage isFetchingNextPage={false} onLoadMore={onLoadMore} />);
    const [second] = activeObservers();
    expect(second).not.toBe(first);

    intersect(second, true);
    expect(onLoadMore).toHaveBeenCalledTimes(1);
  });

  it("stops watching when the component unmounts", () => {
    const { unmount } = render(
      <Harness hasNextPage isFetchingNextPage={false} onLoadMore={vi.fn()} />,
    );
    const [observer] = activeObservers();

    unmount();
    expect(observer.disconnected).toBe(true);
  });

  it("does nothing where IntersectionObserver is unavailable", () => {
    vi.stubGlobal("IntersectionObserver", undefined);
    const onLoadMore = vi.fn();

    expect(() =>
      render(<Harness hasNextPage isFetchingNextPage={false} onLoadMore={onLoadMore} />),
    ).not.toThrow();
    expect(onLoadMore).not.toHaveBeenCalled();
  });
});
