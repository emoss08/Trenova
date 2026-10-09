import type { RowData } from "@tanstack/react-table";
import type { Cell, FilterVariant } from "@trenova/shared/types/data-table";
import type { SelectOption } from "@trenova/shared/types/fields";

/** The selected cells as a rectangle in display order; a gap in the selection is null. */
export type SelectedCellGrid<TData extends RowData> = {
  rows: (Cell<TData, unknown> | null)[][];
  columnIds: string[];
  /** Where the rectangle starts among the shown rows and visible columns. */
  anchorRow: number;
  anchorColumn: number;
};

/** One cell's change: what it holds now and what it is to hold. */
export type CellWrite<TData extends RowData> = {
  rowId: string;
  columnId: string;
  row: TData;
  value: unknown;
  previousValue: unknown;
};

/** Why a cell in the range was not written. */
export type CellWriteSkip = "read-only" | "unreadable";

export type CellWritePlan<TData extends RowData> = {
  writes: CellWrite<TData>[];
  skipped: Record<CellWriteSkip, number>;
};

type GridSource<TData extends RowData> = {
  rows: readonly { getVisibleCells: () => Cell<TData, unknown>[] }[];
  selectedIds: ReadonlySet<string>;
};

/**
 * The selection as rows and columns in the order the table shows them, from the first
 * row and column holding a selected cell to the last. A cell inside that rectangle the
 * person did not select is null, so a ctrl-click selection never writes between its parts.
 */
export function selectedCellGrid<TData extends RowData>({
  rows,
  selectedIds,
}: GridSource<TData>): SelectedCellGrid<TData> {
  const empty: SelectedCellGrid<TData> = { rows: [], columnIds: [], anchorRow: 0, anchorColumn: 0 };
  if (selectedIds.size === 0) return empty;

  let firstRow = -1;
  let minColumn = Number.POSITIVE_INFINITY;
  let maxColumn = -1;
  const pickedRows: Cell<TData, unknown>[][] = [];
  let columnIds: string[] = [];

  for (let rowIndex = 0; rowIndex < rows.length; rowIndex++) {
    const cells = rows[rowIndex].getVisibleCells();
    let rowHasSelection = false;
    for (let index = 0; index < cells.length; index++) {
      if (!selectedIds.has(cells[index].id)) continue;
      rowHasSelection = true;
      if (index < minColumn) minColumn = index;
      if (index > maxColumn) maxColumn = index;
    }
    if (rowHasSelection) {
      if (firstRow < 0) firstRow = rowIndex;
      pickedRows.push(cells);
      if (columnIds.length === 0) columnIds = cells.map((cell) => cell.column.id);
    }
  }

  if (pickedRows.length === 0) return empty;

  return {
    anchorRow: firstRow,
    anchorColumn: minColumn,
    columnIds: columnIds.slice(minColumn, maxColumn + 1),
    rows: pickedRows.map((cells) =>
      cells
        .slice(minColumn, maxColumn + 1)
        .map((cell) => (selectedIds.has(cell.id) ? cell : null)),
    ),
  };
}

/**
 * Splits pasted text into rows and cells as a spreadsheet copies them: tabs between
 * cells, line breaks between rows, and a quoted cell may hold either. The trailing
 * line break a spreadsheet adds is not an empty last row.
 */
export function parseClipboardGrid(text: string): string[][] {
  const rows: string[][] = [];
  let row: string[] = [];
  let cell = "";
  let quoted = false;
  let atCellStart = true;

  for (let index = 0; index < text.length; index++) {
    const char = text[index];
    if (quoted) {
      if (char === '"') {
        if (text[index + 1] === '"') {
          cell += '"';
          index++;
        } else {
          quoted = false;
        }
      } else {
        cell += char;
      }
      continue;
    }
    if (char === '"' && atCellStart) {
      quoted = true;
      atCellStart = false;
      continue;
    }
    if (char === "\t") {
      row.push(cell);
      cell = "";
      atCellStart = true;
      continue;
    }
    if (char === "\r" || char === "\n") {
      if (char === "\r" && text[index + 1] === "\n") index++;
      row.push(cell);
      rows.push(row);
      row = [];
      cell = "";
      atCellStart = true;
      continue;
    }
    cell += char;
    atCellStart = false;
  }

  if (cell !== "" || row.length > 0) {
    row.push(cell);
    rows.push(row);
  }
  return rows;
}

