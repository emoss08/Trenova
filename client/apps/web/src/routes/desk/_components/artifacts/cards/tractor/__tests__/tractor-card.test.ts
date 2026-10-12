import type {
  DeskTractor,
  DeskTractorInspection,
  DeskTractorPosition,
} from "@/lib/graphql/tractor-card";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import {
  compassPoint,
  inspectionKind,
  inspectionReading,
  positionReading,
  tractorChecks,
  tractorFacts,
  tractorPeople,
} from "../tractor-card";

/** Fills a message's numbered places the way the catalog would, plurals read by their count. */
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

function tractor(overrides: Partial<DeskTractor> = {}): DeskTractor {
  return {
    id: "tr_1",
    code: "T-101",
    status: "Available",
    make: "Freightliner",
    model: "Cascadia",
    year: 2022,
    vin: "3AKJHHDR5NSNA1234",
    licensePlateNumber: "TX-48812",
    registrationNumber: "REG-99812",
    registrationExpiry: NOW + 200 * DAY,
    externalId: "",
    fuelType: "Diesel",
    iftaQualified: true,
    ownershipType: "CompanyOwned",
    lessorName: "",
    leaseReference: "",
    leaseEndDate: null,
    createdAt: 1_700_000_000,
    state: { abbreviation: "TX" },
    equipmentType: { id: "et_1", code: "SLP", description: "Sleeper tractor" },
    equipmentManufacturer: { id: "em_1", name: "Freightliner" },
    fleetCode: { id: "fc_1", code: "DAL" },
    primaryWorker: { id: "wrk_1", firstName: "Rosa", lastName: "Diaz" },
    secondaryWorker: null,
    ownerWorker: null,
    lastKnownLocation: null,
    ...overrides,
  } as DeskTractor;
}

function inspection(overrides: Partial<DeskTractorInspection> = {}): DeskTractorInspection {
  return {
    id: "vinsp_1",
    inspectionType: "preTrip",
    safetyStatus: "safe",
    workerName: "Rosa Diaz",
    startedAt: NOW - 2 * DAY,
    location: "Dallas, TX",
    odometerMeters: 293_000_000,
    defectCount: 0,
    unresolvedDefectCount: 0,
    ...overrides,
  };
}

function position(overrides: Partial<DeskTractorPosition> = {}): DeskTractorPosition {
  return {
    latitude: 32.7767,
    longitude: -96.797,
    headingDegrees: 40,
    speedMph: 58.4,
    engineState: "On",
    fuelPercent: 62,
    odometerMeters: 293_100_000,
    formattedLocation: "I-35E near Waxahachie, TX",
    recordedAt: NOW - 120,
    ...overrides,
  };
}

describe("tractorPeople", () => {
  it("lists the driver and co-driver", () => {
    const people = tractorPeople(
      tractor({ secondaryWorker: { id: "wrk_2", firstName: "Sam", lastName: "Lee" } }),
      t,
    );
    expect(people).toEqual([
      { key: "primary", id: "wrk_1", name: "Rosa Diaz", role: "Primary driver" },
      { key: "secondary", id: "wrk_2", name: "Sam Lee", role: "Co-driver" },
    ]);
  });

  it("lists an owner-operator who does not drive it on their own row", () => {
    const people = tractorPeople(
      tractor({
        ownershipType: "OwnerOperator",
        ownerWorker: { id: "wrk_9", firstName: "Ana", lastName: "Ruiz" },
      }),
      t,
    );
    expect(people.map((person) => [person.id, person.role])).toEqual([
      ["wrk_1", "Primary driver"],
      ["wrk_9", "Owner-operator"],
    ]);
  });

  it("lists an owner-operator who drives it once, as both", () => {
    const people = tractorPeople(
      tractor({
        ownershipType: "OwnerOperator",
        ownerWorker: { id: "wrk_1", firstName: "Rosa", lastName: "Diaz" },
      }),
      t,
    );
    expect(people).toEqual([
      {
        key: "primary",
        id: "wrk_1",
        name: "Rosa Diaz",
        role: "Owner-operator and primary driver",
      },
    ]);
  });

  it("ignores an owner on file for a tractor the company owns", () => {
    const people = tractorPeople(
      tractor({ ownerWorker: { id: "wrk_9", firstName: "Ana", lastName: "Ruiz" } }),
      t,
    );
    expect(people.map((person) => person.id)).toEqual(["wrk_1"]);
  });

  it("does not list the same worker twice", () => {
    const people = tractorPeople(
      tractor({ secondaryWorker: { id: "wrk_1", firstName: "Rosa", lastName: "Diaz" } }),
      t,
    );
    expect(people).toHaveLength(1);
  });

  it("names a worker with no name on file", () => {
    const people = tractorPeople(
      tractor({ primaryWorker: { id: "wrk_1", firstName: "", lastName: "" } }),
      t,
    );
    expect(people[0]?.name).toBe("Unnamed worker");
  });
});

