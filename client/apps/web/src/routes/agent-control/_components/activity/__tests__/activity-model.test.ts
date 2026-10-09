import type { AgentActivitySummary } from "@/lib/graphql/agent-activity";
import { describe, expect, it } from "vitest";
import { activityFacts, failedTodayFilters, runIsWorking } from "../activity-model";

const summary: AgentActivitySummary = {
  since: 1_800_000_000,
  runs: 14,
  runsFailed: 2,
  runsWorking: 1,
  runsAwaiting: 3,
  pendingProposals: 4,
  oldestPendingAt: 1_800_003_600,
  openExceptions: 2,
  decisionWindowDays: 7,
  decided: 8,
  approvedAsProposed: 0.75,
};

describe("activityFacts", () => {
  it("reads the day's runs, what waits, and how long the oldest has waited", () => {
    expect(activityFacts(summary, 1_800_007_200)).toEqual({
      runs: 14,
      failed: 2,
      working: 1,
      pending: 4,
      oldestWaitSeconds: 3600,
      openExceptions: 2,
      approvedAsProposed: 0.75,
      decisionWindowDays: 7,
    });
  });

  // What the reader may not see is not spoken of as nothing.
  it("keeps counts the reader may not read apart from zero", () => {
    const facts = activityFacts(
      {
        ...summary,
        pendingProposals: null,
        oldestPendingAt: null,
        openExceptions: null,
        decided: null,
        approvedAsProposed: null,
      },
      1_800_007_200,
    );

    expect(facts.pending).toBeNull();
    expect(facts.oldestWaitSeconds).toBeNull();
    expect(facts.openExceptions).toBeNull();
    expect(facts.approvedAsProposed).toBeNull();
  });

  it("never says a proposal waited less than nothing", () => {
    expect(activityFacts(summary, 1_800_000_000).oldestWaitSeconds).toBe(0);
  });
});

describe("runIsWorking", () => {
  it("is true only before a run reaches a decision, an end or a failure", () => {
    expect(runIsWorking("Pending")).toBe(true);
    expect(runIsWorking("GatheringContext")).toBe(true);
    expect(runIsWorking("Diagnosing")).toBe(true);
    expect(runIsWorking("AwaitingDecision")).toBe(false);
    expect(runIsWorking("Completed")).toBe(false);
    expect(runIsWorking("Failed")).toBe(false);
  });
});

describe("failedTodayFilters", () => {
  it("narrows the runs to those that failed since the start of the day", () => {
    expect(failedTodayFilters(1_800_000_000)).toEqual([
      { field: "status", operator: "eq", value: "Failed" },
      { field: "createdAt", operator: "gte", value: 1_800_000_000 },
    ]);
  });
});
