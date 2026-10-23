import { describe, expect, it } from "vitest";
import {
  rateAgreementAccessorialSchema,
  rateAgreementRuleSchema,
  rateAgreementSchema,
  rateMatrixDimensionSchema,
  rateMatrixSchema,
} from "../rate";

/**
 * The smallest payload the schema accepts, so each test states only the fact
 * it is about. A lane is a tariff and that tariff's input values; nothing on
 * it prices on its own any more.
 */
function rule(overrides: Record<string, unknown> = {}): Record<string, unknown> {
  return {
    effectiveFrom: 1_700_000_000,
    formulaTemplateId: "ft_per_mile",
    ...overrides,
  };
}

function issuePaths(result: ReturnType<typeof rateAgreementRuleSchema.safeParse>): string[] {
  return result.success ? [] : result.error.issues.map((issue) => issue.path.join("."));
}

describe("rateAgreementRuleSchema tariff and inputs", () => {
  it("accepts a lane that names its tariff and sets no inputs, defaulting them to none", () => {
    const result = rateAgreementRuleSchema.safeParse(rule());

    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.inputs).toEqual({});
    }
  });

  it("rejects a lane without a tariff, since nothing else on it can price", () => {
    expect(issuePaths(rateAgreementRuleSchema.safeParse(rule({ formulaTemplateId: "" })))).toEqual([
      "formulaTemplateId",
    ]);
    expect(
      issuePaths(rateAgreementRuleSchema.safeParse(rule({ formulaTemplateId: undefined }))),
    ).toEqual(["formulaTemplateId"]);
  });

  it("keeps every input value in its wire form, units and records included", () => {
    const inputs = {
      rate: { $q: "2.35", u: "$/mi" },
      minimumCharge: { $q: "450", u: "$" },
      breaks: [
        { from: { $q: "0", u: "lb" }, rate: { $q: "42.00", u: "$/cwt" }, minimum: null },
        {
          from: { $q: "500", u: "lb" },
          rate: { $q: "35.50", u: "$/cwt" },
          minimum: { $q: "95", u: "$" },
        },
      ],
      hazmat: true,
      service: "Guaranteed",
      stops: { $n: "3" },
      startsOn: { $date: "2026-10-22" },
    };

    const result = rateAgreementRuleSchema.safeParse(rule({ inputs }));

    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.inputs).toEqual(inputs);
    }
  });

  it("drops an input a form left unset, so the lane keeps inheriting it", () => {
    const result = rateAgreementRuleSchema.parse(
      rule({ inputs: { rate: { $q: "2.35", u: "$/mi" }, minimumCharge: undefined } }),
    );

    expect(result.inputs).toEqual({ rate: { $q: "2.35", u: "$/mi" } });
  });

  it("rejects an input value that is a bare JSON number, which would lose its precision", () => {
    const result = rateAgreementRuleSchema.safeParse(rule({ inputs: { rate: 2.35 } }));

    expect(result.success).toBe(false);
    expect(issuePaths(result)[0]).toMatch(/^inputs\.rate/);
  });

  it("drops the removed pricing columns instead of carrying them back to the server", () => {
    const result = rateAgreementRuleSchema.safeParse(
      rule({ rate: 2.55, rateMatrixId: "rmx_1", minCharge: 850, breaks: [], roundingMode: "" }),
    );

    expect(result.success).toBe(true);
    if (result.success) {
      for (const removed of ["rate", "rateMatrixId", "minCharge", "breaks", "roundingMode"]) {
        expect(result.data).not.toHaveProperty(removed);
      }
    }
  });
});

/**
 * What the Go API actually emits, which the schemas must read back. A nil
 * pulid marshals as JSON null (shared/pulid/pulid.go MarshalJSON), and the
 * agreement endpoints echo submitted rules verbatim, so that shape reaches the
 * client on every save, not just on degenerate rows.
 */
describe("rateAgreementRuleSchema server echo tolerance", () => {
  it("accepts a rule echoed back with a null id, as an unsaved lane is", () => {
    const result = rateAgreementRuleSchema.safeParse(rule({ id: null }));

    expect(result.success).toBe(true);
  });
});

