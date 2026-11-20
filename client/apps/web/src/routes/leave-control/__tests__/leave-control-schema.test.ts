import { leaveControlSchema, type LeaveControl } from "@trenova/shared/types/worker-leave";
import { describe, expect, it } from "vitest";
import { savedLeaveControl } from "./leave-control-fixture";

function refusals(record: LeaveControl) {
  const result = leaveControlSchema.safeParse(record);
  return result.success
    ? []
    : result.error.issues.map((issue) => [issue.path.join("."), issue.message]);
}

describe("leave control schema", () => {
  it("accepts the server's defaults as the query sends them", () => {
    expect(refusals(savedLeaveControl)).toEqual([]);
  });

  it("accepts a more generous policy, to two decimals", () => {
    expect(
      refusals({
        ...savedLeaveControl,
        entitlementWeeks: "16.5",
        militaryCaregiverWeeks: "30",
        workweekHours: "37.5",
      }),
    ).toEqual([]);
  });

  it.each([
    ["entitlementWeeks", "11.99", "FMLA entitles an eligible employee to at least twelve weeks"],
    ["militaryCaregiverWeeks", "25", "Military caregiver leave is at least twenty-six weeks"],
    ["workweekHours", "0", "A workweek must be more than zero hours"],
    ["workweekHours", "168.5", "A workweek cannot be more than 168 hours"],
    ["workweekHours", "40.125", "Enter a number with at most two decimals"],
  ] as const)("refuses %s of %s", (field, value, message) => {
    expect(refusals({ ...savedLeaveControl, [field]: value })).toEqual([[field, message]]);
  });

  it.each([
    ["eligibilityMonths", -1, "Months of service cannot be negative"],
    ["eligibilityHours", 8761, "A year holds at most 8,760 hours"],
    [
      "certificationDueDays",
      14,
      "The employee must be given at least fifteen days (29 CFR 825.305(b))",
    ],
  ] as const)("refuses %s of %d", (field, value, message) => {
    expect(refusals({ ...savedLeaveControl, [field]: value })).toEqual([[field, message]]);
  });

  // Over 52 weeks the entitlement is also longer than the caregiver leave beside it, and both
  // rows say so.
  it("refuses an entitlement over 52 weeks", () => {
    expect(refusals({ ...savedLeaveControl, entitlementWeeks: "52.01" })).toEqual([
      ["entitlementWeeks", "Leave cannot be more than 52 weeks a year"],
      [
        "militaryCaregiverWeeks",
        "Military caregiver leave cannot be shorter than the ordinary entitlement",
      ],
    ]);
  });

  it("refuses military caregiver leave shorter than the ordinary entitlement", () => {
    expect(
      refusals({ ...savedLeaveControl, entitlementWeeks: "30", militaryCaregiverWeeks: "26" }),
    ).toEqual([
      [
        "militaryCaregiverWeeks",
        "Military caregiver leave cannot be shorter than the ordinary entitlement",
      ],
    ]);
  });
});
