import { describe, expect, it } from "vitest";
import { collectIdentifierTokens, lintExpression } from "../expr-lint";
import { buildKnownIdentifiers } from "../known-identifiers";

const known = buildKnownIdentifiers(undefined, [
  { name: "fuelPct", type: "Number", description: "", required: false },
]);

describe("collectIdentifierTokens", () => {
  it("collects plain and dotted identifiers with positions", () => {
    const tokens = collectIdentifierTokens("baseRate + customer.name");
    expect(tokens.map((t) => t.name)).toEqual(["baseRate", "customer.name"]);
    expect(tokens[0].from).toBe(0);
    expect(tokens[0].to).toBe(8);
  });

  it("marks call targets", () => {
    const tokens = collectIdentifierTokens("round(baseRate, 2)");
    expect(tokens[0]).toMatchObject({ name: "round", isCall: true });
    expect(tokens[1]).toMatchObject({ name: "baseRate", isCall: false });
  });

  it("treats a name followed by whitespace and a paren as a call", () => {
    const tokens = collectIdentifierTokens("round (1.5)");
    expect(tokens[0].isCall).toBe(true);
  });

  it("skips identifiers inside strings and comments", () => {
    const tokens = collectIdentifierTokens(
      '"notAVariable" + baseRate // trailing bogusName\n/* bogusBlock */ + weight',
    );
    expect(tokens.map((t) => t.name)).toEqual(["baseRate", "weight"]);
  });
});

describe("lintExpression", () => {
  it("returns nothing for a valid expression", () => {
    expect(lintExpression("round(baseRate * totalDistance, 2)", known)).toEqual([]);
  });

  it("returns nothing for an empty expression", () => {
    expect(lintExpression("   ", known)).toEqual([]);
  });

  it("accepts declared custom variables", () => {
    expect(lintExpression("baseRate * (1 + fuelPct / 100)", known)).toEqual([]);
  });

  it("warns on an unknown variable with a suggestion", () => {
    const diagnostics = lintExpression("baseRaet * 2", known);
    expect(diagnostics).toHaveLength(1);
    expect(diagnostics[0].severity).toBe("warning");
    expect(diagnostics[0].message).toContain("baseRaet");
    expect(diagnostics[0].message).toContain("baseRate");
  });

  it("errors on an unknown function with a suggestion", () => {
    const diagnostics = lintExpression("rond(baseRate, 2)", known);
    expect(diagnostics.some((d) => d.severity === "error" && d.message.includes("round"))).toBe(
      true,
    );
  });

  it("does not flag expr runtime builtins", () => {
    expect(lintExpression("len(customer.name)", known)).toEqual([]);
  });

  it("flags unbalanced brackets", () => {
    const diagnostics = lintExpression("round(baseRate", known);
    expect(diagnostics.some((d) => d.message.includes("Unclosed"))).toBe(true);
  });

  it("flags a trailing operator", () => {
    const diagnostics = lintExpression("baseRate * ", known);
    expect(diagnostics.some((d) => d.message.includes("ends with an operator"))).toBe(true);
  });

  it("does not treat keywords as unknown variables", () => {
    expect(lintExpression("hasHazmat and true", known)).toEqual([]);
  });
});
