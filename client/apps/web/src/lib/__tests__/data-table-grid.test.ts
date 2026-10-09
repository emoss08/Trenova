import { beforeEach, describe, expect, it } from "vitest";
import { gridCellMove } from "../data-table-grid";

const COLUMNS = ["select", "name", "status", "amount"];
let body: HTMLTableSectionElement;

function cell(row: number, column: string): HTMLTableCellElement {
  return body.querySelector<HTMLTableCellElement>(
    `tr[data-row-index="${row}"] td[data-column-id="${column}"]`,
  )!;
}

function key(name: string, modifiers: Partial<KeyboardEvent> = {}) {
  return { key: name, shiftKey: false, ctrlKey: false, metaKey: false, altKey: false, ...modifiers };
}

beforeEach(() => {
  const table = document.createElement("table");
  body = document.createElement("tbody");
  for (let row = 0; row < 25; row++) {
    const tr = document.createElement("tr");
    tr.dataset.rowIndex = String(row);
    tr.id = `r${row}`;
    for (const column of COLUMNS) {
      const td = document.createElement("td");
      td.setAttribute("role", "gridcell");
      td.dataset.columnId = column;
      tr.append(td);
    }
    body.append(tr);
    if (row === 3) {
      const expanded = document.createElement("tr");
      expanded.innerHTML = '<td colspan="4">details</td>';
      body.append(expanded);
    }
  }
  table.append(body);
  document.body.replaceChildren(table);
});

describe("gridCellMove", () => {
  it("moves one cell with each arrow, skipping rows that are not grid rows", () => {
    expect(gridCellMove(key("ArrowRight"), body, cell(0, "name"), 10)?.cell).toBe(cell(0, "status"));
    expect(gridCellMove(key("ArrowLeft"), body, cell(0, "name"), 10)?.cell).toBe(cell(0, "select"));
    expect(gridCellMove(key("ArrowDown"), body, cell(3, "name"), 10)?.cell).toBe(cell(4, "name"));
    expect(gridCellMove(key("ArrowUp"), body, cell(4, "name"), 10)?.cell).toBe(cell(3, "name"));
  });

  it("stays on the cell at an edge so the key never scrolls the page", () => {
    expect(gridCellMove(key("ArrowUp"), body, cell(0, "name"), 10)).toEqual({ cell: cell(0, "name") });
    expect(gridCellMove(key("ArrowRight"), body, cell(0, "amount"), 10)).toEqual({
      cell: cell(0, "amount"),
    });
  });

  it("grows the selection with Shift and an arrow", () => {
    expect(gridCellMove(key("ArrowDown", { shiftKey: true }), body, cell(1, "status"), 10)).toEqual({
      cell: cell(2, "status"),
      extend: "down",
    });
  });

  it("goes to the row's ends with Home and End and the grid's corners with a modifier", () => {
    expect(gridCellMove(key("Home"), body, cell(5, "status"), 10)?.cell).toBe(cell(5, "select"));
    expect(gridCellMove(key("End"), body, cell(5, "status"), 10)?.cell).toBe(cell(5, "amount"));
    expect(gridCellMove(key("Home", { ctrlKey: true }), body, cell(5, "status"), 10)?.cell).toBe(
      cell(0, "select"),
    );
    expect(gridCellMove(key("End", { metaKey: true }), body, cell(5, "status"), 10)?.cell).toBe(
      cell(24, "amount"),
    );
  });

  it("moves a page of rows with Page Up and Page Down, stopping at the ends", () => {
    expect(gridCellMove(key("PageDown"), body, cell(2, "name"), 10)?.cell).toBe(cell(12, "name"));
    expect(gridCellMove(key("PageDown"), body, cell(20, "name"), 10)?.cell).toBe(cell(24, "name"));
    expect(gridCellMove(key("PageUp"), body, cell(4, "name"), 10)?.cell).toBe(cell(0, "name"));
  });

  it("leaves other keys and Alt chords to the rest of the table", () => {
    expect(gridCellMove(key("Enter"), body, cell(0, "name"), 10)).toBeNull();
    expect(gridCellMove(key("ArrowDown", { altKey: true }), body, cell(0, "name"), 10)).toBeNull();
  });
});
