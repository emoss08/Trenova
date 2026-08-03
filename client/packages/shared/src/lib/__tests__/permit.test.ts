import { describe, expect, it } from "vitest";
import {
  describeCommodityDimensions,
  describeRequirement,
  dimensionRows,
  escortSummary,
  formatFeetInches,
  formatPounds,
  hasOpenRequirements,
  isOversize,
  pickupIsTooSoon,
  restrictionSummary,
  unverifiedJurisdictions,
} from "@trenova/shared/lib/permit";
import type {
  AssessedJurisdiction,
  PermitAssessment,
  PermitRequirement,
} from "@trenova/shared/types/permit";

function jurisdiction(
  overrides: Partial<AssessedJurisdiction> & { stateCode: string },
): AssessedJurisdiction {
  return {
    stateId: `us_${overrides.stateCode}`,
    stateName: overrides.stateCode,
    sequence: 0,
    verificationState: "Unverified",
    maxWidthFeet: 8.5,
    maxHeightFeet: 13.5,
    maxLengthFeet: 53,
    maxWeightPounds: 80000,
    widthHeadroomFeet: 0,
    heightHeadroomFeet: 0,
    lengthHeadroomFeet: 0,
    weightHeadroomPounds: 0,
    requiresPermit: false,
    restrictions: [],
    ...overrides,
  };
}

function assessment(overrides: Partial<PermitAssessment> = {}): PermitAssessment {
  return {
    measurements: { widthFeet: 0, heightFeet: 0, lengthFeet: 0, weightPounds: 0 },
    jurisdictions: [],
    requirements: [],
    open: [],
    expiringBeforeDelivery: [],
    totalEscorts: 0,
    totalEstimatedFee: 0,
    maxLeadTimeDays: 0,
    earliestPickup: 0,
    routeResolved: true,
    feeIsBaseOnly: true,
    ...overrides,
  };
}

function requirement(
  overrides: Partial<PermitRequirement> & { stateCode: string },
): PermitRequirement {
  const { stateCode, ...rest } = overrides;

  return {
    id: `pmr_${stateCode}`,
    shipmentId: "shp_1",
    stateId: `us_${stateCode}`,
    status: "Open",
    routeSequence: 0,
    exceedances: [],
    escorts: [],
    restrictions: [],
    leadTimeDays: 0,
    validityDays: 0,
    estimatedFee: null,
    isSuperload: false,
    satisfiedByPermitId: null,
    waivedById: null,
    derivedAt: 0,
    provenance: {
      ruleId: `jrl_${stateCode}`,
      stateCode,
      stateName: stateCode,
      verificationState: "Unverified",
      maxWidthFeet: 8.5,
      maxHeightFeet: 13.5,
      maxLengthFeet: 53,
      maxWeightPounds: 80000,
    },
    ...rest,
  };
}

describe("formatFeetInches", () => {
  it("renders whole feet without an inches component", () => {
    expect(formatFeetInches(12)).toBe("12'");
  });

  it("splits a decimal foot into feet and inches", () => {
    expect(formatFeetInches(12.5)).toBe(`12' 6"`);
    expect(formatFeetInches(8.5)).toBe(`8' 6"`);
    expect(formatFeetInches(13.75)).toBe(`13' 9"`);
  });

  it("keeps the sign on a negative headroom", () => {
    expect(formatFeetInches(-4.25)).toBe(`-4' 3"`);
  });

  it("rolls eleven-and-a-half inches up to the next whole foot", () => {
    expect(formatFeetInches(12.99)).toBe("13'");
  });

  it("returns a placeholder for a non-finite measurement", () => {
    expect(formatFeetInches(Number.NaN)).toBe("—");
  });
});

describe("formatPounds", () => {
  it("groups thousands and names the unit", () => {
    expect(formatPounds(90000)).toBe("90,000 lbs");
  });
});

describe("describeCommodityDimensions", () => {
  it("renders a fully dimensioned line as L × W × H", () => {
    expect(describeCommodityDimensions({ lengthFeet: 12, widthFeet: 8.5, heightFeet: 9 })).toBe(
      `12' × 8' 6" × 9'`,
    );
  });

  it("marks a missing side rather than hiding the whole line", () => {
    expect(describeCommodityDimensions({ lengthFeet: 12, widthFeet: null, heightFeet: 9 })).toBe(
      `12' × ? × 9'`,
    );
  });

  it("collapses to a dash only when no side was entered", () => {
    expect(describeCommodityDimensions({})).toBe("—");
    expect(describeCommodityDimensions(null)).toBe("—");
  });
});

