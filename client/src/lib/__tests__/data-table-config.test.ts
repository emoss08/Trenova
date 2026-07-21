import type { TableConfig } from "@/types/table-configuration";
import type { Column, Row } from "@tanstack/react-table";
import { describe, expect, it } from "vitest";
import { compileFormatRules, isTableConfigEqual, stringifyUnknown } from "../data-table";

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
      isTableConfigEqual(makeConfig({ columnPinning: { left: ["name"], right: [] } }), makeConfig()),
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

    expect(evaluate?.(makeRow({ status: "active" }))).toContain("emerald");
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

    expect(evaluate?.(makeRow({ amount: 200 }))).toContain("red");
    expect(evaluate?.(makeRow({ amount: 75 }))).toContain("amber");
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
