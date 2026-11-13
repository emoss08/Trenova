/**
 * The rail's frame, as class lists. They live apart from the rail's component
 * so the Desk's loading screen draws the same frame without loading the rail.
 *
 * A rail is pinned or floating (Desk settings, Appearance). Pinned, it sits
 * beside the page, folds to nothing, and on a phone is a sheet over the page.
 * Floating, it is a card over the page at every width, shown when asked for
 * and put away once a place in it is chosen.
 */

// i18n-ignore: class names, never shown to a person
const SHEET_OPEN =
  "data-open:translate-x-0 data-open:shadow-[var(--dsk-lift-hi),0_0_0_1px_var(--dsk-b-sub)]";

// i18n-ignore: class names, never shown to a person
const RAIL_BASE =
  "group/rail flex min-h-0 min-w-0 flex-col overflow-hidden bg-dsk-rail font-sans **:font-sans";

// i18n-ignore: class names, never shown to a person
const RAIL_PINNED = [
  "w-58 border-r border-dsk-b-sub",
  "min-[721px]:transition-[width,border-color] min-[721px]:duration-[240ms] min-[721px]:ease-settle",
  "min-[721px]:data-folded:w-0 min-[721px]:data-folded:border-r-transparent",
  "max-[720px]:fixed max-[720px]:inset-y-0 max-[720px]:left-0 max-[720px]:z-40",
  "max-[720px]:w-[min(300px,86vw)] max-[720px]:-translate-x-[102%]",
  "max-[720px]:transition-transform max-[720px]:duration-[320ms] max-[720px]:ease-settle",
  SHEET_OPEN.replaceAll("data-open:", "max-[720px]:data-open:"),
].join(" ");

// i18n-ignore: class names, never shown to a person
const RAIL_FLOATING = [
  "fixed top-2 bottom-2 left-2 z-40 w-58 max-[720px]:w-[min(300px,86vw)]",
  // The lift carries its own hairline ring, so the card takes no border of its own.
  "rounded-xl data-open:shadow-(--dsk-lift-hi)",
  "-translate-x-[calc(100%+1rem)] transition-transform duration-[320ms] ease-settle",
  "data-open:translate-x-0",
].join(" ");

export type RailMode = "pinned" | "floating";

/** The rail itself: where it sits and how it comes and goes. */
export function railClass(mode: RailMode): string {
  return [RAIL_BASE, mode === "floating" ? RAIL_FLOATING : RAIL_PINNED].join(" ");
}

// i18n-ignore: class names, never shown to a person
const INNER_BASE = "flex min-h-0 flex-1 flex-col px-1 pt-1.5";

// i18n-ignore: class names, never shown to a person
const INNER_PINNED = [
  "w-[231px] max-[720px]:w-full",
  "min-[721px]:transition-opacity min-[721px]:duration-[160ms] min-[721px]:ease-[ease]",
  "min-[721px]:group-data-folded/rail:opacity-0",
].join(" ");

/** What the rail holds, at a fixed width so folding does not reflow it. */
export function railInnerClass(mode: RailMode): string {
  return [INNER_BASE, mode === "floating" ? "w-full" : INNER_PINNED].join(" ");
}

// i18n-ignore: class names, never shown to a person
export const RAIL_TOP_CLASS = "flex h-13 flex-none items-center gap-2 px-2";

/** The rail's body, between its top and its foot: the room its scroll area fills. */
// i18n-ignore: class names, never shown to a person
export const RAIL_BODY_CLASS = "min-h-0 flex-1";

/** The gutter of what scrolls in the body, the same under every view of the rail. */
// i18n-ignore: class names, never shown to a person
export const RAIL_GUTTER_CLASS = "relative px-2 pt-0.5 pb-4";
