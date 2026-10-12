import { refusalLabel, refusalTone, type ToolRefusal } from "@/components/assistant/activity";
import { humanizeKey } from "@/components/assistant/readable-values";
import type { AgentToolVerdictRow } from "@/lib/graphql/agent-scorecard";
import type { ToolVerdict } from "@/types/assistant";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";

export type VerdictReason = AgentToolVerdictRow["topReasons"][number];

/** How many of one tool's calls ended one way, and the reasons given most often. */
export type VerdictTally = {
  verdict: string;
  calls: number;
  reasons: VerdictReason[];
};

/** One tool's calls over the window, split by how each ended. */
export type ToolCallRecord = {
  toolName: string;
  calls: number;
  /** Calls that did not go through: refused by the runtime, failed, or of unknown outcome. */
  failing: number;
  /** `failing` over `calls`, 0 to 1. */
  failingShare: number;
  /** Most successful first, then refusals, then failures. */
  verdicts: VerdictTally[];
};

export type VerdictTone = "success" | "info" | "neutral" | "warning" | "danger";

/**
 * The order a tool's verdicts are drawn in: what went through, then what the
 * runtime turned away, then what broke. A verdict this client has not heard
 * of sorts after all of them.
 */
const VERDICT_ORDER: readonly ToolVerdict[] = [
  "ran",
  "proposed",
  "simulated",
  "denied",
  "invalid",
  "duplicate",
  "over_budget",
  "failed",
  "unknown",
];

const VERDICT_RANK: ReadonlyMap<string, number> = new Map(
  VERDICT_ORDER.map((verdict, index) => [verdict, index]),
);

/** The calls that did what they were asked: run, or filed as a proposal or a preview. */
const WENT_THROUGH: ReadonlySet<string> = new Set<ToolVerdict>(["ran", "proposed", "simulated"]);

const REFUSALS: ReadonlySet<string> = new Set<ToolRefusal>([
  "denied",
  "invalid",
  "duplicate",
  "over_budget",
]);

function isRefusal(verdict: string): verdict is ToolRefusal {
  return REFUSALS.has(verdict);
}

/** Whether a call that ended this way did what it was asked. */
export function wentThrough(verdict: string): boolean {
  return WENT_THROUGH.has(verdict);
}

function verdictRank(verdict: string): number {
  return VERDICT_RANK.get(verdict) ?? VERDICT_ORDER.length;
}

function byCallsThenReason(a: VerdictReason, b: VerdictReason): number {
  return b.calls - a.calls || a.reason.localeCompare(b.reason);
}

/** Folds a repeated reason into one line, so two rows for one verdict read as one. */
function mergeReasons(
  into: readonly VerdictReason[],
  added: readonly VerdictReason[],
): VerdictReason[] {
  const counts = new Map<string, number>();
  for (const reason of [...into, ...added]) {
    counts.set(reason.reason, (counts.get(reason.reason) ?? 0) + reason.calls);
  }

  return Array.from(counts, ([reason, calls]) => ({ reason, calls })).sort(byCallsThenReason);
}

/**
 * The scorecard's verdict rows gathered by tool, worst first: most calls that
 * did not go through, then the highest share of them, then the busiest tool,
 * then by name so the order is stable. A row with no calls says nothing and
 * is left out.
 */
export function toolCallRecords(rows: readonly AgentToolVerdictRow[]): ToolCallRecord[] {
  const byTool = new Map<string, ToolCallRecord>();

  for (const row of rows) {
    if (row.calls <= 0) {
      continue;
    }
    let record = byTool.get(row.toolName);
    if (!record) {
      record = { toolName: row.toolName, calls: 0, failing: 0, failingShare: 0, verdicts: [] };
      byTool.set(row.toolName, record);
    }
    record.calls += row.calls;
    if (!wentThrough(row.verdict)) {
      record.failing += row.calls;
    }
    const tally = record.verdicts.find((entry) => entry.verdict === row.verdict);
    if (tally) {
      tally.calls += row.calls;
      tally.reasons = mergeReasons(tally.reasons, row.topReasons);
    } else {
      record.verdicts.push({
        verdict: row.verdict,
        calls: row.calls,
        reasons: [...row.topReasons].sort(byCallsThenReason),
      });
    }
  }

  const records = Array.from(byTool.values());
  for (const record of records) {
    record.failingShare = record.failing / record.calls;
    record.verdicts.sort(
      (a, b) =>
        verdictRank(a.verdict) - verdictRank(b.verdict) ||
        b.calls - a.calls ||
        a.verdict.localeCompare(b.verdict),
    );
  }

  return records.sort(
    (a, b) =>
      b.failing - a.failing ||
      b.failingShare - a.failingShare ||
      b.calls - a.calls ||
      a.toolName.localeCompare(b.toolName),
  );
}

/** Every tool's calls added up, for the panel's one-line account. */
export function toolCallTotals(records: readonly ToolCallRecord[]): {
  calls: number;
  failing: number;
} {
  let calls = 0;
  let failing = 0;
  for (const record of records) {
    calls += record.calls;
    failing += record.failing;
  }

  return { calls, failing };
}

/**
 * A verdict in the reader's words. The runtime's refusals keep the words the
 * conversation already uses for them, so a refusal reads the same in a
 * thread and on the scorecard.
 */
export function verdictLabel(verdict: string, t: TranslateFn): string {
  if (isRefusal(verdict)) {
    return refusalLabel(verdict, t);
  }
  switch (verdict) {
    case "ran":
      return t("Ran");
    case "proposed":
      return t("Proposed");
    case "simulated":
      return t("Simulated");
    case "failed":
      return t("Failed");
    case "unknown":
      return t("Outcome unknown");
    default:
      return humanizeKey(verdict);
  }
}

/**
 * What a verdict says about the call, as a tone. A call that ran is the good
 * outcome; a proposal is under way with a person; a preview and a skipped
 * repeat harmed nothing; the other refusals are something to fix; a failure,
 * or a call nobody knows the outcome of, is the worst.
 */
export function verdictTone(verdict: string): VerdictTone {
  if (isRefusal(verdict)) {
    return refusalTone(verdict);
  }
  switch (verdict) {
    case "ran":
      return "success";
    case "proposed":
      return "info";
    case "simulated":
      return "neutral";
    default:
      return "danger";
  }
}

/** A share as a whole percent; a share above zero never rounds down to "0%". */
export function formatFailingShare(share: number): string {
  if (!Number.isFinite(share) || share <= 0) {
    return "0%";
  }

  return `${Math.max(1, Math.round(share * 100))}%`;
}
