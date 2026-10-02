import { formatDisplayValue } from "@/components/assistant/readable-values";
import { buildCsv, spreadsheetSafeText } from "@/lib/data-table-export";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { TableViewArtifact, TableViewRow } from "./artifact-payloads";

/**
 * A table artifact as CSV, cell for cell what a person reads in the pane: the
 * column labels, a status in words, a figure formatted, a date in their own
 * timezone, and a hole where the row has no value. The rows are the rows
 * given, so a filtered table copies what is shown. A record's id was never a
 * column and is not written. Free text that a spreadsheet would run as a
 * formula is kept as text.
 */
export function tableViewCsv(view: TableViewArtifact, t: TranslateFn): string {
  const columns = view.columns.map((column) => ({
    id: column.key,
    header: column.label,
    getValue: (row: TableViewRow) => {
      const value = row.values[column.key];
      if (value === undefined || value === null) {
        return "";
      }
      const text = formatDisplayValue(column.type, value, t);

      return column.type === "text" || column.type === "longText"
        ? (spreadsheetSafeText(text) ?? "")
        : text;
    },
  }));

  return buildCsv(view.rows, columns);
}
