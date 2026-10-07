import type { TriggerMode } from "@/types/assistant";

export const TRIGGER_LABELS: Record<TriggerMode, string> = {
  Chat: "Chat",
  Scheduled: "Scheduled",
  Event: "On an event",
  Continuous: "Continuous",
};

/** What each shelf of the roster is for, in a line. */
export const TRIGGER_NOTES: Record<TriggerMode, string> = {
  Chat: "Answer people in the assistant",
  Scheduled: "Run on a timetable",
  Event: "Run when something happens",
  Continuous: "Run again and again while enabled",
};
