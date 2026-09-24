import { pageDraftSchema, type FormulaProposal } from "@/types/page-draft";
import type { VariableDefinition } from "@trenova/shared/types/formula-template";
import { describe, expect, it } from "vitest";
import { formulaDraftFromEditor, proposalForEditor } from "../formula-draft";

/*
 * pagedraft.Formula on the server: a schema id and template type of key
 * characters, an expression of at most 10000 runes, and at most 50 variables,
 * each named like an identifier, typed Number, String or Boolean, described in
 * at most 500 runes and defaulting to a scalar.
 */
function variable(overrides: Partial<VariableDefinition>): VariableDefinition {
  return {
    name: "fuelRate",
    type: "Number",
    description: "",
    required: false,
    defaultValue: undefined,
    ...overrides,
  };
}

describe("formulaDraftFromEditor", () => {
  it("hands the assistant what the editor holds", () => {
    const draft = formulaDraftFromEditor({
      templateId: "ft_1",
      schemaId: "shipment",
      templateType: "FreightCharge",
      expression: "totalDistance * fuelRate",
      variables: [variable({ description: "Per mile", defaultValue: 0.5 })],
    });

    expect(pageDraftSchema.safeParse(draft).success).toBe(true);
    expect(draft).toEqual({
      surface: "formula",
      formula: {
        templateId: "ft_1",
        schemaId: "shipment",
        templateType: "FreightCharge",
        expression: "totalDistance * fuelRate",
        variables: [
          { name: "fuelRate", type: "Number", description: "Per mile", defaultValue: 0.5 },
        ],
      },
    });
  });

  it("leaves the template out while it is unsaved, and falls back to the shipment schema", () => {
    const draft = formulaDraftFromEditor({
      templateId: null,
      schemaId: "",
      templateType: "FreightCharge",
      expression: "",
      variables: [],
    });

    expect(draft.formula).not.toHaveProperty("templateId");
    expect(draft.formula?.schemaId).toBe("shipment");
  });

  it("keeps every variable within what the server accepts", () => {
    const many = Array.from({ length: 60 }, (_, index) => variable({ name: `v${index}` }));
    const draft = formulaDraftFromEditor({
      templateId: null,
      schemaId: "shipment",
      templateType: "FreightCharge",
      expression: "x".repeat(10_050),
      variables: [
        variable({ name: "1bad" }),
        variable({ name: "has space" }),
        variable({ name: "pickupDate", type: "Date", description: "When it ships" }),
        variable({ name: "tags", type: "Array", defaultValue: ["a"] }),
        variable({ name: "pickupDate", type: "String" }),
        variable({ name: "note", type: "String", defaultValue: "é".repeat(600) }),
        variable({
          name: "flag",
          type: "Boolean",
          defaultValue: false,
          description: "d".repeat(600),
        }),
        ...many,
      ],
    });
    const formula = draft.formula!;

    expect(Array.from(formula.expression)).toHaveLength(10_000);
    expect(formula.variables).toHaveLength(50);
    expect(formula.variables.map((item) => item.name).slice(0, 5)).toEqual([
      "pickupDate",
      "tags",
      "note",
      "flag",
      "v0",
    ]);
    expect(formula.variables[0]).toEqual({
      name: "pickupDate",
      type: "String",
      description: "A Date value. When it ships",
      defaultValue: undefined,
    });
    expect(formula.variables[1].type).toBe("String");
    expect(formula.variables[1].defaultValue).toBeUndefined();
    expect(formula.variables[2].defaultValue).toBeUndefined();
    expect(formula.variables[3].defaultValue).toBe(false);
    expect(Array.from(formula.variables[3].description)).toHaveLength(500);
    expect(pageDraftSchema.safeParse(draft).success).toBe(true);
  });
});

describe("proposalForEditor", () => {
  it("turns a proposal into the editor's expression and variable definitions", () => {
    const proposal: FormulaProposal = {
      schemaId: "shipment",
      expression: "max(totalDistance * perMile, minimum)",
      variables: [
        { name: "perMile", type: "Number", description: "Rate per mile", defaultValue: 2.85 },
        { name: "minimum", type: "Number", description: "", defaultValue: null },
      ],
      explanation: "Per mile with a floor.",
      check: { valid: true, result: "350", error: "" },
      scenarios: [],
    };

    expect(proposalForEditor(proposal)).toEqual({
      expression: "max(totalDistance * perMile, minimum)",
      variableDefinitions: [
        {
          name: "perMile",
          type: "Number",
          description: "Rate per mile",
          required: false,
          defaultValue: 2.85,
        },
        {
          name: "minimum",
          type: "Number",
          description: "",
          required: false,
          defaultValue: undefined,
        },
      ],
    });
  });
});
