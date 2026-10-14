import { describe, expect, it } from "vitest";
import {
  factoringNoaDisplayStatus,
  factoringNoaPhase,
  factoringVerdictPhase,
  formatBasisPoints,
  formatMinutesOfDay,
  parseMinutesOfDay,
  percentToBasisPoints,
  readyOverADay,
} from "@/lib/factoring";

const DAY = 24 * 60 * 60;

describe("factoring rates", () => {
  it("reads basis points as the percentage the contract states", () => {
    expect(formatBasisPoints(8500)).toBe("85%");
    expect(formatBasisPoints(275)).toBe("2.75%");
    expect(formatBasisPoints(0)).toBe("0%");
  });

  it("turns a typed percentage into whole basis points without float drift", () => {
    expect(percentToBasisPoints(2.75)).toBe(275);
    expect(percentToBasisPoints(0.29)).toBe(29);
    expect(percentToBasisPoints(100)).toBe(10_000);
  });
});

describe("factoring submission time", () => {
  it("round-trips minutes after midnight through a 24-hour clock time", () => {
    expect(formatMinutesOfDay(0)).toBe("00:00");
    expect(formatMinutesOfDay(570)).toBe("09:30");
    expect(formatMinutesOfDay(1439)).toBe("23:59");
    expect(parseMinutesOfDay("09:30")).toBe(570);
    expect(parseMinutesOfDay("23:59")).toBe(1439);
  });

  it("refuses a value that is not a time of day", () => {
    expect(parseMinutesOfDay("24:00")).toBeNull();
    expect(parseMinutesOfDay("9:75")).toBeNull();
    expect(parseMinutesOfDay("")).toBeNull();
  });
});

describe("notice of assignment status", () => {
  const now = 1_800_000_000;

  it("reads an acknowledged notice whose signed copy has lapsed as expired", () => {
    expect(factoringNoaDisplayStatus({ status: "Acknowledged", expiresAt: now }, now)).toBe(
      "Expired",
    );
    expect(factoringNoaPhase("Expired")).toBe("failed");
  });

  it("keeps an acknowledged notice in force until it expires, or forever without an expiry", () => {
    expect(factoringNoaDisplayStatus({ status: "Acknowledged", expiresAt: now + 1 }, now)).toBe(
      "Acknowledged",
    );
    expect(factoringNoaDisplayStatus({ status: "Acknowledged", expiresAt: null }, now)).toBe(
      "Acknowledged",
    );
  });

  it("never reads an expiry onto a notice that was not acknowledged", () => {
    expect(factoringNoaDisplayStatus({ status: "Sent", expiresAt: now - 1 }, now)).toBe("Sent");
    expect(factoringNoaDisplayStatus({ status: "Released", expiresAt: now - 1 }, now)).toBe(
      "Released",
    );
  });

  it("puts a sent notice in front of a person and a released one out of the way", () => {
    expect(factoringNoaPhase("Sent")).toBe("awaiting");
    expect(factoringNoaPhase("Acknowledged")).toBe("complete");
    expect(factoringNoaPhase("Released")).toBe("closed");
  });
});

describe("factoring readiness", () => {
  it("colours each verdict by what the biller does next", () => {
    expect(factoringVerdictPhase("Ready")).toBe("complete");
    expect(factoringVerdictPhase("NotReady")).toBe("attention");
    expect(factoringVerdictPhase("Excluded")).toBe("closed");
  });

  it("counts an invoice as ready over a day from the full day on, and never when not ready", () => {
    const now = 1_800_000_000;
    expect(readyOverADay(now - DAY, now)).toBe(true);
    expect(readyOverADay(now - DAY + 1, now)).toBe(false);
    expect(readyOverADay(null, now)).toBe(false);
  });
});
