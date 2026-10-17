import { describe, expect, it } from "vitest";
import { snoozePresets, tomorrowMorning } from "../snooze-presets";

function at(iso: string): number {
  return Math.floor(Date.parse(iso) / 1000);
}

describe("snooze times", () => {
  it("lands on the person's own wall clock: five today, eight tomorrow, eight on Monday", () => {
    const wednesdayAfternoon = at("2026-10-07T19:39:00Z");
    expect(snoozePresets(wednesdayAfternoon, "America/Denver")).toEqual([
      { key: "laterToday", until: at("2026-10-07T23:00:00Z") },
      { key: "tomorrow", until: at("2026-10-08T14:00:00Z") },
      { key: "monday", until: at("2026-10-12T14:00:00Z") },
    ]);
  });

  it("reads tomorrow from the person's own date, not the server's or UTC's", () => {
    const tuesdayEveningInDenver = at("2026-10-07T02:00:00Z");
    expect(snoozePresets(tuesdayEveningInDenver, "America/Denver")).toEqual([
      { key: "tomorrow", until: at("2026-10-07T14:00:00Z") },
      { key: "monday", until: at("2026-10-12T14:00:00Z") },
    ]);
  });

  it("drops later today after five, and Monday when tomorrow is Monday", () => {
    const sundayEvening = at("2026-10-11T22:00:00Z");
    expect(snoozePresets(sundayEvening, "UTC")).toEqual([
      { key: "tomorrow", until: at("2026-10-12T08:00:00Z") },
    ]);
  });

  it("takes Monday as next week's from a Monday", () => {
    const monday = at("2026-10-12T09:00:00Z");
    expect(snoozePresets(monday, "UTC").find((preset) => preset.key === "monday")?.until).toBe(
      at("2026-10-19T08:00:00Z"),
    );
  });

  it("keeps eight in the morning across a clock change", () => {
    const beforeFallBack = at("2026-10-31T12:00:00Z");
    expect(tomorrowMorning(beforeFallBack, "Europe/London")).toBe(at("2026-11-01T08:00:00Z"));
    expect(tomorrowMorning(at("2026-10-24T12:00:00Z"), "Europe/London")).toBe(
      at("2026-10-25T08:00:00Z"),
    );
  });
});
