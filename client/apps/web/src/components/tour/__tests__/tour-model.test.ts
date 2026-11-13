import { describe, expect, it } from "vitest";
import {
  isTourSeen,
  nextStepIndex,
  tourTarget,
  tourTargetSelector,
  type TourStep,
} from "../tour-model";

const tour = { key: "freight-outlook", version: 2 };

describe("isTourSeen", () => {
  it("counts a view of this version or a later one", () => {
    expect(isTourSeen([{ tour: "freight-outlook", tourVersion: 2 }], tour)).toBe(true);
    expect(isTourSeen([{ tour: "freight-outlook", tourVersion: 5 }], tour)).toBe(true);
  });

  it("shows a tour again once its steps have moved on", () => {
    expect(isTourSeen([{ tour: "freight-outlook", tourVersion: 1 }], tour)).toBe(false);
  });

  it("never counts another page's tour, or nothing at all", () => {
    expect(isTourSeen([{ tour: "console", tourVersion: 9 }], tour)).toBe(false);
    expect(isTourSeen([], tour)).toBe(false);
  });
});

describe("nextStepIndex", () => {
  const steps: TourStep[] = [
    { title: "Welcome", body: "" },
    { target: "figures", title: "Figures", body: "" },
    { target: "moves", title: "Moves", body: "" },
    { target: "pace", title: "Pace", body: "" },
  ];
  const all = () => true;
  const withoutMoves = (target: string) => target !== "moves";

  it("walks forward and back one step at a time", () => {
    expect(nextStepIndex(steps, 0, 1, all)).toBe(1);
    expect(nextStepIndex(steps, 2, -1, all)).toBe(1);
  });

  it("passes over a part that is not on the page, in both directions", () => {
    expect(nextStepIndex(steps, 1, 1, withoutMoves)).toBe(3);
    expect(nextStepIndex(steps, 3, -1, withoutMoves)).toBe(1);
  });

  it("has nowhere to go past either end", () => {
    expect(nextStepIndex(steps, 3, 1, all)).toBeNull();
    expect(nextStepIndex(steps, 0, -1, all)).toBeNull();
    expect(nextStepIndex(steps, 1, 1, () => false)).toBeNull();
  });

  it("always reaches a step that points at nothing", () => {
    expect(nextStepIndex(steps, 1, -1, () => false)).toBe(0);
  });
});

describe("tourTarget", () => {
  it("marks an element the way the selector finds it", () => {
    expect(tourTarget("figures")).toEqual({ "data-tour": "figures" });
    expect(tourTargetSelector("figures")).toBe('[data-tour="figures"]');
  });
});
