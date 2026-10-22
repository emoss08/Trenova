import { knownFor } from "@/components/formula-editor/__tests__/formula-fixtures";
import { buildKnownIdentifiers } from "@/components/formula-editor/known-identifiers";
import { formulaCatalog } from "@trenova/shared/rules/formula";
import { rulesSync } from "@trenova/shared/rules/runtime";
import { describe, expect, it } from "vitest";
import { analyzeExpression, keyFromLabel } from "../expression-analysis";

const rules = rulesSync();
const known = knownFor([
  { name: "fuelSurchargePercent", type: "Number", description: "", required: false },
]);

describe("analyzeExpression", () => {
  it("separates schema reads, input reads and functions, and never counts a let as a read", () => {
    const analysis = analyzeExpression(
      rules,
      "let linehaul = baseRate * totalDistance;\nmax(linehaul * (1 + fuelSurchargePercent / 100), 250)",
      known,
    );

    expect(analysis.errorCount).toBe(0);
    expect([...analysis.schemaReads]).toEqual(["baseRate", "totalDistance"]);
    expect([...analysis.inputReads]).toEqual(["fuelSurchargePercent"]);
    expect([...analysis.functionCalls]).toEqual(["max"]);
    expect(analysis.reads).toEqual(["fuelSurchargePercent", "baseRate", "totalDistance"]);
    expect(analysis.bindings).toEqual([
      { name: "linehaul", type: "Number", reads: 1, from: 0, to: 39, nameFrom: 4, nameTo: 12 },
    ]);
    expect(analysis.statements.map((statement) => [statement.kind, statement.from])).toEqual([
      ["let", 0],
      ["result", 41],
    ]);
    expect(analysis.resultType).toBe("Number");
  });

  it("reports an unknown name as a problem the studio can turn into an input", () => {
    const analysis = analyzeExpression(rules, "baseRate * weekendFee", known);
    expect(analysis.problems).toHaveLength(1);
    expect(analysis.problems[0]).toMatchObject({ code: "unknown-variable", name: "weekendFee" });
    expect(analysis.problems[0].fixes).toEqual([
      {
        kind: "declare",
        label: { message: "Declare {0} as a {1} input", args: ["weekendFee", "Number"] },
        name: "weekendFee",
        type: "Number",
      },
    ]);
    expect(analysis.errorCount).toBe(1);
  });

  it("starts a flag named is, has or requires as a Boolean input", () => {
    for (const name of ["isWeekend", "hasLiftgate", "requiresEscort"]) {
      const [problem] = analyzeExpression(rules, `${name} ? 10 : 0`, known).problems;
      expect(problem.fixes).toContainEqual(
        expect.objectContaining({ kind: "declare", type: "Boolean" }),
      );
    }
  });

  it("treats a blank or comment-only expression as empty with nothing to list", () => {
    for (const source of ["   ", "// rate per mile, to come"]) {
      const analysis = analyzeExpression(rules, source, known);
      expect(analysis.empty).toBe(true);
      expect(analysis.problems).toEqual([]);
      expect(analysis.errorCount).toBe(0);
    }
  });

  it("tells a whole number and a duration from an amount", () => {
    expect(analyzeExpression(rules, "len(stops)", known).resultType).toBe("Integer");
    const duration = analyzeExpression(rules, 'duration("1h")', known);
    expect(duration.resultType).toBe("Duration");
    expect(duration.problems.map((problem) => problem.code)).toEqual(["result-type"]);
  });

  it("lists the rate tables the formula reads, and refuses a table named by a variable", () => {
    expect(analyzeExpression(rules, 'lookup("fuel", totalDistance)', known).lookupTables).toEqual([
      { name: "fuel", function: "lookup", from: 7, to: 13 },
    ]);
    expect(
      analyzeExpression(rules, "lookup(proNumber, 1)", known).problems.map((p) => p.code),
    ).toEqual(["rate-table"]);
  });

  it("stops at the parser's limit with one error", () => {
    const source = Array.from({ length: 1_200 }, () => "totalDistance").join(" + ");
    const analysis = analyzeExpression(rules, source, known);
    expect(analysis.problems.filter((problem) => problem.code === "limit")).toHaveLength(1);
    expect(analysis.errorCount).toBeGreaterThan(0);
  });

  it("holds back name checks while the schema is pending, and says so", () => {
    const pending = buildKnownIdentifiers(formulaCatalog(rules), undefined, []);
    const analysis = analyzeExpression(rules, "weekendFee * 2", pending);
    expect(analysis.pending).toBe(true);
    expect(analysis.problems).toEqual([]);
  });

  it("checks a 1,000-node formula on every keystroke well inside a frame", () => {
    // 500 reads summed in a balanced tree: 999 nodes, just under the parser's
    // 1,000 (a flat chain would stop at the 128-level depth limit first).
    const tree = (leaves: number, at = 0): string => {
      if (leaves === 1) return at % 2 === 0 ? "totalDistance" : "baseRate";
      const left = Math.ceil(leaves / 2);
      return `(${tree(left, at)} + ${tree(leaves - left, at + left)})`;
    };
    const source = tree(500);
    const fresh = () => knownFor();
    analyzeExpression(rules, source, fresh());
    const runs = 20;
    const started = performance.now();
    for (let run = 0; run < runs; run += 1) analyzeExpression(rules, source, fresh());
    const perCheck = (performance.now() - started) / runs;

    expect(analyzeExpression(rules, source, known).problems).toEqual([]);
    expect(perCheck).toBeLessThan(16);
  });
});

describe("keyFromLabel", () => {
  it("camel-cases a label into a line key", () => {
    expect(keyFromLabel("Fuel surcharge")).toBe("fuelSurcharge");
    expect(keyFromLabel("Stops after two!")).toBe("stopsAfterTwo");
    expect(keyFromLabel("2nd stop")).toBe("line2ndStop");
    expect(keyFromLabel("  ")).toBe("");
  });
});
