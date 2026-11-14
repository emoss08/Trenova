/**
 * The classes of a settings page's sections, shared by the page and its loading state so
 * the two cannot drift: what a section is sits on the left, its controls in a panel on the
 * right, and the sections stack with a gap between them.
 */
export const settingsSectionsClass = "flex flex-col gap-6";

export const settingsSectionClass =
  "grid grid-cols-1 gap-x-10 gap-y-3 lg:grid-cols-[minmax(0,19rem)_minmax(0,42rem)]";

/** Set down by the panel's own top padding, so the title starts level with the first control. */
export const settingsSectionHeadClass = "flex flex-col gap-1 lg:pt-4";

/**
 * The panel's fields. A field's own floor (`FormControl` keeps room for an error line) is
 * dropped here: the gap between fields is the rhythm, and a switch is not padded out.
 */
export const settingsPanelClass = "flex flex-col gap-4 [&_[data-slot=form-control]]:min-h-0";

/** Fields that belong to the switch above them, set in under its label. */
export const settingsNestedClass = "flex flex-col gap-4 pl-10";

/** The rule the tab row sits on. */
export const settingsTabRowClass = "overflow-x-auto border-b border-border";

export const settingsTabsClass = "gap-6";
