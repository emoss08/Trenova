import type { WorkerSafetyEventRow } from "@/lib/graphql/worker-safety";
import { describe, expect, it } from "vitest";
import {
  decimalAmount,
  disciplineStair,
  filterEvents,
  pointsGauge,
  sortEvents,
} from "../safety-model";

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

describe("pointsGauge", () => {
  it("runs the track two past the at-risk line and fills one segment per event carrying points, oldest first", () => {
    const gauge = pointsGauge({ ...thresholds, activePoints: 7 }, [
      event({ id: "inspection", occurredAt: 300, activePoints: 4, pointsExpireAt: 3_000 }),
      event({ id: "citation", occurredAt: 100, activePoints: 0, points: 2 }),
      event({ id: "incident", occurredAt: 200, activePoints: 3, pointsExpireAt: 2_000 }),
    ]);

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
    const gauge = pointsGauge({ ...thresholds, activePoints: 5 }, [
      event({ id: "a", activePoints: 2, pointsExpireAt: 9_000 }),
      event({ id: "b", activePoints: 3, pointsExpireAt: 9_000 }),
    ]);
    expect(gauge.rollOffs).toEqual([{ date: 9_000, points: 5 }]);
  });

  it("widens the track past the points when they pass it, and turns to at risk", () => {
    const gauge = pointsGauge({ ...thresholds, activePoints: 15 }, []);
    expect(gauge.max).toBe(15);
    expect(gauge.fill).toBe(1);
    expect(gauge.tone).toBe("risk");
  });

  it("is calm and empty with no active points", () => {
    const gauge = pointsGauge({ ...thresholds, activePoints: 0 }, [event({ points: 3 })]);
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

describe("decimalAmount", () => {
  it("reads the server's decimal text and treats empty and zero as none", () => {
    expect(decimalAmount("150.00")).toBe(150);
    expect(decimalAmount("0")).toBeNull();
    expect(decimalAmount("")).toBeNull();
    expect(decimalAmount(null)).toBeNull();
    expect(decimalAmount("abc")).toBeNull();
  });
});
