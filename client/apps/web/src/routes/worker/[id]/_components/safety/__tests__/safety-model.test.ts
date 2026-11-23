import type { ActivePointEvent, WorkerSafetyEventRow } from "@/lib/graphql/worker-safety";
import { translate } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import {
  actionSummary,
  disciplineStair,
  eventAmountMinor,
  filterEvents,
  formatEventAmount,
  pointsGauge,
  sortEvents,
} from "../safety-model";

const t = ((text: string, ...args: unknown[]) =>
  args.reduce<string>((out, arg, index) => out.replace(`{${index}}`, String(arg)), text)) as never;

function event(overrides: Partial<WorkerSafetyEventRow>): WorkerSafetyEventRow {
  return {
    id: "wsev",
    status: "Closed",
    occurredAt: 1_700_000_000,
    activePoints: 0,
    points: 0,
    pointsExpireAt: null,
    kind: "Incident",
    ...overrides,
  } as WorkerSafetyEventRow;
}

const thresholds = { pointsWatchThreshold: 6, pointsAtRiskThreshold: 10 };

/** One event the scorecard says still carries points, as `activePointEvents` reports it. */
function carrying(overrides: Partial<ActivePointEvent>): ActivePointEvent {
  return {
    id: "wsev",
    kind: "Incident",
    severity: "Minor",
    preventable: false,
    inspectionResult: null,
    inspectionLevel: null,
    occurredAt: 1_700_000_000,
    activePoints: 1,
    pointsExpireAt: null,
    ...overrides,
  } as ActivePointEvent;
}

describe("pointsGauge", () => {
  it("runs the track two past the at-risk line and fills one segment per event carrying points, oldest first", () => {
    const gauge = pointsGauge({
      ...thresholds,
      activePoints: 7,
      activePointEvents: [
        carrying({ id: "inspection", occurredAt: 300, activePoints: 4, pointsExpireAt: 3_000 }),
        carrying({ id: "incident", occurredAt: 200, activePoints: 3, pointsExpireAt: 2_000 }),
      ],
    });

    expect(gauge.max).toBe(12);
    expect(gauge.tone).toBe("watch");
    expect(gauge.fill).toBeCloseTo(7 / 12);
    expect(gauge.segments.map((segment) => [segment.id, segment.points])).toEqual([
      ["incident", 3],
      ["inspection", 4],
    ]);
    expect(gauge.segments[0].share).toBeCloseTo(3 / 7);
    expect(gauge.rollOffs).toEqual([
      { date: 3_000, points: 4 },
      { date: 2_000, points: 3 },
    ]);
  });

  it("adds up points that roll off on the same day", () => {
    const gauge = pointsGauge({
      ...thresholds,
      activePoints: 5,
      activePointEvents: [
        carrying({ id: "a", activePoints: 2, pointsExpireAt: 9_000 }),
        carrying({ id: "b", activePoints: 3, pointsExpireAt: 9_000 }),
      ],
    });
    expect(gauge.rollOffs).toEqual([{ date: 9_000, points: 5 }]);
  });

  it("names every event the scorecard counts, however few of the events list has loaded", () => {
    const gauge = pointsGauge({
      ...thresholds,
      activePoints: 9,
      activePointEvents: [
        carrying({ id: "old", occurredAt: 100, activePoints: 6, pointsExpireAt: 4_000 }),
        carrying({ id: "recent", occurredAt: 500, activePoints: 3, pointsExpireAt: 5_000 }),
      ],
    });
    expect(gauge.segments.map((segment) => [segment.id, segment.share])).toEqual([
      ["old", 6 / 9],
      ["recent", 3 / 9],
    ]);
    expect(gauge.rollOffs).toEqual([
      { date: 5_000, points: 3 },
      { date: 4_000, points: 6 },
    ]);
  });

  it("widens the track past the points when they pass it, and turns to at risk", () => {
    const gauge = pointsGauge({ ...thresholds, activePoints: 15, activePointEvents: [] });
    expect(gauge.max).toBe(15);
    expect(gauge.fill).toBe(1);
    expect(gauge.tone).toBe("risk");
  });

  it("is calm and empty with no active points", () => {
    const gauge = pointsGauge({
      ...thresholds,
      activePoints: 0,
      activePointEvents: [carrying({ activePoints: 0 })],
    });
    expect(gauge.tone).toBe("calm");
    expect(gauge.segments).toEqual([]);
    expect(gauge.rollOffs).toEqual([]);
  });
});

