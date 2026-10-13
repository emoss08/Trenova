import { describe, expect, it } from "vitest";
import { ApiRequestError } from "@trenova/shared/lib/api";
import type { PublicTrackingStop } from "@trenova/shared/types/carrier-tracking";
import { checkCallPostFailure, nextStopAction } from "../carrier-tracking";

function problem(
  status: number,
  type: string,
  errors?: Array<{ field: string; message: string }>,
  detail?: string,
): ApiRequestError {
  return new ApiRequestError(status, {
    type: `https://trenova.app/problems/${type}`,
    title: "Error",
    status,
    detail,
    errors,
  });
}

function stop(overrides: Partial<PublicTrackingStop>): PublicTrackingStop {
  return {
    id: "stp_1",
    sequence: 0,
    type: "Pickup",
    locationName: "DC 12",
    city: "Dallas",
    stateCode: "TX",
    windowStart: 1_800_000_000,
    windowEnd: null,
    arrivedAt: null,
    departedAt: null,
    ...overrides,
  };
}

describe("checkCallPostFailure", () => {
  it("reads only a token field error as a dead link", () => {
    expect(
      checkCallPostFailure(
        problem(400, "validation-error", [
          { field: "token", message: "This tracking link is no longer valid" },
        ]),
      ),
    ).toEqual({ kind: "invalid" });
  });

  it("hands the carrier's own mistakes back as messages, not a dead link", () => {
    expect(
      checkCallPostFailure(
        problem(400, "validation-error", [
          { field: "delayReason", message: "Say why the load is delayed" },
          { field: "reportedAt", message: "The reported time cannot be in the future" },
        ]),
      ),
    ).toEqual({
      kind: "rejected",
      messages: ["Say why the load is delayed", "The reported time cannot be in the future"],
    });
  });

  it("falls back to the problem detail when a rejection names no field", () => {
    expect(
      checkCallPostFailure(
        problem(404, "resource-not-found", undefined, "Stop not found on this load"),
      ),
    ).toEqual({ kind: "rejected", messages: ["Stop not found on this load"] });
  });

  it("reads a business rule violation as a finished load", () => {
    expect(
      checkCallPostFailure(
        problem(422, "business-rule-violation", undefined, "This load is no longer being tracked"),
      ),
    ).toEqual({ kind: "closed", message: "This load is no longer being tracked" });
  });

  it("never presents throttling, a server error or a network failure as a dead link", () => {
    expect(checkCallPostFailure(problem(429, "rate-limit-exceeded"))).toEqual({
      kind: "throttled",
    });
    expect(checkCallPostFailure(problem(503, "internal-error"))).toEqual({ kind: "unavailable" });
    expect(checkCallPostFailure(new TypeError("Failed to fetch"))).toEqual({ kind: "unavailable" });
  });
});

describe("nextStopAction", () => {
  it("offers the arrival at the first stop nobody has reached", () => {
    const pickup = stop({ id: "stp_1", sequence: 0 });
    const delivery = stop({ id: "stp_2", sequence: 1, type: "Delivery" });

    expect(nextStopAction([delivery, pickup])).toEqual({ stop: pickup, kind: "Arrival" });
  });

  it("offers the departure from a stop the truck is still at", () => {
    const pickup = stop({ id: "stp_1", sequence: 0, arrivedAt: 1_800_000_100 });
    const delivery = stop({ id: "stp_2", sequence: 1 });

    expect(nextStopAction([pickup, delivery])).toEqual({ stop: pickup, kind: "Departure" });
  });

  it("moves on once a stop is left, and offers nothing when every stop is", () => {
    const pickup = stop({
      id: "stp_1",
      sequence: 0,
      arrivedAt: 1_800_000_100,
      departedAt: 1_800_000_200,
    });
    const delivery = stop({ id: "stp_2", sequence: 1 });

    expect(nextStopAction([pickup, delivery])).toEqual({ stop: delivery, kind: "Arrival" });
    expect(
      nextStopAction([
        pickup,
        { ...delivery, arrivedAt: 1_800_000_300, departedAt: 1_800_000_400 },
      ]),
    ).toBeNull();
    expect(nextStopAction([])).toBeNull();
  });
});
