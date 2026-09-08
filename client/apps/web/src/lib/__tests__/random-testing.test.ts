import { describe, expect, it } from "vitest";
import {
  drawShortOfTarget,
  entryTally,
  filterDraws,
  periodKeyFor,
  periodSlotsForYear,
  poolCalendar,
  poolProgress,
  programmeOverview,
  roundStatusLabel,
  type RandomDrawLike,
} from "../random-testing";

const pool = {
  id: "pool_1",
  code: "DOT",
  name: "DOT drivers",
  status: "Active",
  period: "Quarterly",
  drugRatePercent: 50,
  alcoholRatePercent: 10,
  isDefault: true,
  meetsDotMinimums: true,
};

// 2026-08-15T00:00:00Z: inside Q3, so Q1 and Q2 are behind us and Q4 ahead.
const NOW = Date.UTC(2026, 7, 15) / 1000;

function draw(over: Partial<RandomDrawLike> = {}): RandomDrawLike {
  return {
    id: "draw_1",
    poolId: "pool_1",
    periodKey: "2026-Q1",
    periodStart: Date.UTC(2026, 0, 1) / 1000,
    periodEnd: Date.UTC(2026, 3, 1) / 1000,
    status: "Final",
    poolSize: 40,
    drugTarget: 5,
    alcoholTarget: 1,
    drugSelected: 5,
    alcoholSelected: 1,
    drawnAt: Date.UTC(2026, 0, 4) / 1000,
    ...over,
  };
}

const draws = [
  draw(),
  draw({
    id: "draw_2",
    periodKey: "2026-Q2",
    periodStart: Date.UTC(2026, 3, 1) / 1000,
    periodEnd: Date.UTC(2026, 6, 1) / 1000,
    status: "Cancelled",
    drawnAt: Date.UTC(2026, 3, 2) / 1000,
  }),
  draw({
    id: "draw_3",
    periodKey: "2026-Q3",
    periodStart: Date.UTC(2026, 6, 1) / 1000,
    periodEnd: Date.UTC(2026, 9, 1) / 1000,
    status: "Draft",
    drugSelected: 3,
    drawnAt: Date.UTC(2026, 6, 3) / 1000,
  }),
];

describe("periodSlotsForYear", () => {
  // The keys must be byte-for-byte what the server stamps on a draw, or a
  // drawn round would show as never drawn.
  it("names the slots the way the server does, with UTC bounds", () => {
    expect(periodSlotsForYear("Quarterly", 2026).map((slot) => slot.key)).toEqual([
      "2026-Q1",
      "2026-Q2",
      "2026-Q3",
      "2026-Q4",
    ]);
    expect(periodSlotsForYear("Monthly", 2026)[2]).toEqual({
      key: "2026-M03",
      label: "Mar",
      start: Date.UTC(2026, 2, 1) / 1000,
      end: Date.UTC(2026, 3, 1) / 1000,
    });
    expect(periodSlotsForYear("SemiAnnual", 2026).map((slot) => slot.key)).toEqual([
      "2026-H1",
      "2026-H2",
    ]);
    expect(periodSlotsForYear("Annual", 2026).map((slot) => slot.key)).toEqual(["2026"]);
    expect(periodSlotsForYear("Whenever", 2026)).toEqual([]);
  });

  it("keys a moment to its round", () => {
    expect(periodKeyFor("Quarterly", NOW)).toBe("2026-Q3");
    expect(periodKeyFor("Monthly", NOW)).toBe("2026-M08");
    expect(periodKeyFor("SemiAnnual", NOW)).toBe("2026-H2");
    expect(periodKeyFor("Annual", NOW)).toBe("2026");
  });
});

describe("poolCalendar", () => {
  // A voided round leaves its slot empty: the period still has to be drawn,
  // and a slot that reads "drawn" because of a voided draw hides that.
  it("marks each round drawn, owed, missed or ahead, ignoring voided draws", () => {
    const calendar = poolCalendar(pool, draws, NOW);
    expect(calendar.map((slot) => [slot.key, slot.state, slot.voided])).toEqual([
      ["2026-Q1", "final", 0],
      ["2026-Q2", "missed", 1],
      ["2026-Q3", "draft", 0],
      ["2026-Q4", "upcoming", 0],
    ]);
    expect(calendar[2].draw?.id).toBe("draw_3");
  });

  it("owes the current round when nothing has been drawn for it", () => {
    const calendar = poolCalendar(pool, [draws[0]], NOW);
    expect(calendar[2].state).toBe("due");
  });

  it("does not read another pool's draws", () => {
    const other = { ...pool, id: "pool_2" };
    expect(poolCalendar(other, draws, NOW).map((slot) => slot.state)).toEqual([
      "missed",
      "missed",
      "due",
      "upcoming",
    ]);
  });
});

