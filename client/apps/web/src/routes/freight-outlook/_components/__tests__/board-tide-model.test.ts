import { describe, expect, it } from "vitest";
import {
  BAR_MAX,
  DRIVERS_LANE,
  LOADS_LANE,
  barHeights,
  columnStep,
  laneScale,
  laneTop,
  nowLabelAnchor,
} from "../board-tide-model";

describe("laneTop", () => {
  it("is the first round number at or above the highest value", () => {
    expect(laneTop(150)).toBe(150);
    expect(laneTop(147)).toBe(150);
    expect(laneTop(28)).toBe(30);
    expect(laneTop(12)).toBe(15);
    expect(laneTop(58)).toBe(60);
    expect(laneTop(42)).toBe(50);
  });

  it("keeps whole steps for a handful of loads, and a scale to draw on for none", () => {
    expect(laneTop(5)).toBe(5);
    expect(laneTop(2)).toBe(2);
    expect(laneTop(2.4)).toBe(3);
    expect(laneTop(0)).toBe(1);
  });
});

describe("laneScale", () => {
  it("puts zero on the lane's bottom row and the top of its scale on its top row", () => {
    const at = laneScale(LOADS_LANE, 6);

    expect(at(0)).toBe(LOADS_LANE.bottom);
    expect(at(6)).toBe(LOADS_LANE.top);
    expect(LOADS_LANE.bottom - LOADS_LANE.top).toBeGreaterThan(
      DRIVERS_LANE.bottom - DRIVERS_LANE.top,
    );
    expect(at(3)).toBe((LOADS_LANE.top + LOADS_LANE.bottom) / 2);
  });

  it("never draws outside its lane", () => {
    const at = laneScale(DRIVERS_LANE, 150);

    expect(at(400)).toBe(DRIVERS_LANE.top);
    expect(at(-3)).toBe(DRIVERS_LANE.bottom);
  });

  it("keeps the two lanes clear of each other", () => {
    expect(DRIVERS_LANE.top - LOADS_LANE.bottom).toBeGreaterThanOrEqual(14);
  });
});

describe("barHeights", () => {
  it("scales a lane to its own largest count", () => {
    const idle = barHeights([126, 63, 0]);

    expect(idle(126)).toBe(BAR_MAX);
    expect(idle(63)).toBe(BAR_MAX / 2);
  });

  it("draws one load as one load, not as a full bar", () => {
    const short = barHeights([1, 0, 0]);

    expect(short(1)).toBe(BAR_MAX / 4);
    expect(short(4)).toBe(BAR_MAX);
  });

  it("keeps a small count visible beside a large one, and draws nothing for none", () => {
    const idle = barHeights([400, 3]);

    expect(idle(3)).toBe(2);
    expect(idle(0.4)).toBe(0);
    expect(idle(0)).toBe(0);
  });
});

describe("nowLabelAnchor", () => {
  it("sits after its line until that would run past the plot", () => {
    expect(nowLabelAnchor(280, 1200)).toBe("start");
    expect(nowLabelAnchor(1116, 1200)).toBe("start");
    expect(nowLabelAnchor(1117, 1200)).toBe("end");
    expect(nowLabelAnchor(1200, 1200)).toBe("end");
  });
});

describe("columnStep", () => {
  it("shares what is left between the label and value columns", () => {
    expect(columnStep({ width: 1588, label: 280, value: 108, count: 12 })).toBe(100);
    expect(columnStep({ width: 200, label: 280, value: 108, count: 12 })).toBe(0);
    expect(columnStep({ width: 1588, label: 280, value: 108, count: 0 })).toBe(1200);
  });
});
