import { describe, expect, it } from "vitest";
import {
  credentialHealthMeta,
  describeDaysUntil,
  shortDaysUntil,
  suggestExpiryUnix,
} from "../credential";

describe("credentialHealthMeta", () => {
  it("ranks the worst health first and maps every health to a badge variant", () => {
    expect(credentialHealthMeta("Missing").rank).toBeLessThan(credentialHealthMeta("Expired").rank);
    expect(credentialHealthMeta("Expired").rank).toBeLessThan(
      credentialHealthMeta("ExpiringSoon").rank,
    );
    expect(credentialHealthMeta("ExpiringSoon").rank).toBeLessThan(
      credentialHealthMeta("Valid").rank,
    );
    expect(credentialHealthMeta("Valid").badgeVariant).toBe("active");
    expect(credentialHealthMeta("ExpiringSoon").badgeVariant).toBe("warning");
    expect(credentialHealthMeta("Expired").badgeVariant).toBe("inactive");
    expect(credentialHealthMeta("Missing").badgeVariant).toBe("outline");
  });
});

describe("describeDaysUntil", () => {
  it("phrases whole-day distances the way the server counts them", () => {
    expect(describeDaysUntil(null)).toBe("No expiry");
    expect(describeDaysUntil(undefined)).toBe("No expiry");
    expect(describeDaysUntil(0)).toBe("Expires today");
    expect(describeDaysUntil(1)).toBe("Expires tomorrow");
    expect(describeDaysUntil(14)).toBe("14 days left");
    expect(describeDaysUntil(-1)).toBe("Expired yesterday");
    expect(describeDaysUntil(-9)).toBe("Expired 9 days ago");
  });

  it("has a compact form for chips", () => {
    expect(shortDaysUntil(null)).toBe("—");
    expect(shortDaysUntil(0)).toBe("Today");
    expect(shortDaysUntil(7)).toBe("7d");
    expect(shortDaysUntil(-3)).toBe("-3d");
  });
});

describe("suggestExpiryUnix", () => {
  const jan31 = Date.UTC(2026, 0, 31) / 1000;

  it("adds whole months and clamps to the last day of a shorter month", () => {
    expect(suggestExpiryUnix(jan31, 1)).toBe(Date.UTC(2026, 1, 28) / 1000);
    expect(suggestExpiryUnix(jan31, 24)).toBe(Date.UTC(2028, 0, 31) / 1000);
  });

  it("returns null without a usable validity", () => {
    expect(suggestExpiryUnix(jan31, null)).toBeNull();
    expect(suggestExpiryUnix(jan31, 0)).toBeNull();
    expect(suggestExpiryUnix(null, 12)).toBeNull();
  });
});
