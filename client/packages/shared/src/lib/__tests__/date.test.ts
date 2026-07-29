import { afterEach, describe, expect, it } from "vitest";
import {
  daysUntil,
  formatUnixDate,
  formatUnixDateMedium,
  formatUnixDateTime,
  formatUnixDateTimeMedium,
  formatUnixDateTimeOrDash,
  formatUnixDateTimeShort,
  formatUnixTime,
  formatUnixTimeWithSeconds,
  fromUserWallClock,
  getTodayDate,
  getUserDatePreferences,
  resolveUserTimeFormat,
  resolveUserTimezone,
  setUserDatePreferences,
  toUserWallClock,
  dateToUnixTimestamp,
  toDate,
  toUnixTimeStamp,
  formatSplitDateTime,
  generateDateOnlyString,
  generateDateOnly,
  isValidDateOnlyFormat,
  generateDateTimeString,
  generateDateTimeStringFromUnixTimestamp,
  formatToUserTimezone,
  formatCurrentUserTime,
  formatDurationFromSeconds,
  formatSecondsAgo,
  inclusiveDays,
  formatRange,
  getStartOfDay,
  getEndOfDay,
  getStartOfMonth,
  getEndOfMonth,
  getCommonDatePresets,
} from "@trenova/shared/lib/date";

const JAN_15_2024_UTC = 1705276800;

afterEach(() => {
  setUserDatePreferences({});
});

describe("user date preferences", () => {
  it("falls back to the browser zone and a 12-hour clock", () => {
    expect(getUserDatePreferences()).toEqual({ timezone: "auto", timeFormat: "12-hour" });
    expect(resolveUserTimezone()).toBe(Intl.DateTimeFormat().resolvedOptions().timeZone);
    expect(resolveUserTimeFormat()).toBe("12-hour");
  });

  it("resolves the stored preference, and lets an explicit argument win", () => {
    setUserDatePreferences({ timezone: "America/Denver", timeFormat: "24-hour" });
    expect(resolveUserTimezone()).toBe("America/Denver");
    expect(resolveUserTimezone("America/New_York")).toBe("America/New_York");
    expect(resolveUserTimeFormat()).toBe("24-hour");
    expect(resolveUserTimeFormat("12-hour")).toBe("12-hour");
  });

  it("treats a partial update as a reset to the fallback", () => {
    setUserDatePreferences({ timezone: "America/Denver" });
    expect(getUserDatePreferences()).toEqual({
      timezone: "America/Denver",
      timeFormat: "12-hour",
    });
  });

  it("renders stored timestamps in the preferred zone without a call-site argument", () => {
    setUserDatePreferences({ timezone: "America/Denver" });
    expect(formatUnixDateTime(JAN_15_2024_UTC)).toBe("1/14/2024, 5:00 PM");
    expect(formatToUserTimezone(JAN_15_2024_UTC)).toContain("MST");
    expect(formatToUserTimezone(JAN_15_2024_UTC)).toContain("01/14/2024");
  });
});

describe("unix display helpers", () => {
  it("formats each shape in the preferred zone", () => {
    setUserDatePreferences({ timezone: "America/Denver" });
    expect(formatUnixDate(JAN_15_2024_UTC)).toBe("1/14/2024");
    expect(formatUnixDateMedium(JAN_15_2024_UTC)).toBe("Jan 14, 2024");
    expect(formatUnixDateTimeMedium(JAN_15_2024_UTC)).toBe("Jan 14, 2024, 5:00 PM");
    expect(formatUnixDateTimeShort(JAN_15_2024_UTC)).toBe("Jan 14, 5:00 PM");
    expect(formatUnixTime(JAN_15_2024_UTC)).toBe("5:00 PM");
  });

  it("honours the clock preference", () => {
    setUserDatePreferences({ timezone: "America/Denver", timeFormat: "24-hour" });
    expect(formatUnixTime(JAN_15_2024_UTC)).toBe("17:00");
    expect(formatUnixTimeWithSeconds(JAN_15_2024_UTC)).toBe("17:00:00");
  });

  it("treats unset timestamps as unset", () => {
    expect(formatUnixDate(null)).toBe("N/A");
    expect(formatUnixDate(0)).toBe("N/A");
    expect(formatUnixDateTimeOrDash(undefined)).toBe("-");
    expect(formatUnixDateTimeOrDash(JAN_15_2024_UTC)).not.toBe("-");
    expect(formatUnixDateMedium(null, { fallback: "—" })).toBe("—");
    expect(formatUnixDateTimeOrDash(null, { fallback: "Not recorded" })).toBe("Not recorded");
  });
});