const TRUE_WORDS = new Set(["true", "yes", "y", "1", "on"]);
const FALSE_WORDS = new Set(["false", "no", "n", "0", "off"]);
const NUMBER_NOISE = /[\s,$€£¥%]/g;

type Coerced = { ok: true; value: unknown } | { ok: false };

/**
 * Reads pasted text as the value a column holds. Numbers lose currency marks and
 * thousands separators, a select matches an option by value or label, and a date
 * becomes the Unix seconds the table stores. Text that cannot be read is refused
 * rather than written as something else.
 */
export function coerceClipboardText(
  text: string,
  variant: FilterVariant,
  options?: readonly SelectOption[],
): Coerced {
  const trimmed = text.trim();
  if (trimmed === "") return variant === "boolean" ? { ok: false } : { ok: true, value: null };

  switch (variant) {
    case "number": {
      const negative = /^\(.*\)$/.test(trimmed);
      const parsed = Number(trimmed.replace(/[()]/g, "").replace(NUMBER_NOISE, ""));
      if (!Number.isFinite(parsed)) return { ok: false };
      return { ok: true, value: negative ? -parsed : parsed };
    }
    case "boolean": {
      const word = trimmed.toLowerCase();
      if (TRUE_WORDS.has(word)) return { ok: true, value: true };
      if (FALSE_WORDS.has(word)) return { ok: true, value: false };
      return { ok: false };
    }
    case "select": {
      const needle = trimmed.toLowerCase();
      const match = options?.find(
        (option) =>
          String(option.value).toLowerCase() === needle || option.label.toLowerCase() === needle,
      );
      return match ? { ok: true, value: match.value } : { ok: false };
    }
    case "date": {
      if (/^\d{9,11}$/.test(trimmed)) return { ok: true, value: Number(trimmed) };
      const parsed = Date.parse(trimmed);
      return Number.isFinite(parsed) ? { ok: true, value: Math.floor(parsed / 1000) } : { ok: false };
    }
    case "record":
      return { ok: false };
    default:
      return { ok: true, value: trimmed };
  }
}

function sameValue(a: unknown, b: unknown): boolean {
  return Object.is(a, b) || ((a === null || a === undefined) && (b === null || b === undefined));
}

function emptyPlan<TData extends RowData>(): CellWritePlan<TData> {
  return { writes: [], skipped: { "read-only": 0, unreadable: 0 } };
}

function addWrite<TData extends RowData>(
  plan: CellWritePlan<TData>,
  cell: Cell<TData, unknown>,
  value: unknown,
) {
  if (!cell.getCanEdit()) {
    plan.skipped["read-only"]++;
    return;
  }
  const previousValue = cell.getValue();
  if (sameValue(value, previousValue)) return;
  plan.writes.push({
    rowId: cell.row.id,
    columnId: cell.column.id,
    row: cell.row.original,
    value,
    previousValue,
  });
}

/** Copies the top selected cell of each column into every selected cell below it. */
export function planFillDown<TData extends RowData>(
  grid: SelectedCellGrid<TData>,
): CellWritePlan<TData> {
  const plan = emptyPlan<TData>();
  for (let column = 0; column < grid.columnIds.length; column++) {
    let source: Cell<TData, unknown> | null = null;
    for (const row of grid.rows) {
      const cell = row[column];
      if (!cell) continue;
      if (!source) {
        source = cell;
        continue;
      }
      addWrite(plan, cell, source.getValue());
    }
  }
  return plan;
}

