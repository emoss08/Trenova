import { describe, expect, it } from "vitest";
import type { ExtractionRollout } from "@/lib/graphql/extraction-rollout";
import {
  rolloutDraftOf,
  rolloutInput,
  rolloutProblems,
  rolloutState,
  type RolloutDraft,
} from "../rollout-model";

function draft(overrides: Partial<RolloutDraft> = {}): RolloutDraft {
  return {
    enabled: true,
    providerId: "aip_candidate",
    percent: "10",
    maxAccuracyDropPoints: "5",
    maxRejectionIncreasePoints: "10",
    ...overrides,
  };
}

function rollout(overrides: Partial<ExtractionRollout> = {}): ExtractionRollout {
  return {
    enabled: true,
    providerId: "aip_candidate",
    percent: 10,
    maxAccuracyDropPoints: 5,
    maxRejectionIncreasePoints: 10,
    serving: true,
    startedAt: 1_790_000_000,
    haltedAt: null,
    haltReason: null,
    haltCandidateRate: 0,
    haltBaselineRate: 0,
    updatedById: null,
    version: 2,
    updatedAt: 1_790_000_000,
    ...overrides,
  } as ExtractionRollout;
}

describe("rolloutProblems", () => {
  it("accepts a rollout that is off with no candidate", () => {
    expect(rolloutProblems(draft({ enabled: false, providerId: "" }))).toEqual({});
  });

  it("needs a candidate only while the rollout is on", () => {
    expect(rolloutProblems(draft({ providerId: "" }))).toEqual({
      providerId: "Choose the AI provider to roll out",
    });
  });

  it.each([
    ["percent", "0"],
    ["percent", "101"],
    ["percent", "2.5"],
    ["percent", ""],
    ["maxAccuracyDropPoints", "0"],
    ["maxAccuracyDropPoints", "51"],
    ["maxRejectionIncreasePoints", "0"],
    ["maxRejectionIncreasePoints", "101"],
  ] as const)("refuses %s of %s", (field, value) => {
    const problems = rolloutProblems(draft({ [field]: value }));
    expect(Object.keys(problems)).toEqual([field]);
  });
});

describe("rolloutInput", () => {
  it("turns a valid draft into the mutation input", () => {
    expect(rolloutInput(draft({ percent: " 25 " }), 4)).toEqual({
      enabled: true,
      providerId: "aip_candidate",
      percent: 25,
      maxAccuracyDropPoints: 5,
      maxRejectionIncreasePoints: 10,
      version: 4,
    });
  });

  it("sends no candidate when none is chosen", () => {
    expect(rolloutInput(draft({ enabled: false, providerId: "" }), 1)?.providerId).toBeNull();
  });

  it("has nothing to send while the draft has problems", () => {
    expect(rolloutInput(draft({ percent: "0" }), 1)).toBeNull();
  });

  it("round-trips a saved rollout", () => {
    const saved = rollout({ percent: 40 });
    expect(rolloutInput(rolloutDraftOf(saved), saved.version)).toMatchObject({
      percent: 40,
      version: 2,
    });
  });
});

describe("rolloutState", () => {
  it("names each state", () => {
    expect(rolloutState(rollout({ providerId: null, serving: false, enabled: false }))).toBe(
      "unset",
    );
    expect(rolloutState(rollout({ serving: false, enabled: false }))).toBe("off");
    expect(rolloutState(rollout())).toBe("serving");
    expect(rolloutState(rollout({ serving: false, haltedAt: 1_790_000_500 }))).toBe("halted");
  });
});
