"use no memo";
import type { Virtualizer } from "@tanstack/react-virtual";
import { useCallback } from "react";

/**
 * A ref for a virtual row that measures it once the commit that attached it is over.
 *
 * Measured during the commit, a row above the viewport that is taller than its
 * estimate moves the scroll position and asks React to flush the new positions
 * at once, which React refuses mid-commit, so a frame paints with the rows where
 * the estimate put them. A microtask still runs before the browser paints, so
 * the corrected positions land in the same frame.
 */
export function useDeferredMeasure<TScroll extends Element | Window, TItem extends Element>(
  virtualizer: Virtualizer<TScroll, TItem>,
) {
  return useCallback(
    (node: TItem | null) => {
      if (node === null) {
        virtualizer.measureElement(null);
        return;
      }
      queueMicrotask(() => {
        if (node.isConnected) {
          virtualizer.measureElement(node);
        }
      });
    },
    [virtualizer],
  );
}