describe("toUserWallClock / fromUserWallClock", () => {
  it("round-trips an instant through the user's wall clock", () => {
    setUserDatePreferences({ timezone: "America/Denver" });
    const wallClock = toUserWallClock(JAN_15_2024_UTC)!;
    expect(wallClock.getFullYear()).toBe(2024);
    expect(wallClock.getMonth()).toBe(0);
    expect(wallClock.getDate()).toBe(14);
    expect(wallClock.getHours()).toBe(17);
    expect(fromUserWallClock(wallClock)).toBe(JAN_15_2024_UTC);
  });

  it("returns undefined for unset values", () => {
    expect(toUserWallClock(undefined)).toBeUndefined();
    expect(fromUserWallClock(undefined)).toBeUndefined();
    expect(fromUserWallClock(new Date("invalid"))).toBeUndefined();
  });
});

describe("daysUntil", () => {
  it("counts calendar days in the user's zone", () => {
    setUserDatePreferences({ timezone: "America/Denver" });
    const now = Math.floor(Date.now() / 1000);
    expect(daysUntil(now)).toBe(0);
    expect(daysUntil(now + 86400)).toBe(1);
    expect(daysUntil(now - 86400)).toBe(-1);
  });
});

describe("getTodayDate", () => {
  it("anchors to midnight in the user's zone", () => {
    setUserDatePreferences({ timezone: "America/Denver" });
    expect(getTodayDate()).toBe(getStartOfDay(new Date(), "America/Denver"));
    expect(getTodayDate("UTC")).toBe(getStartOfDay(new Date(), "UTC"));
  });
});

describe("dateToUnixTimestamp", () => {
  it("converts known date to known timestamp", () => {
    const date = new Date("2024-01-15T00:00:00Z");
    expect(dateToUnixTimestamp(date)).toBe(1705276800);
  });

  it("throws on invalid date", () => {
    expect(() => dateToUnixTimestamp(new Date("invalid"))).toThrow(
      "Invalid date provided to dateToUnixTimestamp",
    );
  });

  it("floors milliseconds", () => {
    const date = new Date(1705276800999);
    expect(dateToUnixTimestamp(date)).toBe(1705276800);
  });
});

describe("toDate", () => {
  it("converts valid timestamp to Date", () => {
    const result = toDate(1705276800);
    expect(result).toBeInstanceOf(Date);
    expect(result!.toISOString()).toBe("2024-01-15T00:00:00.000Z");
  });

  it("returns undefined for undefined input", () => {
    expect(toDate(undefined)).toBeUndefined();
  });

  it("returns undefined for NaN", () => {
    expect(toDate(NaN)).toBeUndefined();
  });

  it("converts 0 (Unix epoch) to Date", () => {
    const result = toDate(0);
    expect(result).toBeInstanceOf(Date);
    expect(result!.toISOString()).toBe("1970-01-01T00:00:00.000Z");
  });
});

describe("toUnixTimeStamp", () => {
  it("converts valid date", () => {
    const date = new Date("2024-01-15T00:00:00Z");
    expect(toUnixTimeStamp(date)).toBe(1705276800);
  });

  it("returns undefined for undefined input", () => {
    expect(toUnixTimeStamp(undefined)).toBeUndefined();
  });

  it("round-trips with toDate", () => {
    const timestamp = 1705276800;
    const date = toDate(timestamp);
    expect(toUnixTimeStamp(date!)).toBe(timestamp);
  });
});

