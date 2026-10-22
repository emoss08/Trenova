import {
  laneCoverage,
  laneKeys,
  laneSpecificities,
  parseLaneKeys,
  type LaneRuleInput,
} from "@trenova/shared/rules/lane";
import { rulesSync } from "@trenova/shared/rules/runtime";
import { corpusCase, expectCorpusOutput } from "@trenova/shared/rules/__tests__/corpus";
import { describe, expect, it } from "vitest";

function rulesOf(name: string) {
  const testCase = corpusCase("lane", name);
  return { testCase, rules: (testCase.input as { rules: LaneRuleInput[] }).rules };
}

describe("laneKeys", () => {
  it("keys anywhere, record prefixes, folded cities, postal prefixes and radius lanes", () => {
    const { testCase, rules } = rulesOf(
      "TS laneKeyPreview: anywhere, prefixes, folded cities, postal prefixes and radius",
    );
    expectCorpusOutput(testCase, { rules: laneKeys(rulesSync(), rules) });
  });

  // The stored key must match what the engine stores, so a postal code keeps its inner spaces
  // and a leading dash, where the old browser copy stripped them.
  it("keeps a postal code's inner spaces and leading dash, as the engine stores them", () => {
    const { testCase, rules } = rulesOf(
      "rategeo: postal codes keep a leading dash, their inner spaces and fold case like Go",
    );
    expectCorpusOutput(testCase, { rules: laneKeys(rulesSync(), rules) });
  });

  it("sends a half-typed rule as the fields the lane rules read", () => {
    const [keys] = laneKeys(rulesSync(), [
      {
        originScopeType: "State",
        originScopeValue: "us_tx",
        priority: Number.NaN,
        effectiveFrom: 1.5,
        minWeight: "abc",
      },
    ]);
    expect(keys).toEqual({
      origin: "ST:us_tx",
      destination: "ANY",
      key: "ST:us_tx>ANY",
      stored: "ST:us_tx>ANY",
      reverse: null,
      segments: [
        { text: "ST:us_tx", scope: "State", id: "us_tx", prefix: "ST:", suffix: "" },
        { text: "ANY" },
      ],
    });
  });
});

describe("laneSpecificities", () => {
  it("scores geography above every condition combined", () => {
    const { testCase, rules } = rulesOf(
      "TS laneSpecificity: anywhere is zero, the ends weigh the same, conditions never outweigh one step of geography",
    );
    expectCorpusOutput(testCase, { scores: laneSpecificities(rulesSync(), rules) });
  });

  it("reads a blank bound as unset", () => {
    const { testCase, rules } = rulesOf(
      "specificity: a blank or null bound is unset, and null flags are off",
    );
    expectCorpusOutput(testCase, { scores: laneSpecificities(rulesSync(), rules) });
  });
});

describe("parseLaneKeys", () => {
  it("splits stored keys into record and literal ends", () => {
    const testCase = corpusCase(
      "lane",
      "parse (TS laneKeyScopeIds, laneKeyStringDisplay): ID-backed ends and literal ends",
    );
    const { keys } = testCase.input as { keys: string[] };
    expectCorpusOutput(testCase, {
      keys: parseLaneKeys(rulesSync(), keys).map((segments) => ({ segments })),
    });
  });
});

describe("laneCoverage", () => {
  it("names a duplicate lane with the message to show", () => {
    const { testCase, rules } = rulesOf(
      "coverage: two equally narrow lanes leave the winner to a tie-break",
    );
    expectCorpusOutput(testCase, { issues: laneCoverage(rulesSync(), rules) });
  });

  // The old browser check flagged whichever lane came second, even when it had the higher
  // priority and was the one that applies.
  it("shadows the lower-priority lane wherever it sits", () => {
    const { testCase, rules } = rulesOf(
      "coverage (fixes TS): the lower-priority lane is the shadowed one wherever it sits",
    );
    expectCorpusOutput(testCase, { issues: laneCoverage(rulesSync(), rules) });
  });

  it("does not compare lanes whose effective windows never overlap", () => {
    const { testCase, rules } = rulesOf(
      "coverage: lanes whose effective windows do not overlap never compete",
    );
    expectCorpusOutput(testCase, { issues: laneCoverage(rulesSync(), rules) });
  });

  it("meets a bidirectional lane with the lane written the other way round", () => {
    const { testCase, rules } = rulesOf(
      "coverage: a bidirectional lane meets the lane written the other way round",
    );
    expectCorpusOutput(testCase, { issues: laneCoverage(rulesSync(), rules) });
  });
});
