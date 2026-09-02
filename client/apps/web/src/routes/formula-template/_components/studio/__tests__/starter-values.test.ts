import type { FormulaTemplate, StandardTemplate } from "@trenova/shared/types/formula-template";
import { describe, expect, it } from "vitest";
import { copyValuesFrom, starterValuesFrom } from "../starter-values";

describe("starterValuesFrom", () => {
  it("takes the expression and variables from a catalog entry and resets the rest", () => {
    const standard: StandardTemplate = {
      name: "Per Mile",
      description: "Rate per mile",
      type: "FreightCharge",
      expression: "baseRate * totalDistance",
      schemaId: "shipment",
      variableDefinitions: [
        { name: "fuel", type: "Number", description: "", required: false, defaultValue: 18 },
      ],
    };

    expect(starterValuesFrom(standard)).toEqual({
      schemaId: "shipment",
      expression: "baseRate * totalDistance",
      variableDefinitions: standard.variableDefinitions,
      breakdownDefinitions: [],
      minCharge: null,
      maxCharge: null,
      roundingMode: "HalfUp",
      roundingPrecision: 2,
    });
  });
});

describe("copyValuesFrom", () => {
  it("copies everything that computes a charge, and nothing that identifies the source", () => {
    const source = {
      id: "ft_1",
      name: "Acme Lane",
      description: "Acme's lane rate",
      type: "FreightCharge",
      expression: "lookup('lane', laneCode)",
      schemaId: "shipment",
      status: "Active",
      variableDefinitions: [],
      breakdownDefinitions: [{ name: "linehaul", label: "Linehaul", expression: "1" }],
      minCharge: 250,
      maxCharge: null,
      roundingMode: "Up",
      roundingPrecision: 0,
      currentVersionNumber: 7,
    } as FormulaTemplate;

    const values = copyValuesFrom(source);
    expect(values.schemaId).toBe("shipment");
    expect(values.expression).toBe("lookup('lane', laneCode)");
    expect(values.breakdownDefinitions).toEqual(source.breakdownDefinitions);
    expect(values.minCharge).toBe(250);
    expect(values.roundingMode).toBe("Up");
    expect(values.roundingPrecision).toBe(0);
    expect("id" in values).toBe(false);
    expect("name" in values).toBe(false);
    expect("status" in values).toBe(false);
  });
});