describe("formatSplitDateTime", () => {
  it("returns defaults for undefined timestamp", () => {
    expect(formatSplitDateTime(undefined)).toEqual({ date: "-", time: "" });
  });

  it("formats with 12-hour time", () => {
    const result = formatSplitDateTime(1705276800, "12-hour", "UTC");
    expect(result.time).toMatch(/AM|PM/);
    expect(result.date).toBeTruthy();
  });

  it("formats with 24-hour time", () => {
    const result = formatSplitDateTime(1705276800, "24-hour", "UTC");
    expect(result.time).toMatch(/\d{2}:\d{2}/);
    expect(result.time).not.toMatch(/AM|PM/);
  });

  it("returns date part with year", () => {
    const result = formatSplitDateTime(1705276800, "24-hour", "UTC");
    expect(result.date).toContain("2024");
  });

  it("formats timestamp 0 (Unix epoch) instead of returning default", () => {
    const result = formatSplitDateTime(0);
    expect(result.date).not.toBe("-");
    expect(result.time).not.toBe("");
  });

  it("respects timezone parameter", () => {
    const result = formatSplitDateTime(1705276800, "12-hour", "America/New_York");
    expect(result.date).toContain("2024");
    expect(result.time).toMatch(/PM/);
  });
});

describe("generateDateOnlyString", () => {
  it("formats known date in MM/dd/yyyy format", () => {
    const date = new Date(2024, 0, 15);
    const result = generateDateOnlyString(date);
    expect(result).toBe("01/15/2024");
  });

  it("throws on invalid date", () => {
    expect(() => generateDateOnlyString(new Date("invalid"))).toThrow(
      "Invalid date provided to generateDateOnlyString",
    );
  });

  it("returns MM/dd/yyyy format", () => {
    const date = new Date(2024, 0, 5);
    const result = generateDateOnlyString(date);
    expect(result).toMatch(/^\d{2}\/\d{2}\/\d{4}$/);
  });
});

describe("generateDateOnly", () => {
  it("parses valid date string to midnight Date", () => {
    const result = generateDateOnly("January 15, 2024");
    expect(result).toBeInstanceOf(Date);
    expect(result!.getHours()).toBe(0);
    expect(result!.getMinutes()).toBe(0);
    expect(result!.getSeconds()).toBe(0);
  });

  it("returns null for empty string", () => {
    expect(generateDateOnly("")).toBeNull();
  });

  it("returns null for null input", () => {
    expect(generateDateOnly(null as any)).toBeNull();
  });

  it("returns null for unparseable string", () => {
    expect(generateDateOnly("not a date at all xyz")).toBeNull();
  });
});

describe("isValidDateOnlyFormat", () => {
  it("returns true for valid format", () => {
    expect(isValidDateOnlyFormat("01/15/2024")).toBe(true);
  });

  it("returns false for wrong format", () => {
    expect(isValidDateOnlyFormat("2024-01-15")).toBe(false);
  });

  it("returns false for empty string", () => {
    expect(isValidDateOnlyFormat("")).toBe(false);
  });
});

describe("generateDateTimeString", () => {
  const date = new Date("2024-01-15T14:30:45Z");

  it("defaults to the user's clock preference", () => {
    setUserDatePreferences({ timeFormat: "24-hour" });
    const result = generateDateTimeString(date);
    expect(result).toMatch(/\d{2}\/\d{2}\/\d{4}/);
    expect(result).not.toMatch(/AM|PM/);
    setUserDatePreferences({});
    expect(generateDateTimeString(date)).toMatch(/AM|PM/);
  });

  it("uses 12-hour format when specified", () => {
    const result = generateDateTimeString(date, { timeFormat: "12-hour" });
    expect(result).toMatch(/AM|PM/);
  });

  it("includes seconds when specified", () => {
    const result = generateDateTimeString(date, { showSeconds: true });
    expect(result).toMatch(/:\d{2}:\d{2}/);
  });

  it("throws on invalid date", () => {
    expect(() => generateDateTimeString(new Date("invalid"))).toThrow(
      "Invalid date provided to generateDateTimeString",
    );
  });
});

