import {
  blockedSourceEvents,
  journalSourceEventSchema,
  requiredSourceEvents,
  type JournalSourceEvent,
  type Recognition,
} from "@/types/accounting-control";

export {
  blockedSourceEvents,
  RECOGNITION_BY_BASIS,
  requiredSourceEvents,
  type Recognition,
} from "@/types/accounting-control";

/**
 * The auto-post events once recognition moves from one pair of policies to another: the
 * events the old policies required give way to the ones the new policies require, the
 * events the new revenue policy blocks are dropped, and every other choice is kept.
 */
export function sourceEventsForRecognition(
  events: readonly JournalSourceEvent[],
  from: Recognition,
  to: Recognition,
): JournalSourceEvent[] {
  const required = requiredSourceEvents(to);
  const dropped = new Set<JournalSourceEvent>([
    ...requiredSourceEvents(from).filter((entry) => !required.includes(entry)),
    ...blockedSourceEvents(to.revenue),
  ]);
  const kept = events.filter((entry) => !dropped.has(entry));
  const next = new Set<JournalSourceEvent>([...kept, ...required]);

  return journalSourceEventSchema.options.filter((entry) => next.has(entry));
}
