import { describe, expect, it } from "vitest";
import { safetyViolationFormSchema } from "../fleet-safety";

function baseViolation() {
  return {
    basic: "HOSCompliance" as const,
    code: "395.8",
    description: "No record of duty status",
    severityWeight: 5,
    outOfService: false,
  };
}

describe("safetyViolationFormSchema", () => {
  it("accepts a violation off an inspection report", () => {
    expect(safetyViolationFormSchema.safeParse(baseViolation()).success).toBe(true);
  });

  // The FMCSA weights run 1 to 10 and the published tables are what a clerk
  // keys from. A weight nobody could look up is not evidence of anything.
  it("refuses a weight outside the published scale", () => {
    expect(
      safetyViolationFormSchema.safeParse({ ...baseViolation(), severityWeight: 0 }).success,
    ).toBe(false);
    expect(
      safetyViolationFormSchema.safeParse({ ...baseViolation(), severityWeight: 11 }).success,
    ).toBe(false);
  });

  // The field is bound to a numeric control. A schema that accepted the string
  // a select would have written would let the two disagree silently.
  it("refuses a weight that arrived as text", () => {
    expect(
      safetyViolationFormSchema.safeParse({ ...baseViolation(), severityWeight: "5" }).success,
    ).toBe(false);
  });

  it("refuses a BASIC outside the seven", () => {
    expect(
      safetyViolationFormSchema.safeParse({ ...baseViolation(), basic: "Paperwork" }).success,
    ).toBe(false);
  });
});
