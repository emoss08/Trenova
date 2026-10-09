import { describe, expect, it, vi } from "vitest";
import type { Cell } from "@trenova/shared/types/data-table";
import {
  applyCellWrites,
  coerceClipboardText,
  invertWrites,
  parseClipboardGrid,
  planFillDown,
  planPaste,
  selectedCellGrid,
  type CellWrite,
} from "../data-table-cell-fill";

type Row = { id: string; status: string; weight: number | null; code: string };

type ColumnSpec = {
  id: keyof Row;
  editable: boolean;
  filterType?: string;
  filterOptions?: { value: string; label: string }[];
};

const COLUMNS: ColumnSpec[] = [
  { id: "code", editable: false },
  {
    id: "status",
    editable: true,
    filterType: "select",
    filterOptions: [
      { value: "Active", label: "Active" },
      { value: "Inactive", label: "Inactive" },
    ],
  },
  { id: "weight", editable: true, filterType: "number" },
];

function fakeCell(row: Row, spec: ColumnSpec): Cell<Row, unknown> {
  return {
    id: `${row.id}_${spec.id}`,
    row: { id: row.id, original: row },
    column: {
      id: spec.id,
      columnDef: { meta: { filterType: spec.filterType, filterOptions: spec.filterOptions } },
    },
    getValue: () => row[spec.id],
    getCanEdit: () => spec.editable,
  } as unknown as Cell<Row, unknown>;
}

function fakeRows(data: Row[]) {
  return data.map((row) => {
    const cells = COLUMNS.map((spec) => fakeCell(row, spec));
    return { getVisibleCells: () => cells };
  });
}

const DATA: Row[] = [
  { id: "r1", status: "Active", weight: 100, code: "A" },
  { id: "r2", status: "Inactive", weight: 200, code: "B" },
  { id: "r3", status: "Inactive", weight: null, code: "C" },
  { id: "r4", status: "Active", weight: 400, code: "D" },
];

describe("selectedCellGrid", () => {
  it("returns the selection's rectangle in display order with its anchor", () => {
    const grid = selectedCellGrid({
      rows: fakeRows(DATA),
      selectedIds: new Set(["r3_weight", "r2_status", "r2_weight", "r3_status"]),
    });

    expect(grid.anchorRow).toBe(1);
    expect(grid.anchorColumn).toBe(1);
    expect(grid.columnIds).toEqual(["status", "weight"]);
    expect(grid.rows.map((cells) => cells.map((cell) => cell?.id ?? null))).toEqual([
      ["r2_status", "r2_weight"],
      ["r3_status", "r3_weight"],
    ]);
  });

  it("leaves a cell between two separate selections empty", () => {
    const grid = selectedCellGrid({
      rows: fakeRows(DATA),
      selectedIds: new Set(["r1_status", "r2_weight"]),
    });

    expect(grid.rows.map((cells) => cells.map((cell) => cell?.id ?? null))).toEqual([
      ["r1_status", null],
      [null, "r2_weight"],
    ]);
  });

  it("returns nothing when nothing is selected", () => {
    expect(selectedCellGrid({ rows: fakeRows(DATA), selectedIds: new Set() }).rows).toEqual([]);
  });
});

describe("parseClipboardGrid", () => {
  it("splits tabs into cells and line breaks into rows, ignoring the trailing break", () => {
    expect(parseClipboardGrid("a\tb\r\nc\td\r\n")).toEqual([
      ["a", "b"],
      ["c", "d"],
    ]);
  });

  it("keeps a quoted cell's tab, line break and doubled quote", () => {
    expect(parseClipboardGrid('"one\ttwo\nthree ""x"""\tnext')).toEqual([
      ['one\ttwo\nthree "x"', "next"],
    ]);
  });

  it("keeps empty cells in place", () => {
    expect(parseClipboardGrid("\tb\t\n")).toEqual([["", "b", ""]]);
  });

  it("reads an empty clipboard as no rows", () => {
    expect(parseClipboardGrid("")).toEqual([]);
  });
});

