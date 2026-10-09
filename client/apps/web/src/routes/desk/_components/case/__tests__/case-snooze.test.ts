import { describe, expect, it } from "vitest";
import { snoozePresets, wallClockInputToUnix } from "../case-snooze";
import { nextWake } from "../use-snooze-clock";

// Wednesday 2026-09-16 15:00 in New York (19:00 UTC).
const NOW = Date.UTC(2026, 8, 16, 19, 0, 0) / 1000;
const ZONE = "America/New_York";

describe("snoozePresets", () => {
  it("ends on the person's own calendar, not the browser's", () => {
    const [later, tomorrow, nextWeek] = snoozePresets(NOW, ZONE);

    expect(later).toEqual({ key: "later", until: NOW + 3 * 3600 });
    expect(tomorrow.until).toBe(Date.UTC(2026, 8, 17, 12, 0, 0) / 1000);
    expect(nextWeek.until).toBe(Date.UTC(2026, 8, 21, 12, 0, 0) / 1000);
  });
});

describe("wallClockInputToUnix", () => {
  it("reads a datetime-local value in the person's zone", () => {
    expect(wallClockInputToUnix("2026-09-17T08:30", ZONE)).toBe(
      Date.UTC(2026, 8, 17, 12, 30, 0) / 1000,
    );
    expect(wallClockInputToUnix("", ZONE)).toBeNull();
    expect(wallClockInputToUnix("tomorrow", ZONE)).toBeNull();
  });
});

describe("nextWake", () => {
  it("is the soonest snooze still to end", () => {
    expect(nextWake([null, NOW - 5, NOW + 90, NOW + 30, undefined], NOW)).toBe(NOW + 30);
    expect(nextWake([NOW - 5], NOW)).toBeNull();
  });
});
