import { describe, expect, it } from "vitest";
import {
  costTotals,
  defaultPlanYear,
  emptyActivePlans,
  endingSoon,
  enrollmentStanding,
  groupPlansByType,
  matchesEnrollmentSearch,
  planYearsOf,
  recentDeclines,
  startingSoon,
} from "../benefits-console";

const DAY = 86_400;
const NOW = Date.UTC(2026, 8, 7, 15, 30) / 1000;

function plan(
  over: Partial<{
    id: string;
    status: string;
    planType: string;
    planYear: number;
    employeeCostMinor: number;
    employerCostMinor: number;
  }> = {},
) {
  return {
    id: "bpl_1",
    status: "Active",
    planType: "Medical",
    planYear: 2026,
    employeeCostMinor: 12_000,
    employerCostMinor: 48_000,
    ...over,
  };
}

function cost(
  over: Partial<{
    planId: string;
    enrolled: number;
    waived: number;
    employeeCostMinor: number;
    employerCostMinor: number;
  }> = {},
) {
  return {
    planId: "bpl_1",
    enrolled: 10,
    waived: 2,
    employeeCostMinor: 120_000,
    employerCostMinor: 480_000,
    ...over,
  };
}

function enrollment(
  over: Partial<{
    id: string;
    status: string;
    effectiveFrom: number;
    effectiveTo: number | null;
  }> = {},
) {
  return {
    id: "ben_1",
    status: "Active",
    effectiveFrom: NOW - 100 * DAY,
    effectiveTo: null,
    ...over,
  };
}

describe("plan years", () => {
  it("lists the years on file newest first, without repeats", () => {
    expect(
      planYearsOf([plan({ planYear: 2025 }), plan({ planYear: 2026 }), plan({ planYear: 2025 })]),
    ).toEqual([2026, 2025]);
  });

  // Next year's plans are often priced before this year is over; the page
  // still opens on the year being administered.
  it("opens on this year when it has a plan, else the newest year", () => {
    expect(defaultPlanYear([2027, 2026, 2025], 2026)).toBe(2026);
    expect(defaultPlanYear([2027, 2025], 2026)).toBe(2027);
    expect(defaultPlanYear([], 2026)).toBeNull();
  });
});

describe("costTotals", () => {
  it("adds people and money across the rows", () => {
    expect(
      costTotals([
        cost(),
        cost({
          planId: "bpl_2",
          enrolled: 4,
          waived: 0,
          employeeCostMinor: 8_000,
          employerCostMinor: 2_000,
        }),
      ]),
    ).toEqual({
      enrolled: 14,
      waived: 2,
      employeeMinor: 128_000,
      employerMinor: 482_000,
      totalMinor: 610_000,
    });
  });
});

describe("groupPlansByType", () => {
  it("groups in the administrator's order with the kind's own totals", () => {
    const groups = groupPlansByType(
      [
        plan({ id: "d1", planType: "Dental" }),
        plan({ id: "m1", planType: "Medical" }),
        plan({ id: "m2", planType: "Medical" }),
        plan({ id: "x1", planType: "Custom" }),
      ],
      [cost({ planId: "m1", enrolled: 3 }), cost({ planId: "m2", enrolled: 7, waived: 1 })],
    );
    expect(groups.map((group) => group.type)).toEqual(["Medical", "Dental", "Custom"]);
    expect(groups[0]).toMatchObject({ label: "Medical", enrolled: 10, waived: 3 });
    expect(groups[0].plans.map(({ plan: p }) => p.id)).toEqual(["m2", "m1"]);
    expect(groups[1].plans[0].cost).toBeNull();
  });

  it("puts archived plans after active ones whatever their size", () => {
    const groups = groupPlansByType(
      [plan({ id: "old", status: "Inactive" }), plan({ id: "new" })],
      [cost({ planId: "old", enrolled: 50 }), cost({ planId: "new", enrolled: 1 })],
    );
    expect(groups[0].plans.map(({ plan: p }) => p.id)).toEqual(["new", "old"]);
  });

  it("finds active plans nobody has joined", () => {
    expect(
      emptyActivePlans(
        [plan({ id: "a" }), plan({ id: "b" }), plan({ id: "c", status: "Inactive" })],
        [cost({ planId: "a", enrolled: 0 }), cost({ planId: "b", enrolled: 2 })],
      ).map((p) => p.id),
    ).toEqual(["a"]);
  });
});

describe("enrollmentStanding", () => {
  // The stored status says Active for cover dated next month and cover that
  // ends on Friday alike; the page has to tell them apart.
  it("reads the calendar as well as the status", () => {
    expect(enrollmentStanding(enrollment(), NOW)).toBe("covered");
    expect(enrollmentStanding(enrollment({ effectiveFrom: NOW + DAY }), NOW)).toBe("starting");
    expect(enrollmentStanding(enrollment({ effectiveTo: NOW + 10 * DAY }), NOW)).toBe("ending");
    expect(enrollmentStanding(enrollment({ effectiveTo: NOW + 31 * DAY }), NOW)).toBe("covered");
    expect(enrollmentStanding(enrollment({ effectiveTo: NOW - DAY }), NOW)).toBe("ended");
    expect(enrollmentStanding(enrollment({ status: "Waived" }), NOW)).toBe("declined");
    expect(enrollmentStanding(enrollment({ status: "Ended", effectiveFrom: NOW + DAY }), NOW)).toBe(
      "ended",
    );
  });

  it("lists what is starting soonest first and what is ending soonest first", () => {
    const entries = [
      enrollment({ id: "later", effectiveFrom: NOW + 20 * DAY }),
      enrollment({ id: "sooner", effectiveFrom: NOW + 2 * DAY }),
      enrollment({ id: "ends-late", effectiveTo: NOW + 25 * DAY }),
      enrollment({ id: "ends-soon", effectiveTo: NOW + 3 * DAY }),
      enrollment({ id: "steady" }),
    ];
    expect(startingSoon(entries, NOW).map((e) => e.id)).toEqual(["sooner", "later"]);
    expect(endingSoon(entries, NOW).map((e) => e.id)).toEqual(["ends-soon", "ends-late"]);
  });

  it("lists declines most recent first", () => {
    const entries = [
      enrollment({ id: "old", status: "Waived", effectiveFrom: NOW - 50 * DAY }),
      enrollment({ id: "covered" }),
      enrollment({ id: "new", status: "Waived", effectiveFrom: NOW - 5 * DAY }),
    ];
    expect(recentDeclines(entries).map((e) => e.id)).toEqual(["new", "old"]);
  });
});

describe("matchesEnrollmentSearch", () => {
  it("matches the worker's name or terminal, and nothing without a worker", () => {
    const entry = {
      worker: { firstName: "Ada", lastName: "Byrne", fleetCode: { code: "SOUTH" } },
    };
    expect(matchesEnrollmentSearch(entry, "byr")).toBe(true);
    expect(matchesEnrollmentSearch(entry, "south")).toBe(true);
    expect(matchesEnrollmentSearch(entry, "north")).toBe(false);
    expect(matchesEnrollmentSearch({ worker: null }, "ada")).toBe(false);
    expect(matchesEnrollmentSearch({ worker: null }, "")).toBe(true);
  });
});
