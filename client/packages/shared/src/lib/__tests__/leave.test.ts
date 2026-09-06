import { describe, expect, it } from "vitest";
import {
  certificationOutstanding,
  certificationTone,
  entitlementUsedPercent,
  formatLeaveHours,
  leaveCaseStatusTone,
  measurementMethodLabel,
} from "../leave";

describe("certificationOutstanding", () => {
  // Received and waived are settled; the rest are still owed.
  it("covers every state the office is still waiting on", () => {
    expect(certificationOutstanding("Requested")).toBe(true);
    expect(certificationOutstanding("Insufficient")).toBe(true);
    expect(certificationOutstanding("Overdue")).toBe(true);
    expect(certificationOutstanding("Received")).toBe(false);
    expect(certificationOutstanding("Waived")).toBe(false);
    expect(certificationOutstanding("NotRequired")).toBe(false);
  });
});

describe("entitlementUsedPercent", () => {
  it("is the share of the entitlement used", () => {
    expect(entitlementUsedPercent("120", "480")).toBe(25);
    expect(entitlementUsedPercent("0", "480")).toBe(0);
    expect(entitlementUsedPercent("480", "480")).toBe(100);
  });

  // Designating leave after the fact can push usage past the entitlement. A bar
  // running past its own end reads as a rendering fault, not as a fact.
  it("clamps over-use to a full bar", () => {
    expect(entitlementUsedPercent("560", "480")).toBe(100);
  });

  it("is zero when there is no entitlement to divide by", () => {
    expect(entitlementUsedPercent("40", "0")).toBe(0);
    expect(entitlementUsedPercent("40", "unknown")).toBe(0);
  });
});

describe("formatLeaveHours", () => {
  // Intermittent leave is taken in the smallest increment the employer uses, so
  // a quarter-hour must survive the trip.
  it("keeps a fractional hour", () => {
    expect(formatLeaveHours("2.25")).toBe("2.25");
    expect(formatLeaveHours("480")).toBe("480");
    expect(formatLeaveHours("1200.5")).toBe("1,200.5");
  });

  it("passes a non-numeric value through untouched", () => {
    expect(formatLeaveHours("unknown")).toBe("unknown");
  });
});

describe("labels and tones", () => {
  it("names each measurement method", () => {
    expect(measurementMethodLabel("RollingBackward")).toBe("Rolling twelve months looking back");
    expect(measurementMethodLabel("SomethingNew")).toBe("SomethingNew");
  });

  it("grades case and certification states", () => {
    expect(leaveCaseStatusTone("Approved")).toBe("active");
    expect(leaveCaseStatusTone("Denied")).toBe("inactive");
    expect(leaveCaseStatusTone("Pending")).toBe("warning");
    expect(certificationTone("Overdue")).toBe("inactive");
    expect(certificationTone("Received")).toBe("active");
  });
});
