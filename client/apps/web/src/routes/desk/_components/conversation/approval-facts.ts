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
  /** Records the change would refuse as they stand, each with why. */
  refused: { label: string; reason: string }[];
  /** The whole change would be refused: the server's sentence for why. */
  wouldFail: string | null;
  /** Records that changed since the agent drafted the change. */
  changedSince: number;
};

/** The field a write over many records uses to say what happens to each. */
const OUTCOME_FIELD = "outcome";
const REFUSED_PREFIX = "Refused:";

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
    return { count: 0, resource: "", field: null, refused: [], wouldFail: null, changedSince: 0 };
  }
  const changes = preview.changes.filter((change) => !change.withheld);
  const count = changes.length + preview.withheldCount;
  const resource = changes[0]?.resource ?? "";
  const refused = changes.flatMap((change) => {
    const outcome = change.fields.find((field) => field.path === OUTCOME_FIELD);
    const text = typeof outcome?.after === "string" ? outcome.after : "";
    return text.startsWith(REFUSED_PREFIX)
      ? [
          {
            label: change.label || change.entityId || "",
            reason: text.slice(REFUSED_PREFIX.length).trim(),
          },
        ]
      : [];
  });
  const fail = preview.warnings.find((warning) => warning.code === "would_fail");
  const wouldFail = fail ? fail.message : null;
  const changedSince = changes.filter((change) =>
    change.fields.some((field) => field.changedSinceProposed),
  ).length;
  const extra = { refused, wouldFail, changedSince };
  const first = changes[0]?.fields.find((field) => !field.withheld && field.path !== OUTCOME_FIELD);
  if (!first) {
    return { count, resource, field: null, ...extra };
  }

  const same = changes.map((change) => change.fields.find((field) => field.path === first.path));
  if (same.some((field) => field === undefined)) {
    return { count, resource, field: null, ...extra };
  }
  const befores = new Set(same.map((field) => valueText(field?.before, field?.beforeRef, t)));
  const afters = new Set(same.map((field) => valueText(field?.after, field?.afterRef, t)));

  return {
    count,
    resource,
    ...extra,
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

/**
 * Refusals counted by reason, most common first: "22 locked by a dispute ·
 * 11 in a closed period". The reasons are the write's own sentences.
 */
export function refusalReasons(refused: readonly { reason: string }[]): [string, number][] {
  const counts = new Map<string, number>();
  for (const item of refused) {
    counts.set(item.reason, (counts.get(item.reason) ?? 0) + 1);
  }
  return [...counts.entries()].sort((a, b) => b[1] - a[1]);
}
