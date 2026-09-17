import { describe, expect, it } from "vitest";
import {
  carrierAssignmentPayloadSchema,
  carrierEligibilitySchema,
  emptyCarrierAssignmentPayload,
  moveCoverageTypeSchema,
} from "./shipment";

function makePayload(overrides: Record<string, unknown> = {}): Record<string, unknown> {
  return {
    ...emptyCarrierAssignmentPayload,
    carrierId: "car_01",
    baseRate: 1500,
    ...overrides,
  };
}

describe("carrierAssignmentPayloadSchema", () => {
  it("accepts plain numbers from the form inputs", () => {
    const parsed = carrierAssignmentPayloadSchema.parse(
      makePayload({
        baseRate: 1500.25,
        fuelSurcharge: 120.5,
        accessorials: [{ accessorialChargeId: null, description: "Lumper", amount: 75 }],
      }),
    );
    expect(parsed.baseRate).toBe(1500.25);
    expect(parsed.fuelSurcharge).toBe(120.5);
    expect(parsed.accessorials[0].amount).toBe(75);
  });

  it("coerces GraphQL decimal strings on the replace-prefill path to numbers", () => {
    const parsed = carrierAssignmentPayloadSchema.parse(
      makePayload({
        baseRate: "1500.2500",
        fuelSurcharge: "120.50",
        accessorials: [{ accessorialChargeId: null, description: "Detention", amount: "75.00" }],
      }),
    );
    expect(parsed.baseRate).toBe(1500.25);
    expect(parsed.fuelSurcharge).toBe(120.5);
    expect(parsed.accessorials[0].amount).toBe(75);
  });

  it("keeps an absent fuel surcharge nullish rather than coercing it", () => {
    const parsed = carrierAssignmentPayloadSchema.parse(makePayload({ fuelSurcharge: null }));
    expect(parsed.fuelSurcharge).toBeNull();
  });

  it("rejects negative amounts whether they arrive as strings or numbers", () => {
    expect(() =>
      carrierAssignmentPayloadSchema.parse(makePayload({ baseRate: "-1.00" })),
    ).toThrowError(/Base rate cannot be negative/);
    expect(() =>
      carrierAssignmentPayloadSchema.parse(makePayload({ fuelSurcharge: -5 })),
    ).toThrowError(/Fuel surcharge cannot be negative/);
    expect(() =>
      carrierAssignmentPayloadSchema.parse(
        makePayload({
          accessorials: [{ accessorialChargeId: null, description: "Lumper", amount: "-75" }],
        }),
      ),
    ).toThrowError(/Amount cannot be negative/);
  });

  it("rejects empty and non-numeric strings for the base rate", () => {
    expect(() => carrierAssignmentPayloadSchema.parse(makePayload({ baseRate: "" }))).toThrowError(
      /Base rate is required/,
    );
    expect(() =>
      carrierAssignmentPayloadSchema.parse(makePayload({ baseRate: "abc" })),
    ).toThrowError(/Base rate is required/);
  });
});

describe("carrierEligibilitySchema", () => {
  it("normalizes null blockers and warnings to empty arrays (Go nil slices serialize as null)", () => {
    const parsed = carrierEligibilitySchema.parse({ blockers: null, warnings: null });
    expect(parsed.blockers).toEqual([]);
    expect(parsed.warnings).toEqual([]);
  });

  it("normalizes absent blockers and warnings to empty arrays", () => {
    const parsed = carrierEligibilitySchema.parse({});
    expect(parsed.blockers).toEqual([]);
    expect(parsed.warnings).toEqual([]);
  });

  it("passes populated arrays through unchanged", () => {
    const parsed = carrierEligibilitySchema.parse({
      blockers: ["Carrier is inactive"],
      warnings: ["Cargo insurance below load value"],
    });
    expect(parsed.blockers).toEqual(["Carrier is inactive"]);
    expect(parsed.warnings).toEqual(["Cargo insurance below load value"]);
  });

  it("normalizes absent advisories and findings to empty arrays", () => {
    const parsed = carrierEligibilitySchema.parse({ advisories: null });
    expect(parsed.advisories).toEqual([]);
    expect(parsed.findings).toEqual([]);
  });

  it("keeps structured findings with their source and override requirement", () => {
    const parsed = carrierEligibilitySchema.parse({
      blockers: ["Operating authority is revoked"],
      warnings: [],
      advisories: ["Carrier intelligence has not been refreshed recently"],
      findings: [
        {
          code: "authority.revoked",
          source: "Intelligence",
          severity: "Blocker",
          message: "Operating authority is revoked",
          requiresOverride: false,
        },
        {
          code: "intel.snapshot_stale",
          source: "Intelligence",
          severity: "Advisory",
          message: "Carrier intelligence has not been refreshed recently",
        },
      ],
    });
    expect(parsed.findings).toHaveLength(2);
    expect(parsed.findings[0]?.source).toBe("Intelligence");
    expect(parsed.findings[1]?.requiresOverride).toBe(false);
  });
});

describe("moveCoverageTypeSchema", () => {
  it("accepts every coverage a move can carry, uncovered included", () => {
    for (const coverage of ["unassigned", "driver", "carrier"]) {
      expect(moveCoverageTypeSchema.parse(coverage)).toBe(coverage);
    }
  });

  it("rejects a coverage the server never writes", () => {
    expect(() => moveCoverageTypeSchema.parse("broker")).toThrow();
    expect(() => moveCoverageTypeSchema.parse("")).toThrow();
  });
});
