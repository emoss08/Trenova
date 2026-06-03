import { describe, expect, it } from "vitest";
import { serviceFailure214PartnerSettingsSchema } from "./edi";

describe("serviceFailure214PartnerSettingsSchema", () => {
  it("parses valid settings with defaults", () => {
    const parsed = serviceFailure214PartnerSettingsSchema.parse({
      enabled: true,
      sendOnReviewed: true,
      statusCode: "SD",
      acceptedReasonCodes: ["NS", "CA"],
    });

    expect(parsed).toMatchObject({
      enabled: true,
      sendOnReviewed: true,
      sendOnResolved: false,
      mandatoryOnReviewed: false,
      mandatoryOnResolved: false,
      statusCode: "SD",
      requireStatusReasonCode: false,
      requireLocation: false,
      requireLocationName: false,
      requireCityState: false,
      requirePostalCode: false,
      requireTimeCode: false,
      requireStop: false,
      requireProNumber: false,
      requireBol: false,
      acceptedReasonCodes: ["NS", "CA"],
    });
  });

  it("rejects invalid types and unknown keys", () => {
    expect(() =>
      serviceFailure214PartnerSettingsSchema.parse({
        enabled: "true",
        acceptedReasonCodes: ["NS", 42],
        unknown: true,
      }),
    ).toThrow();
  });
});
