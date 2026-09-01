import { describe, expect, it } from "vitest";
import { buildKnownIdentifiers, isKnownFunction, isKnownVariable } from "../known-identifiers";

describe("buildKnownIdentifiers", () => {
  it("falls back to the bundled shipment schema when no server schema exists", () => {
    const known = buildKnownIdentifiers(undefined);
    expect(known.variablePaths.has("totalDistance")).toBe(true);
    expect(known.functionNames.has("round")).toBe(true);
  });

  it("uses the server schema when it has variables", () => {
    const known = buildKnownIdentifiers({
      variables: [
        {
          name: "newServerVariable",
          type: "number",
          description: "",
          category: "shipment",
          nullable: false,
          computed: false,
          enum: undefined,
        },
      ],
      functions: [
        { name: "lookup", signature: "lookup(t, k)", description: "", example: "", category: "" },
      ],
    });
    expect(known.variablePaths.has("newServerVariable")).toBe(true);
    expect(known.variablePaths.has("totalDistance")).toBe(false);
  });

  it("adds custom variables with a custom category", () => {
    const known = buildKnownIdentifiers(undefined, [
      { name: "fuelPct", type: "Number", description: "", required: false },
    ]);
    expect(known.variablePaths.has("fuelPct")).toBe(true);
    expect(known.variables.find((v) => v.name === "fuelPct")?.custom).toBe(true);
  });
});

describe("isKnownVariable", () => {
  const known = buildKnownIdentifiers(undefined);

  it("recognizes exact paths and dotted-root matches", () => {
    expect(isKnownVariable(known, "customer.name")).toBe(true);
    expect(isKnownVariable(known, "customer.somethingElse")).toBe(true);
    expect(isKnownVariable(known, "bogus")).toBe(false);
  });
});

describe("isKnownFunction", () => {
  const known = buildKnownIdentifiers(undefined);

  it("recognizes Trenova functions and expr builtins", () => {
    expect(isKnownFunction(known, "round")).toBe(true);
    expect(isKnownFunction(known, "len")).toBe(true);
    expect(isKnownFunction(known, "bogusFn")).toBe(false);
  });
});