describe("generateDateTimeStringFromUnixTimestamp", () => {
  it("returns N/A for undefined", () => {
    expect(generateDateTimeStringFromUnixTimestamp(undefined)).toBe("N/A");
  });

  it("returns N/A for NaN", () => {
    expect(generateDateTimeStringFromUnixTimestamp(NaN)).toBe("N/A");
  });

  it("formats valid timestamp", () => {
    const result = generateDateTimeStringFromUnixTimestamp(1705276800);
    expect(result).toContain("2024");
  });

  it("formats 0 (Unix epoch) instead of returning N/A", () => {
    const result = generateDateTimeStringFromUnixTimestamp(0);
    expect(result).not.toBe("N/A");
    expect(result).not.toBe("-");
    expect(result.length).toBeGreaterThan(0);
  });
});

describe("formatToUserTimezone", () => {
  it("returns N/A for NaN", () => {
    expect(formatToUserTimezone(NaN)).toBe("N/A");
  });

  it("formats 0 (Unix epoch) instead of returning N/A", () => {
    const result = formatToUserTimezone(0, {}, "UTC");
    expect(result).not.toBe("N/A");
    expect(result).toBeTruthy();
  });

  it("formats with explicit timezone", () => {
    const result = formatToUserTimezone(1705276800, {}, "America/New_York");
    expect(result).toBeTruthy();
    expect(result).not.toBe("N/A");
  });

  it("respects showDate=false", () => {
    const result = formatToUserTimezone(
      1705276800,
      { showDate: false, showTime: true, showTimeZone: false },
      "UTC",
    );
    expect(result).not.toMatch(/2024/);
  });

  it("respects showTime=false", () => {
    const result = formatToUserTimezone(1705276800, { showDate: true, showTime: false }, "UTC");
    expect(result).toMatch(/2024/);
  });

  it("respects 12-hour time format", () => {
    const result = formatToUserTimezone(
      1705276800,
      { timeFormat: "12-hour", showDate: false, showTime: true, showTimeZone: false },
      "UTC",
    );
    expect(result).toMatch(/AM|PM/);
  });

  it("respects 24-hour time format", () => {
    const result = formatToUserTimezone(
      1705276800,
      { timeFormat: "24-hour", showDate: false, showTime: true, showTimeZone: false },
      "UTC",
    );
    expect(result).not.toMatch(/AM|PM/);
  });
});

describe("formatCurrentUserTime", () => {
  const date = new Date("2026-05-04T19:26:00Z");

  it("formats 24-hour time with seconds and timezone", () => {
    expect(formatCurrentUserTime(date, "24-hour", "America/Chicago")).toBe("14:26:00 CT");
  });

  it("formats 12-hour time with seconds and timezone", () => {
    expect(formatCurrentUserTime(date, "12-hour", "America/Chicago")).toBe("02:26:00 PM CT");
  });

  it("returns N/A for invalid dates", () => {
    expect(formatCurrentUserTime(new Date("invalid"))).toBe("N/A");
  });
});

describe("formatDurationFromSeconds", () => {
  it("returns 0m for 0 seconds", () => {
    expect(formatDurationFromSeconds(0)).toBe("0m");
  });

  it("returns 0m for negative", () => {
    expect(formatDurationFromSeconds(-5)).toBe("0m");
  });

  it("returns 1m for 60 seconds", () => {
    expect(formatDurationFromSeconds(60)).toBe("1m");
  });

  it("returns 1h for 3600 seconds", () => {
    expect(formatDurationFromSeconds(3600)).toBe("1h");
  });

  it("returns 1d for 86400 seconds", () => {
    expect(formatDurationFromSeconds(86400)).toBe("1d");
  });

  it("returns combined format for 90061", () => {
    expect(formatDurationFromSeconds(90061)).toBe("1d 1h 1m");
  });

  it("returns < 1m for sub-minute durations", () => {
    expect(formatDurationFromSeconds(30)).toBe("< 1m");
  });

  it("returns < 1m for 1 second", () => {
    expect(formatDurationFromSeconds(1)).toBe("< 1m");
  });
});

