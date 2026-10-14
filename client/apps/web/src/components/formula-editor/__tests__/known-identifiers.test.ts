import { formulaSchemaResponseSchema } from "@trenova/shared/types/formula-template";
import { describe, expect, it } from "vitest";
import {
  buildKnownIdentifiers,
  functionInsertion,
  isKnownFunction,
  isKnownVariable,
} from "../known-identifiers";
import { describedShipment } from "./described-shipment";

describe("buildKnownIdentifiers", () => {
  it("knows only the template's inputs until the engine's schema has loaded", () => {
    const known = buildKnownIdentifiers(undefined, [{ name: "fuelPct", type: "Number" }]);
    expect(known.ready).toBe(false);
    expect([...known.variablePaths]).toEqual(["fuelPct"]);
    expect(known.functionNames.size).toBe(0);
    expect(known.disabled.size).toBe(0);
  });

  it("takes functions, builtins and the disabled list from the engine", () => {
    const known = buildKnownIdentifiers(describedShipment);
    expect(known.ready).toBe(true);
    expect(known.variablePaths.has("totalDistance")).toBe(true);
    expect(known.shapes.get("round")).toMatchObject({ minArgs: 1, maxArgs: 2, returns: "Number" });
    expect(known.shapes.get("min")?.maxArgs).toBe(Number.POSITIVE_INFINITY);
    expect(known.shapes.get("filter")?.predicate).toBe(true);
    expect(known.shapes.get("sum")?.predicate).toBe(false);
    for (const builtin of ["reverse", "toPairs", "fromPairs", "bitnot"]) {
      expect(known.functionNames.has(builtin), builtin).toBe(true);
    }
    expect(known.functions.some((fn) => fn.name === "reverse")).toBe(false);
    expect(known.disabled.get("now")).toEqual({
      name: "now",
      reason: "nondeterministic",
      replacement: "ratingTime",
    });
  });

  it("uses the server schema when it has variables", () => {
    const known = buildKnownIdentifiers(
      formulaSchemaResponseSchema.parse({
        schemaId: "shipment",
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
      }),
    );
    expect(known.variablePaths.has("newServerVariable")).toBe(true);
    expect(known.variablePaths.has("totalDistance")).toBe(false);
  });

  it("adds custom variables with a custom category", () => {
    const known = buildKnownIdentifiers(describedShipment, [
      { name: "fuelPct", type: "Number", description: "", required: false },
    ]);
    expect(known.variablePaths.has("fuelPct")).toBe(true);
    expect(known.variables.find((v) => v.name === "fuelPct")?.custom).toBe(true);
  });
});

describe("isKnownVariable", () => {
  const known = buildKnownIdentifiers(describedShipment);

  it("recognizes exact paths and dotted-root matches", () => {
    expect(isKnownVariable(known, "customer.name")).toBe(true);
    expect(isKnownVariable(known, "customer.somethingElse")).toBe(true);
    expect(isKnownVariable(known, "bogus")).toBe(false);
  });
});

describe("isKnownFunction", () => {
  const known = buildKnownIdentifiers(describedShipment);

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
