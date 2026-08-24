import { describe, expect, it } from "vitest";
import {
  emptySpotTenderLine,
  spotTenderPayloadSchema,
  tenderSchema,
  waterfallTenderResultSchema,
} from "./tender";

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

  it("ignores a leftover email on an EDI line and strips it from the parsed payload", () => {
    const parsed = spotTenderPayloadSchema.parse(
      makeSpotPayload({
        lines: [
          {
            ...emptySpotTenderLine,
            carrierId: "car_01",
            channel: "EDI",
            email: "not-an-email",
          },
        ],
      }),
    );
    expect(parsed.lines[0].email).toBe("");
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

/**
 * The Go handler returns CreateWaterfallResult, which embeds the tender itself —
 * so the body stays a superset of a plain tender and any consumer still parsing
 * with tenderSchema keeps working.
 */
function makeWaterfallResponse(overrides: Record<string, unknown> = {}): Record<string, unknown> {
  return {
    id: "ten_01",
    organizationId: "org_01",
    businessUnitId: "bu_01",
    shipmentId: "shp_01",
    shipmentMoveId: "smv_01",
    routingGuideId: "rgd_01",
    mode: "Waterfall",
    status: "Active",
    currentRank: 0,
    cancellationReason: null,
    acceptedOfferId: null,
    acceptedAt: null,
    exhaustedAt: null,
    canceledAt: null,
    offers: [],
    ...overrides,
  };
}

describe("waterfallTenderResultSchema", () => {
  it("keeps the plain tender schema parsing the new response shape", () => {
    const response = makeWaterfallResponse({
      screening: {
        skipped: [
          { carrierName: "Lapsed Logistics", rank: 1, reasons: ["Carrier status is DoNotUse"] },
        ],
        warned: [],
      },
    });

    expect(() => tenderSchema.parse(response)).not.toThrow();
  });

  it("normalizes an absent screening summary to null", () => {
    const parsed = waterfallTenderResultSchema.parse(makeWaterfallResponse());
    expect(parsed.screening).toBeNull();
  });

  it("parses the screening summary with its reasons", () => {
    const parsed = waterfallTenderResultSchema.parse(
      makeWaterfallResponse({
        screening: {
          skipped: [
            { carrierName: "Lapsed Logistics", rank: 1, reasons: ["Blocked", "Expired policy"] },
          ],
          warned: [{ carrierName: "Expiring Express", rank: 2, reasons: ["Policy expires soon"] }],
        },
      }),
    );

    expect(parsed.screening?.skipped).toHaveLength(1);
    expect(parsed.screening?.skipped[0].reasons).toEqual(["Blocked", "Expired policy"]);
    expect(parsed.screening?.warned[0].carrierName).toBe("Expiring Express");
  });
});
