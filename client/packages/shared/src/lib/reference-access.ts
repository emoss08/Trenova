import { defineLabels, translateLabel } from "@trenova/shared/i18n/labels";
import type { ReferenceKind } from "@trenova/shared/types/generated/reference-views";

/**
 * What a kind of record is called in "You don't have access to {0}": a plural noun read
 * mid-sentence, so it starts in lower case. A kind not listed here reads as its name in words
 * (see docs/engineering/reference-views.md).
 */
const REFERENCE_KIND_ACCESS_LABELS = defineLabels({
  agent_definition: "agents",
  carrier: "carriers",
  customer: "customers",
  driver_settlement: "driver settlements",
  escrow_account: "escrow accounts",
  fiscal_period: "fiscal periods",
  fleet_code: "fleet codes",
  fuel_card: "fuel cards",
  invoice: "invoices",
  invoice_dispute: "invoice disputes",
  location: "locations",
  location_category: "location categories",
  pay_advance: "pay advances",
  performance_review: "performance reviews",
  service_failure_reason_code: "service failure reason codes",
  shift_template: "shift templates",
  shipment: "shipments",
  tractor: "tractors",
  trailer: "trailers",
  user: "users",
  worker: "workers",
  worker_checklist: "worker checklists",
  worker_credential_type: "credential types",
} satisfies Partial<Record<ReferenceKind, string>>);

/** The kind of record in words, as it reads in the withheld-field tooltip. */
export function referenceKindAccessLabel(kind: ReferenceKind): string {
  if (Object.hasOwn(REFERENCE_KIND_ACCESS_LABELS, kind)) {
    return REFERENCE_KIND_ACCESS_LABELS[kind as keyof typeof REFERENCE_KIND_ACCESS_LABELS];
  }
  return translateLabel(kind.replaceAll("_", " "));
}

/**
 * A field of a record an operation reads as its root, or as the caller's own under a
 * self-scoped `my*` field (selected without `@requiresRead`).
 *
 * The server withholds a field outside a kind's reference view only where the record is
 * linked from another one, answering it `null`; the schema marks such a field nullable for
 * every reader because the same field is withheld there. Where the record is the root of the
 * read, its own permission check covers every field, so `null` here is a broken contract,
 * not an access state, and it fails loudly rather than drawing a zero or an empty value.
 */
export function rootField<T>(value: T | null | undefined, field: string): T {
  if (value === null || value === undefined) {
    throw new Error(`${field} was withheld on a record read as its operation's root`);
  }
  return value;
}

/** A root record with the named fields known to be present (see `rootField`). */
export type RootFields<T, K extends keyof T> = Omit<T, K> & { [P in K]-?: NonNullable<T[P]> };

/**
 * `rootField` for several fields of one root record at once: the record with each named field
 * checked and typed as present. `typeName` names the GraphQL type in the error.
 */
export function rootFields<T extends object, const K extends keyof T & string>(
  record: T,
  typeName: string,
  fields: readonly K[],
): RootFields<T, K> {
  const checked: Partial<Record<K, unknown>> = {};
  for (const field of fields) {
    checked[field] = rootField(record[field], `${typeName}.${field}`);
  }
  return { ...record, ...checked } as RootFields<T, K>;
}
