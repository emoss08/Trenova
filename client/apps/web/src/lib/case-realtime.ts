import type {
  QueryKeyRoot,
  ResourceInvalidationEvent,
} from "@trenova/shared/hooks/realtime-patching";
import type { AssistantThreadList } from "@/types/assistant";

/**
 * Where a Desk case is cached: the case header of the open conversation and
 * the rail, which shows each case's state.
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

const boundByList = new WeakMap<AssistantThreadList, ReadonlySet<string>>();

/** The records the person's cases are about, read once per thread list. */
export function boundCaseRecords(list: AssistantThreadList | undefined): ReadonlySet<string> {
  if (!list) {
    return new Set();
  }
  const known = boundByList.get(list);
  if (known) {
    return known;
  }

  const bound = new Set<string>();
  for (const thread of list.items) {
    if (thread.case && thread.subjectId) {
      bound.add(thread.subjectId);
    }
  }
  boundByList.set(list, bound);

  return bound;
}

/**
 * Whether a change to a record reaches one of the person's cases. The data
 * channel carries every change in the organization, so the case is refetched
 * only when the event names its record: by id, or through the shipment,
 * invoice or document resource the entity points at.
 */
export function touchesCase(event: ResourceInvalidationEvent, bound: ReadonlySet<string>): boolean {
  if (bound.size === 0 || !CASE_RECORD_RESOURCES.has(event.resource)) {
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
