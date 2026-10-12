import { useCallback } from "react";
import { create } from "zustand";

/**
 * A page that floats a bar along the bottom of the screen, as Triage's action
 * bar does, says how far up the screen it reaches, and the assistant's beacon
 * stands above the highest one while it sits in a bottom corner. The beacon
 * moves rather than the bar because the bar is the page's own next step and
 * the beacon can sit anywhere: it can be dragged to any corner and widens
 * while it says what it is doing, so no gutter the bar kept free of it would
 * hold at every width.
 */

/** The space between the top of a page's bar and the beacon standing over it. */
export const LAUNCHER_GAP = 12;

type ClearanceState = {
  /** How far up from the bottom of the screen each mounted bar reaches, in pixels. */
  bars: ReadonlyMap<symbol, number>;
  report: (id: symbol, reach: number) => void;
  withdraw: (id: symbol) => void;
};

export const useLauncherClearanceStore = create<ClearanceState>()((set) => ({
  bars: new Map(),
  report: (id, reach) =>
    set((state) => {
      if (state.bars.get(id) === reach) {
        return state;
      }
      const bars = new Map(state.bars);
      bars.set(id, reach);
      return { bars };
    }),
  withdraw: (id) =>
    set((state) => {
      if (!state.bars.has(id)) {
        return state;
      }
      const bars = new Map(state.bars);
      bars.delete(id);
      return { bars };
    }),
}));

/** How far up the highest bar reaches; 0 when no bar is showing. */
export function highestReach(bars: ReadonlyMap<symbol, number>): number {
  let highest = 0;
  for (const reach of bars.values()) {
    highest = Math.max(highest, reach);
  }
  return highest;
}

/**
 * How far from the bottom of the screen a beacon in a bottom corner sits:
 * where its corner puts it, or clear above the highest bar.
 */
export function beaconBottom(rest: number, reach: number): number {
  return reach > 0 ? Math.max(rest, reach + LAUNCHER_GAP) : rest;
}

/**
 * How far up from the bottom of the screen an element reaches. An element
 * that draws nothing (no height, or nothing showing) reaches nowhere.
 */
export function reachOf(rect: Pick<DOMRect, "top" | "height">, viewportHeight: number): number {
  if (rect.height <= 0) {
    return 0;
  }

  return Math.max(0, Math.ceil(viewportHeight - rect.top));
}

export function useLauncherReach(): number {
  return useLauncherClearanceStore((state) => highestReach(state.bars));
}

/**
 * Keeps the beacon clear of a bar that floats along the bottom of the screen
 * for as long as the element it is given is mounted and shows something. Give
 * it the bar's positioned container, not an element that animates in with a
 * transform, so the measure is where the bar settles rather than where its
 * entrance starts. It is a ref, so a bar that only renders now and then is
 * counted exactly while it is there.
 */
export function useClearOfLauncher(): (element: HTMLElement | null) => (() => void) | undefined {
  return useCallback((element: HTMLElement | null) => {
    if (element === null) {
      return undefined;
    }
    const id = Symbol("bottom-bar");
    const { report, withdraw } = useLauncherClearanceStore.getState();
    const measure = () => report(id, reachOf(element.getBoundingClientRect(), window.innerHeight));
    measure();
    const observer = typeof ResizeObserver === "undefined" ? null : new ResizeObserver(measure);
    observer?.observe(element);
    window.addEventListener("resize", measure);

    return () => {
      observer?.disconnect();
      window.removeEventListener("resize", measure);
      withdraw(id);
    };
  }, []);
}
