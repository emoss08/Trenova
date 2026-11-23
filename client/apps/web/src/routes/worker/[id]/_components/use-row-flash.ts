import { useEffect, useRef, useState } from "react";

/** How long the wash lasts; the tokens' `row-changed` and `attention` both run 1600ms. */
const FLASH_MS = 1600;

export type FlashPhase = "a" | "b";

/**
 * Brief washes on rows a person was sent to or just changed. A key flashes `a`, a second flash
 * while it still shows switches to `b`, so the animation replays (the tokens declare two names
 * with the same keyframes for this). Draw it with `data-changed={flashOf(key)}` and the
 * `animate-row-changed-a/b` or `animate-attention-a/b` utilities.
 */
export function useRowFlash() {
  const [phases, setPhases] = useState<Readonly<Record<string, FlashPhase>>>({});
  const timers = useRef(new Map<string, number>());

  useEffect(() => {
    const pending = timers.current;
    return () => {
      for (const timer of pending.values()) window.clearTimeout(timer);
      pending.clear();
    };
  }, []);

  const flash = (key: string) => {
    setPhases((current) => ({ ...current, [key]: current[key] === "a" ? "b" : "a" }));
    const previous = timers.current.get(key);
    if (previous !== undefined) window.clearTimeout(previous);
    timers.current.set(
      key,
      window.setTimeout(() => {
        timers.current.delete(key);
        setPhases((current) => {
          const { [key]: _done, ...rest } = current;
          return rest;
        });
      }, FLASH_MS),
    );
  };

  return { flash, flashOf: (key: string): FlashPhase | undefined => phases[key] };
}

/** The classes a table row takes so `data-changed` washes its cells. */
export const ROW_CHANGED_CLASS =
  "data-[changed=a]:[&>td]:animate-row-changed-a data-[changed=b]:[&>td]:animate-row-changed-b";

/** The classes a block takes so `data-changed` draws the attention ring once. */
export const ATTENTION_CLASS =
  "data-[changed=a]:animate-attention-a data-[changed=b]:animate-attention-b";

/** Brings an element into view where the browser can; jsdom and old engines lack it. */
export function revealElement(element: Element | null | undefined) {
  if (element && typeof element.scrollIntoView === "function") {
    element.scrollIntoView({ block: "nearest", behavior: "smooth" });
  }
}