describe("poolProgress", () => {
  it("sums the year's live rounds and notices one that fell short", () => {
    expect(poolProgress(pool, draws, NOW)).toEqual({
      rounds: 2,
      finalRounds: 1,
      drugSelected: 8,
      drugTarget: 10,
      alcoholSelected: 2,
      alcoholTarget: 2,
      onPace: false,
    });
  });

  it("leaves last year's rounds out", () => {
    const old = draw({
      id: "draw_0",
      periodKey: "2025-Q4",
      periodStart: Date.UTC(2025, 9, 1) / 1000,
      periodEnd: Date.UTC(2026, 0, 1) / 1000,
    });
    expect(poolProgress(pool, [old], NOW).rounds).toBe(0);
  });
});

describe("filterDraws", () => {
  it("narrows by pool and status and puts the newest first", () => {
    const other = draw({ id: "draw_9", poolId: "pool_2", drawnAt: Date.UTC(2026, 7, 1) / 1000 });
    expect(
      filterDraws([...draws, other], { poolId: null, status: "all" }).map((d) => d.id),
    ).toEqual(["draw_9", "draw_3", "draw_2", "draw_1"]);
    expect(
      filterDraws([...draws, other], { poolId: "pool_1", status: "Final" }).map((d) => d.id),
    ).toEqual(["draw_1"]);
    expect(
      filterDraws([...draws, other], { poolId: null, status: "Cancelled" }).map((d) => d.id),
    ).toEqual(["draw_2"]);
  });
});

describe("programmeOverview", () => {
  it("reads the programme across pools", () => {
    const inactive = {
      ...pool,
      id: "pool_2",
      code: "OLD",
      status: "Inactive",
      meetsDotMinimums: false,
    };
    const thin = { ...pool, id: "pool_3", code: "THIN", meetsDotMinimums: false, isDefault: false };
    const overview = programmeOverview([pool, inactive, thin], draws, NOW);
    expect(overview).toEqual({
      pools: 3,
      activePools: 2,
      belowMinimum: 1,
      lastPoolSize: 40,
      lastDrawnAt: Date.UTC(2026, 6, 3) / 1000,
      roundsThisYear: 2,
      draftRounds: 1,
      finalRounds: 1,
      owedNow: 1,
      missed: 2,
      drugSelected: 8,
      drugTarget: 10,
      alcoholSelected: 2,
      alcoholTarget: 2,
    });
  });

  it("has no last draw before anything has been drawn", () => {
    const overview = programmeOverview([pool], [], NOW);
    expect(overview.lastPoolSize).toBeNull();
    expect(overview.owedNow).toBe(1);
  });
});

describe("entryTally", () => {
  it("counts a round's selections by substance and what is still to come", () => {
    expect(
      entryTally([
        { substance: "Drug", status: "Selected" },
        { substance: "Drug", status: "Notified" },
        { substance: "Drug", status: "Completed" },
        { substance: "Drug", status: "Excused" },
        { substance: "Alcohol", status: "Missed" },
      ]),
    ).toEqual([
      {
        substance: "Drug",
        total: 4,
        collected: 1,
        notified: 1,
        excused: 1,
        missed: 0,
        outstanding: 2,
      },
      {
        substance: "Alcohol",
        total: 1,
        collected: 0,
        notified: 0,
        excused: 0,
        missed: 1,
        outstanding: 0,
      },
    ]);
  });
});

describe("labels", () => {
  it("calls a cancelled round voided and flags a short one", () => {
    expect(roundStatusLabel("Cancelled")).toBe("Voided");
    expect(roundStatusLabel("Final")).toBe("Final");
    expect(drawShortOfTarget(draw({ alcoholSelected: 0 }))).toBe(true);
    expect(drawShortOfTarget(draw())).toBe(false);
  });
});
