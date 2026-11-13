import { defineLabels } from "@trenova/shared/i18n/labels";
import type { NotificationCounts, NotificationLens } from "@trenova/shared/types/notification";

/** A part of the panel a person can open: every lens but the driver portal's single list. */
export type NotificationPanelLens = Exclude<NotificationLens, "open">;

export const NOTIFICATION_PANEL_LENSES: readonly NotificationPanelLens[] = [
  "forYou",
  "team",
  "later",
  "done",
];

export const NOTIFICATION_LENS_LABELS: Record<NotificationPanelLens, string> = defineLabels({
  forYou: "For you",
  team: "Team",
  later: "Later",
  done: "Done",
});

/** The server stops counting here, so a count that reaches it is shown as "or more". */
const COUNT_CAP = 1000;

/** A tab's count as its label shows it, or null for a part that is not counted. */
export function lensCount(
  counts: NotificationCounts | undefined,
  lens: NotificationPanelLens,
): string | null {
  if (counts === undefined || lens === "done") return null;

  const count = counts[lens];
  return count >= COUNT_CAP ? `${COUNT_CAP - 1}+` : String(count);
}
