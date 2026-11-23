import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import {
  availabilitySummary,
  daysOut,
  nextApprovedTimeOff,
  shiftClashes,
  weekBars,
} from "../schedule-model";

const t = ((text: string, ...values: unknown[]) =>
  text.replace(/\{(\d+)\}/g, (_, index: string) => String(values[Number(index)]))) as TranslateFn;
const SHORT = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
const shortDay = (day: number) => SHORT[day];

describe("weekBars", () => {
  it("lays a day shift on its working days, Monday first", () => {
    const week = weekBars("0111110", 360, 630);
    expect(week.map((day) => day.day)).toEqual([1, 2, 3, 4, 5, 6, 0]);
    expect(week[0].bars).toEqual([{ startPercent: 25, widthPercent: 43.75 }]);
    expect(week[0].minutes).toBe(630);
    expect(week[5].bars).toEqual([]);
    expect(week[6].minutes).toBe(0);
  });

  it("carries a shift past midnight onto the next day", () => {
    const week = weekBars("1000000", 1200, 630);
    const sunday = week.find((day) => day.day === 0);
    const monday = week.find((day) => day.day === 1);
    expect(sunday?.bars).toEqual([
      { startPercent: (1200 / 1440) * 100, widthPercent: (240 / 1440) * 100 },
    ]);
    expect(monday?.bars).toEqual([{ startPercent: 0, widthPercent: (390 / 1440) * 100 }]);
    expect(monday?.minutes).toBe(0);
  });
});

describe("shiftClashes", () => {
  it("names the working days the worker said they can't work", () => {
    expect(
      shiftClashes("1000011", [
        { dayOfWeek: 0, preference: "Unavailable" },
        { dayOfWeek: 6, preference: "Unavailable" },
        { dayOfWeek: 5, preference: "Preferred" },
        { dayOfWeek: 3, preference: "Unavailable" },
      ]),
    ).toEqual([6, 0]);
  });
});

describe("availabilitySummary", () => {
  it("collapses runs and says weekends in words", () => {
    expect(
      availabilitySummary(
        t,
        [
          { dayOfWeek: 2, preference: "Preferred" },
          { dayOfWeek: 3, preference: "Preferred" },
          { dayOfWeek: 4, preference: "Preferred" },
          { dayOfWeek: 6, preference: "Unavailable" },
          { dayOfWeek: 0, preference: "Unavailable" },
          { dayOfWeek: 1, preference: "Available" },
        ],
        shortDay,
      ),
    ).toBe("Prefers Tue to Thu · not weekends");
  });

  it("says any day when nothing is ruled out, and nothing said when nothing is stated", () => {
    expect(availabilitySummary(t, [{ dayOfWeek: 1, preference: "Preferred" }], shortDay)).toBe(
      "Prefers Mon · any day",
    );
    expect(availabilitySummary(t, [], shortDay)).toBe("Nothing said yet");
    expect(
      availabilitySummary(
        t,
        [
          { dayOfWeek: 1, preference: "Unavailable" },
          { dayOfWeek: 2, preference: "Unavailable" },
        ],
        shortDay,
      ),
    ).toBe("not Mon, Tue");
  });
});

describe("home time", () => {
  it("counts whole days out since the worker was last home", () => {
    expect(daysOut(1_800_000_000, 1_800_000_000 + 8 * 86_400 + 3_600)).toBe(8);
    expect(daysOut(null, 1_800_000_000)).toBeNull();
  });

  it("finds the earliest approved time off that has not ended", () => {
    const now = 1_800_000_000;
    const later = { status: "Approved", startDate: now + 20 * 86_400, endDate: now + 22 * 86_400 };
    const sooner = { status: "Approved", startDate: now + 5 * 86_400, endDate: now + 6 * 86_400 };
    expect(
      nextApprovedTimeOff(
        [
          later,
          { status: "Requested", startDate: now + 86_400, endDate: now + 2 * 86_400 },
          { status: "Approved", startDate: now - 10 * 86_400, endDate: now - 9 * 86_400 },
          sooner,
        ],
        now,
      ),
    ).toBe(sooner);
    expect(nextApprovedTimeOff([], now)).toBeNull();
  });
});
