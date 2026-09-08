import { describe, expect, it } from "vitest";
import {
  buildMonthGrid,
  buildWeekSegments,
  laneCount,
  monthEndUnix,
  monthStartUnix,
  ptoTiming,
  selectionRange,
  shiftMonth,
  spansOnDay,
  splitVisibleSegments,
} from "../calendar-layout";
import { buildWhosOut } from "../whos-out-strip";

const DAY = 86_400;
const local = (y: number, m: number, d: number) => new Date(y, m, d).getTime() / 1000;

describe("buildMonthGrid", () => {
  it("starts on the Sunday before the 1st and covers the whole month in full weeks", () => {
    const weeks = buildMonthGrid(2026, 2, new Date(2026, 2, 10));
    expect(weeks[0].days[0].date).toBe(1);
    expect(weeks[0].days[0].inMonth).toBe(true);
    expect(weeks).toHaveLength(5);
    expect(weeks[4].days[6].inMonth).toBe(false);
    expect(weeks[1].days[2].isToday).toBe(true);
    expect(
      weeks.flatMap((w) => w.days).every((d, i, all) => i === 0 || d.unix > all[i - 1].unix),
    ).toBe(true);
  });

  it("pads leading days from the previous month", () => {
    const weeks = buildMonthGrid(2026, 3, new Date(2026, 0, 1));
    expect(weeks[0].days[0].inMonth).toBe(false);
    expect(weeks[0].days[3].date).toBe(1);
    expect(weeks[0].days[3].inMonth).toBe(true);
  });
});

describe("buildWeekSegments", () => {
  const weeks = buildMonthGrid(2026, 2, new Date(2026, 2, 1));

  it("clips a span that crosses the week boundary and flags the continuation", () => {
    const spanning = { id: "a", startDate: local(2026, 2, 5), endDate: local(2026, 2, 10) };
    const first = buildWeekSegments([spanning], weeks[0]);
    expect(first).toHaveLength(1);
    expect(first[0]).toMatchObject({
      startCol: 4,
      endCol: 6,
      continuesBefore: false,
      continuesAfter: true,
    });

    const second = buildWeekSegments([spanning], weeks[1]);
    expect(second[0]).toMatchObject({
      startCol: 0,
      endCol: 2,
      continuesBefore: true,
      continuesAfter: false,
    });
  });

  it("keeps a span that starts in the previous month on the padded leading days", () => {
    const aprilWeeks = buildMonthGrid(2026, 3, new Date(2026, 3, 1));
    const spanning = { id: "b", startDate: local(2026, 2, 30), endDate: local(2026, 3, 2) };
    const segments = buildWeekSegments([spanning], aprilWeeks[0]);
    expect(segments[0]).toMatchObject({ startCol: 1, endCol: 4 });
  });

  it("packs overlapping spans into separate lanes and reuses lanes when free", () => {
    const items = [
      { id: "long", startDate: local(2026, 2, 1), endDate: local(2026, 2, 4) },
      { id: "overlap", startDate: local(2026, 2, 2), endDate: local(2026, 2, 3) },
      { id: "later", startDate: local(2026, 2, 6), endDate: local(2026, 2, 6) },
    ];
    const segments = buildWeekSegments(items, weeks[0]);
    const byId = Object.fromEntries(segments.map((s) => [s.item.id, s.lane]));
    expect(byId.long).toBe(0);
    expect(byId.overlap).toBe(1);
    expect(byId.later).toBe(0);
    expect(laneCount(segments)).toBe(2);
  });

  it("ignores spans outside the week entirely", () => {
    const segments = buildWeekSegments(
      [{ id: "x", startDate: local(2026, 3, 20), endDate: local(2026, 3, 22) }],
      weeks[0],
    );
    expect(segments).toHaveLength(0);
  });
});

describe("month helpers", () => {
  it("shifts across year boundaries and produces inclusive month ranges", () => {
    expect(shiftMonth(2026, 11, 1)).toEqual({ year: 2027, month: 0 });
    expect(shiftMonth(2026, 0, -1)).toEqual({ year: 2025, month: 11 });
    expect(monthEndUnix(2026, 1) - monthStartUnix(2026, 1)).toBe(28 * DAY - 1);
    expect(selectionRange(10, 5)).toEqual({ start: 5, end: 10 });
  });
});

