import { describe, expect, it } from "vitest";
import {
  coverageByDay,
  coverageOn,
  coverageTone,
  isRotaDensity,
  matchesRotaSearch,
  rotaCellMode,
  peakCoverage,
  rotaComposition,
  rotaConflicts,
  summarizeSwaps,
  templateStats,
  todayColumnIndex,
  unrosteredWorkers,
} from "../scheduling-board";

const DAY = 86_400;
const WEEK_START = Date.UTC(2026, 8, 6) / 1000;

type DayOver = Partial<{
  state: string;
  scheduled: boolean;
  isConflict: boolean;
  durationMinutes: number;
}>;

function day(index: number, over: DayOver = {}) {
  return {
    date: WEEK_START + index * DAY,
    state: "Scheduled",
    scheduled: true,
    startMinute: 480,
    durationMinutes: 480,
    preference: null,
    assignmentCount: 0,
    isConflict: false,
    ...over,
  };
}

function row(name: string, days: ReturnType<typeof day>[], over: Record<string, unknown> = {}) {
  const scheduledDays = days.filter((d) => d.scheduled).length;
  return {
    workerId: `wrk_${name.toLowerCase().replace(/\s+/g, "_")}`,
    name,
    fleetCode: "SOUTH",
    fleetColor: "#2563eb",
    shiftCode: "DAYS",
    shiftName: "Days",
    shiftColor: null,
    days,
    scheduledDays,
    conflicts: days.filter((d) => d.isConflict).length,
    ...over,
  };
}

const week = (pattern: (index: number) => DayOver) =>
  Array.from({ length: 7 }, (_, index) => day(index, pattern(index)));

describe("coverageByDay", () => {
  // Cover is the day's question, not the person's: a row total never says
  // whether Thursday is thin.
  it("counts who is rostered and who can actually work it, per column", () => {
    const rows = [
      row(
        "Ada Byrne",
        week(() => ({})),
      ),
      row(
        "Ben Cole",
        week((i) => (i === 3 ? { state: "TimeOff", isConflict: true } : {})),
      ),
      row(
        "Cal Diaz",
        week((i) => (i === 3 ? { state: "Leave", scheduled: false } : {})),
      ),
    ];
    const coverage = coverageByDay(rows);
    expect(coverage).toHaveLength(7);
    expect(coverage[0]).toMatchObject({ expected: 3, covered: 3, conflicts: 0 });
    expect(coverage[3]).toMatchObject({
      date: WEEK_START + 3 * DAY,
      expected: 2,
      covered: 1,
      conflicts: 1,
      timeOff: 1,
      leave: 1,
    });
  });

  it("keeps columns in date order whatever order the rows arrive in", () => {
    const rows = [row("Zed", [day(2), day(0)])];
    expect(coverageByDay(rows).map((c) => c.date)).toEqual([WEEK_START, WEEK_START + 2 * DAY]);
  });

  it("finds the column an instant falls in", () => {
    const coverage = coverageByDay([
      row(
        "Ada Byrne",
        week(() => ({})),
      ),
    ]);
    expect(coverageOn(coverage, WEEK_START + 2 * DAY + 3600)?.date).toBe(WEEK_START + 2 * DAY);
    expect(coverageOn(coverage, WEEK_START - 1)).toBeNull();
    expect(peakCoverage(coverage)).toBe(1);
  });
});

describe("coverageTone", () => {
  it("calls a day under half the peak thin, and an empty day none", () => {
    expect(coverageTone(0, 10)).toBe("none");
    expect(coverageTone(4, 10)).toBe("thin");
    expect(coverageTone(5, 10)).toBe("strong");
    expect(coverageTone(3, 0)).toBe("strong");
  });
});

describe("rotaComposition", () => {
  it("splits person-days into working, off and away", () => {
    const rows = [
      row(
        "Ada Byrne",
        week((i) =>
          i === 0 || i === 6
            ? { state: "Off", scheduled: false }
            : i === 3
              ? { state: "Assigned" }
              : i === 4
                ? { state: "Leave", scheduled: false }
                : {},
        ),
      ),
    ];
    expect(rotaComposition(rows)).toEqual({ working: 4, off: 2, away: 1, total: 7 });
  });
});

