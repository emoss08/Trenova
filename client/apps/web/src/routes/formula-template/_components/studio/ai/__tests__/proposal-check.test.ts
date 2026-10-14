import { describe, expect, it } from "vitest";
import type { ExpressionAnalysis } from "../../model/expression-analysis";
import { proposalCheck } from "../proposal-check";

type NameAnalysis = Pick<ExpressionAnalysis, "pending" | "errorCount" | "problems">;

function analysis(overrides: Partial<NameAnalysis> = {}): NameAnalysis {
  return { pending: false, errorCount: 0, problems: [], ...overrides };
}

function problems(...messages: string[]): ExpressionAnalysis["problems"] {
  return messages.map((message) => ({
    severity: "error",
    message,
  })) as unknown as ExpressionAnalysis["problems"];
}

const ran = { valid: true, error: "" };

describe("proposalCheck", () => {
  it("reports the engine's own failure before anything the editor found", () => {
    expect(proposalCheck({ valid: false, error: "unknown name foo (1:1)" }, analysis())).toEqual({
      kind: "engine",
      message: "unknown name foo (1:1)",
    });
  });

  it("reports the first error the editor found", () => {
    expect(
      proposalCheck(ran, analysis({ errorCount: 2, problems: problems("first", "second") })),
    ).toEqual({ kind: "problem", message: "first" });
  });

  it("is pending, not ok, while the schema has not arrived", () => {
    expect(proposalCheck(ran, analysis({ pending: true }))).toEqual({ kind: "pending" });
  });

  it("reports a syntax error even while names are unchecked", () => {
    expect(
      proposalCheck(
        ran,
        analysis({ pending: true, errorCount: 1, problems: problems("Expected )") }),
      ),
    ).toEqual({ kind: "problem", message: "Expected )" });
  });

  it("vouches for the names only once the schema has arrived", () => {
    expect(proposalCheck(ran, analysis())).toEqual({ kind: "ok" });
  });
});
