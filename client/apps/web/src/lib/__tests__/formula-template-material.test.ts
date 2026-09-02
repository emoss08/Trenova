import type {
  FormulaTemplate,
  FormulaTemplateFormValues,
} from "@trenova/shared/types/formula-template";
import { describe, expect, it } from "vitest";
import { hasMaterialChange, saveDemotesToDraft } from "../formula-template-material";

function template(overrides: Partial<FormulaTemplate> = {}): FormulaTemplate {
  return {
    id: "ft_1",
    name: "Per Mile",
    description: "Rate per mile",
    type: "FreightCharge",
    expression: "baseRate * totalDistance",
    status: "Active",
    schemaId: "shipment",
    variableDefinitions: [
      { name: "fuel", type: "Number", description: "", required: false, defaultValue: 18 },
    ],
    breakdownDefinitions: [],
    minCharge: 250,
    maxCharge: null,
    roundingMode: "HalfUp",
    roundingPrecision: 2,
    ...overrides,
  } as FormulaTemplate;
}

function values(overrides: Partial<FormulaTemplateFormValues> = {}): FormulaTemplateFormValues {
  return { ...template(), ...overrides } as FormulaTemplateFormValues;
}

describe("hasMaterialChange", () => {
  it("ignores name and description edits", () => {
    expect(
      hasMaterialChange(template(), values({ name: "Renamed", description: "New words" })),
    ).toBe(false);
  });

  it("detects expression, guardrail, rounding, and definition edits", () => {
    expect(hasMaterialChange(template(), values({ expression: "baseRate * 2" }))).toBe(true);
    expect(hasMaterialChange(template(), values({ minCharge: 300 }))).toBe(true);
    expect(hasMaterialChange(template(), values({ roundingPrecision: 0 }))).toBe(true);
    expect(
      hasMaterialChange(
        template(),
        values({
          variableDefinitions: [
            { name: "fuel", type: "Number", description: "", required: false, defaultValue: 20 },
          ],
        }),
      ),
    ).toBe(true);
  });

  it("treats equivalent guardrail spellings as the same", () => {
    expect(hasMaterialChange(template({ maxCharge: null }), values({ maxCharge: undefined }))).toBe(
      false,
    );
  });
});

describe("saveDemotesToDraft", () => {
  it("only applies to approved or in-review templates with a material change", () => {
    expect(saveDemotesToDraft(template({ status: "Active" }), values({ expression: "x" }))).toBe(
      true,
    );
    expect(saveDemotesToDraft(template({ status: "InReview" }), values({ expression: "x" }))).toBe(
      true,
    );
    expect(saveDemotesToDraft(template({ status: "Draft" }), values({ expression: "x" }))).toBe(
      false,
    );
    expect(saveDemotesToDraft(template({ status: "Active" }), values({ name: "Other" }))).toBe(
      false,
    );
    expect(saveDemotesToDraft(null, values())).toBe(false);
  });
});
