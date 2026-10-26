import type { TableConfig } from "@/types/table-configuration";
import type { Column, Header, Row } from "@trenova/shared/types/data-table";
import { describe, expect, it } from "vitest";
import {
  columnLayout,
  compileFormatRules,
  FORMAT_RULE_COLOR_CLASSES,
  fromColumnPinningState,
  isTableConfigEqual,
  stringifyUnknown,
  toColumnPinningState,
  withRequiredPinning,
} from "../data-table";

function makeConfig(overrides: Partial<TableConfig> = {}): TableConfig {
  return {
    fieldFilters: [],
    filterGroups: [],
    joinOperator: "and",
    sort: [],
    pageSize: 10,
    columnVisibility: {},
    columnOrder: [],
    columnSizing: {},
    columnPinning: { left: [], right: [] },
    density: "comfortable",
    formatRules: [],
    ...overrides,
  };
}

describe("isTableConfigEqual", () => {
  it("treats missing visibility entries as visible", () => {
    const a = makeConfig({ columnVisibility: { name: true, status: true } });
    const b = makeConfig({ columnVisibility: {} });

    expect(isTableConfigEqual(a, b)).toBe(true);
  });

  it("detects hidden column differences", () => {
    const a = makeConfig({ columnVisibility: { name: false } });
    const b = makeConfig({ columnVisibility: {} });

    expect(isTableConfigEqual(a, b)).toBe(false);
  });

  it("detects filter differences", () => {
    const a = makeConfig({
      fieldFilters: [{ field: "status", operator: "eq", value: "Active" }],
    });
    const b = makeConfig();

    expect(isTableConfigEqual(a, b)).toBe(false);
  });

  it("detects density and pinning differences", () => {
    expect(isTableConfigEqual(makeConfig({ density: "compact" }), makeConfig())).toBe(false);
    expect(
      isTableConfigEqual(
        makeConfig({ columnPinning: { left: ["name"], right: [] } }),
        makeConfig(),
      ),
    ).toBe(false);
  });

  it("detects sizing differences with missing keys", () => {
    const a = makeConfig({ columnSizing: { name: 200 } });
    const b = makeConfig({ columnSizing: {} });

    expect(isTableConfigEqual(a, b)).toBe(false);
    expect(isTableConfigEqual(a, makeConfig({ columnSizing: { name: 200 } }))).toBe(true);
  });
});

function makeLeafColumn(id: string, apiField: string): Column<Record<string, unknown>, unknown> {
  return {
    id,
    columnDef: {
      accessorKey: id,
      meta: { apiField },
    },
  } as unknown as Column<Record<string, unknown>, unknown>;
}

function makeRow(values: Record<string, unknown>): Row<Record<string, unknown>> {
  return {
    getValue: (columnId: string) => values[columnId],
  } as unknown as Row<Record<string, unknown>>;
}

describe("compileFormatRules", () => {
  const leafColumns = [makeLeafColumn("status", "status"), makeLeafColumn("amount", "amount")];

  it("returns null when there are no usable rules", () => {
    expect(compileFormatRules([], leafColumns)).toBeNull();
    expect(
      compileFormatRules(
        [{ id: "r1", field: "missing", operator: "eq", value: "x", color: "red" }],
        leafColumns,
      ),
    ).toBeNull();
  });

  it("matches eq rules case-insensitively", () => {
    const evaluate = compileFormatRules(
      [{ id: "r1", field: "status", operator: "eq", value: "Active", color: "green" }],
      leafColumns,
    );

    expect(evaluate?.(makeRow({ status: "active" }))).toContain("success");
    expect(evaluate?.(makeRow({ status: "Inactive" }))).toBeUndefined();
  });

  it("first matching rule wins", () => {
    const evaluate = compileFormatRules(
      [
        { id: "r1", field: "amount", operator: "gt", value: 100, color: "red" },
        { id: "r2", field: "amount", operator: "gt", value: 50, color: "amber" },
      ],
      leafColumns,
    );

    expect(evaluate?.(makeRow({ amount: 200 }))).toContain("danger");
    expect(evaluate?.(makeRow({ amount: 75 }))).toContain("warning");
    expect(evaluate?.(makeRow({ amount: 10 }))).toBeUndefined();
  });

  it("supports isnull and isnotnull", () => {
    const evaluate = compileFormatRules(
      [{ id: "r1", field: "status", operator: "isnull", value: null, color: "gray" }],
      leafColumns,
    );

    expect(evaluate?.(makeRow({ status: null }))).toBeDefined();
    expect(evaluate?.(makeRow({ status: "" }))).toBeDefined();
    expect(evaluate?.(makeRow({ status: "Active" }))).toBeUndefined();
  });

  it("ignores numeric comparisons against non-numeric values", () => {
    const evaluate = compileFormatRules(
      [{ id: "r1", field: "amount", operator: "lt", value: 5, color: "blue" }],
      leafColumns,
    );

    expect(evaluate?.(makeRow({ amount: "not-a-number" }))).toBeUndefined();
    expect(evaluate?.(makeRow({ amount: 3 }))).toContain("sky");
  });

  it("paints every colour opaque and deepens it when the row is hovered, on the cursor or opened", () => {
    const tones = {
      red: "danger",
      amber: "warning",
      green: "success",
      blue: "accent-sky",
      purple: "accent-violet",
      gray: "neutral",
    } as const;

    for (const [color, tone] of Object.entries(tones)) {
      expect(FORMAT_RULE_COLOR_CLASSES[color as keyof typeof tones].split(" ")).toEqual([
        `bg-${tone}-subtle`,
        `hover:bg-${tone}-subtle-hover`,
        `data-cursor:bg-${tone}-subtle-hover`,
        `data-expanded:bg-${tone}-subtle-hover`,
      ]);
    }
  });
});

