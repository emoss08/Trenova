import { describe, expect, it } from "vitest";
import {
  formatUtcDate,
  groupOccurrencesByMonth,
  projectHolidaysOntoYear,
  toUtcDateOnly,
  utcMidnight,
} from "./holiday";

const entry = (
  id: string,
  name: string,
  year: number,
  month: number,
  day: number,
  overrides: Partial<{ kind: "Holiday" | "Blackout"; recursAnnually: boolean }> = {},
) => ({
  id,
  name,
  holidayDate: utcMidnight(year, month, day),
  kind: overrides.kind ?? ("Holiday" as const),
  recursAnnually: overrides.recursAnnually ?? true,
});

describe("projectHolidaysOntoYear", () => {
  it("moves recurring entries into the requested year and drops one-offs from other years", () => {
    const entries = [
      entry("h1", "Christmas Day", 2024, 11, 25),
      entry("h2", "Company picnic", 2025, 6, 18, { recursAnnually: false }),
      entry("h3", "Inventory freeze", 2026, 0, 10, { kind: "Blackout", recursAnnually: false }),
    ];

    const occurrences = projectHolidaysOntoYear(entries, 2026);

    expect(occurrences.map((occurrence) => occurrence.entry.id)).toEqual(["h3", "h1"]);
    expect(occurrences[1].date).toBe(utcMidnight(2026, 11, 25));
    expect(occurrences[1].month).toBe(11);
    expect(occurrences[1].day).toBe(25);
  });

  it("lands a recurring 29 February on the 28th in a common year", () => {
    const [leapDay] = projectHolidaysOntoYear([entry("h1", "Leap day", 2024, 1, 29)], 2026);
    expect(leapDay.month).toBe(1);
    expect(leapDay.day).toBe(28);
    const [kept] = projectHolidaysOntoYear([entry("h1", "Leap day", 2024, 1, 29)], 2028);
    expect(kept.day).toBe(29);
  });

  it("orders a shared date holiday-first, then by name", () => {
    const entries = [
      entry("b", "Zulu freeze", 2026, 6, 4, { kind: "Blackout", recursAnnually: false }),
      entry("a", "Independence Day", 2026, 6, 4),
      entry("c", "Alpha freeze", 2026, 6, 4, { kind: "Blackout", recursAnnually: false }),
    ];
    expect(projectHolidaysOntoYear(entries, 2026).map((occurrence) => occurrence.entry.id)).toEqual(
      ["a", "c", "b"],
    );
  });
});

describe("groupOccurrencesByMonth", () => {
  it("returns twelve buckets so empty months still render", () => {
    const grouped = groupOccurrencesByMonth(
      projectHolidaysOntoYear([entry("h1", "Labor Day", 2026, 8, 7)], 2026),
    );
    expect(grouped).toHaveLength(12);
    expect(grouped[8]).toHaveLength(1);
    expect(grouped[0]).toHaveLength(0);
  });
});

describe("date helpers", () => {
  it("formats a UTC-midnight date without drifting a day in western timezones", () => {
    expect(formatUtcDate(utcMidnight(2026, 6, 4))).toBe("Jul 4, 2026");
    expect(formatUtcDate(utcMidnight(2026, 6, 4), { weekday: "short" })).toBe("Sat, Jul 4, 2026");
  });

  it("collapses a user wall-clock instant to the UTC midnight of that calendar day", () => {
    const denverNoon = Date.UTC(2026, 6, 4, 18, 0, 0) / 1000;
    expect(toUtcDateOnly(denverNoon, "America/Denver")).toBe(utcMidnight(2026, 6, 4));
    const tokyoJustAfterMidnight = Date.UTC(2026, 6, 3, 15, 30, 0) / 1000;
    expect(toUtcDateOnly(tokyoJustAfterMidnight, "Asia/Tokyo")).toBe(utcMidnight(2026, 6, 4));
  });
});
