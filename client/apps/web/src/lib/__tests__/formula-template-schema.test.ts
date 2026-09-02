import { formulaTemplateSchema } from "@trenova/shared/types/formula-template";
import { describe, expect, it } from "vitest";

const base = {
  name: "T",
  description: "d",
  type: "FreightCharge" as const,
  expression: "x",
  status: "Draft" as const,
  schemaId: "shipment",
  variableDefinitions: [],
  breakdownDefinitions: [],
  minCharge: null,
  maxCharge: null,
};

describe("formulaTemplateSchema duplicate names", () => {
  it("flags the second declaration of a variable name at its own path", () => {
    const result = formulaTemplateSchema.safeParse({
      ...base,
      variableDefinitions: [
        { name: "fuel", type: "Number" },
        { name: "fuel", type: "Number" },
      ],
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues[0]?.path).toEqual(["variableDefinitions", 1, "name"]);
    }
  });

  it("flags duplicate breakdown names", () => {
    const result = formulaTemplateSchema.safeParse({
      ...base,
      breakdownDefinitions: [
        { name: "linehaul", label: "A", expression: "1" },
        { name: "linehaul", label: "B", expression: "2" },
      ],
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues[0]?.path).toEqual(["breakdownDefinitions", 1, "name"]);
    }
  });

  it("accepts distinct names", () => {
    expect(
      formulaTemplateSchema.safeParse({
        ...base,
        variableDefinitions: [
          { name: "a", type: "Number" },
          { name: "b", type: "Number" },
        ],
      }).success,
    ).toBe(true);
  });
});
