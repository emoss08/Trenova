import { describe, expect, it } from "vitest";
import { rateAgreementAccessorialSchema, rateAgreementRuleSchema, rateMatrixSchema } from "../rate";

/**
 * The smallest payload the schema accepts, so each test states only the
 * pricing-method fact it is about. Mirrors the Go domain's validatePricing:
 * a rule prices through exactly one of a formula template or a rate matrix,
 * and a matrix rule's rates live in the cells.
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

describe("rateAgreementRuleSchema pricing method", () => {
  it("accepts a lane priced by a formula template alone", () => {
    const result = rateAgreementRuleSchema.safeParse(rule({ rate: 2.55 }));

    expect(result.success).toBe(true);
  });

  it("accepts a formula lane without a rate, since a template need not read one", () => {
    const result = rateAgreementRuleSchema.safeParse(rule());

    expect(result.success).toBe(true);
  });

  it("accepts a lane priced by a rate matrix alone", () => {
    const result = rateAgreementRuleSchema.safeParse(
      rule({ formulaTemplateId: null, rateMatrixId: "rmx_zone_tl" }),
    );

    expect(result.success).toBe(true);
  });

  it("rejects a lane that names neither pricing method", () => {
    const result = rateAgreementRuleSchema.safeParse(rule({ formulaTemplateId: null }));

    expect(result.success).toBe(false);
    expect(issuePaths(result)).toContain("formulaTemplateId");
  });

  it("rejects a lane that names both pricing methods", () => {
    const result = rateAgreementRuleSchema.safeParse(rule({ rateMatrixId: "rmx_zone_tl" }));

    expect(result.success).toBe(false);
    expect(issuePaths(result)).toContain("formulaTemplateId");
  });

  it("rejects a rate on a matrix lane, whose rates live in the cells", () => {
    const result = rateAgreementRuleSchema.safeParse(
      rule({ formulaTemplateId: null, rateMatrixId: "rmx_zone_tl", rate: 2.55 }),
    );

    expect(result.success).toBe(false);
    expect(issuePaths(result)).toContain("rate");
  });

  it("rejects weight breaks on a matrix lane", () => {
    const result = rateAgreementRuleSchema.safeParse(
      rule({
        formulaTemplateId: null,
        rateMatrixId: "rmx_zone_tl",
        breaks: [{ fromWeight: 0, toWeight: null, rate: 12, minCharge: null, sortOrder: 0 }],
      }),
    );

    expect(result.success).toBe(false);
    expect(issuePaths(result)).toContain("breaks");
  });

  it("accepts weight breaks on a formula lane", () => {
    const result = rateAgreementRuleSchema.safeParse(
      rule({
        breaks: [{ fromWeight: 0, toWeight: null, rate: 12, minCharge: null, sortOrder: 0 }],
      }),
    );

    expect(result.success).toBe(true);
  });
});

/**
 * What the Go API actually emits, which the schemas must read back. A nil
 * pulid marshals as JSON null (shared/pulid/pulid.go MarshalJSON), and an
 * unset nullzero enum column marshals as "" — and the agreement endpoints
 * echo submitted rules verbatim, so both shapes reach the client on every
 * save, not just on degenerate rows.
 */
describe("rateAgreementRuleSchema server echo tolerance", () => {
  it("accepts a rule echoed back with a null id, as an unsaved lane is", () => {
    const result = rateAgreementRuleSchema.safeParse(rule({ id: null }));

    expect(result.success).toBe(true);
  });

  it("accepts an empty rounding mode, which is how Go writes an unset one", () => {
    const result = rateAgreementRuleSchema.safeParse(rule({ roundingMode: "" }));

    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.roundingMode ?? null).toBeNull();
    }
  });

  it("accepts a weight break echoed back with a null id", () => {
    const result = rateAgreementRuleSchema.safeParse(
      rule({
        breaks: [
          {
            id: null,
            fromWeight: 0,
            toWeight: null,
            rate: 12,
            minCharge: null,
            sortOrder: 0,
          },
        ],
      }),
    );

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

describe("rateMatrixSchema rating method", () => {
  const matrix = {
    code: "ZONE-TL",
    name: "Zone to Zone Truckload",
    formulaTemplateId: "ft_per_mile",
    dimensions: [{ position: 0, kind: "Zone", matchMode: "Exact", label: "" }],
  };

  it("accepts a matrix that names the template pricing its cells", () => {
    expect(rateMatrixSchema.safeParse(matrix).success).toBe(true);
  });

  it("rejects a matrix without a formula template, whose cells would mean nothing", () => {
    const result = rateMatrixSchema.safeParse({ ...matrix, formulaTemplateId: "" });

    expect(result.success).toBe(false);
  });
});
