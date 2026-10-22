import type { FormulaDiagnostic } from "@trenova/shared/types/formula-template";
import { describe, expect, it } from "vitest";
import type { ExpressionAnalysis } from "../../model/expression-analysis";
import { proposalCheck } from "../proposal-check";

type NameAnalysis = Pick<ExpressionAnalysis, "pending" | "problems">;

function analysis(overrides: Partial<NameAnalysis> = {}): NameAnalysis {
  return { pending: false, problems: [], ...overrides };
}

function finding(
  severity: FormulaDiagnostic["severity"],
  message: string,
  from = 0,
): FormulaDiagnostic {
  return { from, to: from + 1, severity, code: "syntax", message, fixes: [] };
}

const ran = { valid: true, error: "" };

describe("proposalCheck", () => {
  it("reports the engine's own failure before anything the editor found", () => {
    expect(
      proposalCheck(
        { valid: false, error: "unknown name foo" },
        analysis({ problems: [finding("error", "first")] }),
      ),
    ).toEqual({ kind: "engine", message: "unknown name foo" });
  });

  it("reports the first error the check found, past any warning before it", () => {
    const first = finding("error", "first", 4);
    expect(
      proposalCheck(
        ran,
        analysis({
          problems: [finding("warning", "unused", 0), first, finding("error", "second", 9)],
        }),
      ),
    ).toEqual({ kind: "problem", problem: first });
  });

  it("is pending, not ok, while the schema has not arrived", () => {
    expect(proposalCheck(ran, analysis({ pending: true }))).toEqual({ kind: "pending" });
  });

  it("reports a syntax error even while names are unchecked", () => {
    const syntax = finding("error", "'{0}' needs a value on its right");
    expect(proposalCheck(ran, analysis({ pending: true, problems: [syntax] }))).toEqual({
      kind: "problem",
      problem: syntax,
    });
  });

  it("vouches for the names only once the schema has arrived", () => {
    expect(proposalCheck(ran, analysis({ problems: [finding("warning", "unused")] }))).toEqual({
      kind: "ok",
    });
  });
});
