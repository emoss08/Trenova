import type { TriggerMode } from "@/types/assistant";
import type { BadgeVariant } from "@trenova/shared/types/badge";
import {
  CalendarClockIcon,
  MessageSquare01Icon,
  Repeat01Icon,
  Tool02Icon,
} from "@trenova/shared/components/icons";
import { defineLabels } from "@trenova/shared/i18n/labels";

export const TRIGGER_LABELS: Record<TriggerMode, string> = defineLabels({
  Chat: "Chat",
  Scheduled: "Scheduled",
  Event: "On an event",
  Continuous: "Continuous",
});

/** What each shelf of the roster is for, in a line. */
export const TRIGGER_NOTES: Record<TriggerMode, string> = defineLabels({
  Chat: "Answer people in the assistant",
  Scheduled: "Run on a timetable",
  Event: "Run when something happens",
  Continuous: "Run again and again while enabled",
});

export const TRIGGER_ICONS: Record<TriggerMode, typeof Tool02Icon> = {
  Chat: MessageSquare01Icon,
  Scheduled: CalendarClockIcon,
  Event: Tool02Icon,
  Continuous: Repeat01Icon,
};

/** A trigger is a category, so it takes an accent, never a tone. */
export const TRIGGER_BADGE: Record<TriggerMode, BadgeVariant> = {
  Chat: "info",
  Scheduled: "accent-teal",
  Event: "accent-amber",
  Continuous: "accent-violet",
};
