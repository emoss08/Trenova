import type { DeskActivityDetail, DeskActivityEntry } from "@/lib/graphql/desk-attention";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";

/** The records a line of the ledger names, the rest counted. */
export function recordLine(t: TranslateFn, entry: DeskActivityEntry): string {
  const named = entry.records.filter((record) => record.readable && record.label !== "");
  const unnamed = entry.records.length - named.length + entry.recordsMore;
  const names = named.map((record) => record.label).join(", ");
  if (names === "") {
    return unnamed > 0 ? t("{0, plural, one {# record} other {# records}}", unnamed) : "";
  }
  return unnamed > 0
    ? t("{0} and {1, plural, one {# more} other {# more}}", names, unnamed)
    : names;
}

/** Who stood behind a write: the person who approved it, or the rule it ran under. */
export function decidedLine(t: TranslateFn, entry: DeskActivityEntry): string {
  switch (entry.outcome) {
    case "Approved":
      return entry.decidedBy ? t("Approved by {0}", entry.decidedBy.name) : t("Approved");
    case "OnItsOwn":
      return t("Ran under AI control");
    case "Previewed":
      return t("Previewed, never made");
    default:
      return entry.detail;
  }
}

/** Where a line of the ledger opens: the write, with why it could run and what it changed. */
export function handledPath(entry: Pick<DeskActivityEntry, "id">): string {
  return `/desk/handled/${encodeURIComponent(entry.id)}`;
}

export type RunTrailStep = { key: string; at: number; text: string };

/**
 * A line of the ledger as it happened: the agent proposing the writes, what let
 * them run, and the server settling them. Every time is the server's.
 */
export function runTrail(t: TranslateFn, detail: DeskActivityDetail): RunTrailStep[] {
  const { entry, changes } = detail;
  if (changes.length === 0) {
    return [];
  }
  const agent = entry.agent?.name ?? t("The agent");
  const proposedAt = Math.min(...changes.map((change) => change.proposedAt));
  const settledAt = Math.max(...changes.map((change) => change.at));
  const failed = changes.filter((change) => change.outcome === "Failed").length;
  const steps: RunTrailStep[] = [
    {
      key: "proposed",
      at: proposedAt,
      text: t("{0} proposed {1, plural, one {# change} other {# changes}}", agent, changes.length),
    },
  ];
  switch (entry.outcome) {
    case "OnItsOwn":
      steps.push({
        key: "rule",
        at: proposedAt,
        text: t("AI control let it run without asking"),
      });
      break;
    case "Approved":
      steps.push({
        key: "approved",
        at: settledAt,
        text: entry.decidedBy
          ? t("{0} approved it", entry.decidedBy.name)
          : t("A person approved it"),
      });
      break;
    default:
      break;
  }
  steps.push({
    key: "settled",
    at: settledAt,
    text:
      entry.outcome === "Previewed"
        ? t("Previewed in shadow; nothing was changed")
        : failed === changes.length
          ? t("The server refused it; nothing changed")
          : failed > 0
            ? t(
                "The server confirmed {0} of {1}; the rest did not go through",
                changes.length - failed,
                changes.length,
              )
            : t(
                "The server confirmed {0, plural, one {the change} other {all # changes}}",
                changes.length,
              ),
  });
  return steps;
}
