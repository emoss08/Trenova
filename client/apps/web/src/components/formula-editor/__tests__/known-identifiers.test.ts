import { describe, expect, it } from "vitest";
import {
  buildKnownIdentifiers,
  functionInsertion,
  isKnownFunction,
  isKnownVariable,
} from "../known-identifiers";

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
        {
          name: "lookup",
          signature: "lookup(t, k)",
          description: "",
          example: "",
          category: "",
          operator: false,
        },
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

describe("functionInsertion", () => {
  it("inserts a call with the cursor between the parentheses", () => {
    expect(functionInsertion({ name: "round", signature: "round(x)", description: "" })).toEqual({
      text: "round()",
      cursor: 6,
    });
  });

  it("inserts an infix operator with a string placeholder selected for typing", () => {
    expect(
      functionInsertion({
        name: "startsWith",
        signature: 'text startsWith "prefix"',
        description: "",
        operator: true,
      }),
    ).toEqual({ text: ' startsWith ""', cursor: 13 });
  });

  it("inserts a slice with the cursor on the start index", () => {
    expect(
      functionInsertion({
        name: "[start:end]",
        signature: "text[start:end]",
        description: "",
        operator: true,
      }),
    ).toEqual({ text: "[0:3]", cursor: 1 });
  });
});