describe("inclusiveDays", () => {
  it("returns 1 for same day", () => {
    const ts = 1705276800;
    expect(inclusiveDays(ts, ts)).toBe(1);
  });

  it("returns 2 for adjacent days", () => {
    const start = 1705276800;
    const end = start + 86400;
    expect(inclusiveDays(start, end)).toBe(2);
  });

  it("returns 7 for week span", () => {
    const start = 1705276800;
    const end = start + 86400 * 6;
    expect(inclusiveDays(start, end)).toBe(7);
  });
});

describe("formatRange", () => {
  it("returns single date without dash for same day", () => {
    const ts = 1705276800;
    const result = formatRange(ts, ts);
    expect(result).not.toContain("–");
  });

  it("returns range with dash for different dates", () => {
    const start = 1705276800;
    const end = start + 86400 * 5;
    const result = formatRange(start, end);
    expect(result).toContain("–");
  });

  it("returns a non-empty string", () => {
    const result = formatRange(1705276800, 1705276800 + 86400);
    expect(result.length).toBeGreaterThan(0);
  });
});

describe("getStartOfDay / getEndOfDay", () => {
  it("end is greater than start", () => {
    const date = new Date("2024-01-15T12:00:00Z");
    const start = getStartOfDay(date, "UTC");
    const end = getEndOfDay(date, "UTC");
    expect(end).toBeGreaterThan(start);
  });

  it("works without timezone", () => {
    const date = new Date("2024-01-15T12:00:00Z");
    const start = getStartOfDay(date);
    const end = getEndOfDay(date);
    expect(end).toBeGreaterThan(start);
  });

  it("returns unix timestamps", () => {
    const date = new Date("2024-01-15T12:00:00Z");
    const start = getStartOfDay(date, "UTC");
    expect(start).toBe(1705276800);
  });
});

describe("getStartOfMonth / getEndOfMonth", () => {
  it("Jan 2024 start is Jan 1", () => {
    const date = new Date("2024-01-15T12:00:00Z");
    const start = getStartOfMonth(date, "UTC");
    const startDate = new Date(start * 1000);
    expect(startDate.getUTCDate()).toBe(1);
    expect(startDate.getUTCMonth()).toBe(0);
  });

  it("end is greater than start", () => {
    const date = new Date("2024-01-15T12:00:00Z");
    const start = getStartOfMonth(date, "UTC");
    const end = getEndOfMonth(date, "UTC");
    expect(end).toBeGreaterThan(start);
  });

  it("works without timezone", () => {
    const date = new Date("2024-01-15T12:00:00Z");
    const start = getStartOfMonth(date);
    const end = getEndOfMonth(date);
    expect(end).toBeGreaterThan(start);
  });
});

describe("getCommonDatePresets", () => {
  it("returns 6 presets", () => {
    const presets = getCommonDatePresets("UTC");
    expect(presets).toHaveLength(6);
  });

  it("each preset returns startDate <= endDate", () => {
    const presets = getCommonDatePresets("UTC");
    for (const preset of presets) {
      const { startDate, endDate } = preset.getValue();
      expect(endDate).toBeGreaterThanOrEqual(startDate);
    }
  });
});

describe("formatSecondsAgo", () => {
  it("reads anything under five seconds as just now", () => {
    expect(formatSecondsAgo(0)).toBe("just now");
    expect(formatSecondsAgo(4)).toBe("just now");
  });

  it("reports seconds, then minutes, then hours", () => {
    expect(formatSecondsAgo(31)).toBe("31s ago");
    expect(formatSecondsAgo(90)).toBe("1m ago");
    expect(formatSecondsAgo(7200)).toBe("2h ago");
  });

  it("falls back to just now for a non-finite input", () => {
    expect(formatSecondsAgo(Number.NaN)).toBe("just now");
  });
});