type PasteTarget<TData extends RowData> = {
  /** Every row the table shows, in order, as cells in column order. */
  rows: readonly Cell<TData, unknown>[][];
  /** Where the selection starts. */
  anchorRow: number;
  anchorColumn: number;
  /** How far the selection reaches from the anchor. */
  selectionRows: number;
  selectionColumns: number;
};

/**
 * Lays pasted cells onto the table from the selection's top-left cell. A single value
 * fills the whole selection; a block whose size divides the selection is repeated to
 * fill it, as a spreadsheet does; anything else is pasted once at its own size and
 * stops at the table's last row and column.
 */
export function planPaste<TData extends RowData>(
  pasted: readonly string[][],
  target: PasteTarget<TData>,
): CellWritePlan<TData> {
  const plan = emptyPlan<TData>();
  const pastedRows = pasted.length;
  const pastedColumns = pasted.reduce((widest, row) => Math.max(widest, row.length), 0);
  if (pastedRows === 0 || pastedColumns === 0) return plan;

  const tiles =
    target.selectionRows % pastedRows === 0 && target.selectionColumns % pastedColumns === 0;
  const height = tiles ? Math.max(target.selectionRows, pastedRows) : pastedRows;
  const width = tiles ? Math.max(target.selectionColumns, pastedColumns) : pastedColumns;

  for (let rowOffset = 0; rowOffset < height; rowOffset++) {
    const cells = target.rows[target.anchorRow + rowOffset];
    if (!cells) break;
    const source = pasted[rowOffset % pastedRows];
    for (let columnOffset = 0; columnOffset < width; columnOffset++) {
      const cell = cells[target.anchorColumn + columnOffset];
      if (!cell) break;
      const text = source[columnOffset % pastedColumns] ?? "";
      const meta = cell.column.columnDef.meta;
      const coerced = coerceClipboardText(
        text,
        (meta?.filterType ?? "text") as FilterVariant,
        meta?.filterOptions as SelectOption[] | undefined,
      );
      if (!cell.getCanEdit()) {
        plan.skipped["read-only"]++;
        continue;
      }
      if (!coerced.ok) {
        plan.skipped.unreadable++;
        continue;
      }
      addWrite(plan, cell, coerced.value);
    }
  }
  return plan;
}

/** The same writes the other way round, to put every cell back as it was. */
export function invertWrites<TData extends RowData>(
  writes: readonly CellWrite<TData>[],
): CellWrite<TData>[] {
  return writes.map((write) => ({
    ...write,
    value: write.previousValue,
    previousValue: write.value,
  }));
}

export type CellWriteOutcome<TData extends RowData> = {
  applied: CellWrite<TData>[];
  failed: { write: CellWrite<TData>; message: string }[];
};

/** How many cells are saved at once, so a large paste does not flood the server. */
const WRITE_CONCURRENCY = 4;

/**
 * Saves each write through the table's commit, a few at a time, and says which were
 * saved and which were refused. One refusal never stops the rest.
 */
export async function applyCellWrites<TData extends RowData>(
  writes: readonly CellWrite<TData>[],
  commit: (write: CellWrite<TData>) => Promise<void> | void,
  onProgress?: (done: number) => void,
): Promise<CellWriteOutcome<TData>> {
  const outcome: CellWriteOutcome<TData> = { applied: [], failed: [] };
  let next = 0;
  let done = 0;

  const worker = async () => {
    while (next < writes.length) {
      const write = writes[next++];
      try {
        await commit(write);
        outcome.applied.push(write);
      } catch (error) {
        outcome.failed.push({
          write,
          message: error instanceof Error ? error.message : String(error),
        });
      }
      done++;
      onProgress?.(done);
    }
  };

  await Promise.all(
    Array.from({ length: Math.min(WRITE_CONCURRENCY, writes.length) }, () => worker()),
  );
  return outcome;
}
