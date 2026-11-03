import { describe, expect, it } from "vitest";
import { createTimelineScale } from "../timeline-scale";

const TUE = Date.UTC(2026, 10, 3) / 1000;
const DAY = 86_400;

describe("createTimelineScale", () => {
  it("lays a day out at its zoom when the sheet has less room than that", () => {
    const scale = createTimelineScale("day", [TUE, TUE + DAY], 900);

    expect(scale.width).toBe(24 * 64);
    expect(scale.x(TUE + 3_600)).toBe(64);
  });

  it("stretches a range to fill a sheet wider than it, so nothing past its end is blank", () => {
    const scale = createTimelineScale("day", [TUE, TUE + DAY], 2_400);

    expect(scale.width).toBe(2_400);
    expect(scale.x(TUE + DAY)).toBe(2_400);
    expect(scale.x(TUE + DAY / 2)).toBe(1_200);
  });
});
