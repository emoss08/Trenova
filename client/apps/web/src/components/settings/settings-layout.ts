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

/** The rule the tab row sits on. */
export const settingsTabRowClass = "overflow-x-auto border-b border-border";

export const settingsTabsClass = "gap-5";
