export type GridDirection = "up" | "down" | "left" | "right";

export type GridCellMove = {
  cell: HTMLTableCellElement;
  /** Set when the selection grows toward the cell rather than moving to it. */
  extend?: GridDirection;
};

type GridKey = Pick<KeyboardEvent, "key" | "shiftKey" | "ctrlKey" | "metaKey" | "altKey">;

const CELL = 'td[role="gridcell"]';
const ARROWS: Record<string, GridDirection> = {
  ArrowUp: "up",
  ArrowDown: "down",
  ArrowLeft: "left",
  ArrowRight: "right",
};

function gridRows(body: HTMLElement): HTMLTableRowElement[] {
  return Array.from(body.querySelectorAll<HTMLTableRowElement>("tr[data-row-index]"));
}

function cellsOf(row: HTMLTableRowElement): HTMLTableCellElement[] {
  return Array.from(row.querySelectorAll<HTMLTableCellElement>(`:scope > ${CELL}`));
}

function cellInColumn(row: HTMLTableRowElement | undefined, columnId: string | undefined) {
  if (!row || !columnId) return null;
  return cellsOf(row).find((cell) => cell.dataset.columnId === columnId) ?? null;
}

/**
 * Where a key moves focus within a grid of rows and cells, as the ARIA grid pattern
 * lays it out: arrows one cell, Home and End to the row's ends (with Ctrl or ⌘ to the
 * grid's corners), Page Up and Page Down a page of rows. Shift with an arrow grows the
 * selection instead. At an edge focus stays put, so the key never reaches the page.
 * Keys the grid does not move on return null.
 */
export function gridCellMove(
  event: GridKey,
  body: HTMLElement,
  from: HTMLTableCellElement,
  pageRows: number,
): GridCellMove | null {
  if (event.altKey) return null;
  const row = from.closest<HTMLTableRowElement>("tr[data-row-index]");
  if (!row) return null;
  const rows = gridRows(body);
  const rowAt = rows.indexOf(row);
  const cells = cellsOf(row);
  const columnAt = cells.indexOf(from);
  const columnId = from.dataset.columnId;
  const mod = event.ctrlKey || event.metaKey;

  const direction = ARROWS[event.key];
  if (direction) {
    if (mod) return null;
    let target: HTMLTableCellElement | null = null;
    if (direction === "left") target = cells[columnAt - 1] ?? null;
    else if (direction === "right") target = cells[columnAt + 1] ?? null;
    else target = cellInColumn(rows[rowAt + (direction === "down" ? 1 : -1)], columnId);
    if (!target) return { cell: from };
    return event.shiftKey ? { cell: target, extend: direction } : { cell: target };
  }

  switch (event.key) {
    case "Home": {
      const targetRow = mod ? rows[0] : row;
      const target = targetRow ? cellsOf(targetRow)[0] : undefined;
      return { cell: target ?? from };
    }
    case "End": {
      const targetRow = mod ? rows.at(-1) : row;
      const target = targetRow ? cellsOf(targetRow).at(-1) : undefined;
      return { cell: target ?? from };
    }
    case "PageDown":
    case "PageUp": {
      if (rows.length === 0) return { cell: from };
      const step = event.key === "PageDown" ? pageRows : -pageRows;
      const targetAt = Math.max(0, Math.min(rows.length - 1, rowAt + step));
      return { cell: cellInColumn(rows[targetAt], columnId) ?? from };
    }
    default:
      return null;
  }
}
