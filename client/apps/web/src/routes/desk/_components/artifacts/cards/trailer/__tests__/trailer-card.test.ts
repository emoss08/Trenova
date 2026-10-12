import type { DeskTrailer } from "@/lib/graphql/trailer-card";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import {
  annualInspectionCheck,
  maxLoadText,
  nextInspectionDue,
  trailerChecks,
  trailerFacts,
} from "../trailer-card";

const t = ((message: string, ...args: unknown[]) =>
  message
    .replace(
      /\{(\d+), plural, one \{([^}]*)\} other \{([^}]*)\}\}/gu,
      (_, index, one: string, other: string) => {
        const count = Number(args[Number(index)]);
        return (count === 1 ? one : other).replace("#", String(count));
      },
    )
    .replace(/\{(\d+)\}/gu, (_, index) => String(args[Number(index)]))) as TranslateFn;

const DAY = 86_400;
const NOW = 1_791_720_000;

function trailer(overrides: Partial<DeskTrailer> = {}): DeskTrailer {
  return {
    id: "trl_1",
    code: "TL-501",
    status: "Available",
    make: "Wabash",
    model: "DuraPlate",
    year: 2021,
    vin: "1JJV532D8ML123456",
    licensePlateNumber: "TX-T5501",
    registrationNumber: "REG-T5501",
    registrationExpiry: NOW + 200 * DAY,
    lastInspectionDate: NOW - 100 * DAY,
    maxLoadWeight: 45_000,
    externalId: "",
    ownershipType: "CompanyOwned",
    lessorName: "",
    leaseReference: "",
    leaseEndDate: null,
    createdAt: 1_700_000_000,
    registrationState: { abbreviation: "TX" },
    equipmentType: { id: "et_dv", code: "DV53", description: "53' dry van" },
    equipmentManufacturer: { id: "em_wb", name: "Wabash" },
    fleetCode: { id: "fc_dal", code: "DAL" },
    ownerWorker: null,
    lastKnownLocation: null,
    ...overrides,
  } as DeskTrailer;
}

describe("nextInspectionDue", () => {
  it("is twelve months after the last inspection, and nothing without one", () => {
    expect(nextInspectionDue(NOW)).toBe(NOW + 365 * DAY);
    expect(nextInspectionDue(null)).toBeNull();
    expect(nextInspectionDue(undefined)).toBeNull();
    expect(nextInspectionDue(0)).toBeNull();
  });
});

describe("annualInspectionCheck", () => {
  it("warns when no inspection is on file", () => {
    expect(annualInspectionCheck(null, NOW, t)).toMatchObject({
      state: "warn",
      detail: "No inspection on file",
    });
  });

  it("fails an inspection more than a year old", () => {
    expect(annualInspectionCheck(NOW - 400 * DAY, NOW, t)).toMatchObject({
      state: "bad",
      title: "Annual inspection overdue",
    });
  });

  it("warns inside the last month, and passes before it", () => {
    expect(annualInspectionCheck(NOW - 350 * DAY, NOW, t)).toMatchObject({
      state: "warn",
      title: "Annual inspection due in 15 days",
    });
    expect(annualInspectionCheck(NOW - 100 * DAY, NOW, t)).toMatchObject({
      state: "ok",
      title: "Annual inspection",
    });
  });
});

describe("trailerChecks", () => {
  it("passes a trailer in service, inspected and registered", () => {
    expect(trailerChecks(trailer(), NOW, t).map((check) => [check.key, check.state])).toEqual([
      ["inspection", "ok"],
      ["registration", "ok"],
    ]);
  });

  it("puts what is wrong first, and reads the lease of a leased trailer", () => {
    const checks = trailerChecks(
      trailer({
        status: "OutOfService",
        lastInspectionDate: NOW - 500 * DAY,
        registrationExpiry: NOW + 5 * DAY,
        ownershipType: "Leased",
        lessorName: "XTRA Lease",
        leaseEndDate: NOW + 100 * DAY,
      }),
      NOW,
      t,
    );
    expect(checks.map((check) => [check.key, check.state])).toEqual([
      ["status", "bad"],
      ["inspection", "bad"],
      ["registration", "warn"],
      ["lease", "ok"],
    ]);
  });
});

describe("maxLoadText and trailerFacts", () => {
  it("writes the maximum load in pounds, and nothing for none", () => {
    expect(maxLoadText(45_000, t)).toBe("45,000 lb");
    expect(maxLoadText(null, t)).toBeNull();
    expect(maxLoadText(0, t)).toBeNull();
  });

  it("lists the maximum load after the manufacturer, with the registration state on the plate", () => {
    const facts = trailerFacts(trailer(), t);
    expect(facts.map((fact) => fact.key)).toEqual([
      "plate",
      "registration",
      "vin",
      "manufacturer",
      "load",
      "ownership",
      "external",
      "since",
    ]);
    expect(facts[0]?.value).toBe("TX-T5501 · TX");
    expect(facts[4]?.value).toBe("45,000 lb");
    expect(trailerFacts(trailer({ maxLoadWeight: null }), t)[4]?.value).toBe("None on file");
  });
});