describe("buildWhosOut", () => {
  it("lists approved workers per day for the next week and skips requested ones", () => {
    const today = local(2026, 2, 2);
    const items = [
      {
        id: "1",
        status: "Approved",
        startDate: today - DAY,
        endDate: today + DAY,
        type: "Vacation",
      },
      { id: "2", status: "Requested", startDate: today, endDate: today, type: "Sick" },
      {
        id: "3",
        status: "Approved",
        startDate: today + DAY * 3,
        endDate: today + DAY * 3,
        type: "Sick",
      },
    ] as never[];
    const days = buildWhosOut(items, today);
    expect(days).toHaveLength(7);
    expect(days[0].label).toBe("Today");
    expect(days[0].out.map((p) => p.id)).toEqual(["1"]);
    expect(days[1].out.map((p) => p.id)).toEqual(["1"]);
    expect(days[3].out.map((p) => p.id)).toEqual(["3"]);
    expect(days[6].out).toHaveLength(0);
  });
});

describe("splitVisibleSegments", () => {
  const weeks = buildMonthGrid(2026, 2, new Date(2026, 2, 1));
  const items = [
    { id: "a", startDate: local(2026, 2, 1), endDate: local(2026, 2, 3) },
    { id: "b", startDate: local(2026, 2, 1), endDate: local(2026, 2, 1) },
    { id: "c", startDate: local(2026, 2, 1), endDate: local(2026, 2, 2) },
    { id: "d", startDate: local(2026, 2, 2), endDate: local(2026, 2, 2) },
  ];

  it("keeps every lane when they fit under the cap", () => {
    const segments = buildWeekSegments(items, weeks[0]);
    const { visible, overflow } = splitVisibleSegments(segments, 3);
    expect(visible).toHaveLength(4);
    expect(overflow).toEqual([0, 0, 0, 0, 0, 0, 0]);
  });

  it("drops lanes past the cap and counts the hidden spans on each day they cover", () => {
    const segments = buildWeekSegments(items, weeks[0]);
    const { visible, overflow } = splitVisibleSegments(segments, 2);
    expect(visible.map((s) => s.item.id).sort()).toEqual(["a", "c"]);
    expect(overflow).toEqual([1, 1, 0, 0, 0, 0, 0]);
  });
});

describe("spansOnDay", () => {
  it("returns the spans covering a day, earliest start first", () => {
    const items = [
      { id: "late", startDate: local(2026, 2, 9), endDate: local(2026, 2, 12) },
      { id: "early", startDate: local(2026, 2, 5), endDate: local(2026, 2, 10) },
      { id: "outside", startDate: local(2026, 2, 11), endDate: local(2026, 2, 11) },
    ];
    expect(spansOnDay(items, local(2026, 2, 10)).map((s) => s.id)).toEqual(["early", "late"]);
    expect(spansOnDay(items, local(2026, 2, 13))).toEqual([]);
  });
});

describe("ptoTiming", () => {
  const today = local(2026, 2, 10);

  it("describes upcoming, active and past spans relative to today", () => {
    expect(
      ptoTiming({ startDate: local(2026, 2, 11), endDate: local(2026, 2, 12) }, today),
    ).toEqual({ tone: "upcoming", label: "Starts tomorrow" });
    expect(
      ptoTiming({ startDate: local(2026, 2, 14), endDate: local(2026, 2, 15) }, today).label,
    ).toBe("Starts in 4 days");
    const active = ptoTiming({ startDate: local(2026, 2, 9), endDate: local(2026, 2, 12) }, today);
    expect(active.tone).toBe("active");
    expect(active.label).toMatch(/^Out now · back /);
    expect(ptoTiming({ startDate: local(2026, 2, 1), endDate: local(2026, 2, 9) }, today)).toEqual({
      tone: "past",
      label: "Ended yesterday",
    });
    expect(
      ptoTiming({ startDate: local(2026, 2, 1), endDate: local(2026, 2, 4) }, today).label,
    ).toBe("Ended 6 days ago");
  });

  it("treats a span ending today as still active", () => {
    expect(ptoTiming({ startDate: local(2026, 2, 10), endDate: today }, today).tone).toBe("active");
  });
});
