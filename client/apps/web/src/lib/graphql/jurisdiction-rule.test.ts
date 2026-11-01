import { beforeEach, describe, expect, it, vi } from "vitest";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { JurisdictionRule } from "@/types/jurisdiction-rule";
import type { JurisdictionRuleOverride } from "@/types/jurisdiction-rule-override";
import {
  deleteJurisdictionRuleOverride,
  toJurisdictionRuleInput,
  toJurisdictionRuleOverrideInput,
  verifyJurisdictionRule,
} from "./jurisdiction-rule";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

describe("toJurisdictionRuleInput", () => {
  it("sends fees as decimal strings and never the verification the server owns", () => {
    const input = toJurisdictionRuleInput({
      id: "jr_1",
      stateId: "st_ga",
      status: "Active",
      maxWidthFeet: 8.5,
      maxHeightFeet: 13.5,
      maxLengthFeet: 75,
      maxWeightPounds: 80000,
      daylightOnly: true,
      rushHourRestricted: false,
      weekendRestricted: false,
      holidayRestricted: true,
      permitLeadTimeDays: 1,
      permitValidityDays: 5,
      permitBaseFee: 30,
      permitPerMileFee: null,
      verificationState: "Verified",
      version: 7,
    } as JurisdictionRule);

    expect(input).toEqual(
      expect.objectContaining({
        stateId: "st_ga",
        permitBaseFee: "30",
        permitPerMileFee: null,
        superloadWidthFeet: null,
        version: 7,
      }),
    );
    expect(input).not.toHaveProperty("verificationState");
  });
});

describe("toJurisdictionRuleOverrideInput", () => {
  it("leaves every limit the override does not set as null, deferring to the statute", () => {
    expect(
      toJurisdictionRuleOverrideInput({
        stateId: "st_tn",
        maxWidthFeet: 8,
        reason: "Our insurer caps width at eight feet",
      } as JurisdictionRuleOverride),
    ).toEqual({
      stateId: "st_tn",
      maxWidthFeet: 8,
      maxHeightFeet: null,
      maxLengthFeet: null,
      maxWeightPounds: null,
      permitLeadTimeDays: null,
      daylightOnly: null,
      holidayRestricted: null,
      reason: "Our insurer caps width at eight feet",
      version: null,
    });
  });
});

describe("jurisdiction rule writes", () => {
  beforeEach(() => {
    vi.mocked(requestGraphQL).mockReset();
  });

  it("verifies a rule against the source it was checked against", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({
      verifyJurisdictionRule: { id: "jr_1", verificationState: "Verified" },
    });

    await verifyJurisdictionRule("jr_1", {
      verificationState: "Verified",
      sourceNote: "Checked against the GA DOT oversize page",
    });

    expect(vi.mocked(requestGraphQL).mock.calls[0]?.[0]).toEqual(
      expect.objectContaining({
        operationName: "VerifyJurisdictionRule",
        variables: {
          id: "jr_1",
          input: {
            verificationState: "Verified",
            sourceNote: "Checked against the GA DOT oversize page",
            sourceUrl: null,
          },
        },
      }),
    );
  });

  it("deletes an override by id", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({ deleteJurisdictionRuleOverride: true });

    await deleteJurisdictionRuleOverride("jro_1");

    expect(vi.mocked(requestGraphQL).mock.calls[0]?.[0]).toEqual(
      expect.objectContaining({
        operationName: "DeleteJurisdictionRuleOverride",
        variables: { id: "jro_1" },
      }),
    );
  });
});
