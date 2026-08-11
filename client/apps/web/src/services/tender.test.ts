import { describe, expect, it } from "vitest";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { spotTenderPayloadSchema, type SpotTenderPayload } from "@trenova/shared/types/tender";
import { isOverridableInsuranceWarningError, toSpotTenderRequestBody } from "./tender";

const PROBLEM_BASE = "https://api.trenova.app/problems/";

function payload(overrides: Partial<SpotTenderPayload> = {}): SpotTenderPayload {
  return spotTenderPayloadSchema.parse({
    shipmentMoveId: "smv_1",
    mode: "SpotBroadcast",
    lines: [
      {
        carrierId: "car_1",
        rateMethod: "Flat",
        rate: 1500,
        offerTtlSeconds: 7200,
        channel: "Email",
        email: "",
      },
    ],
    ...overrides,
  });
}

function businessError(params?: Record<string, string>): ApiRequestError {
  return new ApiRequestError(422, {
    type: `${PROBLEM_BASE}business-rule-violation`,
    title: "Business rule violation",
    status: 422,
    detail: "Carrier has insurance warnings. Confirm the override to proceed",
    params,
  });
}

describe("toSpotTenderRequestBody", () => {
  it("defaults overrideInsuranceWarnings to false", () => {
    const body = toSpotTenderRequestBody(payload());
    expect(body.overrideInsuranceWarnings).toBe(false);
  });

  it("carries an explicit override through to the request body", () => {
    const body = toSpotTenderRequestBody(payload({ overrideInsuranceWarnings: true }));
    expect(body.overrideInsuranceWarnings).toBe(true);
  });
});

describe("isOverridableInsuranceWarningError", () => {
  it("matches a business error flagged overridable", () => {
    expect(isOverridableInsuranceWarningError(businessError({ overridable: "true" }))).toBe(true);
  });

  it("rejects a business error without the overridable param", () => {
    expect(isOverridableInsuranceWarningError(businessError())).toBe(false);
    expect(isOverridableInsuranceWarningError(businessError({ carrierId: "car_1" }))).toBe(false);
  });

  it("rejects non-business problems even when the param is present", () => {
    const validation = new ApiRequestError(422, {
      type: `${PROBLEM_BASE}validation-error`,
      title: "Validation failed",
      status: 422,
      params: { overridable: "true" },
    });
    expect(isOverridableInsuranceWarningError(validation)).toBe(false);
  });

  it("rejects plain errors", () => {
    expect(isOverridableInsuranceWarningError(new Error("boom"))).toBe(false);
    expect(isOverridableInsuranceWarningError(undefined)).toBe(false);
  });
});
