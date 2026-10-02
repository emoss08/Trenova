import { useEffect, useRef, useState } from "react";
import { useReducedMotion } from "motion/react";

/** How long a figure takes to count from where it was to where it is. */
const DEFAULT_DURATION_MS = 720;

function easeOut(t: number): number {
  return 1 - (1 - t) ** 3;
}

/**
 * Counts a figure from its previous value to its current one over a short
 * ease-out, so a number arriving on a page is seen to arrive rather than
 * to appear. The first value counts up from zero; under reduced motion, or
 * for a figure that is not finite, the value is simply the value.
 */
export function useCountUp(value: number, durationMs = DEFAULT_DURATION_MS): number {
  const reduceMotion = useReducedMotion();
  const [shown, setShown] = useState(reduceMotion ? value : 0);
  const from = useRef(reduceMotion ? value : 0);

  const still = reduceMotion || !Number.isFinite(value);

  useEffect(() => {
    if (still) {
      from.current = value;
      return;
    }
    const start = performance.now();
    const origin = from.current;
    let frame = 0;
    const tick = (now: number) => {
      const progress = Math.min(1, (now - start) / durationMs);
      const next = origin + (value - origin) * easeOut(progress);
      setShown(progress >= 1 ? value : Math.round(next));
      if (progress < 1) {
        frame = requestAnimationFrame(tick);
      } else {
        from.current = value;
      }
    };
    frame = requestAnimationFrame(tick);
    return () => {
      cancelAnimationFrame(frame);
      from.current = value;
    };
  }, [value, durationMs, still]);

  return still ? value : shown;
}
