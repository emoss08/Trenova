import { useCallback, useState } from "react";

export type ElementSize = { width: number; height: number };

const UNMEASURED: ElementSize = { width: 0, height: 0 };

/**
 * The content size of an element, kept current by a ResizeObserver, so something drawn to
 * fit it redraws when its box changes. Zero until the element has been measured.
 */
export function useElementSize<T extends HTMLElement>(): [(node: T | null) => void, ElementSize] {
  const [size, setSize] = useState<ElementSize>(UNMEASURED);
  const ref = useCallback((node: T | null) => {
    if (!node) {
      return;
    }
    const update = (width: number, height: number) => {
      const next = { width: Math.round(width), height: Math.round(height) };
      setSize((prev) => (prev.width === next.width && prev.height === next.height ? prev : next));
    };
    update(node.clientWidth, node.clientHeight);
    if (typeof ResizeObserver === "undefined") {
      return;
    }
    const observer = new ResizeObserver(([entry]) => {
      if (entry) {
        update(entry.contentRect.width, entry.contentRect.height);
      }
    });
    observer.observe(node);

    return () => observer.disconnect();
  }, []);

  return [ref, size];
}
