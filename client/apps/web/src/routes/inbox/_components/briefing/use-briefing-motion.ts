import { useReducedMotion } from "motion/react";
import { useEffect, useRef, useState } from "react";

const COUNT_MS = 900;

/**
 * A number that counts up to its value once `run` is true, easing out over
 * most of a second, then follows later changes from wherever it stands.
 * Under reduced motion it is the value at once.
 */
export function useCountUp(value: number, run = true, durationMs = COUNT_MS): number {
  const reduce = useReducedMotion() ?? false;
  const [shown, setShown] = useState(0);
  const from = useRef(0);

  useEffect(() => {
    if (!run || reduce) {
      return;
    }
    const start = from.current;
    const began = performance.now();
    // Held in an object, since the frame a tick asks for is known only after it runs.
    const pending = { frame: 0 };
    const tick = (now: number) => {
      const progress = Math.min(1, (now - began) / durationMs);
      const eased = 1 - (1 - progress) ** 3;
      const next = Math.round(start + (value - start) * eased);
      from.current = next;
      setShown(next);
      if (progress < 1) {
        pending.frame = requestAnimationFrame(tick);
      }
    };
    pending.frame = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(pending.frame);
  }, [value, run, reduce, durationMs]);

  return reduce ? value : shown;
}
