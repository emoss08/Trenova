import { describe, expect, it } from "vitest";
import { stopProgress, type StopStripStop } from "../stop-strip-cell";

const stop = (overrides: Partial<StopStripStop>): StopStripStop => ({
  id: overrides.id ?? "s",
  kind: "delivery",
  state: "upcoming",
  late: false,
  place: "Dallas",
  due: null,
  arrived: null,
  ...overrides,
});

describe("stopProgress", () => {
  it("counts the stops behind the load and names the one it is headed for", () => {
    const progress = stopProgress(
      [
        stop({ id: "a", kind: "pickup", state: "done", due: 0, arrived: 0 }),
        stop({ id: "b", state: "current", due: 10_000 }),
        stop({ id: "c", due: 20_000 }),
      ],
      2_500,
    );

    expect(progress).toMatchObject({ total: 3, done: 1, late: 0 });
    expect(progress.next?.id).toBe("b");
    expect(progress.legShare).toBe(0.25);
    expect(progress.minutesToNext).toBe(125);
  });

  it("reports a stop past due as negative minutes", () => {
    const progress = stopProgress(
      [stop({ id: "a", state: "done", due: 0 }), stop({ id: "b", state: "current", due: 600, late: true })],
      1_800,
    );

    expect(progress.minutesToNext).toBe(-20);
    expect(progress.late).toBe(1);
  });

  it("leaves canceled stops out of the count", () => {
    const progress = stopProgress(
      [
        stop({ id: "a", state: "done" }),
        stop({ id: "b", state: "canceled", late: true }),
        stop({ id: "c", state: "done" }),
      ],
      0,
    );

    expect(progress).toMatchObject({ total: 2, done: 2, late: 0, next: null });
  });

  it("has no leg under way before the load leaves its first stop", () => {
    const progress = stopProgress(
      [stop({ id: "a", kind: "pickup", due: 3_600 }), stop({ id: "b", due: 7_200 })],
      0,
    );

    expect(progress.legShare).toBe(0);
    expect(progress.next?.id).toBe("a");
    expect(progress.minutesToNext).toBe(60);
  });
});
