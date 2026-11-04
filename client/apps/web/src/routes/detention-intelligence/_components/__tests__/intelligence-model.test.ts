import type {
  DetentionFacility,
  DetentionIntelligence,
} from "@/lib/graphql/detention-intelligence";
import { describe, expect, it } from "vitest";
import {
  afternoonShare,
  arrivalPeak,
  concentration,
  foldArrivals,
  keptAfterDriverPay,
  rankFacilities,
  weekDaysInWindow,
  recoverySteps,
} from "../intelligence-model";

function facility(locationId: string, pastFreeMinutes: number): DetentionFacility {
  return { locationId, pastFreeMinutes } as DetentionFacility;
}

describe("concentration", () => {
  it("takes the fewest facilities, most hours first, that reach three quarters of the hours", () => {
    const cut = concentration(
      [facility("a", 600), facility("b", 200), facility("c", 100), facility("d", 100)],
      1_000,
    );

    expect(cut.count).toBe(2);
    expect(cut.minutes).toBe(800);
    expect(cut.share).toBe(0.8);
    expect([...cut.ids]).toEqual(["a", "b"]);
  });

  it("stops at ten facilities when detention is spread too thin to reach the share", () => {
    const spread = Array.from({ length: 40 }, (_, index) => facility(`f${index}`, 100));
    const cut = concentration(spread, 4_000);

    expect(cut.count).toBe(10);
    expect(cut.minutes).toBe(1_000);
    expect(cut.share).toBe(0.25);
  });

  it("counts against the total, so facilities past the list's limit keep the share honest", () => {
    const cut = concentration([facility("a", 300), facility("b", 200)], 1_000);

    expect(cut.count).toBe(2);
    expect(cut.share).toBe(0.5);
  });
});

describe("foldArrivals", () => {
  it("folds each hour into the band dispatch books in, Monday first", () => {
    const grid = foldArrivals([
      { weekday: 1, hour: 6, stopCount: 10, detainedCount: 8 },
      { weekday: 1, hour: 7, stopCount: 10, detainedCount: 6 },
      { weekday: 7, hour: 23, stopCount: 2, detainedCount: 1 },
      { weekday: 3, hour: 0, stopCount: 4, detainedCount: 0 },
    ]);

    expect(grid).toHaveLength(7);
    expect(grid[0]).toHaveLength(8);
    expect(grid[0]?.[1]).toEqual({ stopCount: 20, detainedCount: 14 });
    expect(grid[6]?.[7]).toEqual({ stopCount: 2, detainedCount: 1 });
    expect(grid[2]?.[0]).toEqual({ stopCount: 4, detainedCount: 0 });
  });
});

describe("arrivalPeak", () => {
  it("names the most detained band that had enough arrivals to say", () => {
    const grid = foldArrivals([
      { weekday: 1, hour: 6, stopCount: 20, detainedCount: 15 },
      { weekday: 2, hour: 13, stopCount: 20, detainedCount: 4 },
      // Every arrival detained, but too few to rank.
      { weekday: 6, hour: 2, stopCount: 3, detainedCount: 3 },
    ]);

    expect(arrivalPeak(grid)).toEqual({ weekday: 0, band: 1, share: 0.75 });
    expect(afternoonShare(grid)).toBe(0.2);
  });

  it("finds no peak when no band had enough arrivals", () => {
    const grid = foldArrivals([{ weekday: 1, hour: 6, stopCount: 4, detainedCount: 4 }]);

    expect(arrivalPeak(grid)).toBeNull();
  });
});

describe("recoverySteps", () => {
  it("leaves out the steps that lost no money, keeping the server's order", () => {
    const recovery = {
      accrued: "1000",
      billed: "700",
      invoiced: "600",
      collected: "500",
      steps: [
        { bucket: "Late", occurrenceCount: 2, amount: "100" },
        { bucket: "Notice", occurrenceCount: 0, amount: "0" },
        { bucket: "Waived", occurrenceCount: 3, amount: "200" },
      ],
    } as DetentionIntelligence["recovery"];

    expect(recoverySteps(recovery).map((step) => step.bucket)).toEqual(["Late", "Waived"]);
  });
});

describe("rankFacilities", () => {
  const rows = [
    {
      locationId: "a",
      pastFreeMinutes: 600,
      dockMinutes: 100,
      detainedCount: 5,
      stopCount: 50,
      p90DwellMinutes: 200,
      billedAmount: "900",
      netMargin: "300",
    },
    {
      locationId: "b",
      pastFreeMinutes: 300,
      dockMinutes: 280,
      detainedCount: 9,
      stopCount: 10,
      p90DwellMinutes: 480,
      billedAmount: "400",
      netMargin: "-250",
    },
    {
      locationId: "c",
      pastFreeMinutes: 100,
      dockMinutes: 50,
      detainedCount: 3,
      stopCount: 4,
      p90DwellMinutes: 300,
      billedAmount: "1200",
      netMargin: "40",
    },
  ] as DetentionFacility[];
  const order = (rank: Parameters<typeof rankFacilities>[1]) =>
    rankFacilities(rows, rank).map((row) => row.locationId);

  it("puts the most first for every figure but net margin", () => {
    expect(order("hours")).toEqual(["a", "b", "c"]);
    expect(order("dock")).toEqual(["b", "a", "c"]);
    expect(order("detained")).toEqual(["b", "c", "a"]);
    expect(order("p90")).toEqual(["b", "c", "a"]);
    expect(order("billed")).toEqual(["c", "a", "b"]);
  });

  it("puts the biggest loss first when ranking by net margin", () => {
    expect(order("net")).toEqual(["b", "c", "a"]);
  });

  it("leaves the list it was given as it was", () => {
    rankFacilities(rows, "billed");
    expect(rows.map((row) => row.locationId)).toEqual(["a", "b", "c"]);
  });
});

describe("keptAfterDriverPay", () => {
  const page = (billed: string, driverPayAmount: string | null) =>
    ({
      // Net margin on every billable charge is larger than what was kept: it counts money that
      // was forgiven, disputed or never billed.
      totals: { driverPayAmount, netMargin: "2431" },
      recovery: { billed },
    }) as unknown as DetentionIntelligence;

  it("is what was billed less what drivers were paid, not the net margin on billable charges", () => {
    expect(keptAfterDriverPay(page("2141", "2053"))).toBe(88);
  });

  it("goes below zero when drivers were paid more than was billed", () => {
    expect(keptAfterDriverPay(page("500", "800"))).toBe(-300);
  });

  it("is withheld from a reader who cannot see driver pay", () => {
    expect(keptAfterDriverPay(page("2141", null))).toBeNull();
  });
});

describe("weekDaysInWindow", () => {
  // 1_900_000_000 is a Sunday afternoon in Chicago; 90 days on is a Saturday afternoon.
  const window = {
    from: 1_900_000_000,
    to: 1_900_000_000 + 90 * 86_400,
    timezone: "America/Chicago",
  };
  const firstMonday = 1_899_417_600;
  const week = 7 * 86_400;

  it("counts only the Sunday of a first week the window opens on its last day", () => {
    expect(weekDaysInWindow(firstMonday, window)).toBe(1);
  });

  it("counts every day of a week wholly inside the window", () => {
    expect(weekDaysInWindow(firstMonday + 5 * week, window)).toBe(7);
  });

  it("counts Monday to Saturday of a last week the window ends inside", () => {
    expect(weekDaysInWindow(firstMonday + 13 * week, window)).toBe(6);
  });

  it("counts nothing for a week outside the window", () => {
    expect(weekDaysInWindow(firstMonday - week, window)).toBe(0);
  });
});
