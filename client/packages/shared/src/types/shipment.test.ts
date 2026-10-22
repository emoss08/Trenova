import { describe, expect, it } from "vitest";
import {
  carrierAssignmentPayloadSchema,
  carrierEligibilitySchema,
  emptyCarrierAssignmentPayload,
  moveCoverageTypeSchema,
  ratingDetailSchema,
  shipmentFreightLineSchema,
  shipmentUpdateSchema,
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

describe("shipment tariff rating", () => {
  const base = {
    serviceTypeId: "st_1",
    shipmentTypeId: "sht_1",
    customerId: "cus_1",
  };

  it("carries the tariff's input values in their wire form and no base rate", () => {
    const tariffInputs = { rate: { $q: "2.35", u: "$/mi" }, extraDropFee: { $q: "75", u: "$" } };
    const parsed = shipmentUpdateSchema.parse({ ...base, tariffInputs, baseRate: 1500 });

    expect(parsed.tariffInputs).toEqual(tariffInputs);
    expect(parsed).not.toHaveProperty("baseRate");
    expect(parsed).not.toHaveProperty("ratingUnit");
  });

  it("defaults a shipment's tariff inputs to none", () => {
    expect(shipmentUpdateSchema.parse(base).tariffInputs).toEqual({});
  });

  it("reads a freight line with its quantity and rate, and a manual line without them", () => {
    expect(
      shipmentFreightLineSchema.parse({
        label: "Linehaul",
        quantity: "812",
        quantityUnit: "mi",
        rate: "2.35",
        rateUnit: "$/mi",
        amount: "1908.20",
      }),
    ).toMatchObject({ label: "Linehaul", quantity: 812, rate: 2.35, amount: 1908.2 });
    expect(
      shipmentFreightLineSchema.parse({
        label: "Freight",
        quantity: null,
        rate: null,
        amount: "1500",
      }),
    ).toMatchObject({ label: "Freight", quantity: null, rate: null, amount: 1500 });
    // A manual rate as the server writes it: quantity and rate are omitted, not null.
    expect(shipmentFreightLineSchema.parse({ label: "Freight", amount: "1500" })).toEqual({
      label: "Freight",
      amount: 1500,
    });
  });

  // The bill as the server stores it (tarifftypes.Bill): decimals with their unit text, line
  // kinds capitalised, no label on an adjustment, an exchange to the billing currency.
  const storedBill = {
    currency: "CAD",
    lines: [
      {
        kind: "Charge",
        label: "Linehaul",
        freight: true,
        amount: "1908.2",
        quantity: "812",
        quantityUnit: "mi",
        rate: "2.35",
        rateUnit: "$/mi",
      },
      {
        kind: "Charge",
        label: "Extra drops",
        code: "XDROP",
        amount: "75",
        quantity: "1",
        rate: "75",
        rateUnit: "$",
      },
      { kind: "Minimum", freight: true, amount: "16.8" },
    ],
    notes: [{ text: "Two drops" }],
    freightTotal: "1925",
    accessorialTotal: "75",
    total: "2000",
    exchange: { currency: "USD", rate: "0.73" },
  };

  it("reads the stored bill into the wire form the bill views draw", () => {
    const parsed = ratingDetailSchema.parse({
      formulaTemplateId: "ft_1",
      formulaTemplateName: "Dry van TL",
      ratedAt: 1_700_000_000,
      bill: storedBill,
      inputs: { rate: { $q: "2.35", u: "$/mi" } },
      skipped: [{ code: "FSC", source: "FuelProgram" }],
    });

    expect(parsed.bill).toEqual({
      currency: "CAD",
      lines: [
        {
          kind: "charge",
          label: "Linehaul",
          code: null,
          freight: true,
          amount: { $q: "1908.2", u: "$" },
          quantity: { $q: "812", u: "mi" },
          rate: { $q: "2.35", u: "$/mi" },
          range: null,
        },
        {
          kind: "charge",
          label: "Extra drops",
          code: "XDROP",
          freight: false,
          amount: { $q: "75", u: "$" },
          quantity: { $n: "1" },
          rate: { $q: "75", u: "$" },
          range: null,
        },
        {
          kind: "minimum",
          label: null,
          code: null,
          freight: true,
          amount: { $q: "16.8", u: "$" },
          quantity: null,
          rate: null,
          range: null,
        },
      ],
      notes: [{ text: "Two drops", range: null }],
      freightTotal: { $q: "1925", u: "$" },
      accessorialTotal: { $q: "75", u: "$" },
      total: { $q: "2000", u: "$" },
      converted: null,
      exchange: { currency: "USD", rate: "0.73" },
    });
    expect(parsed.inputs).toEqual({ rate: { $q: "2.35", u: "$/mi" } });
    expect(parsed.skipped).toEqual([{ code: "FSC", source: "FuelProgram" }]);
  });

  it("reads an already-read bill again unchanged, as a form reset from a parsed record does", () => {
    const once = ratingDetailSchema.parse({
      formulaTemplateId: "ft_1",
      formulaTemplateName: "Dry van TL",
      ratedAt: 1_700_000_000,
      bill: storedBill,
    });
    const twice = ratingDetailSchema.parse(once);

    expect(twice.bill).toEqual(once.bill);
  });

  it("reads a refusal, which leaves the rating with no bill", () => {
    const parsed = ratingDetailSchema.parse({
      formulaTemplateId: "ft_1",
      formulaTemplateName: "Dry van TL",
      ratedAt: 1_700_000_000,
      refusal: "This rate needs the shipment's weight",
    });

    expect(parsed.bill ?? null).toBeNull();
    expect(parsed.refusal).toBe("This rate needs the shipment's weight");
  });
});
