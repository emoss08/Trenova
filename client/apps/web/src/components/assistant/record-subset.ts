import type { PreviewRecordChange, ProposalPreview } from "@/lib/graphql/agent-preview";
import type { ProposalField } from "@/types/assistant";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { operationLabel } from "./proposal-preview/preview-format";

/** One record a subset field offers, as a row a person ticks or unticks. */
export type SubsetChoice = { id: string; label: string };

/** Past this many rows the list takes a filter. */
export const SUBSET_FILTER_THRESHOLD = 8;

/**
 * The value the agent proposed for a subset parameter: its ids as the tool
 * received them, in order, duplicates and all. What is kept is always drawn
 * from this, so a list the server bounded never drops the ids it did not
 * show.
 */
export function proposedSubsetIds(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value.filter((item): item is string => typeof item === "string");
}

/**
 * The rows a subset field lists: the server's choices, or the proposed ids by
 * themselves when a server sent none.
 */
export function subsetChoices(field: ProposalField, proposed: readonly string[]): SubsetChoice[] {
  if (field.choices && field.choices.length > 0) {
    return field.choices;
  }

  const seen = new Set<string>();
  const choices: SubsetChoice[] = [];
  for (const raw of proposed) {
    const id = raw.trim();
    if (id !== "" && !seen.has(id)) {
      seen.add(id);
      choices.push({ id, label: id });
    }
  }

  return choices;
}

/** The ids a draft keeps, read back from the text the form holds. */
export function keptSubsetIds(draft: string): Set<string> {
  return new Set(
    draft
      .split(",")
      .map((id) => id.trim())
      .filter((id) => id !== ""),
  );
}

/**
 * The draft for a set of kept ids: the proposed value with every dropped id
 * taken out, in the order proposed. A draft that keeps everything is the
 * proposal itself, so it is no change at all.
 */
export function subsetDraft(proposed: readonly string[], kept: ReadonlySet<string>): string {
  return proposed.filter((id) => kept.has(id.trim())).join(", ");
}

/**
 * How many of the proposed records are kept, named by what they are. Only
 * the ids the proposal carries count, so the total is the proposal's.
 */
export function subsetCountLabel(
  resource: string | null | undefined,
  kept: number,
  total: number,
  t: TranslateFn,
): string {
  if (resource === "shipment") {
    return total === 1 ? t("{0} of 1 shipment", kept) : t("{0} of {1} shipments", kept, total);
  }

  return total === 1 ? t("{0} of 1 record", kept) : t("{0} of {1} records", kept, total);
}

const OUTCOME_PATH = "outcome";

/**
 * What a preview says happens to one record, in a line: the tool's own
 * outcome when it states one, otherwise the fields it would change.
 */
export function recordOutcome(change: PreviewRecordChange, t: TranslateFn): string {
  const stated = change.fields.find((item) => item.path === OUTCOME_PATH && !item.withheld);
  if (typeof stated?.after === "string" && stated.after.trim() !== "") {
    return stated.after;
  }

  const changed = change.fields.filter((item) => !item.withheld).map((item) => item.label);
  if (change.operation !== "Update" || changed.length === 0) {
    return operationLabel(change.operation, t);
  }

  return t("Would change {0}", changed.join(", "));
}

/**
 * Each record's outcome by id, from the previews in the order given; a later
 * preview's word on a record replaces an earlier one. A withheld record names
 * no id, so it is never matched to a row.
 */
export function previewOutcomes(
  previews: readonly (ProposalPreview | null | undefined)[],
  t: TranslateFn,
): Map<string, string> {
  const outcomes = new Map<string, string>();
  for (const current of previews) {
    for (const change of current?.changes ?? []) {
      if (change.withheld || !change.entityId) {
        continue;
      }
      outcomes.set(change.entityId, recordOutcome(change, t));
    }
  }

  return outcomes;
}

/** The rows whose label or id holds the filter, case aside. */
export function filterSubsetChoices(
  choices: readonly SubsetChoice[],
  filter: string,
): readonly SubsetChoice[] {
  const needle = filter.trim().toLowerCase();
  if (needle === "") {
    return choices;
  }

  return choices.filter(
    (choice) =>
      choice.label.toLowerCase().includes(needle) || choice.id.toLowerCase().includes(needle),
  );
}
