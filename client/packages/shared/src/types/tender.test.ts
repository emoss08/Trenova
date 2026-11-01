import { describe, expect, it } from "vitest";
import {
  bounceTenderPayloadSchema,
  emptySpotTenderLine,
  publicTenderCounterPayloadSchema,
  publicTenderOfferSchema,
  recordTenderResponsePayloadSchema,
  spotTenderPayloadSchema,
  tenderSchema,
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

/** A waterfall tender as the server returns it. */
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

const offerResponse = (overrides: Record<string, unknown> = {}) => ({
  id: "tof_01",
  tenderId: "ten_01",
  carrierId: "car_01",
  rank: 1,
  rateMethod: "Flat",
  rate: "1500",
  offerTtlSeconds: 7200,
  channel: "Email",
  status: "Sent",
  ...overrides,
});

describe("tenderSchema bounce and counter fields", () => {
  it("defaults the bounce count, escalation round and approval flag on a tender that predates them", () => {
    const parsed = tenderSchema.parse(makeWaterfallResponse({ offers: [offerResponse()] }));

    expect(parsed.bounceCount).toBe(0);
    expect(parsed.escalationRound).toBe(0);
    expect(parsed.offers?.[0]?.counterNeedsApproval).toBe(false);
    expect(parsed.offers?.[0]?.counterRate ?? null).toBeNull();
  });

  it("reads a countered offer, a revoked one and an escalated tender as the server writes them", () => {
    const parsed = tenderSchema.parse(
      makeWaterfallResponse({
        bounceCount: 1,
        escalatedFromId: "ten_00",
        escalationRound: 2,
        offers: [
          offerResponse({
            status: "Countered",
            counterRate: "1850.5",
            counterNote: "Fuel is up",
            counterMarginPercent: "7.5",
            counterNeedsApproval: true,
          }),
          offerResponse({
            id: "tof_02",
            status: "Revoked",
            revokedAt: 1_720_000_000,
            revokeReason: "Truck broke down",
          }),
          offerResponse({ id: "tof_03", status: "CounterRejected", counterRate: "2000" }),
        ],
      }),
    );

    expect(parsed.escalationRound).toBe(2);
    expect(parsed.escalatedFromId).toBe("ten_00");
    expect(parsed.offers?.[0]).toMatchObject({
      status: "Countered",
      counterRate: 1850.5,
      counterMarginPercent: 7.5,
      counterNeedsApproval: true,
    });
    expect(parsed.offers?.[1]).toMatchObject({
      status: "Revoked",
      revokeReason: "Truck broke down",
    });
    expect(parsed.offers?.[2]?.status).toBe("CounterRejected");
  });
});

describe("recordTenderResponsePayloadSchema", () => {
  it("requires a positive counter rate below ten million only when the answer is a counter", () => {
    expect(recordTenderResponsePayloadSchema.safeParse({ action: "Accept" }).success).toBe(true);

    const missing = recordTenderResponsePayloadSchema.safeParse({ action: "Counter" });
    expect(missing.success).toBe(false);
    expect(missing.error?.issues.map((issue) => [issue.path.join("."), issue.message])).toEqual([
      ["counterRate", "Counter rate is required"],
    ]);

    expect(
      recordTenderResponsePayloadSchema.safeParse({ action: "Counter", counterRate: "9999999.99" })
        .success,
    ).toBe(true);
  });

  it("caps the counter note at 500 characters", () => {
    const result = recordTenderResponsePayloadSchema.safeParse({
      action: "Counter",
      counterRate: "1850",
      counterNote: "x".repeat(501),
    });
    expect(result.error?.issues[0]?.path).toEqual(["counterNote"]);
  });
});

describe("bounceTenderPayloadSchema", () => {
  it("requires a reason that is more than whitespace and trims it", () => {
    expect(bounceTenderPayloadSchema.safeParse({ reason: "   " }).success).toBe(false);
    expect(bounceTenderPayloadSchema.parse({ reason: "  Truck broke down " })).toEqual({
      reason: "Truck broke down",
    });
  });

  it("caps the reason at 500 characters", () => {
    expect(bounceTenderPayloadSchema.safeParse({ reason: "x".repeat(501) }).success).toBe(false);
  });
});

describe("public tender offer counter", () => {
  it("reads an offer page from a server that predates counters as not taking them", () => {
    const parsed = publicTenderOfferSchema.parse({
      carrierName: "Blue Ridge Freight",
      shipmentProNumber: "PRO-1",
      originSummary: "Dallas, TX",
      destinationSummary: "Tulsa, OK",
      pickupWindow: "",
      deliveryWindow: "",
      equipmentSummary: "",
      weightSummary: "",
      rateAmount: "$1,850.00",
      rateMethodLabel: "Flat Rate",
      expiresAt: 0,
      responded: false,
    });

    expect(parsed.counterAllowed).toBe(false);
    expect(parsed.rateValue).toBe("");
  });

  it("trims the rate and keeps the note for the counter request", () => {
    expect(publicTenderCounterPayloadSchema.parse({ rate: " 2100 ", note: "Fuel" })).toEqual({
      rate: "2100",
      note: "Fuel",
    });
    expect(publicTenderCounterPayloadSchema.safeParse({ rate: "abc" }).success).toBe(false);
  });
});