describe("rateAgreementAccessorialSchema server echo tolerance", () => {
  it("accepts an empty rate unit on a flat accessorial, which Go writes for an unset one", () => {
    const result = rateAgreementAccessorialSchema.safeParse({
      accessorialChargeId: "acc_detention",
      method: "Flat",
      rateUnit: "",
      amount: 75,
    });

    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.rateUnit ?? null).toBeNull();
    }
  });
});

describe("rateAgreementAccessorialSchema condition", () => {
  const accessorial = { accessorialChargeId: "acc_residential", method: "Flat", amount: 95 };

  it("keeps a condition file's source as written", () => {
    const source = 'condition "Residential"\n\nlane.destination.zip3 in ["100", "112"]';
    const result = rateAgreementAccessorialSchema.parse({ ...accessorial, condition: source });

    expect(result.condition).toBe(source);
  });

  it("reads an empty or absent condition as none, which applies to every shipment", () => {
    expect(rateAgreementAccessorialSchema.parse({ ...accessorial, condition: "" }).condition).toBe(
      null,
    );
    expect(rateAgreementAccessorialSchema.parse(accessorial).condition).toBe(null);
  });
});

describe("rateAgreementSchema contract inputs", () => {
  const agreement = {
    code: "ACME-2026",
    name: "Acme",
    customerId: "cus_1",
    effectiveFrom: 1_700_000_000,
  };

  it("defaults the agreement's contract-wide inputs to none", () => {
    expect(rateAgreementSchema.parse(agreement).inputs).toEqual({});
  });

  it("keeps the agreement's inputs in their wire form", () => {
    const inputs = { minimumCharge: { $q: "450", u: "$" } };

    expect(rateAgreementSchema.parse({ ...agreement, inputs }).inputs).toEqual(inputs);
  });
});

describe("rateMatrixSchema value type", () => {
  const matrix = {
    code: "ZONE-TL",
    name: "Zone to Zone Truckload",
    valueType: "Money/cwt",
    dimensions: [{ position: 0, kind: "Zone", matchMode: "Exact", label: "" }],
  };

  it("accepts a matrix that states what its cells hold", () => {
    expect(rateMatrixSchema.parse(matrix).valueType).toBe("Money/cwt");
  });

  it("accepts every value type the server stores: money per any rate unit, percent and counts", () => {
    for (const valueType of ["Money/ft3", "Money/d", "Money/kg", "Percent", "Number", "Int"]) {
      expect(rateMatrixSchema.parse({ ...matrix, valueType }).valueType).toBe(valueType);
    }
  });

  it("defaults the value type to money", () => {
    const { valueType: _, ...rest } = matrix;

    expect(rateMatrixSchema.parse(rest).valueType).toBe("Money");
  });

  it("rejects a value type that is not a Tariff type", () => {
    expect(rateMatrixSchema.safeParse({ ...matrix, valueType: "Money/furlong" }).success).toBe(
      false,
    );
  });

  it("needs one to four dimensions", () => {
    const axis = { position: 0, kind: "Zone", matchMode: "Exact" };

    expect(rateMatrixSchema.safeParse({ ...matrix, dimensions: [] }).success).toBe(false);
    expect(
      rateMatrixSchema.safeParse({
        ...matrix,
        dimensions: [0, 1, 2, 3, 3].map((position) => ({ ...axis, position })),
      }).success,
    ).toBe(false);
  });
});

describe("rateMatrixDimensionSchema lookup modes", () => {
  it("defaults an axis to exact keys and strict bands", () => {
    const parsed = rateMatrixDimensionSchema.parse({
      position: 0,
      kind: "Zone",
      matchMode: "Exact",
    });
    expect(parsed).toMatchObject({ keyNormalization: "None", rangeOverflow: "Error" });
  });

  it("accepts the normalisation and overflow modes the server knows", () => {
    expect(
      rateMatrixDimensionSchema.parse({
        position: 0,
        kind: "Zone",
        matchMode: "Exact",
        keyNormalization: "Zip3",
        rangeOverflow: "ClampToTopBand",
      }),
    ).toMatchObject({ keyNormalization: "Zip3", rangeOverflow: "ClampToTopBand" });
  });
});
