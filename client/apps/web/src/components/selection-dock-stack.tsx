import { useEffect, useId, useSyncExternalStore } from "react";

let open: readonly string[] = [];
const listeners = new Set<() => void>();

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function publish(next: readonly string[]) {
  open = next;
  for (const listener of listeners) listener();
}

/** One step up the stack: a dock's height and the gap above it. */
export const SELECTION_DOCK_STEP_PX = 50;

/**
 * The place a selection dock takes among the docks open now, counting from the
 * bottom: two surfaces with something selected at once (the queue's items and an
 * item's documents) stack their docks, the first opened lowest, instead of drawing
 * one over the other. The dock stays where it is rendered, so one inside a modal
 * sheet is still inside it.
 */
export function useSelectionDockSlot(showing: boolean): number {
  const id = useId();

  useEffect(() => {
    if (!showing) return;
    publish([...open, id]);
    return () => publish(open.filter((entry) => entry !== id));
  }, [id, showing]);

  const index = useSyncExternalStore(
    subscribe,
    () => open.indexOf(id),
    () => -1,
  );
  return Math.max(0, index);
}
