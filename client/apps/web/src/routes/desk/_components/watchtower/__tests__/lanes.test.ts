import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import {
  clockCaption,
  clockRemaining,
  formatCountdown,
  groupByLane,
  isHot,
  laneOf,
  minutesLeft,
  showsCountdown,
} from "../lanes";
import { watchtowerItem } from "./item-fixture";

const t = ((message: string, ...args: unknown[]) => {
  if (message.startsWith("{0, plural")) {
    return `${String(args[0])} days`;
  }
  return message.replace(/\{(\d)\}/g, (_, index: string) => String(args[Number(index)]));
}) as TranslateFn;

const NOW = 1_800_000_000;

describe("laneOf", () => {
  it("puts four hours or less, and anything past, in act now", () => {
    expect(laneOf(-51)).toBe("now");
    expect(laneOf(0)).toBe("now");
    expect(laneOf(240)).toBe("now");
  });

  it("puts the rest of the day in today and anything later in coming up", () => {
    expect(laneOf(241)).toBe("today");
    expect(laneOf(1440)).toBe("today");
    expect(laneOf(1441)).toBe("later");
  });

  it("puts an item with no deadline in for your information", () => {
    expect(laneOf(null)).toBe("fyi");
  });
});

describe("formatCountdown", () => {
  it("says it the way the design does", () => {
    expect(formatCountdown(t, 0)).toBe("Now");
    expect(formatCountdown(t, -51)).toBe("51m running");
    expect(formatCountdown(t, 45)).toBe("45m");
    expect(formatCountdown(t, 190)).toBe("3h 10m");
    expect(formatCountdown(t, 9 * 1440)).toBe("9 days");
    expect(formatCountdown(t, null)).toBeNull();
  });
});

describe("queue row countdowns", () => {
  it("shows the countdown under a day and draws it hot inside an hour", () => {
    expect(showsCountdown(1439)).toBe(true);
    expect(showsCountdown(1440)).toBe(false);
    expect(showsCountdown(null)).toBe(false);
    expect(isHot(60)).toBe(true);
    expect(isHot(61)).toBe(false);
    expect(isHot(-5)).toBe(true);
  });
});

describe("minutesLeft", () => {
  it("counts whole minutes to the deadline, negative once it has passed", () => {
    expect(minutesLeft(NOW + 190 * 60 + 30, NOW)).toBe(190);
    expect(minutesLeft(NOW - 51 * 60, NOW)).toBe(-51);
    expect(minutesLeft(null, NOW)).toBeNull();
  });
});

describe("the deadline clock", () => {
  it("says what happens and when it is still to come, or just what happens once it has", () => {
    const item = { dueLabel: "Minneapolis dock closes" };
    expect(clockCaption(t, item, 190)).toBe("Minneapolis dock closes in");
    expect(clockCaption(t, item, 0)).toBe("Minneapolis dock closes");
    expect(clockCaption(t, { dueLabel: "Detention is accruing" }, -51)).toBe(
      "Detention is accruing",
    );
    expect(clockCaption(t, { dueLabel: null }, 30)).toBe("Due in");
    expect(clockCaption(t, { dueLabel: null }, -1)).toBe("Overdue");
  });

  it("closes over the last four hours", () => {
    expect(clockRemaining(480)).toBe(1);
    expect(clockRemaining(120)).toBe(0.5);
    expect(clockRemaining(-30)).toBe(0);
  });
});

describe("groupByLane", () => {
  it("reads soonest first within a lane, leaves empty lanes out and puts undated items last", () => {
    const undated = watchtowerItem({ id: "wt_fyi", dueAt: null });
    const later = watchtowerItem({ id: "wt_later", dueAt: NOW + 9 * 86_400 });
    const overdue = watchtowerItem({ id: "wt_overdue", dueAt: NOW - 51 * 60 });
    const soon = watchtowerItem({ id: "wt_soon", dueAt: NOW + 190 * 60 });

    const lanes = groupByLane([undated, later, soon, overdue], NOW);

    expect(lanes.map((lane) => [lane.lane, lane.items.map((item) => item.id)])).toEqual([
      ["now", ["wt_overdue", "wt_soon"]],
      ["later", ["wt_later"]],
      ["fyi", ["wt_fyi"]],
    ]);
  });
});
