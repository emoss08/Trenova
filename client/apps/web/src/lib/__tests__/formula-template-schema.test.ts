import { formulaTemplateSchema } from "@trenova/shared/types/formula-template";
import { describe, expect, it } from "vitest";

const SOURCE = 'tariff "T"\n\ncharge "Linehaul" = $100\n';

describe("formulaTemplateSchema", () => {
  it("requires a name and a source with something in it", () => {
    const result = formulaTemplateSchema.safeParse({ name: "  ", source: "  \n" });
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues.map((issue) => issue.path)).toEqual([["name"], ["source"]]);
    }
  });

  it("reads the server's derived inputs, imports and test count, defaulting the absent", () => {
    const parsed = formulaTemplateSchema.parse({
      name: "T",
      source: SOURCE,
      inputs: [
        {
          name: "rate",
          doc: null,
          type: "Money/mi",
          descriptor: { kind: "quantity", dimension: "Money/Distance", unit: "$/mi" },
          required: false,
          default: { $q: "2.35", u: "$/mi" },
          range: { from: 10, to: 14 },
        },
      ],
      imports: null,
    });

    expect(parsed).toMatchObject({
      name: "T",
      description: "",
      status: "Draft",
      testCount: 0,
      imports: [],
    });
    expect(parsed.inputs[0]?.descriptor).toEqual({
      kind: "quantity",
      dimension: "Money/Distance",
      unit: "$/mi",
    });
  });

  it("refuses a declaration whose default is not a wire value", () => {
    const result = formulaTemplateSchema.safeParse({
      name: "T",
      source: SOURCE,
      inputs: [
        {
          name: "rate",
          type: "Money",
          descriptor: { kind: "money" },
          required: false,
          default: 2.35,
        },
      ],
    });
    expect(result.success).toBe(false);
  });
});