describe("stringifyUnknown", () => {
  it("stringifies primitives without object coercion", () => {
    expect(stringifyUnknown("abc")).toBe("abc");
    expect(stringifyUnknown(42)).toBe("42");
    expect(stringifyUnknown(true)).toBe("true");
    expect(stringifyUnknown(null)).toBe("");
    expect(stringifyUnknown(undefined)).toBe("");
    expect(stringifyUnknown({ a: 1 })).toBe('{"a":1}');
    expect(stringifyUnknown([1, 2])).toBe("[1,2]");
  });
});

describe("column pinning wire <-> table state", () => {
  it("maps persisted left/right onto logical start/end", () => {
    expect(toColumnPinningState({ left: ["select", "pro"], right: ["actions"] })).toEqual({
      start: ["select", "pro"],
      end: ["actions"],
    });
  });

  it("maps logical start/end back onto persisted left/right", () => {
    expect(fromColumnPinningState({ start: ["select"], end: ["actions", "menu"] })).toEqual({
      left: ["select"],
      right: ["actions", "menu"],
    });
  });

  it("does not swap the two sides on a round trip", () => {
    const persisted = { left: ["a"], right: ["b"] };
    expect(fromColumnPinningState(toColumnPinningState(persisted))).toEqual(persisted);
  });

  it("keeps a one-sided pin on its own side", () => {
    expect(fromColumnPinningState(toColumnPinningState({ left: [], right: ["actions"] }))).toEqual({
      left: [],
      right: ["actions"],
    });
  });

  it("defaults absent pinning to empty sides", () => {
    expect(toColumnPinningState(undefined)).toEqual({ start: [], end: [] });
    expect(toColumnPinningState(null)).toEqual({ start: [], end: [] });
    expect(fromColumnPinningState(undefined)).toEqual({ left: [], right: [] });
  });
});

describe("withRequiredPinning", () => {
  it("keeps a table's required columns pinned under a saved view that pins nothing", () => {
    expect(
      withRequiredPinning({ left: [], right: [] }, { left: ["select", "lane"], right: [] }),
    ).toEqual({ left: ["select", "lane"], right: [] });
  });

  it("puts the required columns first and keeps the view's own pins after them", () => {
    expect(
      withRequiredPinning(
        { left: ["pro", "lane"], right: ["actions"] },
        { left: ["select", "lane"], right: [] },
      ),
    ).toEqual({ left: ["select", "lane", "pro"], right: ["actions"] });
  });

  it("moves a required column off the side the view pinned it to", () => {
    expect(
      withRequiredPinning({ left: [], right: ["lane"] }, { left: ["lane"], right: [] }),
    ).toEqual({ left: ["lane"], right: [] });
  });

  it("returns the view's pinning when the table requires none", () => {
    const pinning = { left: ["pro"], right: [] };
    expect(withRequiredPinning(pinning, undefined)).toEqual(pinning);
  });
});

describe("columnLayout", () => {
  function makeHeader(
    id: string,
    size: number,
    pinned: false | "start" | "end",
    offset = 0,
  ): Header<Record<string, unknown>> {
    return {
      getSize: () => size,
      column: {
        id,
        getIsPinned: () => pinned,
        getStart: () => offset,
        getAfter: () => offset,
      },
    } as unknown as Header<Record<string, unknown>>;
  }

  it("totals the columns pinned to each side apart from the ones that scroll", () => {
    const layout = columnLayout([
      makeHeader("select", 40, "start", 0),
      makeHeader("lane", 220, "start", 40),
      makeHeader("status", 120, false),
      makeHeader("pro", 160, false),
      makeHeader("actions", 48, "end", 0),
    ]);

    expect(layout.totalSize).toBe(588);
    expect(layout.pinnedStartSize).toBe(260);
    expect(layout.pinnedEndSize).toBe(48);
  });

  it("reports no pinned width when nothing is pinned", () => {
    const layout = columnLayout([makeHeader("status", 120, false), makeHeader("pro", 160, false)]);

    expect(layout.pinnedStartSize).toBe(0);
    expect(layout.pinnedEndSize).toBe(0);
  });
});
