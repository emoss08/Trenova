// Numbers and dates are as much a part of a translation as the words. These assert the
// formatters actually follow the active locale, which is what the hardcoded "en-US" in
// lib/utils.ts and lib/date.ts used to prevent.
import { setLocale } from "@trenova/shared/i18n/runtime";
import {
  formatDuration,
  formatList,
  formatNumber,
  formatRelativeTime,
} from "@trenova/shared/i18n/format";
import { formatCurrency, formatPercent } from "@trenova/shared/lib/utils";
import { afterEach, describe, expect, it } from "vitest";

afterEach(async () => {
  await setLocale("en");
});

describe("number formatting follows the active locale", () => {
  it("groups digits the English way by default", () => {
    expect(formatNumber(1234567.5)).toBe("1,234,567.5");
  });

  it("uses Spanish grouping and decimal separators", async () => {
    await setLocale("es");
    // es-419 groups with commas and marks decimals with a period, unlike es-ES.
    expect(formatNumber(1234567.5)).toBe("1,234,567.5");
  });

  it("formats currency per locale", async () => {
    expect(formatCurrency(1234.56)).toBe("$1,234.56");
    await setLocale("zh-CN");
    expect(formatCurrency(1234.56)).toContain("1,234.56");
  });

  it("keeps the existing percent contract while localizing the format", () => {
    expect(formatPercent(12.5)).toBe("12.5%");
    expect(formatPercent(-3.25, 2)).toBe("-3.25%");
  });
});

describe("list and relative-time formatting", () => {
  it("joins a list in English", () => {
    expect(formatList(["a", "b", "c"])).toBe("a, b, and c");
  });

  it("joins a list in Spanish", async () => {
    await setLocale("es");
    expect(formatList(["a", "b", "c"])).toBe("a, b y c");
  });

  it("renders relative time in English", () => {
    expect(formatRelativeTime(-86_400)).toBe("yesterday");
    expect(formatRelativeTime(7_200)).toBe("in 2 hours");
  });

  it("renders relative time in Traditional Chinese", async () => {
    await setLocale("zh-TW");
    expect(formatRelativeTime(-86_400)).toBe("昨天");
  });
});

describe("duration formatting", () => {
  it("writes the two largest units a span covers", () => {
    expect(formatDuration(134)).toBe("2 min, 14 sec");
    expect(formatDuration(3_900)).toBe("1 hr, 5 min");
    expect(formatDuration(90_061)).toBe("1 day, 1 hr");
  });

  // "2 hr, 30 sec" reads as a precise figure when it is a rounded one; a span stops at
  // its first empty unit instead.
  it("stops at the first empty unit after the largest", () => {
    expect(formatDuration(7_230)).toBe("2 hr");
  });

  it("writes a span under a minute in seconds, and nothing as zero seconds", () => {
    expect(formatDuration(38)).toBe("38 sec");
    expect(formatDuration(0)).toBe("0 sec");
    expect(formatDuration(Number.NaN)).toBe("0 sec");
  });

  it("writes the units in the active language", async () => {
    await setLocale("es");
    expect(formatDuration(134)).toBe("2 min y 14 s");
  });
});