describe("dimensionRows", () => {
  it("reports the tightest jurisdiction per dimension, not the first", () => {
    const rows = dimensionRows(
      assessment({
        measurements: {
          widthFeet: 8.25,
          heightFeet: 13,
          lengthFeet: 48,
          weightPounds: 74000,
        },
        jurisdictions: [
          jurisdiction({
            stateCode: "GA",
            sequence: 0,
            maxWidthFeet: 8.5,
            widthHeadroomFeet: 0.25,
            heightHeadroomFeet: 0.5,
            lengthHeadroomFeet: 5,
            weightHeadroomPounds: 6000,
          }),
          jurisdiction({
            stateCode: "TN",
            sequence: 1,
            maxWidthFeet: 8.5,
            widthHeadroomFeet: 0.25,
            heightHeadroomFeet: 0.1,
            lengthHeadroomFeet: 2,
            weightHeadroomPounds: 1000,
          }),
        ],
      }),
    );

    const height = rows.find((row) => row.key === "height");
    expect(height?.governingStateCode).toBe("TN");
    expect(height?.headroom).toBeCloseTo(0.1);
    expect(height?.exceeded).toBe(false);
  });

  it("marks a dimension exceeded when the tightest headroom goes negative", () => {
    const rows = dimensionRows(
      assessment({
        measurements: {
          widthFeet: 12.5,
          heightFeet: 13,
          lengthFeet: 48,
          weightPounds: 74000,
        },
        jurisdictions: [
          jurisdiction({
            stateCode: "GA",
            widthHeadroomFeet: -4,
            heightHeadroomFeet: 0.5,
            lengthHeadroomFeet: 5,
            weightHeadroomPounds: 6000,
          }),
        ],
      }),
    );

    const width = rows.find((row) => row.key === "width");
    expect(width?.exceeded).toBe(true);
    expect(width?.headroom).toBe(-4);
    expect(width?.actual).toBe(12.5);
    expect(width?.limit).toBe(8.5);
  });

  it("returns a limitless row when no jurisdiction resolved", () => {
    const rows = dimensionRows(
      assessment({
        measurements: {
          widthFeet: 12.5,
          heightFeet: 0,
          lengthFeet: 0,
          weightPounds: 0,
        },
      }),
    );

    expect(rows).toHaveLength(4);
    expect(rows[0]?.limit).toBeNull();
    expect(rows[0]?.headroom).toBeNull();
    expect(rows[0]?.governingStateCode).toBeNull();
    expect(rows[0]?.exceeded).toBe(false);
  });

  it("returns nothing without an assessment", () => {
    expect(dimensionRows(null)).toEqual([]);
  });
});

describe("describeRequirement", () => {
  it("names the state, the dimension, and the amount over", () => {
    expect(
      describeRequirement(
        requirement({
          stateCode: "GA",
          exceedances: [
            { trigger: "Width", limit: 8.5, actual: 12.5, overBy: 4, superload: false },
          ],
        }),
      ),
    ).toBe(`GA permit required (width 4' over)`);
  });

  it("renders an overweight exceedance in pounds, not feet", () => {
    expect(
      describeRequirement(
        requirement({
          stateCode: "OH",
          exceedances: [
            {
              trigger: "Weight",
              limit: 80000,
              actual: 92000,
              overBy: 12000,
              superload: false,
            },
          ],
        }),
      ),
    ).toBe("OH permit required (weight 12,000 lbs over)");
  });

  it("lists every exceedance rather than only the first", () => {
    const described = describeRequirement(
      requirement({
        stateCode: "NE",
        exceedances: [
          { trigger: "Width", limit: 8.5, actual: 12.5, overBy: 4, superload: false },
          { trigger: "Height", limit: 13.5, actual: 14.5, overBy: 1, superload: false },
        ],
      }),
    );

    expect(described).toContain("width");
    expect(described).toContain("height");
  });

  it("falls back to a bare sentence when no exceedance was recorded", () => {
    expect(describeRequirement(requirement({ stateCode: "GA" }))).toBe("GA permit required");
  });
});

