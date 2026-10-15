import { useCallback, useState } from "react";

function overflows(node: HTMLElement) {
  return node.scrollWidth > node.clientWidth;
}

/**
 * Whether a single-line element's text is cut off by its own width, kept current by a
 * ResizeObserver so a tooltip that repeats the text appears only while the text is clipped.
 */
export function useIsTruncated<T extends HTMLElement>(): [(node: T | null) => void, boolean] {
  const [truncated, setTruncated] = useState(false);
  const ref = useCallback((node: T | null) => {
    if (!node) {
      return;
    }
    setTruncated(overflows(node));
    if (typeof ResizeObserver === "undefined") {
      return;
    }
    const observer = new ResizeObserver(() => setTruncated(overflows(node)));
    observer.observe(node);

    return () => observer.disconnect();
  }, []);

  return [ref, truncated];
}
