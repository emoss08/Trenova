import { useLayoutEffect, useState, type RefObject } from "react";

/**
 * Whether an element is narrower than `width` pixels, kept current as it
 * resizes. For a layout that answers its own column rather than the window:
 * a reader beside a list is narrow on a wide screen.
 */
export function useWidthBelow(ref: RefObject<HTMLElement | null>, width: number): boolean {
  const [below, setBelow] = useState(false);

  useLayoutEffect(() => {
    const element = ref.current;
    if (element === null) {
      return;
    }
    setBelow(element.getBoundingClientRect().width < width);
    if (typeof ResizeObserver === "undefined") {
      return;
    }
    const observer = new ResizeObserver((entries) => {
      const entry = entries[entries.length - 1];
      if (entry !== undefined) {
        setBelow(entry.contentRect.width < width);
      }
    });
    observer.observe(element);

    return () => observer.disconnect();
  }, [ref, width]);

  return below;
}
