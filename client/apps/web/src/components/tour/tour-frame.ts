import { autoUpdate, getOverflowAncestors } from "@floating-ui/react-dom";
import { useEffect, useState, useSyncExternalStore } from "react";
import { grow, overlap, placeCard, sameBox, type Box, type Extent } from "./tour-geometry";
import { TOUR_TARGET_ATTRIBUTE, tourTargetSelector, type TourSide } from "./tour-model";

/** The clear space left around a lit part, inside its outline. */
const LIT_PADDING = 6;

/** A fixed box for the card to hang off, re-measured with the target's own scrolling. */
type CardAnchor = { getBoundingClientRect: () => DOMRect; contextElement: Element };

/** A step on screen: what is lit, and where its card hangs. */
export type TourFrame = {
  lit: Box;
  side: TourSide;
  gap: number;
  anchorBox: Box;
  anchor: CardAnchor;
};

type Measured = { target: Element; frame: TourFrame | null };

/**
 * The part a step points at, when a person can see and use it: on the page, drawn, and not
 * inside something switched off (a folded panel keeps its contents but marks them inert).
 */
export function findTourTarget(name: string): Element | null {
  const element = document.querySelector(tourTargetSelector(name));
  if (!element || element.closest("[inert]")) return null;
  if (typeof element.checkVisibility === "function" && !element.checkVisibility()) return null;
  return element;
}

function windowBox(): Box {
  return { x: 0, y: 0, width: window.innerWidth, height: window.innerHeight };
}

/**
 * The part of the window a target can show through: the window, cut down by every ancestor
 * that scrolls or clips it. Null when nothing of it can show.
 */
export function clipOf(target: Element): Box | null {
  let clip: Box | null = windowBox();
  for (const ancestor of getOverflowAncestors(target)) {
    if (clip === null) return null;
    if (ancestor instanceof Element) clip = overlap(clip, ancestor.getBoundingClientRect());
  }
  return clip;
}

/**
 * As much of a target as is on screen. A list longer than the box it scrolls in reports its
 * whole length as its own box, most of it under whatever sits above and below the list; only
 * the part that shows is lit and only that part places the card.
 */
export function visibleBox(target: Element): Box | null {
  const clip = clipOf(target);
  return clip && overlap(target.getBoundingClientRect(), clip);
}

/** Whether a target is longer than the space it shows through, so it cannot be centered. */
export function outgrowsItsClip(target: Element): boolean {
  const clip = clipOf(target);
  return clip !== null && target.getBoundingClientRect().height > clip.height;
}

const PAGE_CHANGES: MutationObserverInit = {
  childList: true,
  subtree: true,
  attributes: true,
  attributeFilter: [TOUR_TARGET_ATTRIBUTE, "inert", "hidden"],
};

function watchPage(onChange: () => void): () => void {
  const observer = new MutationObserver(onChange);
  observer.observe(document.body, PAGE_CHANGES);
  return () => observer.disconnect();
}

function watchNothing(): () => void {
  return () => {};
}

/**
 * The element a step points at, kept current while the step is open: a part drawn a moment
 * after the tour opened (a panel the page unfolds for it) is picked up, and one that goes away
 * is let go.
 */
export function useStepTarget(name: string | undefined): Element | null {
  return useSyncExternalStore(
    name ? watchPage : watchNothing,
    () => (name ? findTourTarget(name) : null),
    () => null,
  );
}

/**
 * Which of a tour's parts are drawn right now, kept current while the tour is open, so the
 * steps it passes over and the button that ends it follow the page as it changes.
 */
export function useDrawnTargets(names: readonly string[], watching: boolean): ReadonlySet<string> {
  const drawn = useSyncExternalStore(
    watching ? watchPage : watchNothing,
    () => (watching ? names.filter((name) => findTourTarget(name) !== null).join("\n") : ""),
    () => "",
  );
  return new Set(drawn === "" ? [] : drawn.split("\n"));
}

/**
 * Where a step's target is on screen and where its card goes, followed as the page scrolls,
 * resizes or shifts under it. Null while there is no target or none of it shows.
 */
export function useTourFrame(
  target: Element | null,
  preferred: TourSide,
  card: Extent,
): TourFrame | null {
  const [measured, setMeasured] = useState<Measured | null>(null);
  const { width, height } = card;

  useEffect(() => {
    if (!target) return;
    const measure = () => {
      const visible = visibleBox(target);
      const viewport = { width: window.innerWidth, height: window.innerHeight };
      setMeasured((current) => {
        const before = current?.target === target ? current.frame : undefined;
        if (!visible) return before === null ? current : { target, frame: null };
        const lit = grow(visible, LIT_PADDING);
        const placed = placeCard({ lit, viewport, card: { width, height }, preferred });
        const unchanged =
          before &&
          sameBox(before.lit, lit) &&
          sameBox(before.anchorBox, placed.anchor) &&
          before.side === placed.side &&
          before.gap === placed.gap;
        if (unchanged) return current;
        const anchorBox = placed.anchor;
        return {
          target,
          frame: {
            lit,
            side: placed.side,
            gap: placed.gap,
            anchorBox,
            anchor: {
              getBoundingClientRect: () => DOMRect.fromRect(anchorBox),
              contextElement: target,
            },
          },
        };
      });
    };
    // Nothing floats beside the target here, so the page's own body stands in for it: only
    // the target's box, its scrolling ancestors and the window are worth watching.
    return autoUpdate(target, document.body, measure, { layoutShift: true });
  }, [target, preferred, width, height]);

  return measured?.target === target ? measured.frame : null;
}
