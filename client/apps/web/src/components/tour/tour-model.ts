import type { TranslateFn } from "@trenova/shared/i18n/runtime";

export type TourSide = "top" | "bottom" | "left" | "right";

/** One stop of a tour: what it points at and what it says about it. */
export type TourStep = {
  /**
   * The name given to the element with `tourTarget(name)`. A step with no target, or whose
   * target is not on the page, is shown in the middle of the screen.
   */
  target?: string;
  title: string;
  body: string;
  /** The side of the target the card prefers; it moves when there is no room. */
  side?: TourSide;
};

export type TourDefinition = {
  /** Lowercase words joined by hyphens: "freight-outlook". The server stores it. */
  key: string;
  /** Raise it when the steps change enough that people who saw the tour should see it again. */
  version: number;
  steps: (t: TranslateFn) => TourStep[];
};

export const TOUR_TARGET_ATTRIBUTE = "data-tour";

/** Marks an element as a place a tour can point at: `<section {...tourTarget("figures")}>`. */
export function tourTarget(name: string): { "data-tour": string } {
  return { [TOUR_TARGET_ATTRIBUTE]: name };
}

export function tourTargetSelector(name: string): string {
  return `[${TOUR_TARGET_ATTRIBUTE}="${name}"]`;
}

/** Whether a person has already been shown this tour as it stands now. */
export function isTourSeen(
  views: readonly { tour: string; tourVersion: number }[],
  tour: Pick<TourDefinition, "key" | "version">,
): boolean {
  return views.some((view) => view.tour === tour.key && view.tourVersion >= tour.version);
}

/**
 * The next step to show from `from` going forward (1) or back (-1), passing over steps whose
 * target is not on the page: a section a person cannot see has nothing to be shown. Null when
 * there is none, which going forward means the tour is over.
 */
export function nextStepIndex(
  steps: readonly TourStep[],
  from: number,
  direction: 1 | -1,
  onPage: (target: string) => boolean,
): number | null {
  for (let index = from + direction; index >= 0 && index < steps.length; index += direction) {
    const target = steps[index]?.target;
    if (target === undefined || onPage(target)) return index;
  }
  return null;
}
