import type { RowData } from "@tanstack/react-table";
import type { Column } from "@trenova/shared/types/data-table";
import { escapeHtml } from "@trenova/shared/lib/html";
import { buildExportColumns, formatCsvValue, type ExportColumn } from "./data-table-export";

/** What goes on the clipboard: tab-separated text for any app, an HTML table for a spreadsheet. */
export type ClipboardGrid = {
  text: string;
  html: string;
};

/** One cell as plain text: tabs and line breaks would split it into more cells or rows. */
function clipboardText(value: unknown): string {
  return formatCsvValue(value).replace(/[\t\r\n]+/g, " ");
}

/**
 * How each column reads as text, the same as an export writes it: a column's own
 * export value when it has one, else what its accessor holds.
 */
export function clipboardColumns<TData extends RowData>(
  leafColumns: Column<TData, unknown>[],
): Map<string, ExportColumn<TData>> {
  return new Map(buildExportColumns(leafColumns, false).map((column) => [column.id, column]));
}

/**
 * A grid of rows and columns as clipboard text. A column with no text form (a
 * checkbox, an actions menu) is left out rather than copied as a blank.
 */
export function buildClipboardGrid<TData extends RowData>(
  rows: readonly TData[],
  columnIds: readonly string[],
  columns: Map<string, ExportColumn<TData>>,
  options: { includeHeader?: boolean } = {},
): ClipboardGrid {
  const copied = columnIds
    .map((id) => columns.get(id))
    .filter((column): column is ExportColumn<TData> => column !== undefined);

  const lines: string[] = [];
  const htmlRows: string[] = [];
  if (options.includeHeader) {
    lines.push(copied.map((column) => clipboardText(column.header)).join("\t"));
    htmlRows.push(
      `<tr>${copied.map((column) => `<th>${escapeHtml(column.header)}</th>`).join("")}</tr>`,
    );
  }
  for (const row of rows) {
    const values = copied.map((column) => clipboardText(column.getValue(row)));
    lines.push(values.join("\t"));
    htmlRows.push(`<tr>${values.map((value) => `<td>${escapeHtml(value)}</td>`).join("")}</tr>`);
  }

  return {
    text: lines.join("\n"),
    html: `<table>${htmlRows.join("")}</table>`,
  };
}

/** Puts a grid on the clipboard as both text and HTML, or as text where HTML is refused. */
export async function writeClipboardGrid(grid: ClipboardGrid): Promise<void> {
  if (typeof ClipboardItem !== "undefined" && navigator.clipboard?.write) {
    try {
      await navigator.clipboard.write([
        new ClipboardItem({
          "text/plain": new Blob([grid.text], { type: "text/plain" }),
          "text/html": new Blob([grid.html], { type: "text/html" }),
        }),
      ]);
      return;
    } catch {
      // Some browsers refuse HTML on the clipboard; plain text still pastes into a grid.
    }
  }
  await navigator.clipboard.writeText(grid.text);
}
