import { describe, expect, it } from "vitest";
import {
  annualCostMinor,
  benefitPlanTypeLabel,
  coverageTierLabel,
  employerSharePercent,
  enrollmentStatusLabel,
  enrollmentStatusTone,
  formatMinor,
} from "../benefits";

describe("labels", () => {
  it("names plan types and coverage tiers in the words a person uses", () => {
    expect(benefitPlanTypeLabel("Life")).toBe("Life insurance");
    expect(coverageTierLabel("EmployeeSpouse")).toBe("Employee and spouse");
    // "Waived" is what the column says; "Declined" is what happened.
    expect(enrollmentStatusLabel("Waived")).toBe("Declined");
  });

  it("passes an unknown value through rather than blanking it", () => {
    expect(benefitPlanTypeLabel("Pet")).toBe("Pet");
    expect(coverageTierLabel("Household")).toBe("Household");
  });
});

describe("enrollmentStatusTone", () => {
  it("grades the four states", () => {
    expect(enrollmentStatusTone("Active")).toBe("active");
    expect(enrollmentStatusTone("Ended")).toBe("inactive");
    expect(enrollmentStatusTone("Pending")).toBe("warning");
    expect(enrollmentStatusTone("Waived")).toBe("secondary");
  });
});

describe("formatMinor", () => {
  // Integer arithmetic all the way to the formatter: dividing by 100 is the
  // last thing that happens, not the first.
  it("renders minor units as money", () => {
    expect(formatMinor(12_000)).toBe("$120.00");
    expect(formatMinor(0)).toBe("$0.00");
    expect(formatMinor(9_048_000)).toBe("$90,480.00");
  });

  it("keeps the odd cent", () => {
    expect(formatMinor(1_234_567)).toBe("$12,345.67");
  });
});

describe("annualCostMinor", () => {
  it("multiplies the per-period cost by the periods", () => {
    expect(annualCostMinor(12_000, 26)).toBe(312_000);
  });

  // An employer-paid plan costs the worker nothing a year, not a rounding
  // error a year.
  it("is nothing when either side is nothing", () => {
    expect(annualCostMinor(0, 26)).toBe(0);
    expect(annualCostMinor(12_000, 0)).toBe(0);
  });
});

describe("employerSharePercent", () => {
  it("is the employer's share of the package to a tenth", () => {
    expect(employerSharePercent(48_000, 9_048_000)).toBe(0.5);
    expect(employerSharePercent(50, 100)).toBe(50);
  });

  // An empty statement has no split, not an even one.
  it("is zero when there is nothing to divide", () => {
    expect(employerSharePercent(0, 100)).toBe(0);
    expect(employerSharePercent(50, 0)).toBe(0);
  });
});