describe("tractorChecks", () => {
  it("passes a tractor in service with registration ahead and a clean inspection", () => {
    const checks = tractorChecks(tractor(), inspection(), NOW, t);
    expect(checks.map((check) => [check.key, check.state])).toEqual([
      ["registration", "ok"],
      ["inspection", "ok"],
    ]);
    expect(checks[1]?.detail).toMatch(/^Pre-trip · /u);
  });

  it("puts what is wrong first: out of service, expired registration, open defects", () => {
    const checks = tractorChecks(
      tractor({ status: "OutOfService", registrationExpiry: NOW - 3 * DAY }),
      inspection({ unresolvedDefectCount: 2, defectCount: 3 }),
      NOW,
      t,
    );
    expect(checks.map((check) => [check.key, check.state, check.title])).toEqual([
      ["status", "bad", "Out of service"],
      ["registration", "bad", "Registration expired"],
      ["inspection", "bad", "2 open defects"],
    ]);
  });

  it("warns of registration expiring soon and of none on file", () => {
    expect(
      tractorChecks(tractor({ registrationExpiry: NOW + 12 * DAY }), undefined, NOW, t)[0],
    ).toMatchObject({ state: "warn", title: "Registration expires in 12 days" });
    expect(
      tractorChecks(tractor({ registrationExpiry: null }), undefined, NOW, t)[0],
    ).toMatchObject({ state: "warn", detail: "No expiry on file" });
  });

  it("reads the lease only on a leased tractor, naming the lessor", () => {
    const leased = tractorChecks(
      tractor({
        ownershipType: "Leased",
        lessorName: "Penske",
        leaseEndDate: NOW + 45 * DAY,
      }),
      undefined,
      NOW,
      t,
    );
    expect(leased.find((check) => check.key === "lease")).toMatchObject({
      state: "warn",
      title: "Lease ends in 45 days",
    });
    expect(leased.find((check) => check.key === "lease")?.detail).toContain("Penske");

    const owned = tractorChecks(tractor({ leaseEndDate: NOW + DAY }), undefined, NOW, t);
    expect(owned.some((check) => check.key === "lease")).toBe(false);
  });

  it("leaves the inspection out while it is read, and warns when there has been none", () => {
    expect(tractorChecks(tractor(), undefined, NOW, t).some((c) => c.key === "inspection")).toBe(
      false,
    );
    expect(tractorChecks(tractor(), null, NOW, t).at(0)).toMatchObject({
      key: "inspection",
      state: "warn",
      detail: "No driver inspection on record",
    });
  });

  it("fails an inspection the driver marked unsafe even with no defect left open", () => {
    const checks = tractorChecks(tractor(), inspection({ safetyStatus: "UNSAFE" }), NOW, t);
    expect(checks[0]).toMatchObject({ key: "inspection", state: "bad" });
  });

  it("holds a tractor at maintenance as a warning, not a failure", () => {
    expect(tractorChecks(tractor({ status: "AtMaintenance" }), undefined, NOW, t)[0]).toMatchObject(
      { key: "status", state: "warn" },
    );
  });
});

