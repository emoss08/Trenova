import { useCallback, useState } from "react";

/** Whether an element's text runs past its own width, so an ellipsis hides part of it. */
export function isClipped(element: Pick<HTMLElement, "scrollWidth" | "clientWidth">): boolean {
  return element.scrollWidth > element.clientWidth + 1;
}

/**
 * Whether the text in an element is cut off by its ellipsis, kept current as
 * the element is resized. Attach the returned ref to the element that clips,
 * and key that element by its text, so a new text mounts it again and is
 * measured.
 *
 * Where ResizeObserver is missing (a test runner) the text is measured when
 * the element is attached.
 */
export function useClipped<T extends HTMLElement>(): [
  (element: T | null) => (() => void) | undefined,
  boolean,
] {
  const [clipped, setClipped] = useState(false);

  const ref = useCallback((element: T | null) => {
    if (!element) {
      return undefined;
    }
    setClipped(isClipped(element));
    if (typeof ResizeObserver === "undefined") {
      return undefined;
    }
    const observer = new ResizeObserver(() => setClipped(isClipped(element)));
    observer.observe(element);
    return () => observer.disconnect();
  }, []);

  return [ref, clipped];
}
