import { translate } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import { buildLifecycle, describeTenure, monthsBetween } from "../lifecycle-model";

const DAY = 86_400;
const sep12_2022 = Date.UTC(2022, 8, 12, 12) / 1000;
const oct3_2022 = Date.UTC(2022, 9, 3, 12) / 1000;
const nov22_2026 = Date.UTC(2026, 10, 22, 12) / 1000;

const hired = { id: "wee_hire", kind: "Hired", effectiveAt: sep12_2022 };
const onboarding = {
  id: "wck_on",
  kind: "Onboarding",
  status: "Completed",
  startedAt: sep12_2022,
  completedAt: oct3_2022,
  sourceEventId: "wee_hire",
};

describe("buildLifecycle", () => {
  it("draws an active worker: onboarding done with its span, active current with tenure", () => {
    const stops = buildLifecycle(
      { events: [hired], checklists: [onboarding], hireDate: sep12_2022, now: nov22_2026 },
      translate,
    );
    expect(stops.map((stop) => [stop.stage, stop.state])).toEqual([
      ["onboarding", "done"],
      ["active", "current"],
      ["offboarding", "ahead"],
      ["left", "ahead"],
    ]);
    expect(stops[0].meta).toBe("Sep 12, 2022 to Oct 3, 2022 · 21 days");
    expect(stops[1].meta).toBe("since Oct 3, 2022 · 4 years 1 month");
    expect(stops[3].meta).toBe("Still employed");
  });

  it("marks a suspended worker's active stop as held up", () => {
    const suspended = { id: "wee_s", kind: "Suspended", effectiveAt: nov22_2026 - 5 * DAY };
    const stops = buildLifecycle(
      {
        events: [hired, suspended],
        checklists: [onboarding],
        hireDate: sep12_2022,
        now: nov22_2026,
      },
      translate,
    );
    expect(stops[1].danger).toBe(true);
    expect(stops[1].meta).toContain("suspended from Nov 17, 2026");
  });

  it("puts a terminated worker in offboarding while its checklist is open", () => {
    const terminated = { id: "wee_t", kind: "Terminated", effectiveAt: nov22_2026 };
    const offboarding = {
      id: "wck_off",
      kind: "Offboarding",
      status: "Open",
      startedAt: nov22_2026,
      completedAt: null,
      sourceEventId: "wee_t",
    };
    const stops = buildLifecycle(
      {
        events: [hired, terminated],
        checklists: [onboarding, offboarding],
        hireDate: sep12_2022,
        now: nov22_2026,
      },
      translate,
    );
    expect(stops.map((stop) => stop.state)).toEqual(["done", "done", "current", "ahead"]);
    expect(stops[2].danger).toBe(true);
    expect(stops[1].meta).toBe("Oct 3, 2022 to Nov 22, 2026");
    expect(stops[3].meta).toBe("When offboarding settles");
  });

  it("reads the hire date on file when the history has no hire event", () => {
    const stops = buildLifecycle(
      { events: [], checklists: [], hireDate: sep12_2022, now: nov22_2026 },
      translate,
    );
    expect(stops[0]).toMatchObject({ state: "done", meta: "Hired Sep 12, 2022" });
    expect(stops[1]).toMatchObject({
      state: "current",
      meta: "since Sep 12, 2022 · 4 years 2 months",
    });
  });
});

describe("tenure", () => {
  it("counts a month only once its day has come round", () => {
    expect(monthsBetween(sep12_2022, Date.UTC(2022, 9, 11, 12) / 1000)).toBe(0);
    expect(monthsBetween(sep12_2022, Date.UTC(2022, 9, 12, 12) / 1000)).toBe(1);
  });

  it("words years and months with plurals", () => {
    expect(describeTenure(0, translate)).toBe("Under a month");
    expect(describeTenure(1, translate)).toBe("1 month");
    expect(describeTenure(12, translate)).toBe("1 year");
    expect(describeTenure(50, translate)).toBe("4 years 2 months");
  });
});