describe("coerceClipboardText", () => {
  it("reads a number with currency marks, separators and accounting negatives", () => {
    expect(coerceClipboardText("$1,234.50", "number")).toEqual({ ok: true, value: 1234.5 });
    expect(coerceClipboardText("(25)", "number")).toEqual({ ok: true, value: -25 });
    expect(coerceClipboardText("abc", "number")).toEqual({ ok: false });
  });

  it("matches a select option by value or by label, ignoring case", () => {
    const options = [{ value: "A", label: "Available" }];
    expect(coerceClipboardText("available", "select", options)).toEqual({ ok: true, value: "A" });
    expect(coerceClipboardText("a", "select", options)).toEqual({ ok: true, value: "A" });
    expect(coerceClipboardText("Gone", "select", options)).toEqual({ ok: false });
  });

  it("reads a yes or no as a boolean and refuses a blank", () => {
    expect(coerceClipboardText("Yes", "boolean")).toEqual({ ok: true, value: true });
    expect(coerceClipboardText("0", "boolean")).toEqual({ ok: true, value: false });
    expect(coerceClipboardText("", "boolean")).toEqual({ ok: false });
  });

  it("reads a date as Unix seconds", () => {
    expect(coerceClipboardText("2026-01-02T00:00:00Z", "date")).toEqual({
      ok: true,
      value: 1767312000,
    });
    expect(coerceClipboardText("1767312000", "date")).toEqual({ ok: true, value: 1767312000 });
  });

  it("clears a cell for a blank and refuses a name pasted into a record column", () => {
    expect(coerceClipboardText("  ", "text")).toEqual({ ok: true, value: null });
    expect(coerceClipboardText("Acme", "record")).toEqual({ ok: false });
  });
});

describe("planFillDown", () => {
  it("copies each column's top cell into the cells below, skipping unchanged and read-only ones", () => {
    const grid = selectedCellGrid({
      rows: fakeRows(DATA),
      selectedIds: new Set(
        DATA.flatMap((row) => [`${row.id}_code`, `${row.id}_status`, `${row.id}_weight`]),
      ),
    });

    const plan = planFillDown(grid);

    expect(plan.skipped["read-only"]).toBe(3);
    expect(plan.writes.map((write) => [write.rowId, write.columnId, write.value])).toEqual([
      ["r2", "status", "Active"],
      ["r3", "status", "Active"],
      ["r2", "weight", 100],
      ["r3", "weight", 100],
      ["r4", "weight", 100],
    ]);
    expect(plan.writes[3].previousValue).toBeNull();
  });
});

describe("planPaste", () => {
  const rows = fakeRows(DATA).map((row) => row.getVisibleCells());

  it("lays a block onto the table from the anchor and stops at the last row", () => {
    const plan = planPaste([["Active", "7"], ["Inactive", "8"], ["Active", "9"]], {
      rows,
      anchorRow: 2,
      anchorColumn: 1,
      selectionRows: 1,
      selectionColumns: 1,
    });

    expect(plan.writes.map((write) => [write.rowId, write.columnId, write.value])).toEqual([
      ["r3", "status", "Active"],
      ["r3", "weight", 7],
      ["r4", "status", "Inactive"],
      ["r4", "weight", 8],
    ]);
  });

  it("fills a whole selection with a single value", () => {
    const plan = planPaste([["5"]], {
      rows,
      anchorRow: 0,
      anchorColumn: 2,
      selectionRows: 3,
      selectionColumns: 1,
    });

    expect(plan.writes.map((write) => [write.rowId, write.value])).toEqual([
      ["r1", 5],
      ["r2", 5],
      ["r3", 5],
    ]);
  });

  it("counts read-only cells and values that do not fit instead of writing them", () => {
    const plan = planPaste([["X", "Retired", "heavy"]], {
      rows,
      anchorRow: 0,
      anchorColumn: 0,
      selectionRows: 1,
      selectionColumns: 1,
    });

    expect(plan.writes).toEqual([]);
    expect(plan.skipped).toEqual({ "read-only": 1, unreadable: 2 });
  });
});

describe("applyCellWrites", () => {
  const writes: CellWrite<Row>[] = ["r1", "r2", "r3"].map((rowId) => ({
    rowId,
    columnId: "weight",
    row: DATA[0],
    value: 1,
    previousValue: 0,
  }));

  it("keeps going past a refused write and reports which were saved", async () => {
    const commit = vi.fn(async (write: CellWrite<Row>) => {
      if (write.rowId === "r2") throw new Error("Weight is locked");
    });
    const progress = vi.fn();

    const outcome = await applyCellWrites(writes, commit, progress);

    expect(commit).toHaveBeenCalledTimes(3);
    expect(outcome.applied.map((write) => write.rowId).sort()).toEqual(["r1", "r3"]);
    expect(outcome.failed).toEqual([{ write: writes[1], message: "Weight is locked" }]);
    expect(progress).toHaveBeenLastCalledWith(3);
  });

  it("puts every cell back with the writes inverted", () => {
    expect(invertWrites(writes.slice(0, 1))).toEqual([
      { rowId: "r1", columnId: "weight", row: DATA[0], value: 0, previousValue: 1 },
    ]);
  });
});
