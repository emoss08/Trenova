import { describe, expect, it } from "vitest";
import { collectIdentifierTokens, lintExpression } from "../expr-lint";
import { buildKnownIdentifiers } from "../known-identifiers";
import { describedShipment } from "@/components/formula-editor/__tests__/described-shipment";
import { formulaSchemaResponseSchema } from "@trenova/shared/types/formula-template";

const known = buildKnownIdentifiers(describedShipment, [
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

  it("flags an unknown variable with a suggestion", () => {
    const diagnostics = lintExpression("baseRaet * 2", known);
    expect(diagnostics).toHaveLength(1);
    expect(diagnostics[0].severity).toBe("error");
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
    expect(diagnostics.some((d) => d.message.includes("never closed"))).toBe(true);
  });

  it("flags a trailing operator", () => {
    const diagnostics = lintExpression("baseRate * ", known);
    expect(diagnostics.some((d) => d.message.includes("needs a value on its right"))).toBe(true);
  });

  it("does not treat keywords as unknown variables", () => {
    expect(lintExpression("(hasHazmat and true) ? 1 : 0", known)).toEqual([]);
  });
});

describe("lintExpression with expr string operators", () => {
  it("does not flag startsWith, endsWith, contains, or matches as unknown names", () => {
    const diagnostics = lintExpression(
      '(origin.state startsWith "T" and customer.name contains "Acme" and origin.zip matches "^7" and destination.state endsWith "X") ? 1 : 0',
      known,
    );
    expect(diagnostics).toEqual([]);
  });

  it("accepts operators the server publishes even when the static list lacks them", () => {
    const serverKnown = buildKnownIdentifiers(
      formulaSchemaResponseSchema.parse({
        schemaId: "shipment",
        variables: [
          {
            name: "code",
            type: "string",
            description: "",
            category: "shipment",
            nullable: false,
            computed: false,
            enum: undefined,
          },
        ],
        functions: [
          {
            name: "soundsLike",
            signature: 'text soundsLike "x"',
            description: "",
            example: "",
            category: "string",
            operator: true,
          },
        ],
      }),
    );
    expect(lintExpression('code soundsLike "abc" ? 1 : 0', serverKnown)).toEqual([]);
  });
});

describe("let bindings", () => {
  it("treats a name the expression binds with let as known wherever it is read", () => {
    const known = buildKnownIdentifiers(describedShipment, []);
    const diagnostics = lintExpression(
      "let linehaul = baseRate * totalDistance;\nlet fuel = linehaul * 0.18;\nlinehaul + fuel",
      known,
    );
    expect(diagnostics).toEqual([]);
  });

  it("still flags a name nothing binds", () => {
    const known = buildKnownIdentifiers(describedShipment, []);
    const diagnostics = lintExpression("let linehaul = baseRate;\nlinehaul + weekendFee", known);
    expect(diagnostics.map((d) => d.message)).toEqual([expect.stringContaining("weekendFee")]);
  });
});
