import type { AgentQualityOverview, AgentSuiteRun } from "@/lib/graphql/agent-quality";
import { describe, expect, it } from "vitest";
import { withPresets } from "../../agent-control-options";
import { caseOutcome, pointsChange, qualityHeroFacts } from "../quality-model";

const overview = {
  windowDays: 30,
  ratingsVisible: true,
  satisfaction: 0.82,
  ratings: 140,
  qualityScore: 0.87,
  agentsWithCases: 3,
  worstRegression: null,
} as unknown as AgentQualityOverview;

function run(overrides: Partial<AgentSuiteRun>): AgentSuiteRun {
  return {
    agentDefinitionId: "agdef_1",
    agentName: "Billing desk",
    qualityScore: 0.8,
    baselineScore: 0.86,
    regression: true,
    ...overrides,
  } as AgentSuiteRun;
}

describe("pointsChange", () => {
  it("is the score against the recent median in whole points, signed", () => {
    expect(pointsChange(run({ qualityScore: 0.8, baselineScore: 0.86 }))).toBe(-6);
    expect(pointsChange(run({ qualityScore: 0.9, baselineScore: 0.88 }))).toBe(2);
    expect(pointsChange(run({ qualityScore: 0.9, baselineScore: 0.9 }))).toBe(0);
  });

  it("compares nothing when either side is missing", () => {
    expect(pointsChange(run({ baselineScore: null }))).toBeNull();
    expect(pointsChange(run({ qualityScore: null }))).toBeNull();
    expect(pointsChange(null)).toBeNull();
  });
});

describe("qualityHeroFacts", () => {
  it("reads the score, what people liked and the agent that fell furthest", () => {
    expect(qualityHeroFacts({ ...overview, worstRegression: run({}) })).toEqual({
      score: 0.87,
      liked: 0.82,
      noCases: false,
      worst: { agentId: "agdef_1", agentName: "Billing desk", points: -6 },
    });
  });

  // Satisfaction a person may not read, or that nobody gave, is not spoken of.
  it("says nothing of what people liked when ratings are hidden or there are none", () => {
    expect(qualityHeroFacts({ ...overview, ratingsVisible: false }).liked).toBeNull();
    expect(qualityHeroFacts({ ...overview, ratings: 0, satisfaction: null }).liked).toBeNull();
  });

  it("knows when no agent has a golden set", () => {
    const facts = qualityHeroFacts({ ...overview, qualityScore: null, agentsWithCases: 0 });
    expect(facts.score).toBeNull();
    expect(facts.noCases).toBe(true);
    expect(facts.worst).toBeNull();
  });
});

describe("caseOutcome", () => {
  it("passes a completed case whose checks passed and fails one below the bar", () => {
    expect(
      caseOutcome({
        status: "Completed",
        checks: { passed: true, hardFailure: false, checks: [] },
      }),
    ).toBe("passed");
    expect(
      caseOutcome({
        status: "Completed",
        checks: { passed: false, hardFailure: false, checks: [] },
      }),
    ).toBe("failed");
  });

  it("fails a case that could not run, and leaves one not asked or not yet scored", () => {
    expect(caseOutcome({ status: "Failed", checks: null })).toBe("failed");
    expect(caseOutcome({ status: "Skipped", checks: null })).toBe("unasked");
    expect(caseOutcome({ status: "Pending", checks: null })).toBe("unasked");
    expect(caseOutcome({ status: "Completed", checks: null })).toBe("unasked");
  });
});

describe("withPresets", () => {
  it("adds a value set elsewhere to the presets, in order", () => {
    expect(withPresets([3, 5, 10], 7)).toEqual([3, 5, 7, 10]);
    expect(withPresets([3, 5, 10], 5)).toEqual([3, 5, 10]);
    expect(withPresets([3, 5, 10], Number.NaN)).toEqual([3, 5, 10]);
  });
});
