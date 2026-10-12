import { useReducedMotion } from "motion/react";
import { useEffect, useRef, useState, type RefObject } from "react";

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
    let frame = 0;
    const tick = (now: number) => {
      const progress = Math.min(1, (now - began) / durationMs);
      const eased = 1 - (1 - progress) ** 3;
      const next = Math.round(start + (value - start) * eased);
      from.current = next;
      setShown(next);
      if (progress < 1) {
        frame = requestAnimationFrame(tick);
      }
    };
    frame = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(frame);
  }, [value, run, reduce, durationMs]);

  return reduce ? value : shown;
}

/**
 * Whether the briefing's motion should rest: while it is scrolled out of
 * view or the tab is hidden, nothing on it moves, so a loop never runs for
 * nobody. The returned flag is meant for a `data-paused` attribute the
 * stylesheet reads; the starfield reads it to stop drawing.
 */
export function usePausedWhenAway<T extends Element>(ref: RefObject<T | null>): boolean {
  const [onScreen, setOnScreen] = useState(true);
  const [visible, setVisible] = useState(
    () => typeof document === "undefined" || document.visibilityState !== "hidden",
  );

  useEffect(() => {
    const element = ref.current;
    if (element === null || typeof IntersectionObserver === "undefined") {
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        const entry = entries.at(-1);
        if (entry !== undefined) {
          setOnScreen(entry.isIntersecting);
        }
      },
      { threshold: 0 },
    );
    observer.observe(element);
    return () => observer.disconnect();
  }, [ref]);

  useEffect(() => {
    const onChange = () => setVisible(document.visibilityState !== "hidden");
    document.addEventListener("visibilitychange", onChange);
    return () => document.removeEventListener("visibilitychange", onChange);
  }, []);

  return !onScreen || !visible;
}
