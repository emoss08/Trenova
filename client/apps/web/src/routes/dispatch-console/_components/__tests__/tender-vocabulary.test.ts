import { describe, expect, it } from "vitest";
import {
  formatOfferCountdown,
  formatOfferRate,
  tenderChipMeta,
  type TenderChipSummary,
} from "../tender-vocabulary";

const baseSummary: TenderChipSummary = {
  status: "Active",
  mode: "Waterfall",
  currentRank: 2,
  offerCount: 5,
  currentCarrierName: "Knight Swift",
  currentOfferExpiresAt: 1_700_000_000,
};

describe("tenderChipMeta", () => {
  it("leads an active waterfall with the ladder position and current carrier", () => {
    expect(tenderChipMeta(baseSummary)).toEqual({
      label: "Waterfall 2/5 · Knight Swift",
      tone: "info",
    });
  });

  it("omits the position when the summary carries no offers", () => {
    expect(tenderChipMeta({ ...baseSummary, offerCount: 0, currentCarrierName: "" }).label).toBe(
      "Waterfall",
    );
  });

  it("uses the spot labels for spot modes", () => {
    expect(tenderChipMeta({ ...baseSummary, mode: "SpotBroadcast" }).label).toContain(
      "Spot broadcast",
    );
    expect(tenderChipMeta({ ...baseSummary, mode: "SpotSequential" }).label).toContain(
      "Spot sequential",
    );
  });

  it("marks the failure states as attention", () => {
    expect(tenderChipMeta({ ...baseSummary, status: "NeedsReview" })).toEqual({
      label: "Tender needs review",
      tone: "attention",
    });
    expect(tenderChipMeta({ ...baseSummary, status: "Exhausted" })).toEqual({
      label: "Tender exhausted",
      tone: "attention",
    });
  });

  it("marks an accepted tender as active", () => {
    expect(tenderChipMeta({ ...baseSummary, status: "Accepted" }).tone).toBe("active");
  });
});

describe("formatOfferRate", () => {
  it("formats a flat rate as currency", () => {
    expect(formatOfferRate("1500.00", "Flat")).toBe("$1,500.00");
  });

  it("appends the per-mile qualifier for distance-based rates", () => {
    expect(formatOfferRate("2.5", "PerMile")).toBe("$2.50/mi");
  });

  it("renders an unparseable rate verbatim rather than as NaN", () => {
    expect(formatOfferRate("n/a", "Flat")).toBe("n/a");
  });
});

describe("formatOfferCountdown", () => {
  it("renders nothing for an unknown expiry", () => {
    expect(formatOfferCountdown(null, 1000)).toBe("");
    expect(formatOfferCountdown(undefined, 1000)).toBe("");
    expect(formatOfferCountdown(0, 1000)).toBe("");
  });

  it("renders a compact remaining duration", () => {
    expect(formatOfferCountdown(1000 + 3900, 1000)).toBe("1h 5m left");
    expect(formatOfferCountdown(1000 + 120, 1000)).toBe("2m left");
  });

  it("reads as expired once the deadline elapses", () => {
    expect(formatOfferCountdown(1000, 1000)).toBe("expired");
    expect(formatOfferCountdown(500, 1000)).toBe("expired");
  });
});
