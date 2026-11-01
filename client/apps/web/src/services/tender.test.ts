import { describe, expect, it } from "vitest";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { spotTenderPayloadSchema, type SpotTenderPayload } from "@trenova/shared/types/tender";
import {
  isOverridableInsuranceWarningError,
  toRespondToTenderOfferInput,
  toSpotTenderInput,
} from "./tender";

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

function graphQLError(type: string, params?: Record<string, string>): GraphQLRequestError {
  const message = "Carrier has insurance warnings. Confirm the override to proceed";
  return new GraphQLRequestError({
    kind: "graphql",
    message,
    graphQLErrors: [{ message, extensions: { type, params }, type, params }],
    status: 200,
  });
}

function businessError(params?: Record<string, string>): GraphQLRequestError {
  return graphQLError("business-rule-violation", params);
}

describe("toSpotTenderInput", () => {
  it("sends no email for an EDI line and the typed address for an email line", () => {
    const input = toSpotTenderInput(
      payload({
        lines: [
          {
            carrierId: "car_1",
            rateMethod: "Flat",
            rate: 1500,
            offerTtlSeconds: 7200,
            channel: "EDI",
            email: "ops@carrier.test",
          },
          {
            carrierId: "car_2",
            rateMethod: "Flat",
            rate: 1650.5,
            offerTtlSeconds: 3600,
            channel: "Email",
            email: "dispatch@carrier.test",
          },
        ],
      }),
    );
    expect(input.lines).toEqual([
      {
        carrierId: "car_1",
        rateMethod: "Flat",
        rate: "1500",
        offerTtlSeconds: 7200,
        channel: "EDI",
        email: null,
      },
      {
        carrierId: "car_2",
        rateMethod: "Flat",
        rate: "1650.5",
        offerTtlSeconds: 3600,
        channel: "Email",
        email: "dispatch@carrier.test",
      },
    ]);
  });

  it("defaults overrideInsuranceWarnings to false", () => {
    const body = toSpotTenderInput(payload());
    expect(body.overrideInsuranceWarnings).toBe(false);
  });

  it("carries an explicit override through to the request body", () => {
    const body = toSpotTenderInput(payload({ overrideInsuranceWarnings: true }));
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
    const validation = graphQLError("validation-error", { overridable: "true" });
    expect(isOverridableInsuranceWarningError(validation)).toBe(false);
  });

  it("rejects plain errors", () => {
    expect(isOverridableInsuranceWarningError(new Error("boom"))).toBe(false);
    expect(isOverridableInsuranceWarningError(undefined)).toBe(false);
  });
});

describe("toRespondToTenderOfferInput", () => {
  it("sends an acceptance without the reason or counter left in the form", () => {
    expect(
      toRespondToTenderOfferInput({
        action: "Accept",
        declineReason: "No truck",
        counterRate: "2100",
        counterNote: "Fuel is up",
      }),
    ).toEqual({ action: "Accept", declineReason: "" });
  });

  it("sends a decline with its trimmed reason and no counter", () => {
    expect(
      toRespondToTenderOfferInput({
        action: "Decline",
        declineReason: "  No truck in the area ",
        counterRate: "2100",
      }),
    ).toEqual({ action: "Decline", declineReason: "No truck in the area" });
  });

  it("sends a counter with its rate and note, and no decline reason", () => {
    expect(
      toRespondToTenderOfferInput({
        action: "Counter",
        declineReason: "left over",
        counterRate: " 2100.50 ",
        counterNote: " Fuel is up ",
      }),
    ).toEqual({
      action: "Counter",
      declineReason: "",
      counterRate: "2100.50",
      counterNote: "Fuel is up",
    });
  });

  it("refuses a counter without a positive rate", () => {
    expect(() => toRespondToTenderOfferInput({ action: "Counter", counterRate: "" })).toThrow(
      "Counter rate is required",
    );
    expect(() => toRespondToTenderOfferInput({ action: "Counter", counterRate: "0" })).toThrow(
      "Counter rate must be a positive amount",
    );
    expect(() =>
      toRespondToTenderOfferInput({ action: "Counter", counterRate: "10000000" }),
    ).toThrow("Counter rate is too large");
  });
});
