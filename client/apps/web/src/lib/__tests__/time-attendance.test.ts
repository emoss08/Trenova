import { describe, expect, it } from "vitest";
import {
  countBySegment,
  dayKeyFor,
  dayTrackSpans,
  groupEntriesByDay,
  isOverlong,
  longestRunning,
  minuteOfDay,
  overtimeHeadroom,
  queueSegmentOf,
  rankRunning,
  sheetsInSegment,
  summarizeSheets,
  waitingDays,
  weekCardDays,
} from "../time-attendance";

const HOUR = 3600;
const NOW = Date.UTC(2026, 8, 7, 15, 30) / 1000;

function sheet(
  over: Partial<{
    workerId: string;
    status: string;
    regularMinutes: number;
    overtimeMinutes: number;
    paidLeaveMinutes: number;
  }> = {},
) {
  return {
    workerId: "wrk_1",
    status: "Submitted",
    regularMinutes: 2400,
    overtimeMinutes: 0,
    paidLeaveMinutes: 0,
    ...over,
  };
}

describe("summarizeSheets", () => {
  it("adds the three kinds of hours and counts weeks and people", () => {
    const summary = summarizeSheets([
      sheet(),
      sheet({ workerId: "wrk_1", overtimeMinutes: 120 }),
      sheet({ workerId: "wrk_2", paidLeaveMinutes: 480, regularMinutes: 1920 }),
    ]);
    expect(summary).toEqual({
      count: 3,
      workers: 2,
      regularMinutes: 6720,
      overtimeMinutes: 120,
      paidLeaveMinutes: 480,
      totalMinutes: 7320,
      overtimeWeeks: 1,
    });
  });

  it("is all zeros for nothing", () => {
    expect(summarizeSheets([])).toMatchObject({ count: 0, workers: 0, totalMinutes: 0 });
  });
});

describe("queue segments", () => {
  // A week sent back is still somebody's unfinished week. Filing it nowhere
  // is how it went missing from the old queue.
  it("files a sent-back week under Open", () => {
    expect(queueSegmentOf("Rejected")).toBe("Open");
    expect(queueSegmentOf("Open")).toBe("Open");
    expect(queueSegmentOf("Submitted")).toBe("Submitted");
    expect(queueSegmentOf("Approved")).toBe("Approved");
    expect(queueSegmentOf("Locked")).toBe("Locked");
    expect(queueSegmentOf("Draft")).toBeNull();
  });

  it("counts each segment from one list", () => {
    expect(
      countBySegment([
        sheet(),
        sheet({ status: "Open" }),
        sheet({ status: "Rejected" }),
        sheet({ status: "Approved" }),
        sheet({ status: "Unknown" }),
      ]),
    ).toEqual({ Submitted: 1, Open: 2, Approved: 1, Locked: 0 });
  });

  it("narrows a list to a segment", () => {
    const rows = [sheet({ status: "Open" }), sheet({ status: "Rejected" }), sheet()];
    expect(sheetsInSegment(rows, "Open").map((row) => row.status)).toEqual(["Open", "Rejected"]);
  });
});

describe("running entries", () => {
  it("ranks the longest-running punch first", () => {
    const ranked = rankRunning(
      [
        { clockedInAt: NOW - 1 * HOUR },
        { clockedInAt: NOW - 6 * HOUR },
        { clockedInAt: NOW - 3 * HOUR },
      ],
      NOW,
    );
    expect(ranked.map((row) => row.runningMinutes)).toEqual([360, 180, 60]);
    expect(longestRunning([{ clockedInAt: NOW - 90 * 60 }], NOW)?.runningMinutes).toBe(90);
    expect(longestRunning([], NOW)).toBeNull();
  });

  it("calls a punch past twelve hours overlong", () => {
    expect(isOverlong(12 * 60)).toBe(true);
    expect(isOverlong(12 * 60 - 1)).toBe(false);
  });
});

describe("groupEntriesByDay", () => {
  const entry = (
    id: string,
    clockedInAt: number,
    over: Partial<{
      clockedOutAt: number | null;
      paidMinutes: number;
      breakMinutes: number;
    }> = {},
  ) => ({
    id,
    clockedInAt,
    clockedOutAt: clockedInAt + 8 * HOUR,
    paidMinutes: 480,
    breakMinutes: 0,
    ...over,
  });

  it("keys days in the reader's zone, not UTC", () => {
    // 23:30 on Sep 6 in New York is 03:30 on Sep 7 in UTC.
    const lateEvening = Date.UTC(2026, 8, 7, 3, 30) / 1000;
    expect(dayKeyFor(lateEvening, "America/New_York")).toBe("2026-09-06");
    expect(dayKeyFor(lateEvening, "UTC")).toBe("2026-09-07");
  });

  it("groups by day newest first, with the day's paid and break totals", () => {
    const monday = Date.UTC(2026, 8, 7, 8) / 1000;
    const tuesday = Date.UTC(2026, 8, 8, 8) / 1000;
    const days = groupEntriesByDay(
      [
        entry("a", monday, { breakMinutes: 30, paidMinutes: 450 }),
        entry("b", monday + 10 * HOUR, { paidMinutes: 120 }),
        entry("c", tuesday),
      ],
      "UTC",
    );
    expect(days.map((day) => day.key)).toEqual(["2026-09-08", "2026-09-07"]);
    expect(days[1]).toMatchObject({ paidMinutes: 570, breakMinutes: 30, running: false });
    expect(days[1].entries.map((row) => row.id)).toEqual(["b", "a"]);
    expect(days[1].startsAt).toBe(monday);
  });

  it("marks a day with an open punch as running", () => {
    const days = groupEntriesByDay(
      [entry("open", NOW - HOUR, { clockedOutAt: null, paidMinutes: 0 })],
      "UTC",
    );
    expect(days[0].running).toBe(true);
  });
});

