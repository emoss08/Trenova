import { describe, expect, it } from "vitest";
import { checkExpression, engineErrorRange, engineErrorText } from "../expr-check";
import { parseExpression } from "../expr-parser";
import { buildKnownIdentifiers } from "../known-identifiers";
import { describedShipment } from "@/components/formula-editor/__tests__/described-shipment";

const known = buildKnownIdentifiers(describedShipment, [
  { name: "fuelPct", type: "Number", description: "", required: false },
  { name: "hazmatFee", type: "Number", description: "", required: false },
]);

const codes = (expression: string) =>
  checkExpression(expression, known).diagnostics.map((d) => `${d.severity}:${d.code}`);

describe("parseExpression", () => {
  it("parses let lines before the result", () => {
    const program = parseExpression("let a = 1;\nlet b = a * 2;\na + b");
    expect(program.diagnostics).toEqual([]);
    expect(program.statements.map((s) => s.kind)).toEqual(["let", "let", "result"]);
    expect(program.statements[1]).toMatchObject({ name: "b", from: 11 });
  });

  it("anchors an unclosed paren on the paren", () => {
    const program = parseExpression("round(baseRate * 2");
    expect(program.diagnostics[0]).toMatchObject({ from: 5, to: 6 });
    expect(program.diagnostics[0].message).toContain("never closed");
  });

  it("flags a let without its semicolon with a fix", () => {
    const program = parseExpression("let a = 1\na + 1");
    const missing = program.diagnostics.find((d) => d.fix?.insert === ";");
    expect(missing).toBeDefined();
  });

  it("offers == for a stray =", () => {
    const program = parseExpression("totalStops = 2 ? 1 : 0");
    expect(program.diagnostics.some((d) => d.fix?.insert === "==")).toBe(true);
  });

  it("parses the word operators, not, and the ternary", () => {
    const program = parseExpression('customer.name startsWith "A" and not (hasHazmat) ? 1 : 0');
    expect(program.diagnostics).toEqual([]);
    expect(program.statements[0].value.kind).toBe("ternary");
  });

  it("asks for parentheses around ?? mixed with an operator", () => {
    const program = parseExpression("weight ?? 0 * 2");
    expect(program.diagnostics[0]?.message).toContain("parentheses");
  });
});

