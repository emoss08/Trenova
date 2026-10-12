import type {
  QueryKeyRoot,
  ResourceInvalidationEvent,
} from "@trenova/shared/hooks/realtime-patching";
import { threadsOf, type ThreadPages } from "@/lib/thread-list";

/**
 * Where a Desk case is cached: the case header of the open conversation and
 * the lists of conversations, which carry each case's summary.
 */
export const CASE_QUERY_ROOTS: QueryKeyRoot[] = [
  ["assistant", "case"],
  ["assistant", "threads"],
];

/**
 * The resources whose changes can move a case: the shipment, invoice or
 * dispute itself, and what its checklist reads (billing, paperwork, the
 * customer updates written on the shipment).
 */
const CASE_RECORD_RESOURCES = new Set([
  "shipments",
  "invoice",
  "invoice_dispute",
  "billing_queue",
  "document",
  "shipment_comment",
]);

/** The fields of an event's entity that can name the record a case is about. */
const RECORD_FIELDS = ["shipmentId", "invoiceId", "resourceId"] as const;

const boundByList = new WeakMap<ThreadPages, ReadonlySet<string>>();

/** The records the person's cases are about, read once per list of conversations. */
export function boundCaseRecords(list: ThreadPages | undefined): ReadonlySet<string> {
  if (!list) {
    return new Set();
  }
  const known = boundByList.get(list);
  if (known) {
    return known;
  }

  const bound = new Set<string>();
  for (const thread of threadsOf(list)) {
    if (thread.case && thread.subjectId) {
      bound.add(thread.subjectId);
    }
  }
  boundByList.set(list, bound);

  return bound;
}

/**
 * The records the person's cases are about across every list this tab holds:
 * the whole list and each shelf of the rail read so far.
 */
export function boundCaseRecordsIn(lists: readonly ThreadPages[]): ReadonlySet<string> {
  if (lists.length === 1) {
    return boundCaseRecords(lists[0]);
  }
  const bound = new Set<string>();
  for (const list of lists) {
    for (const record of boundCaseRecords(list)) {
      bound.add(record);
    }
  }

  return bound;
}

/** Whether a change is to a kind of record a case can be about or reads, before any case is looked for. */
export function mayTouchCase(event: ResourceInvalidationEvent): boolean {
  return CASE_RECORD_RESOURCES.has(event.resource);
}

/**
 * Whether a change to a record reaches one of the person's cases. The data
 * channel carries every change in the organization, so the case is refetched
 * only when the event names its record: by id, or through the shipment,
 * invoice or document resource the entity points at.
 */
export function touchesCase(event: ResourceInvalidationEvent, bound: ReadonlySet<string>): boolean {
  if (bound.size === 0 || !mayTouchCase(event)) {
    return false;
  }
  if (
    (event.recordId && bound.has(event.recordId)) ||
    (event.entityId && bound.has(event.entityId))
  ) {
    return true;
  }

  const entity = event.entity;
  if (!entity) {
    return false;
  }

  return RECORD_FIELDS.some((field) => {
    const value = entity[field];
    return typeof value === "string" && bound.has(value);
  });
}
