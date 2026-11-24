import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { formatBoardValue } from "../board-card";
import type { DeskBoardMetricCell, DeskBoardMetricTable } from "./metric-types";

/** Whether a metric's table has a row to draw. */
export function tableDraws(
  table: DeskBoardMetricTable | null | undefined,
): table is DeskBoardMetricTable {
  return !!table && table.rows.length > 0 && table.columns.length > 0;
}

const MARKS = new Set(["hot", "off", "w"]);

function cellText(cell: DeskBoardMetricCell | undefined): string {
  if (!cell) return "–";
  if (cell.value) return formatBoardValue(cell.value);
  return cell.text === "" ? "–" : cell.text;
}

/**
 * A metric's small table, when its columns matter: the first column names the row, the rest are
 * values set right in the mono face; a cell is marked hot (it happened), off (nothing there) or
 * w (the one that crosses the line). Wider than its exhibit, it scrolls sideways inside it.
 */
export function BoardMetricTable({ table, title }: { table: DeskBoardMetricTable; title: string }) {
  const t = useT();
  return (
    <div className="dk-bd-scroll dk-bd-bleed">
      <table className="dk-tbl week dk-bd-mtbl" aria-label={title || t("Table")}>
        <thead>
          <tr>
            {table.columns.map((column, index) => (
              <th
                // oxlint-disable-next-line react/no-array-index-key -- columns keep their order
                key={index}
                scope="col"
                className={cn(column.kind === "text" && "dk-txt")}
              >
                {column.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {table.rows.map((row, rowIndex) => (
            // oxlint-disable-next-line react/no-array-index-key -- rows keep their order
            <tr key={rowIndex}>
              {table.columns.map((column, index) => {
                const cell = row[index];
                const mark = cell && MARKS.has(cell.mark) ? cell.mark : "";
                const empty = cellText(cell) === "–";
                return (
                  <td
                    // oxlint-disable-next-line react/no-array-index-key -- one cell per column
                    key={index}
                    className={cn(
                      column.kind === "text" && "dk-txt",
                      mark,
                      empty && mark === "" && "off",
                    )}
                  >
                    {cellText(cell)}
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
