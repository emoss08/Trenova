import type { ProposalPreview } from "@/lib/graphql/agent-preview";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";

/** One field the change sets, said once for every record it touches. */
export type ApprovalField = {
  label: string;
  before: string;
  after: string;
};

/** What the compact approval card says about a change. */
export type ApprovalFacts = {
  /** How many records the change touches. */
  count: number;
  /** The kind of record, as the preview names it; empty when it names none. */
  resource: string;
  /** The field the change sets, when one field is set on every record. */
  field: ApprovalField | null;
};

type PreviewField = ProposalPreview["changes"][number]["fields"][number];

function valueText(
  value: unknown,
  ref: PreviewField["beforeRef"] | PreviewField["afterRef"] | undefined,
  t: TranslateFn,
): string {
  if (ref?.label) {
    return ref.label;
  }
  if (value === null || value === undefined || value === "") {
    return t("None");
  }
  if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }

  return JSON.stringify(value);
}

/**
 * The card's facts from the preview the server computed: the records the
 * change touches and, when every record has the same field set, that field
 * with its old and new value. A value that differs between records reads as
 * "Mixed" rather than as one of them.
 */
export function approvalFacts(preview: ProposalPreview | undefined, t: TranslateFn): ApprovalFacts {
  if (!preview) {
    return { count: 0, resource: "", field: null };
  }
  const changes = preview.changes.filter((change) => !change.withheld);
  const count = changes.length + preview.withheldCount;
  const resource = changes[0]?.resource ?? "";
  const first = changes[0]?.fields.find((field) => !field.withheld);
  if (!first) {
    return { count, resource, field: null };
  }

  const same = changes.map((change) => change.fields.find((field) => field.path === first.path));
  if (same.some((field) => field === undefined)) {
    return { count, resource, field: null };
  }
  const befores = new Set(same.map((field) => valueText(field?.before, field?.beforeRef, t)));
  const afters = new Set(same.map((field) => valueText(field?.after, field?.afterRef, t)));

  return {
    count,
    resource,
    field: {
      label: first.label,
      before: befores.size === 1 ? [...befores][0] : t("Mixed"),
      after: afters.size === 1 ? [...afters][0] : t("Mixed"),
    },
  };
}

/** "11 items", "15 invoices", "1 shipment": the records a change touches, counted. */
export function recordCount(resource: string, count: number, t: TranslateFn): string {
  switch (resource) {
    case "shipment":
      return t("{0, plural, one {# shipment} other {# shipments}}", count);
    case "invoice":
      return t("{0, plural, one {# invoice} other {# invoices}}", count);
    case "customer":
      return t("{0, plural, one {# customer} other {# customers}}", count);
    case "worker":
      return t("{0, plural, one {# driver} other {# drivers}}", count);
    default:
      return t("{0, plural, one {# item} other {# items}}", count);
  }
}
