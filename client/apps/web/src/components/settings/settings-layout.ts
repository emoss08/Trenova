/**
 * The classes of a settings page, shared by the page and its loading state so the two
 * cannot drift. A settings page is one centred column: a row of tabs when it has several
 * jobs, then sections, each a titled list of one setting to a row.
 */
export const settingsColumnClass = "mx-auto w-full max-w-3xl";

/**
 * Sections stack with no gap of their own: each carries its space beneath it, so a section
 * that opens or closes takes its space with it instead of leaving a gap to snap shut.
 */
export const settingsSectionsClass = "flex flex-col";

export const settingsSectionClass = "flex flex-col gap-2 pb-5";

export const settingsSectionHeadClass = "flex flex-col gap-px px-0.5";

/** The list: its rows ruled off from one another, on the card surface. */
export const settingsListClass =
  "divide-y divide-border overflow-hidden rounded-xl border border-border bg-card";

/** Rows that only apply while the row above them is on: set in, on a ground of their own. */
export const settingsNestedClass =
  "divide-y divide-border bg-surface-hover/50 *:data-[slot=setting-row]:pl-8.5";

/** What a section reports rather than asks (what was learned): padded as a row is. */
export const settingsReadoutClass = "flex flex-col gap-3 px-3.5 py-3";

/**
 * A tabbed settings page fills the page's frame: the tab row holds its place at the top and
 * only what is under it scrolls, so the tabs are where they were however far down a person
 * has read. The page is a `PageLayout fill`, and everything between it and the tabs passes
 * the height on (`settingsFillClass`).
 */
export const settingsFillClass = "flex min-h-0 flex-1 flex-col";

export const settingsTabsClass = "min-h-0 flex-1 gap-0";

/** The rule the tab row sits on. It never scrolls away and never shrinks. */
export const settingsTabRowClass = "shrink-0 overflow-x-auto border-b border-border";

/** The scrolling region under the tab row: the full width, so its bar is at the page's edge. */
export const settingsScrollClass = "min-h-0 flex-1";

/**
 * What scrolls: room above the first section, and none held under the last. The save dock
 * makes its own room when it appears (the region ends above it); space kept for it all the
 * time was the one thing a tab that fits had to scroll to, and gave it a bar for nothing.
 * A tab's content slides in sideways; clipping that keeps the slide from giving the region
 * a sideways scroll.
 */
export const settingsScrollBodyClass = "overflow-x-clip pt-5";
