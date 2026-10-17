import { useCallback, useState } from "react";

/**
 * Which way the text is cut off: `inline` for one line clipped by its width,
 * `block` for text clamped to a number of lines and clipped by its height.
 */
export type TruncationAxis = "inline" | "block";

function overflows(node: HTMLElement, axis: TruncationAxis) {
  return axis === "inline"
    ? node.scrollWidth > node.clientWidth
    : node.scrollHeight > node.clientHeight;
}

/**
 * Whether an element's text is cut off by its own box, kept current by a
 * ResizeObserver: a single line clipped by its width, or (`block`) clamped
 * lines clipped by their height. A tooltip that repeats the text, or a "Show
 * more" that releases it, appears only while the text is clipped.
 */
export function useIsTruncated<T extends HTMLElement>(
  axis: TruncationAxis = "inline",
): [(node: T | null) => void, boolean] {
  const [truncated, setTruncated] = useState(false);
  const ref = useCallback(
    (node: T | null) => {
      if (!node) {
        return;
      }
      setTruncated(overflows(node, axis));
      if (typeof ResizeObserver === "undefined") {
        return;
      }
      const observer = new ResizeObserver(() => setTruncated(overflows(node, axis)));
      observer.observe(node);

      return () => observer.disconnect();
    },
    [axis],
  );

  return [ref, truncated];
}
