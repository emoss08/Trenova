import { useEffect, useRef, useState } from "react";

export type InViewOptions = {
  /** The scrolling element to watch within; the viewport when null. */
  root?: Element | null;
  /** How far outside the root still counts as in view, as CSS margins: "240px 0px". */
  rootMargin?: string;
  /** Stay in view once seen, and stop watching: for work that should happen once, such as a read. */
  once?: boolean;
  /**
   * Called each time the browser reports the element coming into view, as it
   * happens rather than on the next render: for work done once per sighting,
   * such as drawing the next batch of a list when its end marker is seen.
   */
  onEnter?: () => void;
};

/** Whether this browser can tell when an element comes into view. */
export function canObserveIntersections(): boolean {
  return typeof IntersectionObserver !== "undefined";
}

/**
 * Whether an element is on screen, or close enough to it to start the work
 * it stands for: reading its data, drawing more of a list. Attach the
 * returned ref to the element.
 *
 * The answer belongs to the element it was given for: when the ref moves to a
 * new element it reads out of view until the browser has looked at the new
 * one. Work done once per sighting goes in `onEnter`, which runs when the
 * browser reports, not on a render that may still hold the last answer.
 *
 * Where IntersectionObserver is missing (an old browser, a test runner)
 * everything reads as in view, so the work happens at once rather than never.
 */
export function useInView<T extends Element>({
  root = null,
  rootMargin = "0px",
  once = false,
  onEnter,
}: InViewOptions = {}): [(node: T | null) => void, boolean] {
  const [node, setNode] = useState<T | null>(null);
  const [visible, setVisible] = useState<Element | null>(null);
  const [seen, setSeen] = useState(false);
  const entered = useRef(onEnter);
  const observable = canObserveIntersections();
  const settled = once && seen;

  useEffect(() => {
    entered.current = onEnter;
  }, [onEnter]);

  useEffect(() => {
    if (!node || !observable || settled) return undefined;

    const observer = new IntersectionObserver(
      (entries) => {
        const entry = entries.find((candidate) => candidate.target === node);
        if (!entry) return;
        if (entry.isIntersecting) entered.current?.();
        if (!once) {
          setVisible(entry.isIntersecting ? node : null);
        } else if (entry.isIntersecting) {
          setSeen(true);
          observer.disconnect();
        }
      },
      { root, rootMargin },
    );
    observer.observe(node);

    return () => observer.disconnect();
  }, [node, observable, once, root, rootMargin, settled]);

  if (!observable) return [setNode, true];

  return [setNode, once ? seen : node !== null && visible === node];
}
