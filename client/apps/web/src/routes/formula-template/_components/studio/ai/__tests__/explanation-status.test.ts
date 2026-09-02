import { describe, expect, it } from "vitest";
import { explanationStatus } from "../explanation-status";

describe("explanationStatus", () => {
  it("is none until an explanation exists", () => {
    expect(explanationStatus({ expression: "a", explainedFor: null, hasExplanation: false })).toBe(
      "none",
    );
  });

  it("is fresh while the expression matches what was explained", () => {
    expect(explanationStatus({ expression: "a", explainedFor: "a", hasExplanation: true })).toBe(
      "fresh",
    );
  });

  it("is stale once the expression moves on, ignoring whitespace", () => {
    expect(
      explanationStatus({ expression: "a * 2", explainedFor: "a", hasExplanation: true }),
    ).toBe("stale");
    expect(
      explanationStatus({ expression: "  a  ", explainedFor: "a", hasExplanation: true }),
    ).toBe("fresh");
  });
});
