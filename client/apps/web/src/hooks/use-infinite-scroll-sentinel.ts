import { useEffect, useRef, useState, type RefCallback } from "react";

const SCROLLABLE_OVERFLOW = /(auto|scroll|overlay)/;

type InfiniteScrollSentinelOptions = {
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  onLoadMore: () => void;
  /** How far ahead of the scroll container's edge the next page starts loading. */
  rootMargin?: string;
};

function findScrollContainer(node: Element): Element | null {
  let current = node.parentElement;
  while (current) {
    const { overflowY } = window.getComputedStyle(current);
    if (SCROLLABLE_OVERFLOW.test(overflowY)) return current;
    current = current.parentElement;
  }
  return null;
}

/**
 * Loads the next page of an infinite list when a sentinel at its end scrolls
 * into view.
 *
 * The observer is rooted at the nearest scroll container rather than the window,
 * because a list inside a panel is clipped by that panel: a margin measured
 * against the window would never let it load ahead of the panel's edge. The
 * observer is rebuilt whenever a page finishes loading, so a sentinel that is
 * still visible — a list shorter than its box — reports again and keeps filling.
 */
export function useInfiniteScrollSentinel<T extends Element>({
  hasNextPage,
  isFetchingNextPage,
  onLoadMore,
  rootMargin = "0px 0px 240px 0px",
}: InfiniteScrollSentinelOptions): RefCallback<T> {
  const [sentinel, setSentinel] = useState<T | null>(null);
  const onLoadMoreRef = useRef(onLoadMore);

  useEffect(() => {
    onLoadMoreRef.current = onLoadMore;
  }, [onLoadMore]);

  useEffect(() => {
    if (!sentinel || !hasNextPage || isFetchingNextPage) return;
    if (typeof IntersectionObserver === "undefined") return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) {
          onLoadMoreRef.current();
        }
      },
      { root: findScrollContainer(sentinel), rootMargin },
    );
    observer.observe(sentinel);

    return () => observer.disconnect();
  }, [sentinel, hasNextPage, isFetchingNextPage, rootMargin]);

  return setSentinel;
}
