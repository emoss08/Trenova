import { formulaCatalog } from "@trenova/shared/rules/formula";
import { rulesSync } from "@trenova/shared/rules/runtime";
import { describe, expect, it } from "vitest";
import {
  buildKnownIdentifiers,
  functionInsertion,
  isKeyword,
  isKnownFunction,
  isKnownVariable,
  isOperatorWord,
  itemFieldDoc,
} from "../known-identifiers";
import { knownFor, shipmentSchema } from "./formula-fixtures";

const catalog = formulaCatalog(rulesSync());

describe("buildKnownIdentifiers", () => {
  it("knows the catalog and the inputs, but no field, until the schema has loaded", () => {
    const known = buildKnownIdentifiers(catalog, undefined, [{ name: "fuelPct", type: "Number" }]);

    expect(known.ready).toBe(false);
    expect(known.schema).toBeNull();
    expect([...known.variablePaths]).toEqual(["fuelPct"]);
    expect(known.callables.has("round")).toBe(true);
    expect(known.disabled.get("now")?.replacement).toBe("ratingTime");
  });

  it("hands the checker the schema's fields, item fields included, and the inputs", () => {
    const known = knownFor([{ name: "fuelPct", type: "Number", description: "", required: false }]);

    expect(known.ready).toBe(true);
    expect(known.schema).toEqual({
      variables: shipmentSchema.variables,
      inputs: [{ name: "fuelPct", type: "Number" }],
    });
  });

  it("lists item fields apart from the record's fields", () => {
    const known = knownFor();

    expect(known.variables.some((variable) => variable.name.includes("[]"))).toBe(false);
    expect(known.itemFields.map((field) => field.name)).toEqual(["stops[].type", "stops[].weight"]);
    expect(known.variableRoots.has("stops[]")).toBe(false);
  });

  it("adds inputs as custom variables after the schema's", () => {
    const known = knownFor([{ name: "fuelPct", type: "Number", description: "", required: false }]);

    expect(known.variables.at(-1)).toEqual({
      name: "fuelPct",
      type: "Number",
      description: "Custom variable",
      category: "custom",
      custom: true,
    });
  });

  it("calls built-ins by name but lists only the documented functions", () => {
    const known = knownFor();

    expect(isKnownFunction(known, "filter")).toBe(true);
    expect(known.functions.some((fn) => fn.name === "filter")).toBe(false);
    expect(isKnownFunction(known, "startsWith")).toBe(false);
    expect(isKnownFunction(known, "bogusFn")).toBe(false);
  });

  it("takes keywords and operator words from the catalog", () => {
    const known = knownFor();

    for (const word of ["let", "if", "else", "nil", "true", "and", "in"]) {
      expect(isKeyword(known, word), word).toBe(true);
    }
    expect(isKeyword(known, "round")).toBe(false);
    expect([...known.operatorNames].sort()).toEqual([
      "contains",
      "endsWith",
      "matches",
      "startsWith",
    ]);
    expect(isOperatorWord(known, "startsWith")).toBe(true);
  });

  it("builds the catalog's part once per catalog", () => {
    expect(knownFor().callables).toBe(knownFor([{ name: "x", type: "Number" }]).callables);
  });
});

describe("isKnownVariable", () => {
  const known = knownFor();

  it("recognizes exact paths and dotted-root matches", () => {
    expect(isKnownVariable(known, "customer.name")).toBe(true);
    expect(isKnownVariable(known, "customer.somethingElse")).toBe(true);
    expect(isKnownVariable(known, "bogus")).toBe(false);
  });
});

describe("itemFieldDoc", () => {
  it("describes an item's field from the list that has it", () => {
    expect(itemFieldDoc(knownFor(), "weight")).toEqual({
      name: "weight",
      type: "number",
      description: "Weight at the stop",
      category: "shipment",
      nullable: true,
      enum: null,
    });
  });

  it("says nothing when lists disagree about the field, or none has it", () => {
    const known = buildKnownIdentifiers(catalog, {
      variables: [
        ...shipmentSchema.variables,
        {
          name: "commodities[].weight",
          type: "string",
          description: "",
          category: "",
          nullable: false,
          computed: false,
        },
      ],
    });
    expect(itemFieldDoc(known, "weight")).toBeNull();
    expect(itemFieldDoc(known, "bogus")).toBeNull();
  });
});

describe("functionInsertion", () => {
  it("inserts a call with the cursor between the parentheses", () => {
    expect(functionInsertion({ name: "round", operator: false })).toEqual({
      text: "round()",
      cursor: 6,
    });
  });

  it("inserts an infix operator with a string placeholder selected for typing", () => {
    expect(functionInsertion({ name: "startsWith", operator: true })).toEqual({
      text: ' startsWith ""',
      cursor: 13,
    });
  });

  it("inserts a slice with the cursor on the start index", () => {
    expect(functionInsertion({ name: "[start:end]", operator: true })).toEqual({
      text: "[0:3]",
      cursor: 1,
    });
  });
});
