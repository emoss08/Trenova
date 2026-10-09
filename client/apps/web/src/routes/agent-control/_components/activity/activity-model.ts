import type { AgentActivitySummary } from "@/lib/graphql/agent-activity";
import type { AgentRunStatus } from "@trenova/graphql/generated/graphql";
import type { FieldFilter } from "@trenova/shared/types/data-table";

/** Activity's figures move with every run; half a minute old is still true. */
export const ACTIVITY_STALE_MS = 30_000;

const WORKING: ReadonlySet<AgentRunStatus> = new Set(["Pending", "GatheringContext", "Diagnosing"]);

/** Whether a run is still under way: it has not reached a decision, an end, or a failure. */
export function runIsWorking(status: AgentRunStatus): boolean {
  return WORKING.has(status);
}

export type ActivityFacts = {
  runs: number;
  failed: number;
  working: number;
  /** Null when the reader may not read proposals. */
  pending: number | null;
  /** How long the longest-waiting proposal has waited, in seconds. */
  oldestWaitSeconds: number | null;
  /** Null when the reader may not read exceptions. */
  openExceptions: number | null;
  /** Approved exactly as proposed over the decision window; null when none was decided. */
  approvedAsProposed: number | null;
  decisionWindowDays: number;
};

/** What Nova says at the head of Activity, from the summary, as of a moment. */
export function activityFacts(summary: AgentActivitySummary, now: number): ActivityFacts {
  return {
    runs: summary.runs,
    failed: summary.runsFailed,
    working: summary.runsWorking,
    pending: summary.pendingProposals ?? null,
    oldestWaitSeconds:
      summary.oldestPendingAt == null ? null : Math.max(0, now - summary.oldestPendingAt),
    openExceptions: summary.openExceptions ?? null,
    approvedAsProposed: summary.approvedAsProposed ?? null,
    decisionWindowDays: summary.decisionWindowDays,
  };
}

/** The runs that failed since the start of the day the summary counts from. */
export function failedTodayFilters(since: number): FieldFilter[] {
  return [
    { field: "status", operator: "eq", value: "Failed" },
    { field: "createdAt", operator: "gte", value: since },
  ];
}

export const PENDING_PROPOSAL_FILTERS: FieldFilter[] = [
  { field: "status", operator: "eq", value: "Pending" },
];

export const OPEN_EXCEPTION_FILTERS: FieldFilter[] = [
  { field: "resolutionState", operator: "in", value: ["Open", "InReview"] },
];
