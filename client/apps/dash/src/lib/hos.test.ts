import { describe, expect, it } from "vitest";

import { HOUR_MS, dutyStatusInfo, gaugeTone, timeAgo, toDateKey } from "./hos";

describe("toDateKey", () => {
  it("formats with zero padding", () => {
    expect(toDateKey(new Date(2026, 0, 5))).toBe("2026-01-05");
    expect(toDateKey(new Date(2026, 10, 28))).toBe("2026-11-28");
  });
});

describe("timeAgo", () => {
  const nowMs = 1_700_000_000_000;
  const nowSeconds = nowMs / 1000;

  it("reads just now under a minute", () => {
    expect(timeAgo(nowSeconds, nowMs)).toBe("just now");
    expect(timeAgo(nowSeconds - 59, nowMs)).toBe("just now");
  });

  it("buckets minutes, hours, and days", () => {
    expect(timeAgo(nowSeconds - 60, nowMs)).toBe("1m ago");
    expect(timeAgo(nowSeconds - 59 * 60, nowMs)).toBe("59m ago");
    expect(timeAgo(nowSeconds - 60 * 60, nowMs)).toBe("1h ago");
    expect(timeAgo(nowSeconds - 23 * 60 * 60, nowMs)).toBe("23h ago");
    expect(timeAgo(nowSeconds - 24 * 60 * 60, nowMs)).toBe("1d ago");
    expect(timeAgo(nowSeconds - 90 * 24 * 60 * 60, nowMs)).toBe("90d ago");
  });

  it("never goes negative for future timestamps", () => {
    expect(timeAgo(nowSeconds + 600, nowMs)).toBe("just now");
  });
});

describe("gaugeTone", () => {
  it("goes critical under one hour", () => {
    expect(gaugeTone(0, "brand")).toBe("critical");
    expect(gaugeTone(HOUR_MS - 1, "brand")).toBe("critical");
  });

  it("warns under two hours", () => {
    expect(gaugeTone(HOUR_MS, "brand")).toBe("warning");
    expect(gaugeTone(2 * HOUR_MS - 1, "brand")).toBe("warning");
  });

  it("keeps the default tone with two or more hours left", () => {
    expect(gaugeTone(2 * HOUR_MS, "brand")).toBe("brand");
    expect(gaugeTone(11 * HOUR_MS, "warning")).toBe("warning");
  });
});

describe("dutyStatusInfo", () => {
  it("maps known duty statuses", () => {
    expect(dutyStatusInfo("driving")).toEqual({ label: "Driving", variant: "info" });
    expect(dutyStatusInfo("sleeperBed")).toEqual({ label: "Sleeper Berth", variant: "purple" });
    expect(dutyStatusInfo("onDuty")).toEqual({ label: "On Duty", variant: "warning" });
  });

  it("title-cases unknown statuses", () => {
    const info = dutyStatusInfo("waitingAtShipper");
    expect(info.variant).toBe("secondary");
    expect(info.label.toLowerCase()).toContain("waiting");
  });

  it("falls back to Unknown when missing", () => {
    expect(dutyStatusInfo(null)).toEqual({ label: "Unknown", variant: "secondary" });
    expect(dutyStatusInfo(undefined)).toEqual({ label: "Unknown", variant: "secondary" });
    expect(dutyStatusInfo("")).toEqual({ label: "Unknown", variant: "secondary" });
  });
});
