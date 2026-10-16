import { describe, expect, it } from "vitest";
import { defaultPageSizeFor, resolvePageSize } from "../use-data-table-state";

describe("table page size", () => {
  it("opens a table that names no sizes at 50 rows", () => {
    expect(defaultPageSizeFor()).toBe(50);
  });

  it("opens a table that names its sizes at the first of them", () => {
    expect(defaultPageSizeFor([25, 50, 100])).toBe(25);
  });

  it("keeps every size the default picker offers", () => {
    expect([25, 50, 100].map((size) => resolvePageSize(size))).toEqual([25, 50, 100]);
  });

  it("runs an old link's size the default picker no longer offers at 50 rows", () => {
    expect([10, 20, 30, 40].map((size) => resolvePageSize(size))).toEqual([50, 50, 50, 50]);
  });

  it("runs a size a table does not offer at that table's first size", () => {
    expect(resolvePageSize(10, [25, 50, 100])).toBe(25);
    expect(resolvePageSize(100, [25, 50, 100])).toBe(100);
  });
});
