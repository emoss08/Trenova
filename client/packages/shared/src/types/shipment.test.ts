import { describe, expect, it } from "vitest";
import {
  carrierAssignmentPayloadSchema,
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
