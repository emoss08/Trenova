import { scrollParent } from "@/lib/dom";
import { useLayoutEffect, useRef } from "react";

/**
 * For a view drawn in place of another inside the same scrolling pane, such as a record
 * opened from a table's row: the view opens at the top of the pane, and again each time
 * `key` changes (stepping to the next record), and once it is gone the pane goes back to
 * where it was, so the view it replaced is found as it was left.
 *
 * Put the returned ref on an element inside the view; the pane is the nearest ancestor
 * that scrolls.
 */
export function useScrollFromTop<T extends HTMLElement>(key: string) {
  const ref = useRef<T>(null);

  // Declared first, so the pane's place is read before the view moves it.
  useLayoutEffect(() => {
    const pane = scrollParent(ref.current);
    if (!pane) return undefined;
    const left = pane.scrollTop;
    return () => {
      // The view it replaced is put back in the same commit; once that commit is
      // done the pane is tall enough to return to where it was.
      queueMicrotask(() => {
        pane.scrollTop = left;
      });
    };
  }, []);

  useLayoutEffect(() => {
    const pane = scrollParent(ref.current);
    if (pane) {
      pane.scrollTop = 0;
    }
  }, [key]);

  return ref;
}
