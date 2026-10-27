import { describe, expect, it } from "vitest";
import {
  deltaOf,
  kpisOf,
  pictureOf,
  sheetColumn,
  shownColumns,
  trendPoints,
  type SheetSource,
} from "../report-sheet";

const DISPLAY = {
  style: "number",
  decimals: 2,
  grouping: true,
  currency: "",
  negative: "minus",
  notation: "standard",
  prefix: "",
  suffix: "",
  dateStyle: "",
  boolStyle: "",
  durationUnit: "",
  durationStyle: "",
  nullText: "",
};

const customer = sheetColumn({
  id: "customer",
  label: "Customer",
  type: "string",
  display: { ...DISPLAY, style: "text" },
});
const shipments = sheetColumn({
  id: "shipments",
  label: "Shipments",
  type: "int",
  display: DISPLAY,
});
const revenue = sheetColumn({
  id: "revenue",
  label: "Total revenue",
  type: "decimal",
  hint: "money",
  display: { ...DISPLAY, style: "currency", currency: "USD" },
});
const share = sheetColumn({
  id: "share",
  label: "Accessorial share",
  type: "decimal",
  hint: "percent",
  display: { ...DISPLAY, style: "percent" },
});
const month = sheetColumn({
  id: "month",
  label: "Month",
  type: "epoch",
  display: { ...DISPLAY, style: "date", dateStyle: "monthYear" },
});

function source(overrides: Partial<SheetSource> = {}): SheetSource {
  return {
    columns: [customer, shipments, revenue, share],
    rows: [
      ["Acme Foods", 162, "241880.50", "0.07"],
      ["Harborline", 148, "198412.00", "0.11"],
      ["Northwind", 121, "164209.75", "0.05"],
    ],
    totals: [null, 431, "604502.25", "0.08"],
    rowCount: 3,
    more: false,
    truncated: false,
    ...overrides,
  };
}

const NO_HINTS = { chart: null, sort: [] };

describe("sheetColumn", () => {
  it("reads a column by its resolved display, and by its catalog type without one", () => {
    expect(revenue).toMatchObject({ numeric: true, dated: false, percent: false });
    expect(share).toMatchObject({ numeric: true, percent: true });
    expect(month).toMatchObject({ numeric: false, dated: true });
    expect(customer).toMatchObject({ numeric: false, dated: false });
    expect(sheetColumn({ id: "n", label: "", type: "decimal", hint: "percent" })).toMatchObject({
      label: "n",
      numeric: true,
      percent: true,
      display: null,
    });
    expect(sheetColumn({ id: "d", label: "Day", type: "epoch" })).toMatchObject({
      numeric: false,
      dated: true,
    });
    expect(sheetColumn({ id: "t", label: "At", type: "time" })).toMatchObject({
      numeric: false,
      dated: true,
    });
    expect(sheetColumn({ id: "f", label: "Ratio", type: "float" })).toMatchObject({
      numeric: true,
    });
  });
});

describe("pictureOf", () => {
  it("ranks a labelled result by the measure it is sorted on, descending", () => {
    expect(pictureOf(source(), { chart: null, sort: [{ column: "revenue", desc: true }] })).toEqual(
      {
        kind: "ranked",
        label: 0,
        measure: 2,
        ranked: true,
      },
    );
  });

  it("draws bars on the first money measure, without claiming a ranking, when nothing sorts it", () => {
    expect(pictureOf(source(), NO_HINTS)).toEqual({
      kind: "ranked",
      label: 0,
      measure: 2,
      ranked: false,
    });
    expect(pictureOf(source({ columns: [customer, shipments, share, share] }), NO_HINTS)).toEqual({
      kind: "ranked",
      label: 0,
      measure: 1,
      ranked: false,
    });
  });

  it("follows the report's own bar, pie or donut chart", () => {
    for (const type of ["bar", "pie", "donut"]) {
      expect(
        pictureOf(source(), {
          chart: { type, x: "customer", series: ["revenue"] },
          sort: [],
        }),
      ).toEqual({ kind: "ranked", label: 0, measure: 2, ranked: false });
    }
  });

  it("leaves a map, a scatter or a lone figure as a table", () => {
    for (const type of ["map", "scatter", "kpi"]) {
      expect(pictureOf(source(), { chart: { type, x: null, series: [] }, sort: [] })).toEqual({
        kind: "table",
      });
    }
  });

  it("does not draw bars for a percentage or a measure below zero", () => {
    expect(
      pictureOf(source(), {
        chart: { type: "bar", x: "customer", series: ["share"] },
        sort: [],
      }),
    ).toEqual({ kind: "table" });
    expect(
      pictureOf(
        source({
          rows: [
            ["Acme", 4, "-1", "0"],
            ["Globex", 2, "1", "0"],
          ],
        }),
        NO_HINTS,
      ),
    ).toEqual({ kind: "table" });
  });

  it("trends a result led by periods, its line the first measure and its columns the second", () => {
    const trend = source({
      columns: [month, revenue, shipments],
      rows: [
        [1793491200, "1198775.60", 1112],
        [1790812800, "1128402.10", 1061],
        [1796083200, "1284310.42", 1142],
      ],
    });
    expect(pictureOf(trend, NO_HINTS)).toEqual({ kind: "trend", period: 0, line: 1, columns: 2 });
    expect(
      pictureOf(trend, {
        chart: { type: "line", x: "month", series: ["shipments"] },
        sort: [],
      }),
    ).toEqual({ kind: "trend", period: 0, line: 2, columns: null });
  });

  it("does not trend a sample that leaves periods out, or fewer than three", () => {
    const trend = {
      columns: [month, revenue],
      rows: [
        [1, "1"],
        [2, "2"],
        [3, "3"],
      ],
    };
    expect(pictureOf(source({ ...trend, more: true }), NO_HINTS)).toEqual({ kind: "table" });
    expect(pictureOf(source({ ...trend, rows: trend.rows.slice(0, 2) }), NO_HINTS)).toEqual({
      kind: "table",
    });
  });

  it("keeps a table when there is no measure or a single row", () => {
    expect(pictureOf(source({ columns: [customer], rows: [["A"], ["B"]] }), NO_HINTS)).toEqual({
      kind: "table",
    });
    expect(pictureOf(source({ rows: [["Acme", 1, "1", "0"]] }), NO_HINTS)).toEqual({
      kind: "table",
    });
  });
});