describe("the 24-hour track", () => {
  it("reads the minute of the day in the reader's zone", () => {
    // 15:30 UTC in September is 11:30 in New York (daylight time) and 17:30 in Berlin.
    expect(minuteOfDay(NOW, "UTC")).toBe(15 * 60 + 30);
    expect(minuteOfDay(NOW, "America/New_York")).toBe(11 * 60 + 30);
    expect(minuteOfDay(NOW, "Europe/Berlin")).toBe(17 * 60 + 30);
  });

  it("lays punches on the track in order, running an open punch to now", () => {
    const eight = Date.UTC(2026, 8, 7, 8) / 1000;
    const spans = dayTrackSpans(
      [
        { id: "late", clockedInAt: eight + 6 * HOUR, clockedOutAt: null },
        { id: "early", clockedInAt: eight, clockedOutAt: eight + 4 * HOUR },
      ],
      NOW,
      "UTC",
    );
    expect(spans.map((span) => span.id)).toEqual(["early", "late"]);
    expect(spans[0]).toMatchObject({ start: 8 / 24, end: 12 / 24, running: false });
    expect(spans[1]).toMatchObject({ start: 14 / 24, end: 15.5 / 24, running: true });
  });

  it("cuts a punch that crosses midnight at the end of its day", () => {
    const tenPm = Date.UTC(2026, 8, 6, 22) / 1000;
    const [span] = dayTrackSpans(
      [{ id: "night", clockedInAt: tenPm, clockedOutAt: tenPm + 6 * HOUR }],
      NOW,
      "UTC",
    );
    expect(span.end).toBe(1);
  });
});

describe("weekCardDays", () => {
  const weekStart = Date.UTC(2026, 8, 6) / 1000;
  const DAY = 86_400;

  it("gives every day of the week a cell, with the punches that fell on it", () => {
    const days = weekCardDays(
      [
        {
          id: "a",
          clockedInAt: weekStart + 8 * HOUR,
          clockedOutAt: weekStart + 16 * HOUR,
          paidMinutes: 450,
          breakMinutes: 30,
        },
        {
          id: "b",
          clockedInAt: weekStart + 17 * HOUR,
          clockedOutAt: weekStart + 19 * HOUR,
          paidMinutes: 120,
          breakMinutes: 0,
        },
        {
          id: "c",
          clockedInAt: weekStart + 2 * DAY + 9 * HOUR,
          clockedOutAt: null,
          paidMinutes: 0,
          breakMinutes: 0,
        },
        {
          id: "elsewhere",
          clockedInAt: weekStart + 9 * DAY,
          clockedOutAt: weekStart + 9 * DAY + HOUR,
          paidMinutes: 60,
          breakMinutes: 0,
        },
      ],
      weekStart,
    );
    expect(days).toHaveLength(7);
    expect(days[0]).toMatchObject({ paidMinutes: 570, breakMinutes: 30, punches: 2 });
    expect(days[2]).toMatchObject({ paidMinutes: 0, punches: 1, running: true });
    expect(days.reduce((sum, day) => sum + day.paidMinutes, 0)).toBe(570);
  });
});

describe("waiting and headroom", () => {
  it("counts whole days a week has waited and never goes negative", () => {
    expect(waitingDays(NOW - 86_400 - HOUR, NOW)).toBe(1);
    expect(waitingDays(NOW - HOUR, NOW)).toBe(0);
    expect(waitingDays(NOW + HOUR, NOW)).toBe(0);
  });

  it("says how far a week is from overtime, or how far past it", () => {
    expect(overtimeHeadroom(1935, 2400)).toEqual({ remaining: 465, over: 0, share: 1935 / 2400 });
    expect(overtimeHeadroom(2700, 2400)).toEqual({ remaining: 0, over: 300, share: 1 });
    expect(overtimeHeadroom(120, 0)).toEqual({ remaining: 0, over: 120, share: 1 });
  });
});
