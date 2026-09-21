/**
 * The comparison the server stores on an evaluation, read from its JSON
 * field. Every count is present once the replay finished; a running or
 * failed evaluation has no comparison at all.
 */
export type ReplayVerdict =
  | "Agreed"
  | "Improved"
  | "Regressed"
  | "Repeated"
  | "Changed"
  | "Undecided"
  | "Added";

export type ReplayMatch = {
  toolName: string;
  originalProposalId?: string;
  originalParams?: Record<string, unknown>;
  replayParams?: Record<string, unknown>;
  originalOutcome?: string;
  verdict: ReplayVerdict;
  changes?: { field: string; from: string; to: string }[];
};

export type ReplayComparison = {
  matches: ReplayMatch[];
  agreed: number;
  improved: number;
  regressed: number;
  repeated: number;
  changed: number;
  added: number;
  undecided: number;
  decided: number;
  score: number | null;
};

const VERDICTS: readonly ReplayVerdict[] = [
  "Agreed",
  "Improved",
  "Regressed",
  "Repeated",
  "Changed",
  "Undecided",
  "Added",
];

function count(value: unknown): number {
  return typeof value === "number" && Number.isFinite(value) ? value : 0;
}

function text(value: unknown): string {
  return typeof value === "string" ? value : "";
}

/** Reads the stored comparison, or null when the evaluation has none. */
export function readComparison(value: unknown): ReplayComparison | null {
  if (!value || typeof value !== "object") {
    return null;
  }
  const raw = value as Record<string, unknown>;
  const matches = Array.isArray(raw.matches)
    ? raw.matches.flatMap((item) => {
        if (!item || typeof item !== "object") return [];
        const match = item as Record<string, unknown>;
        const verdict = match.verdict;
        if (typeof verdict !== "string" || !VERDICTS.includes(verdict as ReplayVerdict)) {
          return [];
        }
        return [
          {
            toolName: typeof match.toolName === "string" ? match.toolName : "",
            originalProposalId:
              typeof match.originalProposalId === "string" ? match.originalProposalId : undefined,
            originalParams:
              match.originalParams && typeof match.originalParams === "object"
                ? (match.originalParams as Record<string, unknown>)
                : undefined,
            replayParams:
              match.replayParams && typeof match.replayParams === "object"
                ? (match.replayParams as Record<string, unknown>)
                : undefined,
            originalOutcome:
              typeof match.originalOutcome === "string" ? match.originalOutcome : undefined,
            verdict: verdict as ReplayVerdict,
            changes: Array.isArray(match.changes)
              ? match.changes.flatMap((change) =>
                  change && typeof change === "object"
                    ? [
                        {
                          field: text((change as Record<string, unknown>).field),
                          from: text((change as Record<string, unknown>).from),
                          to: text((change as Record<string, unknown>).to),
                        },
                      ]
                    : [],
                )
              : undefined,
          } satisfies ReplayMatch,
        ];
      })
    : [];

  return {
    matches,
    agreed: count(raw.agreed),
    improved: count(raw.improved),
    regressed: count(raw.regressed),
    repeated: count(raw.repeated),
    changed: count(raw.changed),
    added: count(raw.added),
    undecided: count(raw.undecided),
    decided: count(raw.decided),
    score: typeof raw.score === "number" && Number.isFinite(raw.score) ? raw.score : null,
  };
}

/**
 * One line for the list: what the replay kept, dropped and added, in the
 * order a reader cares: the bad news first.
 */
export function summarizeComparison(
  comparison: ReplayComparison,
  t: (text: string, ...args: (string | number)[]) => string,
): string {
  const parts: string[] = [];
  if (comparison.regressed > 0) parts.push(t("{0} regressed", comparison.regressed));
  if (comparison.repeated > 0) parts.push(t("{0} repeated", comparison.repeated));
  if (comparison.improved > 0) parts.push(t("{0} improved", comparison.improved));
  if (comparison.agreed > 0) parts.push(t("{0} agreed", comparison.agreed));
  if (comparison.changed > 0) parts.push(t("{0} changed", comparison.changed));
  if (comparison.added > 0) parts.push(t("{0} added", comparison.added));
  if (comparison.undecided > 0) parts.push(t("{0} undecided", comparison.undecided));

  return parts.length === 0 ? t("Nothing to compare") : parts.join(" · ");
}
