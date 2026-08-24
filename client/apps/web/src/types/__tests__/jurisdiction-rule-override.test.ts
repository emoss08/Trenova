import { describe, expect, it } from "vitest";
import { jurisdictionRuleOverrideSchema } from "@/types/jurisdiction-rule-override";

// Fixtures written from Override.Validate in
// services/tms/internal/core/domain/jurisdictionrule/override.go.
const base = {
  stateId: "us_01ARZ3NDEKTSV4RRFFQ69G5FAV",
  reason: "Our trailer fleet is narrower than this state allows",
};

describe("jurisdictionRuleOverrideSchema", () => {
  it("accepts an override that narrows one limit", () => {
    expect(jurisdictionRuleOverrideSchema.safeParse({ ...base, maxWidthFeet: 8 }).success).toBe(
      true,
    );
  });

  // Mirrors Override.HasAnyOverride. A reason attached to nothing is not an
  // override, and the row would change no behaviour.
  it("rejects an override that changes nothing", () => {
    expect(jurisdictionRuleOverrideSchema.safeParse(base).success).toBe(false);
  });

  it("requires a reason of at least ten characters", () => {
    expect(
      jurisdictionRuleOverrideSchema.safeParse({
        ...base,
        maxWidthFeet: 8,
        reason: "narrow",
      }).success,
    ).toBe(false);
  });

  // Every limit is nullable and null means "defer to the statute", so absent
  // has to stay distinguishable from a value. A schema that coerced null to
  // zero would read as an override of zero feet.
  it("treats a null limit as deferring rather than as zero", () => {
    const result = jurisdictionRuleOverrideSchema.safeParse({
      ...base,
      maxWidthFeet: null,
      maxHeightFeet: 12,
    });

    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.maxWidthFeet).toBeNull();
      expect(result.data.maxHeightFeet).toBe(12);
    }
  });

  it("rejects a non-positive limit", () => {
    expect(jurisdictionRuleOverrideSchema.safeParse({ ...base, maxWidthFeet: 0 }).success).toBe(
      false,
    );
    expect(jurisdictionRuleOverrideSchema.safeParse({ ...base, maxWeightPounds: -1 }).success).toBe(
      false,
    );
  });

  it("holds lead time to the server's ceiling", () => {
    expect(
      jurisdictionRuleOverrideSchema.safeParse({ ...base, permitLeadTimeDays: 61 }).success,
    ).toBe(false);
    expect(
      jurisdictionRuleOverrideSchema.safeParse({ ...base, permitLeadTimeDays: 7 }).success,
    ).toBe(true);
  });

  // A restriction alone is a valid override — a carrier can decline to run at
  // night without narrowing any dimension.
  it("accepts a restriction as the only change", () => {
    expect(jurisdictionRuleOverrideSchema.safeParse({ ...base, daylightOnly: true }).success).toBe(
      true,
    );
  });
});
