import type { ReportPreviewColumn } from "@/lib/graphql/reports";
import { formatReportValue } from "@trenova/shared/lib/report-format";
import type { ReportIR } from "@/types/report";
import type { DrillTarget } from "./drill-through-sheet";

/**
 * Turns a clicked cell into the coordinates the server needs to rebuild the
 * record list: every grouping value on that row, plus the column that was
 * clicked so a filtered or pivoted measure carries its own scope down.
 *
 * Returns null when the report is already a record list — there is nothing
 * underneath a row that is itself a record.
 */
export function buildDrillTarget(
  ir: ReportIR,
  columns: ReportPreviewColumn[],
  row: unknown[],
  columnIndex: number,
): DrillTarget | null {
  if (!ir.columns.some((column) => column.kind !== "dimension")) return null;

  const indexById = new Map(columns.map((column, index) => [column.id, index]));
  const dimensions: { columnId: string; value: unknown }[] = [];

  for (const column of ir.columns) {
    if (column.kind !== "dimension") continue;
    const index = indexById.get(column.id);
    if (index === undefined) continue;
    dimensions.push({ columnId: column.id, value: row[index] ?? null });
  }

  const clicked = columns[columnIndex];
  const firstDimension = ir.columns.find((column) => column.kind === "dimension");
  const labelIndex = firstDimension ? indexById.get(firstDimension.id) : undefined;
  const labelColumn = labelIndex === undefined ? undefined : columns[labelIndex];

  return {
    dimensions,
    columnId: clicked?.id,
    columnLabel: clicked?.label,
    rowLabel:
      labelIndex === undefined
        ? undefined
        : formatReportValue(row[labelIndex], labelColumn?.display),
  };
}
