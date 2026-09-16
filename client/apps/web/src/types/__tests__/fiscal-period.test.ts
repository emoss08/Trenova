import { describe, expect, it } from "vitest";
import { fiscalPeriodSchema } from "@/types/fiscal-period";
import { fiscalYearSchema } from "@/types/fiscal-year";

// Contract: services/tms/internal/core/domain/fiscalperiod. A period's status runs
// Inactive → Open → Locked → Closed → PermanentlyClosed, its type is Month, Quarter,
// Week or Adjusting, and operating periods stop at 12 while adjusting periods
// (Period 13/14) sit above them. Generating a year with adjusting entries allowed
// appends an Inactive Adjusting Period 13, so the edit form parses one on every
// such year.

const operatingPeriod = {
  id: "fp_01ARZ3NDEKTSV4RRFFQ69G5FAV",
  organizationId: "org_01ARZ3NDEKTSV4RRFFQ69G5FAV",
  businessUnitId: "bu_01ARZ3NDEKTSV4RRFFQ69G5FAV",
  fiscalYearId: "fy_01ARZ3NDEKTSV4RRFFQ69G5FAV",
  version: 1,
  status: "Open" as const,
  periodType: "Month" as const,
  periodNumber: 12,
  name: "December 2026",
  startDate: 1_795_996_800,
  endDate: 1_798_675_199,
  closedAt: null,
};

const adjustingPeriod = {
  ...operatingPeriod,
  id: "fp_01ARZ3NDEKTSV4RRFFQ69G5FAW",
  status: "Inactive" as const,
  periodType: "Adjusting" as const,
  periodNumber: 13,
  name: "Adjusting Period 2026",
};

describe("fiscalPeriodSchema", () => {
  it.each(["Inactive", "Open", "Locked", "Closed", "PermanentlyClosed"])(
    "accepts the %s status",
    (status) => {
      const parsed = fiscalPeriodSchema.safeParse({ ...operatingPeriod, status });

      expect(parsed.success, JSON.stringify(parsed.error?.issues)).toBe(true);
    },
  );

  it.each(["Month", "Quarter", "Week"])("accepts the %s period type", (periodType) => {
    const parsed = fiscalPeriodSchema.safeParse({ ...operatingPeriod, periodType });

    expect(parsed.success, JSON.stringify(parsed.error?.issues)).toBe(true);
  });

  it("rejects values the server does not define", () => {
    expect(fiscalPeriodSchema.safeParse({ ...operatingPeriod, status: "Draft" }).success).toBe(
      false,
    );
    expect(fiscalPeriodSchema.safeParse({ ...operatingPeriod, periodType: "Year" }).success).toBe(
      false,
    );
  });

  it("accepts adjusting periods 13 and 14", () => {
    for (const periodNumber of [13, 14]) {
      const parsed = fiscalPeriodSchema.safeParse({ ...adjustingPeriod, periodNumber });

      expect(parsed.success, JSON.stringify(parsed.error?.issues)).toBe(true);
    }
  });

  it("accepts a period flagged isAdjusting above 12 whatever its type", () => {
    const parsed = fiscalPeriodSchema.safeParse({
      ...operatingPeriod,
      periodNumber: 13,
      isAdjusting: true,
    });

    expect(parsed.success, JSON.stringify(parsed.error?.issues)).toBe(true);
  });

  it("caps operating periods at 12 and adjusting periods at 14", () => {
    const operating = fiscalPeriodSchema.safeParse({ ...operatingPeriod, periodNumber: 13 });
    const adjusting = fiscalPeriodSchema.safeParse({ ...adjustingPeriod, periodNumber: 15 });

    expect(operating.success).toBe(false);
    expect(operating.error?.issues[0]?.path).toEqual(["periodNumber"]);
    expect(adjusting.success).toBe(false);
    expect(adjusting.error?.issues[0]?.path).toEqual(["periodNumber"]);
  });

  it("rejects a period number below 1", () => {
    expect(fiscalPeriodSchema.safeParse({ ...operatingPeriod, periodNumber: 0 }).success).toBe(
      false,
    );
  });
});

describe("fiscalYearSchema periods", () => {
  it("parses a year whose periods end with an adjusting Period 13", () => {
    const months = Array.from({ length: 12 }, (_, index) => ({
      ...operatingPeriod,
      id: `fp_${index}`,
      periodNumber: index + 1,
    }));

    const parsed = fiscalYearSchema.safeParse({
      id: "fy_01ARZ3NDEKTSV4RRFFQ69G5FAV",
      version: 1,
      status: "Open",
      year: 2026,
      name: "FY 2026",
      description: "",
      startDate: 1_767_225_600,
      endDate: 1_798_761_599,
      isCurrent: true,
      isCalendarYear: true,
      allowAdjustingEntries: true,
      periods: [...months, adjustingPeriod],
    });

    expect(parsed.success, JSON.stringify(parsed.error?.issues)).toBe(true);
    expect(parsed.data?.periods).toHaveLength(13);
  });
});
