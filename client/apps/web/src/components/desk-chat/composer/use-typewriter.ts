import { useEffect, useState } from "react";

const TYPE_MS = 38;
const ERASE_MS = 18;
const HOLD_MS = 2200;
const GAP_MS = 300;

type Typing = { index: number; shown: number; erasing: boolean };

/**
 * Types each line in turn, holds it, erases it and moves to the next, for as
 * long as it is active. Returns what is on screen and the line it belongs to.
 */
export function useTypewriter(lines: readonly string[], active: boolean) {
  const [state, setState] = useState<Typing>({ index: 0, shown: 0, erasing: false });
  const count = lines.length;
  const index = count === 0 ? 0 : state.index % count;
  const full = lines[index] ?? "";

  useEffect(() => {
    if (!active || count === 0) {
      return;
    }
    let delay = state.erasing ? ERASE_MS : TYPE_MS;
    if (!state.erasing && state.shown === full.length) {
      delay = HOLD_MS;
    }
    if (state.erasing && state.shown === 0) {
      delay = GAP_MS;
    }
    const timer = window.setTimeout(() => {
      setState((current) => {
        if (!current.erasing && current.shown < full.length) {
          return { ...current, shown: current.shown + 1 };
        }
        if (!current.erasing) {
          return { ...current, erasing: true };
        }
        if (current.shown > 0) {
          return { ...current, shown: current.shown - 1 };
        }
        return { index: (current.index + 1) % count, shown: 0, erasing: false };
      });
    }, delay);

    return () => window.clearTimeout(timer);
  }, [active, count, full.length, state]);

  return { text: full.slice(0, state.shown), full, index, done: state.shown === full.length };
}
