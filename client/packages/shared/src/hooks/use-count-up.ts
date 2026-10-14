import { useEffect, useRef, useState } from "react";
import { useReducedMotion } from "./use-reduced-motion";

const DEFAULT_DURATION_MS = 900;

/**
 * A number that eases from what is on screen to where it is now, so a figure that changes
 * with the filters is seen to move. It lands at once when motion is reduced.
 */
export function useCountUp(target: number, durationMs = DEFAULT_DURATION_MS): number {
  const reduced = useReducedMotion();
  const [value, setValue] = useState(target);
  const shown = useRef(target);

  useEffect(() => {
    if (reduced || !Number.isFinite(target) || !Number.isFinite(shown.current)) {
      shown.current = target;
      setValue(target);
      return;
    }
    const start = shown.current;
    const began = performance.now();
    let frame = 0;
    const step = (now: number) => {
      const progress = Math.min(1, (now - began) / durationMs);
      const next = start + (target - start) * (1 - (1 - progress) ** 3);
      shown.current = next;
      setValue(next);
      if (progress < 1) {
        frame = requestAnimationFrame(step);
      }
    };
    frame = requestAnimationFrame(step);

    return () => cancelAnimationFrame(frame);
  }, [durationMs, reduced, target]);

  return value;
}
