import { translate } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import { formatAnalysisValue, isNumericKind } from "../analysis-values";

describe("formatAnalysisValue", () => {
  it("reads money in its currency and a percent from its fraction", () => {
    expect(formatAnalysisValue(6000000, "money", translate, "USD")).toBe("$6,000,000.00");
    expect(formatAnalysisValue(0.021, "percent", translate)).toBe("2.1%");
    expect(formatAnalysisValue(32, "count", translate)).toBe("32");
  });

  it("keeps a date on the day the analysis wrote it, whatever the reader's timezone", () => {
    expect(formatAnalysisValue("2026-10-09T19:26:21-07:00", "date", translate)).toBe("Oct 9, 2026");
  });

  it("reads a missing or non-finite value as a dash", () => {
    expect(formatAnalysisValue(null, "money", translate)).toBe("—");
    expect(formatAnalysisValue(Number.NaN, "number", translate)).toBe("—");
    expect(formatAnalysisValue("", "text", translate)).toBe("—");
  });

  it("aligns only the kinds that are numbers", () => {
    expect(isNumericKind("money")).toBe(true);
    expect(isNumericKind("percent")).toBe(true);
    expect(isNumericKind("date")).toBe(false);
    expect(isNumericKind("text")).toBe(false);
  });
});