describe("inspectionKind", () => {
  it("names the provider's kinds, whatever their case", () => {
    expect(inspectionKind("preTrip", t)).toBe("Pre-trip");
    expect(inspectionKind("POSTTRIP", t)).toBe("Post-trip");
    expect(inspectionKind("mechanic", t)).toBe("Mechanic");
    expect(inspectionKind("", t)).toBe("Inspection");
    expect(inspectionKind("yardCheck", t)).toBe("yardCheck");
  });
});

describe("inspectionReading", () => {
  it("reads open defects first, then an unsafe verdict, then fixed defects", () => {
    expect(inspectionReading(inspection({ unresolvedDefectCount: 1, defectCount: 1 }), t)).toEqual({
      state: "bad",
      text: "1 open defect",
    });
    expect(inspectionReading(inspection({ safetyStatus: "unsafe" }), t)).toEqual({
      state: "bad",
      text: "Unsafe",
    });
    expect(inspectionReading(inspection({ defectCount: 2 }), t)).toEqual({
      state: "ok",
      text: "2 defects fixed",
    });
    expect(inspectionReading(inspection(), t)).toEqual({ state: "ok", text: "No defects" });
  });
});

describe("compassPoint", () => {
  it("names the nearest of eight points, wrapping round north", () => {
    expect(compassPoint(0)).toBe("N");
    expect(compassPoint(40)).toBe("NE");
    expect(compassPoint(350)).toBe("N");
    expect(compassPoint(-90)).toBe("W");
    expect(compassPoint(540)).toBe("S");
  });
});

describe("positionReading", () => {
  it("is live and moving on a fresh report above walking pace", () => {
    expect(positionReading(position(), NOW, t)).toEqual({
      live: true,
      motion: "Moving 58 mph NE",
      age: 120,
      place: "I-35E near Waxahachie, TX",
    });
  });

  it("says what the tractor was doing when the report is old", () => {
    const reading = positionReading(position({ recordedAt: NOW - 3 * 3600 }), NOW, t);
    expect(reading.live).toBe(false);
    expect(reading.motion).toBe("Was moving 58 mph NE");
  });

  it("tells idling, standing with the engine on and stopped apart", () => {
    expect(positionReading(position({ speedMph: 0, engineState: "Idle" }), NOW, t).motion).toBe(
      "Idling",
    );
    expect(positionReading(position({ speedMph: 1.2, engineState: "On" }), NOW, t).motion).toBe(
      "Standing, engine on",
    );
    expect(positionReading(position({ speedMph: 0, engineState: null }), NOW, t).motion).toBe(
      "Stopped",
    );
  });

  it("has no place for a report without an address", () => {
    expect(positionReading(position({ formattedLocation: "  " }), NOW, t).place).toBeNull();
    expect(positionReading(position({ formattedLocation: null }), NOW, t).place).toBeNull();
  });

  it("never reads a report from the future as negative age", () => {
    expect(positionReading(position({ recordedAt: NOW + 30 }), NOW, t).age).toBe(0);
  });
});

describe("tractorFacts", () => {
  it("lists what the tractor is, in a dispatcher's order", () => {
    const facts = tractorFacts(tractor(), t);
    expect(facts.map((fact) => [fact.key, fact.value])).toEqual([
      ["plate", "TX-48812 · TX"],
      ["registration", "REG-99812"],
      ["vin", "3AKJHHDR5NSNA1234"],
      ["manufacturer", "Freightliner"],
      ["fuel", "Diesel · IFTA"],
      ["ownership", "Company owned"],
      ["external", "None"],
      ["since", expect.any(String)],
    ]);
  });

  it("says what is not on file, and names a lessor on a leased tractor", () => {
    const facts = tractorFacts(
      tractor({
        licensePlateNumber: "",
        vin: "",
        equipmentManufacturer: null,
        iftaQualified: false,
        ownershipType: "Leased",
        lessorName: "Penske",
        leaseReference: "L-2231",
      }),
      t,
    );
    const value = (key: string) => facts.find((fact) => fact.key === key)?.value;
    expect(value("plate")).toBe("None on file");
    expect(value("vin")).toBe("None on file");
    expect(value("manufacturer")).toBe("None on file");
    expect(value("fuel")).toBe("Diesel · not on IFTA");
    expect(value("ownership")).toBe("Leased · Penske · L-2231");
  });
});
