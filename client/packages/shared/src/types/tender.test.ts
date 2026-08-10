import { describe, expect, it } from "vitest";
import { emptySpotTenderLine, spotTenderPayloadSchema } from "./tender";

function makeSpotPayload(overrides: Record<string, unknown> = {}): Record<string, unknown> {
  return {
    shipmentMoveId: "smv_01",
    mode: "SpotBroadcast",
    lines: [{ ...emptySpotTenderLine, carrierId: "car_01", rate: 1500 }],
    ...overrides,
  };
}

describe("spotTenderPayloadSchema", () => {
  it("accepts plain numbers from the form inputs", () => {
    const parsed = spotTenderPayloadSchema.parse(makeSpotPayload());
    expect(parsed.lines[0].rate).toBe(1500);
  });

  it("coerces decimal strings on the prefill path to numbers", () => {
    const parsed = spotTenderPayloadSchema.parse(
      makeSpotPayload({
        lines: [{ ...emptySpotTenderLine, carrierId: "car_01", rate: "1500.2500" }],
      }),
    );
    expect(parsed.lines[0].rate).toBe(1500.25);
  });

  it("rejects negative rates whether they arrive as strings or numbers", () => {
    expect(() =>
      spotTenderPayloadSchema.parse(
        makeSpotPayload({
          lines: [{ ...emptySpotTenderLine, carrierId: "car_01", rate: -5 }],
        }),
      ),
    ).toThrowError(/Rate cannot be negative/);
    expect(() =>
      spotTenderPayloadSchema.parse(
        makeSpotPayload({
          lines: [{ ...emptySpotTenderLine, carrierId: "car_01", rate: "-1.00" }],
        }),
      ),
    ).toThrowError(/Rate cannot be negative/);
  });

  it("rejects a waterfall mode — spot tenders are broadcast or sequential only", () => {
    expect(() => spotTenderPayloadSchema.parse(makeSpotPayload({ mode: "Waterfall" }))).toThrow();
  });

  it("requires at least one line", () => {
    expect(() => spotTenderPayloadSchema.parse(makeSpotPayload({ lines: [] }))).toThrowError(
      /At least one carrier line is required/,
    );
  });

  it("flags a duplicate carrier on the offending line, mirroring the server", () => {
    const result = spotTenderPayloadSchema.safeParse(
      makeSpotPayload({
        lines: [
          { ...emptySpotTenderLine, carrierId: "car_01", rate: 1000 },
          { ...emptySpotTenderLine, carrierId: "car_01", rate: 1200 },
        ],
      }),
    );
    expect(result.success).toBe(false);
    if (!result.success) {
      const issue = result.error.issues.find(
        (item) => item.message === "Carrier is already on this tender",
      );
      expect(issue?.path).toEqual(["lines", 1, "carrierId"]);
    }
  });

  it("bounds the offer expiry to the server's 5-minute/7-day window", () => {
    expect(() =>
      spotTenderPayloadSchema.parse(
        makeSpotPayload({
          lines: [{ ...emptySpotTenderLine, carrierId: "car_01", offerTtlSeconds: 60 }],
        }),
      ),
    ).toThrowError(/at least 5 minutes/);
    expect(() =>
      spotTenderPayloadSchema.parse(
        makeSpotPayload({
          lines: [{ ...emptySpotTenderLine, carrierId: "car_01", offerTtlSeconds: 604801 }],
        }),
      ),
    ).toThrowError(/cannot exceed 7 days/);
  });

  it("accepts an empty email and rejects a malformed one", () => {
    expect(() =>
      spotTenderPayloadSchema.parse(
        makeSpotPayload({
          lines: [{ ...emptySpotTenderLine, carrierId: "car_01", email: "" }],
        }),
      ),
    ).not.toThrow();
    expect(() =>
      spotTenderPayloadSchema.parse(
        makeSpotPayload({
          lines: [{ ...emptySpotTenderLine, carrierId: "car_01", email: "not-an-email" }],
        }),
      ),
    ).toThrowError(/valid email/);
  });
});