describe("shownColumns", () => {
  it("puts the picture's label first and stops at six columns", () => {
    const columns = [shipments, revenue, customer, share, shipments, revenue, share];
    expect(
      shownColumns(source({ columns }), { kind: "ranked", label: 2, measure: 1, ranked: false }),
    ).toEqual([2, 0, 1, 3, 4, 5]);
  });
});

describe("deltaOf", () => {
  it("is the change as a fraction of the earlier figure", () => {
    expect(deltaOf(108.2, 100, false)).toEqual({
      change: expect.closeTo(0.082, 6),
      direction: "up",
    });
    expect(deltaOf(98.6, 100, false)).toEqual({
      change: expect.closeTo(-0.014, 6),
      direction: "down",
    });
    expect(deltaOf(100, 100, false)).toEqual({ change: 0, direction: "flat" });
    expect(deltaOf(5, -10, false)).toEqual({ change: 1.5, direction: "up" });
  });

  it("states nothing without an earlier figure, from zero, or for a percentage", () => {
    expect(deltaOf(10, null, false)).toBeNull();
    expect(deltaOf(10, 0, false)).toBeNull();
    expect(deltaOf(0.2, 0.1, true)).toBeNull();
  });
});

describe("kpisOf", () => {
  it("gives each measure's grand total and the row count, against the previous run", () => {
    const kpis = kpisOf(source(), pictureOf(source(), NO_HINTS), {
      totals: new Map<string, unknown>([
        ["revenue", "500000"],
        ["shipments", 400],
        ["share", "0.06"],
      ]),
      rowCount: 2,
    });
    expect(kpis).toEqual([
      { kind: "total", column: 1, delta: { change: expect.closeTo(0.0775, 6), direction: "up" } },
      { kind: "total", column: 2, delta: { change: expect.closeTo(0.209, 3), direction: "up" } },
      { kind: "total", column: 3, delta: null },
      { kind: "rows", delta: { change: 0.5, direction: "up" } },
    ]);
  });

  it("compares nothing for a capped run, whose figures are of the kept rows", () => {
    const capped = source({ truncated: true });
    const kpis = kpisOf(capped, pictureOf(capped, NO_HINTS), {
      totals: new Map<string, unknown>([["revenue", "1"]]),
      rowCount: 1,
    });
    expect(kpis.every((kpi) => kpi.kind === "peak" || kpi.delta === null)).toBe(true);
  });

  it("gives a trend's last place to the period its line peaks in", () => {
    const trend = source({
      columns: [month, revenue],
      rows: [
        [1, "10"],
        [2, "30"],
        [3, "20"],
      ],
      totals: [null, "60"],
    });
    expect(kpisOf(trend, pictureOf(trend, NO_HINTS), null)).toEqual([
      { kind: "total", column: 1, delta: null },
      { kind: "peak", period: 0, measure: 1, row: 1 },
    ]);
  });
});

describe("trendPoints", () => {
  it("orders periods oldest first, whatever order the report sorted them in", () => {
    const trend = source({
      columns: [month, revenue, shipments],
      rows: [
        [300, "3", 30],
        [100, "1", null],
        [200, "2", 20],
      ],
    });
    expect(trendPoints(trend, { kind: "trend", period: 0, line: 1, columns: 2 })).toEqual([
      { row: 1, line: 1, column: null },
      { row: 2, line: 2, column: 20 },
      { row: 0, line: 3, column: 30 },
    ]);
  });
});