describe("disciplineStair", () => {
  it("fills every step up to the active one and marks the suggested next step", () => {
    const steps = disciplineStair(
      { highestLevel: "VerbalWarning", suggestedLevel: "WrittenWarning" },
      null,
    );
    expect(steps.map((step) => [step.level, step.reached, step.active, step.next])).toEqual([
      ["Coaching", true, false, false],
      ["VerbalWarning", true, true, false],
      ["WrittenWarning", false, false, true],
      ["FinalWarning", false, false, false],
      ["Suspension", false, false, false],
      ["Termination", false, false, false],
    ]);
  });

  it("reaches nothing without an active action and previews the level being issued", () => {
    const steps = disciplineStair({ highestLevel: null, suggestedLevel: "Coaching" }, "Suspension");
    expect(steps.filter((step) => step.reached)).toEqual([]);
    expect(steps.find((step) => step.next)?.level).toBe("Coaching");
    expect(steps.find((step) => step.preview)?.level).toBe("Suspension");
  });

  it("marks no next step at the top of the stair", () => {
    const steps = disciplineStair(
      { highestLevel: "Termination", suggestedLevel: "Termination" },
      null,
    );
    expect(steps.some((step) => step.next)).toBe(false);
    expect(steps.every((step) => step.reached)).toBe(true);
  });
});

describe("events", () => {
  it("puts open and under-review events first, newest first within each", () => {
    const sorted = sortEvents([
      event({ id: "closed-new", status: "Closed", occurredAt: 900 }),
      event({ id: "open-old", status: "Open", occurredAt: 100 }),
      event({ id: "review-new", status: "UnderReview", occurredAt: 800 }),
      event({ id: "closed-old", status: "Closed", occurredAt: 50 }),
    ]);
    expect(sorted.map((row) => row.id)).toEqual([
      "review-new",
      "open-old",
      "closed-new",
      "closed-old",
    ]);
  });

  it("keeps only open and under-review events under Open", () => {
    const rows = [
      event({ id: "a", status: "Open" }),
      event({ id: "b", status: "Closed" }),
      event({ id: "c", status: "UnderReview" }),
    ];
    expect(filterEvents(rows, "open").map((row) => row.id)).toEqual(["a", "c"]);
    expect(filterEvents(rows, "all")).toHaveLength(3);
  });
});

describe("eventAmountMinor", () => {
  it("reads the server's decimal text in cents and treats empty, zero and junk as none", () => {
    expect(eventAmountMinor("150.00")).toBe(15_000);
    expect(eventAmountMinor("0.29")).toBe(29);
    expect(eventAmountMinor("640")).toBe(64_000);
    expect(eventAmountMinor("0")).toBeNull();
    expect(eventAmountMinor("0.0000")).toBeNull();
    expect(eventAmountMinor("")).toBeNull();
    expect(eventAmountMinor(null)).toBeNull();
    expect(eventAmountMinor("abc")).toBeNull();
  });

  it("writes it as money, and nothing when there is none", () => {
    expect(formatEventAmount("640.00")).toBe("$640.00");
    expect(formatEventAmount(null)).toBeNull();
  });
});

describe("actionSummary", () => {
  it("names who issued an active action, the days off and whether it was acknowledged", () => {
    expect(
      actionSummary(
        {
          status: "Active",
          issuedAt: 1_790_100_000,
          expiresAt: null,
          issuedBy: { id: "usr_1", name: "Dana Whitfield" },
          suspensionDays: 3,
          acknowledgedAt: 1_790_200_000,
        },
        translate,
      ),
    ).toMatch(/ by Dana Whitfield · 3 days off · Acknowledged$/);
  });

  it("says when an active action still waits for the acknowledgement", () => {
    expect(
      actionSummary(
        {
          status: "Active",
          issuedAt: 1_790_100_000,
          expiresAt: null,
          issuedBy: null,
          suspensionDays: null,
          acknowledgedAt: null,
        },
        t,
      ),
    ).toMatch(/waiting for the acknowledgement$/);
  });
});
