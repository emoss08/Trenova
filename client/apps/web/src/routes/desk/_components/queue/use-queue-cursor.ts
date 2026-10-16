import { useReducedMotion } from "@trenova/shared/hooks/use-reduced-motion";
import { useCallback, useEffect, useRef, useState } from "react";

/** How long a card takes to leave before the next one comes in. */
export const QUEUE_LEAVE_MS = 300;

/** Settled leaves to the right, set aside to the left. */
export type QueueLeave = "ok" | "no";

/**
 * Where the person is in a queue. The current item is the one they chose, or
 * the first while they have chosen none or theirs has gone; stepping wraps at
 * either end, the way the queue reads as a loop.
 */
export function useQueueCursor<T extends { id: string }>(items: readonly T[]) {
  const [chosenId, setChosenId] = useState<string | null>(null);
  const found = chosenId === null ? -1 : items.findIndex((item) => item.id === chosenId);
  const index = Math.max(0, found);
  const current = items[index] ?? null;

  const go = useCallback(
    (step: number) => {
      if (items.length === 0) return;
      setChosenId(items[(index + step + items.length) % items.length].id);
    },
    [index, items],
  );

  /** The item that takes the place of one leaving: the next, else the one before. */
  const after = useCallback(
    (leavingIds: readonly string[]): string | null => {
      const rest = items.filter((item) => !leavingIds.includes(item.id));
      return rest[Math.min(index, rest.length - 1)]?.id ?? null;
    },
    [index, items],
  );

  return { current, index, select: setChosenId, go, after };
}

/**
 * A card leaving the queue: it animates out, then `done` runs. Under reduced
 * motion the card is cut rather than moved, so nothing waits on an animation
 * that is not drawn.
 */
export function useQueueLeave() {
  const reduceMotion = useReducedMotion();
  const [leaving, setLeaving] = useState<QueueLeave | null>(null);
  const timer = useRef<number | null>(null);

  useEffect(
    () => () => {
      if (timer.current !== null) {
        window.clearTimeout(timer.current);
      }
    },
    [],
  );

  const leave = useCallback(
    (how: QueueLeave, done: () => void) => {
      if (timer.current !== null) {
        window.clearTimeout(timer.current);
      }
      if (reduceMotion) {
        done();
        return;
      }
      setLeaving(how);
      timer.current = window.setTimeout(() => {
        timer.current = null;
        setLeaving(null);
        done();
      }, QUEUE_LEAVE_MS);
    },
    [reduceMotion],
  );

  return { leaving, leave };
}