describe("escortSummary", () => {
  it("counts a role once across the route and names every state demanding it", () => {
    const summary = escortSummary([
      requirement({
        stateCode: "GA",
        escorts: [
          { role: "Front", trigger: "Width", threshold: 12, actual: 12.5 },
          { role: "Rear", trigger: "Width", threshold: 12, actual: 12.5 },
        ],
      }),
      requirement({
        stateCode: "TN",
        escorts: [{ role: "Front", trigger: "Width", threshold: 10, actual: 12.5 }],
      }),
    ]);

    expect(summary).toHaveLength(2);

    const front = summary.find((entry) => entry.role === "Front");
    expect(front?.label).toBe("Front escort");
    expect(front?.stateCodes).toEqual(["GA", "TN"]);

    const rear = summary.find((entry) => entry.role === "Rear");
    expect(rear?.stateCodes).toEqual(["GA"]);
  });

  it("returns nothing when no jurisdiction requires an escort", () => {
    expect(escortSummary([requirement({ stateCode: "GA" })])).toEqual([]);
  });
});

describe("restrictionSummary", () => {
  it("ignores restrictions from jurisdictions the load clears", () => {
    const summary = restrictionSummary([
      jurisdiction({
        stateCode: "GA",
        requiresPermit: true,
        restrictions: ["DaylightOnly", "Holiday"],
      }),
      jurisdiction({
        stateCode: "TN",
        requiresPermit: false,
        restrictions: ["DaylightOnly", "Weekend"],
      }),
    ]);

    expect(summary).toHaveLength(2);

    const daylight = summary.find((entry) => entry.kind === "DaylightOnly");
    expect(daylight?.label).toBe("Daylight movement only");
    expect(daylight?.stateCodes).toEqual(["GA"]);
    expect(summary.some((entry) => entry.kind === "Weekend")).toBe(false);
  });
});

describe("unverifiedJurisdictions", () => {
  it("flags seeded and disputed rows but not confirmed ones", () => {
    const flagged = unverifiedJurisdictions(
      assessment({
        jurisdictions: [
          jurisdiction({ stateCode: "GA", verificationState: "Verified" }),
          jurisdiction({ stateCode: "TN", verificationState: "Unverified" }),
          jurisdiction({ stateCode: "OH", verificationState: "Disputed" }),
        ],
      }),
    );

    expect(flagged.map((entry) => entry.stateCode)).toEqual(["TN", "OH"]);
  });
});

describe("pickupIsTooSoon", () => {
  const day = 86400;

  it("is true when the booked pickup lands inside the permit lead time", () => {
    expect(
      pickupIsTooSoon(
        assessment({ maxLeadTimeDays: 5, earliestPickup: 1000 + 5 * day }),
        1000 + 2 * day,
      ),
    ).toBe(true);
  });

  it("is false when the pickup clears the lead time exactly", () => {
    expect(
      pickupIsTooSoon(
        assessment({ maxLeadTimeDays: 5, earliestPickup: 1000 + 5 * day }),
        1000 + 5 * day,
      ),
    ).toBe(false);
  });

  it("is false when no jurisdiction imposes a lead time", () => {
    expect(
      pickupIsTooSoon(assessment({ maxLeadTimeDays: 0, earliestPickup: 9_999_999 }), 1000),
    ).toBe(false);
  });

  it("is false when the shipment has no scheduled pickup to compare", () => {
    expect(pickupIsTooSoon(assessment({ maxLeadTimeDays: 5, earliestPickup: 9_999_999 }), 0)).toBe(
      false,
    );
  });
});

describe("assessment predicates", () => {
  it("separates an oversize load from one with an unsatisfied requirement", () => {
    const satisfied = assessment({
      requirements: [requirement({ stateCode: "GA", status: "Satisfied" })],
      open: [],
    });

    expect(isOversize(satisfied)).toBe(true);
    expect(hasOpenRequirements(satisfied)).toBe(false);
  });

  it("reports a legal load as neither oversize nor open", () => {
    expect(isOversize(assessment())).toBe(false);
    expect(hasOpenRequirements(assessment())).toBe(false);
  });
});
