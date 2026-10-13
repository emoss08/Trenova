import { pageDraftSchema, type FormulaProposal } from "@/types/page-draft";
import type { VariableDefinition } from "@trenova/shared/types/formula-template";
import { describe, expect, it } from "vitest";
import {
  applyProposal,
  formulaDraftFromEditor,
  newProposalVariables,
  proposalForEditor,
  type ProposalTarget,
} from "../formula-draft";

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

  it("carries the name, description, breakdown lines and charge policy when the editor has them", () => {
    const draft = formulaDraftFromEditor({
      templateId: null,
      schemaId: "shipment",
      templateType: "FreightCharge",
      name: "  Per mile  ",
      description: "Linehaul per mile",
      expression: "baseRate * totalDistance",
      variables: [],
      breakdown: [
        { name: "linehaul", label: "Linehaul", expression: "baseRate * totalDistance" },
        { name: "", label: "Dropped", expression: "1" },
        { name: "empty", label: "Dropped too", expression: "  " },
      ],
      policy: { minCharge: 250, maxCharge: "", roundingMode: "HalfUp", roundingPrecision: 2 },
    });

    expect(pageDraftSchema.safeParse(draft).success).toBe(true);
    expect(draft.formula).toMatchObject({
      name: "Per mile",
      description: "Linehaul per mile",
      breakdown: [{ name: "linehaul", label: "Linehaul", expression: "baseRate * totalDistance" }],
      policy: { minCharge: "250", maxCharge: null, roundingMode: "HalfUp", roundingPrecision: 2 },
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
      name: "",
      description: "",
      breakdown: [],
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

function proposalWith(overrides: Partial<FormulaProposal>): FormulaProposal {
  return {
    schemaId: "shipment",
    name: "",
    description: "",
    templateType: null,
    breakdown: [],
    expression: "baseRate * totalDistance + weekendFee",
    variables: [
      { name: "weekendFee", type: "Number", description: "Weekend fee", defaultValue: 125 },
    ],
    explanation: "Adds a weekend fee.",
    check: { valid: true, result: "1750", error: "" },
    scenarios: [],
    ...overrides,
  };
}

const editor: ProposalTarget = {
  name: "Linehaul",
  description: "Per mile linehaul",
  type: "FreightCharge",
  expression: "baseRate * totalDistance",
  variableDefinitions: [
    {
      name: "weekendFee",
      type: "Number",
      description: "Set by lanes",
      required: true,
      defaultValue: 90,
    },
    { name: "stopFee", type: "Number", description: "", required: false, defaultValue: 50 },
  ],
  breakdownDefinitions: [
    { name: "linehaul", label: "Linehaul", expression: "baseRate * totalDistance" },
  ],
  minCharge: 250,
  maxCharge: null,
  roundingMode: "HalfUp",
  roundingPrecision: 2,
};

describe("applyProposal", () => {
  it("keeps the editor's charge policy unless the proposal sets one", () => {
    const kept = applyProposal(editor, proposalWith({}));
    expect([kept.minCharge, kept.maxCharge, kept.roundingMode]).toEqual([250, null, "HalfUp"]);

    const moved = applyProposal(
      editor,
      proposalWith({ policy: { minCharge: "300", maxCharge: "2500", roundingMode: "Up" } }),
    );
    expect([moved.minCharge, moved.maxCharge, moved.roundingMode, moved.roundingPrecision]).toEqual(
      [300, 2500, "Up", 2],
    );

    const cleared = applyProposal(editor, proposalWith({ policy: { minCharge: null } }));
    expect(cleared.minCharge).toBeNull();
    expect(cleared.maxCharge).toBeNull();
  });

  it("keeps the editor's own variables and adds only the ones it lacks", () => {
    const next = applyProposal(
      editor,
      proposalWith({
        expression: "baseRate * totalDistance + weekendFee + stopFee",
        variables: [
          { name: "weekendFee", type: "Number", description: "Weekend fee", defaultValue: 125 },
          { name: "reeferFee", type: "Number", description: "Reefer fee", defaultValue: 225 },
        ],
      }),
    );

    expect(next.expression).toBe("baseRate * totalDistance + weekendFee + stopFee");
    expect(next.variableDefinitions).toEqual([
      editor.variableDefinitions[0],
      editor.variableDefinitions[1],
      {
        name: "reeferFee",
        type: "Number",
        description: "Reefer fee",
        required: false,
        defaultValue: 225,
      },
    ]);
  });

  it("lets go of an input the new formula and breakdown no longer read", () => {
    const next = applyProposal(
      editor,
      proposalWith({ expression: "baseRate * totalDistance", variables: [] }),
    );
    expect(next.variableDefinitions).toEqual([]);

    const stillRead = applyProposal(
      editor,
      proposalWith({
        expression: "baseRate * totalDistance",
        variables: [],
        breakdown: [{ name: "stops", label: "Stops", expression: "stopFee" }],
      }),
    );
    expect(stillRead.variableDefinitions.map((variable) => variable.name)).toEqual(["stopFee"]);
  });

  it("keeps the editor's breakdown unless the proposal itemizes, then takes its lines", () => {
    expect(applyProposal(editor, proposalWith({})).breakdownDefinitions).toEqual(
      editor.breakdownDefinitions,
    );
    const lines = [
      { name: "linehaul", label: "Linehaul", expression: "baseRate * totalDistance" },
      { name: "weekend", label: "Weekend pickup", expression: "weekendFee" },
    ];
    expect(applyProposal(editor, proposalWith({ breakdown: lines })).breakdownDefinitions).toEqual(
      lines,
    );
  });

  it("fills name, description and type only where the editor has none", () => {
    const proposal = proposalWith({
      name: "Weekend linehaul",
      description: "Linehaul with a weekend fee",
      templateType: "AccessorialCharge",
    });

    const kept = applyProposal(editor, proposal);
    expect([kept.name, kept.description, kept.type]).toEqual([
      "Linehaul",
      "Per mile linehaul",
      "FreightCharge",
    ]);

    const empty: ProposalTarget = { ...editor, name: "", description: "  " };
    const filled = applyProposal(empty, proposal);
    expect([filled.name, filled.description, filled.type]).toEqual([
      "Weekend linehaul",
      "Linehaul with a weekend fee",
      "AccessorialCharge",
    ]);
  });

  it("lists as new only the proposal's variables the editor does not declare", () => {
    const proposal = proposalWith({
      variables: [
        { name: "weekendFee", type: "Number", description: "", defaultValue: 125 },
        { name: "reeferFee", type: "Number", description: "", defaultValue: 225 },
      ],
    });
    expect(newProposalVariables(editor.variableDefinitions, proposal).map((v) => v.name)).toEqual([
      "reeferFee",
    ]);
  });
});
