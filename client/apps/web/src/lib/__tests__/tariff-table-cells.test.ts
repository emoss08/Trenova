import {
  emptyTableCellCache,
  mergeTableCells,
  uncachedNeeds,
  withTableCells,
} from "@/lib/tariff-table-cells";
import { rulesSync } from "@trenova/shared/rules/runtime";
import { tariffTest } from "@trenova/shared/rules/tariff";
import { tariffContextSchema, tariffTableCellsSchema } from "@trenova/shared/types/tariff";
import { describe, expect, it } from "vitest";

const GRID_SOURCE = `tariff "Grid"

let rate = tables.Grid[lane.destination.zip3, 100 lb] ?? fail "No rate"
charge "Linehaul" (quantity: 100 lb, rate: rate)

test "to A" {
  given lane.destination.zip3 = "a"
  expect total == $20
}

test "to B" {
  given lane.destination.zip3 = "B"
  expect fails "No rate"
}
`;

// What GET /formula-templates/context sends for a table with an exact axis: its shape, no cells
// (Go's nil slice is null) and no loaded combinations yet.
const context = tariffContextSchema.parse({
  libraries: [],
  tables: [
    {
      code: "Grid",
      name: "Zone grid",
      valueType: "Money/cwt",
      axes: [
        { match: "Exact", kind: "Zone", normalization: "Upper" },
        { match: "Range", kind: "WeightBreak", unit: "lb", overflow: "ClampToTopBand" },
      ],
      cells: null,
      loaded: [],
    },
    {
      code: "Curve",
      valueType: "Money",
      axes: [{ match: "Range", kind: "Distance", unit: "mi" }],
      cells: [{ at: [{ min: "0" }], value: "5" }],
      loaded: null,
    },
  ],
  codes: [],
  settings: { distanceUnit: "mi" },
});

// What POST /formula-templates/context/cells answers (tariff.TableCells in Go).
const answerForA = tariffTableCellsSchema.parse({
  stamp: "1:3:1767225600",
  tables: [
    {
      code: "Grid",
      cells: [{ at: [{ key: "A" }, { min: "0" }], value: "20" }],
      loaded: [["A"]],
    },
  ],
});
const answerForB = tariffTableCellsSchema.parse({
  stamp: "1:3:1767225600",
  tables: [{ code: "Grid", cells: [], loaded: [["B"]] }],
});

describe("the on-demand table protocol in the browser", () => {
  it("runs a file's tests to needs, then to a verdict once the cells are laid in", () => {
    const first = tariffTest(rulesSync(), { source: GRID_SOURCE, context });
    expect(first.tests.map((test) => test.status)).toEqual(["needs", "needs"]);
    expect(first.needs).toEqual([
      { table: "Grid", keys: ["A"] },
      { table: "Grid", keys: ["B"] },
    ]);

    let cache = emptyTableCellCache("1:3:1767225600");
    cache = mergeTableCells(cache, answerForA);
    cache = mergeTableCells(cache, answerForB);
    const second = tariffTest(rulesSync(), {
      source: GRID_SOURCE,
      context: withTableCells(context, cache),
    });
    expect(second.tests.map((test) => test.status)).toEqual(["passed", "passed"]);
    expect(second.needs).toEqual([]);
  });
});

describe("mergeTableCells", () => {
  it("adds each combination once, so a repeated answer does not double its cells", () => {
    const once = mergeTableCells(emptyTableCellCache(answerForA.stamp), answerForA);
    const twice = mergeTableCells(once, answerForA);
    expect(twice.tables.get("Grid")?.cells).toHaveLength(1);
    expect([...(twice.tables.get("Grid")?.loaded?.values() ?? [])]).toEqual([["A"]]);
  });

  it("starts over when the answer was read at another stamp", () => {
    const old = mergeTableCells(emptyTableCellCache(answerForA.stamp), answerForA);
    const moved = mergeTableCells(old, { ...answerForB, stamp: "1:4:1767312000" });
    expect(moved.stamp).toBe("1:4:1767312000");
    expect([...(moved.tables.get("Grid")?.loaded?.keys() ?? [])]).toEqual(["B"]);
  });

  it("keeps a table that came whole, and lays it into the context without combinations", () => {
    const whole = mergeTableCells(emptyTableCellCache(answerForA.stamp), {
      stamp: answerForA.stamp,
      tables: [{ code: "Grid", cells: answerForA.tables[0]?.cells ?? [], loaded: undefined }],
    });
    const after = mergeTableCells(whole, answerForB);
    expect(after.tables.get("Grid")?.loaded).toBeNull();
    const grid = withTableCells(context, after).tables.find((table) => table.code === "Grid");
    expect(grid?.loaded).toBeUndefined();
    expect(grid?.cells).toHaveLength(1);
  });
});

describe("uncachedNeeds and withTableCells", () => {
  it("asks only for combinations not loaded yet, each once", () => {
    const cache = mergeTableCells(emptyTableCellCache(answerForA.stamp), answerForA);
    expect(
      uncachedNeeds(cache, [
        { table: "Grid", keys: ["A"] },
        { table: "Grid", keys: ["B"] },
        { table: "Grid", keys: ["B"] },
        { table: "Lanes", keys: ["Z1", "Z2"] },
      ]),
    ).toEqual([
      { table: "Grid", keys: ["B"] },
      { table: "Lanes", keys: ["Z1", "Z2"] },
    ]);
  });

  it("leaves a table the context carries whole alone, and the context itself when nothing loaded", () => {
    expect(withTableCells(context, emptyTableCellCache(""))).toBe(context);
    const cache = mergeTableCells(emptyTableCellCache(answerForA.stamp), {
      stamp: answerForA.stamp,
      tables: [{ code: "Curve", cells: [], loaded: [["X"]] }],
    });
    const curve = withTableCells(context, cache).tables.find((table) => table.code === "Curve");
    expect(curve?.cells).toEqual([{ at: [{ min: "0" }], value: "5" }]);
    expect(curve?.loaded).toBeUndefined();
  });
});
