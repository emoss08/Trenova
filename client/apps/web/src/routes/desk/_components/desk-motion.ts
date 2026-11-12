import type { RailView } from "./rail/desk-rail";

/** Joins a region's name to what leaves it unnamed, and so still. */
function named(region: string): string {
  // i18n-ignore: class names, never shown to a person
  return [
    region,
    "motion-reduce:[view-transition-name:none]",
    "in-[.dk-calm]:[view-transition-name:none]",
  ].join(" ");
}

/**
 * Names the rail's body and the page as the regions a view transition moves
 * when the Desk goes from one of its views to another: the chats, the cases,
 * Triage (desk-motion.css). Each view has a name of its own, so a navigation
 * inside a view keeps the name and nothing moves, and one that changes the
 * view has a region leaving and another arriving. Left unnamed under reduced
 * motion, by the system or by the Desk's own setting.
 */
export const RAIL_TRANSITION_CLASS: Record<RailView, string> = {
  // i18n-ignore: class names, never shown to a person
  chats: named("[view-transition-name:dk-rail-chats]"),
  // i18n-ignore: class names, never shown to a person
  cases: named("[view-transition-name:dk-rail-cases]"),
  // i18n-ignore: class names, never shown to a person
  triage: named("[view-transition-name:dk-rail-triage]"),
};

export const PAGE_TRANSITION_CLASS: Record<RailView, string> = {
  // i18n-ignore: class names, never shown to a person
  chats: named("[view-transition-name:dk-page-chats]"),
  // i18n-ignore: class names, never shown to a person
  cases: named("[view-transition-name:dk-page-cases]"),
  // i18n-ignore: class names, never shown to a person
  triage: named("[view-transition-name:dk-page-triage]"),
};
