import { describe, expect, it } from "vitest";
import { holidayNamesByDate, usFederalHolidays } from "@trenova/shared/lib/holidays";

function dateFor(year: number, name: string): string | undefined {
  return usFederalHolidays(year).find((holiday) => holiday.name === name)?.date;
}

describe("usFederalHolidays", () => {
  it("returns the eleven federal holidays in date order", () => {
    const holidays = usFederalHolidays(2026);

    expect(holidays).toHaveLength(11);
    expect(holidays.map((holiday) => holiday.date)).toEqual(
      [...holidays.map((holiday) => holiday.date)].sort(),
    );
  });

  it("resolves fixed-date holidays", () => {
    expect(dateFor(2026, "New Year's Day")).toBe("2026-01-01");
    expect(dateFor(2026, "Juneteenth")).toBe("2026-06-19");
    expect(dateFor(2026, "Independence Day")).toBe("2026-07-04");
    expect(dateFor(2026, "Veterans Day")).toBe("2026-11-11");
    expect(dateFor(2026, "Christmas Day")).toBe("2026-12-25");
  });

  it("resolves nth-weekday holidays", () => {
    expect(dateFor(2026, "Martin Luther King Jr. Day")).toBe("2026-01-19");
    expect(dateFor(2026, "Presidents' Day")).toBe("2026-02-16");
    expect(dateFor(2026, "Labor Day")).toBe("2026-09-07");
    expect(dateFor(2026, "Columbus Day")).toBe("2026-10-12");
    expect(dateFor(2026, "Thanksgiving Day")).toBe("2026-11-26");
  });

  it("resolves last-weekday holidays, including months that end on the target weekday", () => {
    expect(dateFor(2025, "Memorial Day")).toBe("2025-05-26");
    expect(dateFor(2026, "Memorial Day")).toBe("2026-05-25");
    // May 2027 ends on a Monday, so the last-Monday walk must not skip back a week.
    expect(dateFor(2027, "Memorial Day")).toBe("2027-05-31");
  });

  it("handles a leap year without drifting", () => {
    expect(dateFor(2028, "Presidents' Day")).toBe("2028-02-21");
    expect(dateFor(2028, "Memorial Day")).toBe("2028-05-29");
  });
});

describe("holidayNamesByDate", () => {
  it("labels dates across every requested year", () => {
    const names = holidayNamesByDate([2026, 2027]);

    expect(names.get("2026-11-26")).toBe("Thanksgiving Day");
    expect(names.get("2027-11-25")).toBe("Thanksgiving Day");
    expect(names.get("2026-11-27")).toBeUndefined();
  });

  it("returns an empty map for no years", () => {
    expect(holidayNamesByDate([]).size).toBe(0);
  });
});
