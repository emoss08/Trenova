import { describe, expect, it } from "vitest";
import { createTimelineScale, laneHeight, ROW_HEIGHT } from "../timeline-scale";

const TUE = Date.UTC(2026, 10, 3) / 1000;
const DAY = 86_400;
const DAY_ZOOM = { pxPerHour: 64, tickHours: 2, hourLabels: true };

describe("createTimelineScale", () => {
  it("lays a day out at its zoom when the sheet has less room than that", () => {
    const scale = createTimelineScale([TUE, TUE + DAY], DAY_ZOOM, 900);

    expect(scale.width).toBe(24 * 64);
    expect(scale.x(TUE + 3_600)).toBe(64);
  });

  it("stretches a range to fill a sheet wider than it, so nothing past its end is blank", () => {
    const scale = createTimelineScale([TUE, TUE + DAY], DAY_ZOOM, 2_400);

    expect(scale.width).toBe(2_400);
    expect(scale.x(TUE + DAY)).toBe(2_400);
    expect(scale.x(TUE + DAY / 2)).toBe(1_200);
  });

  it("knows a one-day range from a longer one, so times inside it can drop the weekday", () => {
    expect(createTimelineScale([TUE, TUE + DAY], DAY_ZOOM).singleDay).toBe(true);
    expect(createTimelineScale([TUE, TUE + DAY, TUE + 2 * DAY], DAY_ZOOM).singleDay).toBe(false);
  });
});

describe("laneHeight", () => {
  it("keeps an empty or one-row lane as tall as a one-bar row and grows by stacked rows", () => {
    expect(laneHeight(1)).toBe(ROW_HEIGHT);
    expect(laneHeight(3)).toBe(7 * 2 + 3 * 26 - 6);
  });
});
