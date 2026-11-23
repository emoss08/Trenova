import type { EffectiveWorkerPayAssignment } from "@/lib/graphql/driver-settlement";
import { describe, expect, it } from "vitest";
import { compensationShape } from "../benefits/compensation-bar";
import { formatPayRate, overrideStart, payComponentLines } from "../pay-model";

const assignment = {
  id: "wpa_1",
  workerId: "wrk_1",
  payProfileId: "pp_1",
  effectiveFrom: 1_751_328_000,
  effectiveTo: null,
  splitPercent: "100",
  notes: null,
  rateOverrides: [
    { componentId: "c_loaded", rate: "0.62" },
    { componentId: "c_stop", rate: "25" },
  ],
  payProfile: {
    id: "pp_1",
    name: "OTR mileage, company driver",
    classification: "CompanyDriver",
    guaranteedPeriodMinimumMinor: 120_000,
    components: [
      {
        id: "c_loaded",
        kind: "Linehaul",
        method: "PerLoadedMile",
        description: "Loaded miles",
        rate: "0.60",
        revenueBasis: null,
        bands: [],
        isActive: true,
      },
      {
        id: "c_stop",
        kind: "StopPay",
        method: "PerStop",
        description: "",
        rate: "25.00",
        revenueBasis: null,
        bands: null,
        isActive: true,
      },
      {
        id: "c_old",
        kind: "Bonus",
        method: "PerEvent",
        description: "Retired bonus",
        rate: "50",
        revenueBasis: null,
        bands: null,
        isActive: false,
      },
    ],
  },
} as unknown as EffectiveWorkerPayAssignment;

describe("payComponentLines", () => {
  it("lists the active components, naming one by its kind when it has no description, and keeps only overrides that differ", () => {
    expect(payComponentLines(assignment)).toEqual([
      {
        id: "c_loaded",
        name: "Loaded miles",
        method: "PerLoadedMile",
        profileRate: "0.60",
        override: "0.62",
        bandCount: 0,
      },
      {
        id: "c_stop",
        name: "Stop pay",
        method: "PerStop",
        profileRate: "25.00",
        override: null,
        bandCount: 0,
      },
    ]);
  });

  it("offers the current overrides as the change form's starting rates", () => {
    expect(overrideStart(assignment)).toEqual({ c_loaded: "0.62", c_stop: "25" });
    expect(overrideStart(null)).toEqual({});
  });
});

describe("formatPayRate", () => {
  it("keeps fractional cents on money and writes a percentage of revenue as a percent", () => {
    expect(formatPayRate("PerLoadedMile", "0.625")).toBe("$0.625");
    expect(formatPayRate("PerStop", "25")).toBe("$25.00");
    expect(formatPayRate("PercentOfRevenue", "27.5")).toBe("27.5%");
  });
});

describe("compensationShape", () => {
  it("splits the bar into pay and employer benefits and brackets the contribution over the end of the pay", () => {
    const shape = compensationShape({
      grossPayMinor: 7_124_000,
      employerBenefitMinor: 986_000,
      employeeBenefitMinor: 442_000,
    });
    expect(shape.total).toBe(8_110_000);
    expect(shape.paidShare).toBeCloseTo(0.87842, 4);
    expect(shape.benefitShare).toBeCloseTo(0.12158, 4);
    expect(shape.contributionStart).toBeCloseTo((7_124_000 - 442_000) / 8_110_000, 6);
    expect(shape.contributionShare).toBeCloseTo(442_000 / 8_110_000, 6);
    expect(shape.ticks).toEqual([0, 2_000_000, 4_000_000, 6_000_000, 8_000_000]);
  });

  it("never brackets more than was paid", () => {
    const shape = compensationShape({
      grossPayMinor: 1_000,
      employerBenefitMinor: 1_000,
      employeeBenefitMinor: 5_000,
    });
    expect(shape.contributionStart).toBe(0);
    expect(shape.contributionShare).toBe(0.5);
  });

  it("draws nothing for a year with nothing paid", () => {
    expect(
      compensationShape({ grossPayMinor: 0, employerBenefitMinor: 0, employeeBenefitMinor: 0 }),
    ).toEqual({
      total: 0,
      paidShare: 0,
      benefitShare: 0,
      contributionStart: 0,
      contributionShare: 0,
      ticks: [],
    });
  });
});