describe("rotaConflicts", () => {
  it("lists every conflict soonest first with what won the day", () => {
    const rows = [
      row(
        "Ben Cole",
        week((i) => (i === 5 ? { state: "Unavailable", isConflict: true } : {})),
      ),
      row(
        "Ada Byrne",
        week((i) => (i === 2 ? { state: "TimeOff", isConflict: true } : {})),
      ),
    ];
    expect(rotaConflicts(rows)).toEqual([
      {
        workerId: "wrk_ada_byrne",
        name: "Ada Byrne",
        date: WEEK_START + 2 * DAY,
        state: "TimeOff",
      },
      {
        workerId: "wrk_ben_cole",
        name: "Ben Cole",
        date: WEEK_START + 5 * DAY,
        state: "Unavailable",
      },
    ]);
  });
});

describe("unrosteredWorkers", () => {
  // Somebody on a pattern that rosters nothing this week is the same gap as
  // somebody on no pattern, so both are listed.
  it("lists people with no scheduled day, by name", () => {
    const rows = [
      row(
        "Zed Young",
        week(() => ({ scheduled: false, state: "Off" })),
        { shiftName: null },
      ),
      row(
        "Ada Byrne",
        week(() => ({})),
      ),
      row(
        "Ben Cole",
        week(() => ({ scheduled: false, state: "Off" })),
      ),
    ];
    expect(unrosteredWorkers(rows).map((w) => [w.name, w.shiftName])).toEqual([
      ["Ben Cole", "Days"],
      ["Zed Young", null],
    ]);
  });
});

describe("matchesRotaSearch", () => {
  it("matches name, shift and terminal loosely", () => {
    const person = row("Ada Byrne", []);
    expect(matchesRotaSearch(person, "ada")).toBe(true);
    expect(matchesRotaSearch(person, "days")).toBe(true);
    expect(matchesRotaSearch(person, "SOUTH")).toBe(true);
    expect(matchesRotaSearch(person, "north")).toBe(false);
    expect(
      matchesRotaSearch({ ...person, fleetCode: null, shiftName: null, shiftCode: null }, "x"),
    ).toBe(false);
    expect(matchesRotaSearch(person, "  ")).toBe(true);
  });
});

describe("summarizeSwaps", () => {
  it("separates swaps waiting on the office from those waiting on a colleague", () => {
    expect(
      summarizeSwaps([
        { status: "Accepted" },
        { status: "Accepted" },
        { status: "Proposed" },
        { status: "Approved" },
        { status: "Declined" },
      ]),
    ).toEqual({ awaitingOffice: 2, awaitingColleague: 1 });
  });
});

describe("templateStats", () => {
  it("counts active patterns, the people on them, and the mean working week", () => {
    expect(
      templateStats([
        { status: "Active", activeAssignmentCount: 4, daysOfWeek: "0111110", durationMinutes: 480 },
        { status: "Active", activeAssignmentCount: 2, daysOfWeek: "1000001", durationMinutes: 600 },
        {
          status: "Inactive",
          activeAssignmentCount: 0,
          daysOfWeek: "0111110",
          durationMinutes: 480,
        },
      ]),
    ).toEqual({ active: 2, retired: 1, onPatterns: 6, averageWeeklyMinutes: 1800 });
  });

  it("has no average with no active pattern", () => {
    expect(templateStats([]).averageWeeklyMinutes).toBeNull();
  });
});

describe("todayColumnIndex", () => {
  it("finds today's column and reports -1 outside the range", () => {
    const days = week(() => ({}));
    expect(todayColumnIndex(days, WEEK_START + 3 * DAY + 100)).toBe(3);
    expect(todayColumnIndex(days, WEEK_START + 7 * DAY)).toBe(-1);
  });
});

describe("rotaCellMode", () => {
  // Fourteen or twenty-eight columns have no room for a clock time, so more
  // than one week draws blocks whatever the reader asked for.
  it("draws detail or time for one week and blocks for more", () => {
    expect(rotaCellMode("comfortable", 1)).toBe("detail");
    expect(rotaCellMode("compact", 1)).toBe("time");
    expect(rotaCellMode("comfortable", 2)).toBe("block");
    expect(rotaCellMode("compact", 4)).toBe("block");
  });

  it("only trusts a stored density it recognises", () => {
    expect(isRotaDensity("compact")).toBe(true);
    expect(isRotaDensity("comfortable")).toBe(true);
    expect(isRotaDensity("dense")).toBe(false);
    expect(isRotaDensity(undefined)).toBe(false);
  });
});
