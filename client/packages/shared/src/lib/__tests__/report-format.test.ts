import { describe, expect, it } from "vitest";
import {
  formatReportDate,
  isNumericDisplay,
  reportNumericValue,
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

describe("formatReportDate", () => {
  it("writes the module's epoch seconds in the report's time zone and date style", () => {
    expect(formatReportDate(1772587800, "dateTime", "UTC")).toBe("2026-03-04 01:30");
    expect(formatReportDate(1772587800, "date", "UTC")).toBe("2026-03-04");
    expect(formatReportDate(1772587800, "dateLong", "UTC")).toBe("Mar 4, 2026");
    expect(formatReportDate(1772587800, "monthYear", "UTC")).toBe("Mar 2026");
    expect(formatReportDate(1772587800, "quarter", "UTC")).toBe("Q1 2026");
    expect(formatReportDate(1772587800, "weekday", "UTC")).toBe("Wed 2026-03-04");
    expect(formatReportDate(1772587800, "iso", "UTC")).toBe("2026-03-04T01:30:00.000Z");
    expect(formatReportDate(1772587800, "dateTime", "America/Chicago")).toBe("2026-03-03 19:30");
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

describe("reportNumericValue", () => {
  it("reads numbers and decimal strings, and nothing else, for charts and sorting", () => {
    expect(reportNumericValue(12.5)).toBe(12.5);
    expect(reportNumericValue("1082.07")).toBe(1082.07);
    expect(reportNumericValue(" ")).toBeNull();
    expect(reportNumericValue("abc")).toBeNull();
    expect(reportNumericValue(Number.NaN)).toBeNull();
    expect(reportNumericValue(null)).toBeNull();
  });
});
