import { describe, expect, it } from "vitest";
import { jurisdictionRuleSchema, verifyJurisdictionRuleSchema } from "@/types/jurisdiction-rule";

// Fixtures are written from JurisdictionRule.Validate in
// services/tms/internal/core/domain/jurisdictionrule/rule.go, not from what the
// form happens to produce. The client checks the same rules so a typo lands on
// the field that is wrong instead of returning as a whole-form error.
const base = {
  stateId: "us_01ARZ3NDEKTSV4RRFFQ69G5FAV",
  status: "Active" as const,
  maxWidthFeet: 8.5,
  maxHeightFeet: 13.5,
  maxLengthFeet: 53,
  maxWeightPounds: 80000,
  permitLeadTimeDays: 1,
  permitValidityDays: 5,
};

describe("jurisdictionRuleSchema", () => {
  it("accepts the federal baseline", () => {
    expect(jurisdictionRuleSchema.safeParse(base).success).toBe(true);
  });

  it("requires a jurisdiction", () => {
    expect(jurisdictionRuleSchema.safeParse({ ...base, stateId: "" }).success).toBe(false);
  });

  it.each([
    ["maxWidthFeet", 0],
    ["maxWidthFeet", 61],
    ["maxHeightFeet", 0],
    ["maxHeightFeet", 41],
    ["maxLengthFeet", 0],
    ["maxLengthFeet", 301],
    ["maxWeightPounds", 0],
  ])("rejects %s of %s as outside the sanity rails", (field, value) => {
    expect(jurisdictionRuleSchema.safeParse({ ...base, [field]: value }).success).toBe(false);
  });

  // A superload threshold at or below the permitted maximum would classify
  // every permitted load as a superload.
  it("requires superload thresholds above the permitted maximum", () => {
    expect(jurisdictionRuleSchema.safeParse({ ...base, superloadWidthFeet: 8.5 }).success).toBe(
      false,
    );
    expect(jurisdictionRuleSchema.safeParse({ ...base, superloadWidthFeet: 16 }).success).toBe(
      true,
    );
    expect(
      jurisdictionRuleSchema.safeParse({ ...base, superloadWeightPounds: 80000 }).success,
    ).toBe(false);
    expect(
      jurisdictionRuleSchema.safeParse({ ...base, superloadWeightPounds: 120000 }).success,
    ).toBe(true);
  });

  it("holds permit terms to the server's bounds", () => {
    expect(jurisdictionRuleSchema.safeParse({ ...base, permitLeadTimeDays: 61 }).success).toBe(
      false,
    );
    expect(jurisdictionRuleSchema.safeParse({ ...base, permitLeadTimeDays: 0 }).success).toBe(true);
    expect(jurisdictionRuleSchema.safeParse({ ...base, permitValidityDays: 0 }).success).toBe(
      false,
    );
    expect(jurisdictionRuleSchema.safeParse({ ...base, permitValidityDays: 366 }).success).toBe(
      false,
    );
  });

  it("rejects an effective window that ends before it starts", () => {
    expect(
      jurisdictionRuleSchema.safeParse({
        ...base,
        effectiveStartDate: 2000,
        effectiveEndDate: 1000,
      }).success,
    ).toBe(false);
    expect(
      jurisdictionRuleSchema.safeParse({
        ...base,
        effectiveStartDate: 1000,
        effectiveEndDate: 2000,
      }).success,
    ).toBe(true);
  });

  // Absent is the agreed representation for "we do not know", and it has to
  // stay distinguishable from a value. Rejecting null here would push editors
  // toward inventing a threshold.
  it("treats unknown superload thresholds as absent rather than invalid", () => {
    expect(
      jurisdictionRuleSchema.safeParse({
        ...base,
        superloadWidthFeet: null,
        superloadWeightPounds: null,
      }).success,
    ).toBe(true);
  });
});

describe("verifyJurisdictionRuleSchema", () => {
  it("holds the note to the server's minimum length", () => {
    expect(
      verifyJurisdictionRuleSchema.safeParse({
        verificationState: "Verified",
        sourceNote: "checked",
      }).success,
    ).toBe(false);
    expect(
      verifyJurisdictionRuleSchema.safeParse({
        verificationState: "Verified",
        sourceNote: "a".repeat(10),
      }).success,
    ).toBe(true);
  });

  // Unverified is where a row starts and where an edit returns it; the server
  // rejects setting it directly, so the form must not offer it.
  it("does not accept marking a row unverified", () => {
    expect(
      verifyJurisdictionRuleSchema.safeParse({
        verificationState: "Unverified",
        sourceNote: "Checked against the state permit office handbook",
      }).success,
    ).toBe(false);
  });

  it("accepts disputed", () => {
    expect(
      verifyJurisdictionRuleSchema.safeParse({
        verificationState: "Disputed",
        sourceNote: "State permit office contradicts this width; escalated",
      }).success,
    ).toBe(true);
  });
});
