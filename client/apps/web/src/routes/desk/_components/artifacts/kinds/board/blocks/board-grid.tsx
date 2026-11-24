import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Fragment, type CSSProperties } from "react";
import { signedCount } from "./chart-values";
import type { DeskBoardGrid, DeskBoardGridCell } from "./metric-types";

/** Whether a grid has a row to draw. */
export function gridDraws(grid: DeskBoardGrid | null | undefined): grid is DeskBoardGrid {
  return !!grid && grid.rows.length > 0;
}

const CELL_CLASS: Record<string, string> = {
  short: "short",
  deep: "short deep",
  maybe: "maybe",
};

function count(value: number): string {
  return new Intl.NumberFormat(undefined, { maximumFractionDigits: 0 }).format(value);
}

/** A cell's balance in words: what it has against what it needs, or the range it may fall in. */
function balance(cell: DeskBoardGridCell, t: TranslateFn): { head: string; foot: string } {
  const ranged = cell.needHigh > cell.need;
  if (cell.state === "maybe" && ranged) {
    return {
      head: t(
        "{0} to {1}",
        signedCount(cell.have - cell.need),
        signedCount(cell.have - cell.needHigh),
      ),
      foot: t("{0}/{1}–{2}", count(cell.have), count(cell.need), count(cell.needHigh)),
    };
  }
  return {
    head: signedCount(cell.have - cell.need),
    foot: t("{0}/{1}", count(cell.have), count(cell.need)),
  };
}

/** What the grid shows and its headline number: how many cells are short, and the worst. */
export function gridLabel(grid: DeskBoardGrid, title: string, t: TranslateFn): string {
  const cells = grid.rows.flatMap((row) =>
    row.cells.map((cell, column) => ({ row: row.label, column, cell })),
  );
  const short = cells.filter(({ cell }) => cell.state === "short" || cell.state === "deep");
  const maybe = cells.filter(({ cell }) => cell.state === "maybe").length;
  if (short.length === 0) {
    return maybe === 0
      ? t("{0}: all {1, plural, one {# cell} other {# cells}} covered", title, cells.length)
      : t(
          "{0}: none short, {1, plural, one {# may be short} other {# may be short}}",
          title,
          maybe,
        );
  }
  const worst = short.reduce((low, entry) =>
    entry.cell.have - entry.cell.need < low.cell.have - low.cell.need ? entry : low,
  );
  const head = grid.columns[worst.column];
  return t(
    "{0}: {1} of {2} short, {3} may be short; the worst {4} short, {5} at {6}",
    title,
    short.length,
    cells.length,
    maybe,
    count(worst.cell.need - worst.cell.have),
    worst.row,
    head ? head.label : "",
  );
}

/**
 * Cover against need: a row a place, a column a slot, each cell what it has against what it
 * needs. Covered cells are plain, short ones hatched in warn, the deepest outlined too, and a
 * cell that may be short only outlined. Columns keep a width that holds their values; when the
 * exhibit is narrower than them, the grid scrolls sideways inside it.
 */
export function BoardGrid({ grid, title }: { grid: DeskBoardGrid; title: string }) {
  const t = useT();
  return (
    <div className="dk-bd-scroll">
      <div
        className="dk-bd-grid"
        role="img"
        aria-label={gridLabel(grid, title, t)}
        style={{ "--n": grid.columns.length } as CSSProperties}
      >
        <span className="gh-h" aria-hidden="true" />
        {grid.columns.map((column, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- columns keep their order
          <span key={index} className="gh-h">
            <b>{column.label}</b>
            {column.sublabel}
          </span>
        ))}
        {grid.rows.map((row, rowIndex) => (
          // oxlint-disable-next-line react/no-array-index-key -- rows keep their order
          <Fragment key={rowIndex}>
            <span className="gh-a" title={row.label}>
              {row.label}
            </span>
            {row.cells.map((cell, index) => {
              const text = balance(cell, t);
              return (
                // oxlint-disable-next-line react/no-array-index-key -- a row's cells keep their order
                <span key={index} className={cn("gc", CELL_CLASS[cell.state])}>
                  <b>{text.head}</b>
                  <small>{text.foot}</small>
                </span>
              );
            })}
          </Fragment>
        ))}
      </div>
    </div>
  );
}