describe("checkExpression", () => {
  it("accepts a sound formula", () => {
    expect(
      codes(
        "let fuel = baseRate * totalDistance * (fuelPct / 100);\nbaseRate * totalDistance + fuel",
      ),
    ).toEqual([]);
  });

  it("flags an unknown name with a declare fix and a suggestion", () => {
    const { diagnostics } = checkExpression("baseRaet * weekendFee", known);
    expect(diagnostics.map((d) => d.code)).toEqual(["unknown-variable", "unknown-variable"]);
    expect(diagnostics[0].fixes[0]).toMatchObject({
      kind: "replace",
      changes: [{ insert: "baseRate" }],
    });
    expect(diagnostics[1].fixes.at(-1)).toMatchObject({
      kind: "declare",
      name: "weekendFee",
      type: "Number",
    });
  });

  it("refuses now() and points at ratingTime with a fix", () => {
    const { diagnostics } = checkExpression("now().Year() == 2026 ? 10 : 0", known);
    expect(diagnostics).toHaveLength(1);
    expect(diagnostics[0]).toMatchObject({ severity: "error", code: "disabled-function", from: 0 });
    expect(diagnostics[0].message).toBe(
      "`now()` isn't available in formulas: a formula must price a shipment the same way every time it runs. Use `ratingTime`, the date the contract terms are read at",
    );
    expect(diagnostics[0].fixes).toEqual([
      {
        kind: "replace",
        label: "Change to ratingTime",
        changes: [{ from: 0, to: 5, insert: "ratingTime" }],
      },
    ]);
  });

  it("refuses the functions the engine leaves out, without a suggestion", () => {
    for (const expression of ["len(toJSON(totalStops))", 'fromJSON("1")', 'len(repeat("x", 3))']) {
      const { diagnostics } = checkExpression(expression, known);
      expect(diagnostics.map((d) => d.code)).toEqual(["disabled-function"]);
      expect(diagnostics[0].message).toContain("isn't available in formulas");
      expect(diagnostics[0].fixes).toEqual([]);
    }
  });

  it("infers a flag type for is/has names", () => {
    const { diagnostics } = checkExpression("isTeam ? 1 : 0", known);
    expect(diagnostics[0].fixes.at(-1)).toMatchObject({ kind: "declare", type: "Boolean" });
  });

  it("flags an unknown field under a known root with a suggestion", () => {
    const { diagnostics } = checkExpression("customer.nmae", known);
    expect(diagnostics[0]).toMatchObject({ code: "unknown-field" });
    expect(diagnostics[0].message).toContain("customer.name");
  });

  it("checks arity against the signature", () => {
    expect(codes("pow(2)")).toEqual(["error:arity"]);
    expect(codes("max(1, 2, 3)")).toEqual([]);
    expect(codes("clamp(1, 2)")).toEqual(["error:arity"]);
  });

  it("refuses arithmetic on a flag and suggests the ternary", () => {
    const { diagnostics } = checkExpression("hasHazmat * hazmatFee", known);
    expect(diagnostics[0].code).toBe("type");
    expect(diagnostics[0].message).toContain("(hasHazmat ? amount : 0)");
    expect(diagnostics[0].fixes[0]).toMatchObject({ changes: [{ insert: "(hasHazmat ? 1 : 0)" }] });
  });

  it("requires a true/false test before ?", () => {
    const { diagnostics } = checkExpression("totalStops ? 1 : 0", known);
    expect(diagnostics[0].message).toContain("totalStops > 0");
  });

  it("warns on a nullable field read without a guard, once, with a fix", () => {
    const { diagnostics } = checkExpression("weight * 2 + weight", known);
    expect(diagnostics.map((d) => d.code)).toEqual(["nullable"]);
    expect(diagnostics[0].fixes[0]).toMatchObject({ changes: [{ insert: "(weight ?? 0)" }] });
  });

  it("treats ??, coalesce and a nil test as guards", () => {
    expect(codes("(weight ?? 0) * 2")).toEqual([]);
    expect(codes("coalesce(weight, 0) * 2")).toEqual([]);
    expect(codes("weight == nil ? 0 : weight * 2")).toEqual([]);
  });

  it("reports let problems", () => {
    expect(codes("let a = 1;\nlet a = 2;\na")).toContain("error:let-duplicate");
    expect(codes("let a = 1;\nbaseRate")).toContain("warning:let-unused");
    expect(codes("let a = 1;\nlet b = a;")).toContain("error:let-last");
    expect(codes("b + 1;\nlet b = 2;\nb")).toContain("error:let-before-definition");
  });

  it("requires the formula to come to an amount", () => {
    expect(codes("hasHazmat")).toEqual(["error:result-type"]);
    expect(codes("customer.name")).toEqual(["error:result-type"]);
  });

  it("collects what the expression reads", () => {
    const analysis = checkExpression(
      "let fuel = fuelPct;\nbaseRate * totalDistance + fuel + round(1)",
      known,
    );
    expect(analysis.reads).toEqual(["fuelPct", "baseRate", "totalDistance"]);
    expect([...analysis.functionCalls]).toEqual(["round"]);
    expect(analysis.resultType).toBe("Number");
  });

  it("does not flag expr builtins or the pipe", () => {
    expect(codes("len(customer.name) | abs()")).toEqual([]);
  });
});

describe("arity", () => {
  it("takes each function's arguments from the engine's description", () => {
    const messages = (expression: string) =>
      checkExpression(expression, known)
        .diagnostics.filter((d) => d.code === "arity")
        .map((d) => d.message);

    expect(messages("round(1, 2, 3)")).toEqual([
      "round(x, decimals?) takes 1 to 2 arguments; 3 were given",
    ]);
    expect(messages("max()")).toEqual(["max(...values) takes at least 1 argument; 0 were given"]);
    expect(messages("count([1, 2])")).toEqual([]);
    expect(messages("reverse([1], [2])")).toEqual(["reverse(list) takes 1 argument; 2 were given"]);
  });
});

describe("engine errors", () => {
  it("maps the engine's (line:column) onto the expression", () => {
    const expression = "let a = 1;\na + weekendFee";
    expect(engineErrorRange("unknown name weekendFee (2:5)", expression)).toEqual({
      from: 15,
      to: 25,
    });
  });

  it("strips the wrapping and the position from the message", () => {
    expect(
      engineErrorText(
        "schema error for 'x' during compile: unknown name weekendFee (2:5)\n | a + weekendFee\n | ....^",
      ),
    ).toBe("Unknown name weekendFee");
  });
});
