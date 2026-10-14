import { useCallback, useState } from "react";

/**
 * The content width of an element, kept current by a ResizeObserver so a chart drawn in
 * pixels redraws when its column narrows. Zero until the element has been measured.
 */
export function useElementWidth<T extends HTMLElement>(): [(node: T | null) => void, number] {
  const [width, setWidth] = useState(0);
  const ref = useCallback((node: T | null) => {
    if (!node) {
      return;
    }
    setWidth(Math.round(node.clientWidth));
    if (typeof ResizeObserver === "undefined") {
      return;
    }
    const observer = new ResizeObserver(([entry]) => {
      if (entry) {
        setWidth(Math.round(entry.contentRect.width));
      }
    });
    observer.observe(node);

    return () => observer.disconnect();
  }, []);

  return [ref, width];
}
