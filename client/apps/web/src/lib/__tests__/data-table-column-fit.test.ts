import { describe, expect, it, vi } from "vitest";
import type { Column } from "@trenova/shared/types/data-table";
import {
  MAX_FITTED_COLUMN_WIDTH,
  clampFittedWidth,
  columnSizeVar,
  isColumnResizable,
  measureColumnFits,
} from "../data-table";

type Row = { id: string };

function columnWith(def: { minSize?: number; maxSize?: number }, canResize = true) {
  return { columnDef: def, getCanResize: () => canResize } as unknown as Column<Row>;
}

function tableWithHeads(widths: Record<string, number>) {
  const table = document.createElement("table");
  const row = document.createElement("tr");
  for (const [id, width] of Object.entries(widths)) {
    const head = document.createElement("th");
    head.dataset.columnId = id;
    head.getBoundingClientRect = () => {
      const fitted = table.style.getPropertyValue(columnSizeVar(id)) === "max-content";
      return { width: fitted ? width : 0 } as DOMRect;
    };
    row.append(head);
  }
  table.append(row);
  return table;
}

describe("measureColumnFits", () => {
  it("reads each head while its column is let go to its content", () => {
    const table = tableWithHeads({ name: 212.4, status: 96 });

    expect(measureColumnFits(table, ["name", "status"])).toEqual({ name: 213, status: 96 });
  });

  it("puts every width back as it was, including one that was never set", () => {
    const table = tableWithHeads({ name: 200, status: 90 });
    table.style.setProperty(columnSizeVar("name"), "150px");

    measureColumnFits(table, ["name", "status"]);

    expect(table.style.getPropertyValue(columnSizeVar("name"))).toBe("150px");
    expect(table.style.getPropertyValue(columnSizeVar("status"))).toBe("");
  });

  it("leaves out a column with no head on the page", () => {
    const table = tableWithHeads({ name: 200 });

    expect(measureColumnFits(table, ["name", "missing"])).toEqual({ name: 200 });
  });

  it("puts the widths back even when reading a head throws", () => {
    const table = tableWithHeads({ name: 200 });
    table.style.setProperty(columnSizeVar("name"), "150px");
    const head = table.querySelector("th")!;
    head.getBoundingClientRect = vi.fn(() => {
      throw new Error("layout failed");
    });

    expect(() => measureColumnFits(table, ["name"])).toThrow("layout failed");
    expect(table.style.getPropertyValue(columnSizeVar("name"))).toBe("150px");
  });
});

describe("clampFittedWidth", () => {
  it("keeps a fit inside the column's own bounds", () => {
    expect(clampFittedWidth(columnWith({ minSize: 120, maxSize: 300 }), 80)).toBe(120);
    expect(clampFittedWidth(columnWith({ minSize: 120, maxSize: 300 }), 420)).toBe(300);
  });

  it("never fits a column wider than a table can show", () => {
    expect(clampFittedWidth(columnWith({}), 5000)).toBe(MAX_FITTED_COLUMN_WIDTH);
  });
});

describe("isColumnResizable", () => {
  it("refuses a column fixed at one width or one the table will not resize", () => {
    expect(isColumnResizable(columnWith({ minSize: 40, maxSize: 40 }))).toBe(false);
    expect(isColumnResizable(columnWith({}, false))).toBe(false);
    expect(isColumnResizable(columnWith({ minSize: 40, maxSize: 400 }))).toBe(true);
  });
});
