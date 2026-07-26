import { describe, expect, it } from "vitest";
import {
  formatReportValue,
  isNumericDisplay,
  resolveReportDisplay,
  type ReportColumnDisplay,
} from "../report-format";

function display(overrides: Partial<ReportColumnDisplay> = {}): ReportColumnDisplay {
  return {
    style: "number",
    decimals: 2,
    grouping: true,
    currency: "USD",
    negative: "minus",
    notation: "standard",
    prefix: "",
    suffix: "",
    dateStyle: "dateTime",
    boolStyle: "yesNo",
    durationUnit: "seconds",
    durationStyle: "clock",
    nullText: "",
    ...overrides,
  };
}

describe("formatReportValue", () => {
  it("tames raw quotients into money", () => {
    const money = display({ style: "currency", negative: "parentheses" });
    expect(formatReportValue(1082.0672643987443, money)).toBe("$1,082.07");
    expect(formatReportValue(-1082.0672643987443, money)).toBe("($1,082.07)");
  });

  it("accepts decimals serialized as strings", () => {
    expect(formatReportValue("1082.0672643987443", display())).toBe("1,082.07");
  });

  it("renders percents by scaling to points", () => {
    expect(formatReportValue(0.1834, display({ style: "percent", decimals: 1 }))).toBe("18.3%");
  });

  it("abbreviates with compact notation", () => {
    const compact = display({ notation: "compact", decimals: 1 });
    expect(formatReportValue(1284000, compact)).toBe("1.3M");
    expect(formatReportValue(940, compact)).toBe("940.0");
  });

  it("renders durations in every style", () => {
    const base = { style: "duration" as const, decimals: 1 };
    expect(formatReportValue(94200, display({ ...base, durationStyle: "clock" }))).toBe("26:10");
    expect(formatReportValue(94200, display({ ...base, durationStyle: "compact" }))).toBe(
      "1d 2h 10m",
    );
    expect(formatReportValue(94200, display({ ...base, durationStyle: "verbose" }))).toBe(
      "1 day 2 hours 10 minutes",
    );
    expect(formatReportValue(94200, display({ ...base, durationStyle: "decimalHours" }))).toBe(
      "26.2 h",
    );
    expect(formatReportValue(-9000, display({ ...base, durationStyle: "compact" }))).toBe(
      "-2h 30m",
    );
  });

  it("formats dates in the report timezone", () => {
    const dates = display({ style: "date" });
    expect(formatReportValue(1772587800, dates, "UTC")).toBe("2026-03-04 01:30");
    expect(formatReportValue(1772587800, { ...dates, dateStyle: "date" }, "UTC")).toBe(
      "2026-03-04",
    );
    expect(formatReportValue(1772587800, { ...dates, dateStyle: "monthYear" }, "UTC")).toBe(
      "Mar 2026",
    );
    expect(formatReportValue(1772587800, { ...dates, dateStyle: "quarter" }, "UTC")).toBe(
      "Q1 2026",
    );
    expect(formatReportValue(1772587800, dates, "America/Chicago")).toBe("2026-03-03 19:30");
  });

  it("renders booleans and empty values", () => {
    expect(formatReportValue(true, display({ style: "bool" }))).toBe("Yes");
    expect(formatReportValue(false, display({ style: "bool", boolStyle: "check" }))).toBe("✗");
    expect(formatReportValue(null, display({ nullText: "—" }))).toBe("—");
    expect(formatReportValue("", display({ style: "text", nullText: "—" }))).toBe("—");
  });

  it("renders raw when no display contract is present", () => {
    expect(formatReportValue(1234.5, undefined)).toBe("1234.5");
    expect(formatReportValue(1234.5, display({ style: "" }))).toBe("1234.5");
  });
});

describe("resolveReportDisplay", () => {
  it("derives defaults from type and format hint", () => {
    expect(resolveReportDisplay({ type: "decimal", hint: "money" })).toMatchObject({
      style: "currency",
      decimals: 2,
      negative: "parentheses",
    });
    expect(resolveReportDisplay({ type: "int", hint: "count" })).toMatchObject({
      style: "number",
      decimals: 0,
    });
    expect(resolveReportDisplay({ type: "decimal", hint: "distance" })).toMatchObject({
      suffix: " mi",
      decimals: 1,
    });
    expect(resolveReportDisplay({ type: "epoch" }).style).toBe("date");
    expect(resolveReportDisplay({ type: "bool" }).style).toBe("bool");
    expect(resolveReportDisplay({ type: "string" }).style).toBe("text");
  });

  it("applies the bucket date style and author overrides", () => {
    expect(resolveReportDisplay({ type: "epoch", dateStyle: "monthYear" }).dateStyle).toBe(
      "monthYear",
    );
    expect(
      resolveReportDisplay({
        type: "decimal",
        hint: "money",
        overrides: { style: "number", decimals: 0, suffix: " units" },
      }),
    ).toMatchObject({ style: "number", decimals: 0, suffix: " units" });
  });
});

describe("isNumericDisplay", () => {
  it("marks value columns for right alignment", () => {
    expect(isNumericDisplay(display())).toBe(true);
    expect(isNumericDisplay(display({ style: "currency" }))).toBe(true);
    expect(isNumericDisplay(display({ style: "date" }))).toBe(false);
    expect(isNumericDisplay(undefined)).toBe(false);
  });
});

describe("banded columns", () => {
  it("renders a grouped lower bound as the range it stands for", () => {
    const money = display({
      style: "currency",
      band: { width: 0, edges: [0, 1000, 5000] },
    });

    expect(formatReportValue(0, money)).toBe("$0.00 – $1,000.00");
    expect(formatReportValue(1000, money)).toBe("$1,000.00 – $5,000.00");
    // The top range is open-ended, which is what the query actually did.
    expect(formatReportValue(5000, money)).toBe("$5,000.00+");
  });

  it("keeps even ranges bounded on both sides", () => {
    const counts = display({ style: "number", decimals: 0, band: { width: 5, edges: [] } });

    expect(formatReportValue(0, counts)).toBe("0 – 5");
    expect(formatReportValue(25, counts)).toBe("25 – 30");
    expect(formatReportValue(-10, counts)).toBe("-10 – -5");
  });

  it("formats both ends through the column's own display", () => {
    const hour = 3600;
    const dwell = display({
      style: "duration",
      decimals: 0,
      band: { width: 0, edges: [0, 2 * hour, 4 * hour] },
    });

    expect(formatReportValue(0, dwell)).toBe("0:00 – 2:00");
    expect(formatReportValue(4 * hour, dwell)).toBe("4:00+");
  });

  it("ignores a band that cannot describe a range", () => {
    expect(formatReportValue(1000, display({ band: { width: 0, edges: [] } }))).toBe("1,000.00");
    expect(formatReportValue(1000, display({ band: { width: 0, edges: [500] } }))).toBe("1,000.00");
  });

  it("carries the band through resolveReportDisplay", () => {
    const resolved = resolveReportDisplay({
      type: "int",
      hint: "weight",
      band: { width: 5000, edges: [] },
    });

    expect(resolved.band).toEqual({ width: 5000, edges: [] });
    expect(formatReportValue(10000, resolved)).toBe("10,000 – 15,000 lbs");
  });
});
