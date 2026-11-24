import { Fragment, type CSSProperties } from "react";
import { LineBlank } from "../board-parts";
import { CHART_HEIGHTS, rankedHeight, spansHeight } from "./chart-values";

/** Rows a form reserves when neither the kept reply nor the catalog says how many. */
const DEFAULT_ROWS = 4;
/** Parts a split reserves keys for when it is not told. */
const DEFAULT_PARTS = 3;
/** Columns a grid or table sketch draws; the height does not depend on them. */
const SKETCH_COLUMNS = 7;

/**
 * The height a chart form draws at, in pixels, for `rows` rows (ranked bars and spans grow with
 * their rows; the plotted forms are one height). Null for a form that is not an SVG chart.
 */
export function chartSketchHeight(form: string, rows: number): number | null {
  switch (form) {
    case "trend":
      return CHART_HEIGHTS.trend;
    case "bars":
      return rows > 0 ? rankedHeight(rows) : CHART_HEIGHTS.bars;
    case "columns":
      return CHART_HEIGHTS.columns;
    case "dots":
      return CHART_HEIGHTS.dots;
    case "spans":
      return spansHeight(rows || DEFAULT_ROWS);
    default:
      return null;
  }
}

function Shimmers({ count, className }: { count: number; className?: string }) {
  return Array.from({ length: count }, (_, index) => (
    // oxlint-disable-next-line react/no-array-index-key -- shimmer slots are positional
    <i key={index} className={className ?? "ui-shimmer"} />
  ));
}

/**
 * A metric form while it is read, the same height as the form drawn: a chart's box at the
 * height its form draws; a grid, table, log or split in its own boxes with a blank in each, so
 * their fixed rows give the same height. `rows` is what the kept reply drew, or the catalog's
 * sketch, or zero for neither.
 */
export function FormSketch({ form, rows }: { form: string; rows: number }) {
  const height = chartSketchHeight(form, rows);
  if (height !== null) {
    return (
      <i
        className="dk-bd-wait-box ui-shimmer"
        data-sketch={form}
        style={{ height }}
        aria-hidden="true"
      />
    );
  }
  const count = rows || DEFAULT_ROWS;
  switch (form) {
    case "grid":
      return (
        <div className="dk-bd-scroll" data-sketch={form} aria-hidden="true">
          <div className="dk-bd-grid" style={{ "--n": SKETCH_COLUMNS } as CSSProperties}>
            <span className="gh-h" />
            {Array.from({ length: SKETCH_COLUMNS }, (_, index) => (
              // oxlint-disable-next-line react/no-array-index-key -- shimmer slots are positional
              <span key={index} className="gh-h">
                <LineBlank className="w-8" />
              </span>
            ))}
            {Array.from({ length: count }, (_, row) => (
              // oxlint-disable-next-line react/no-array-index-key -- shimmer rows are positional
              <Fragment key={row}>
                <span className="gh-a">
                  <LineBlank className="w-24" />
                </span>
                <Shimmers count={SKETCH_COLUMNS} />
              </Fragment>
            ))}
          </div>
        </div>
      );
    case "table":
      return (
        <div className="dk-bd-scroll dk-bd-bleed" data-sketch={form} aria-hidden="true">
          <table className="dk-tbl week dk-bd-mtbl">
            <thead>
              <tr>
                {Array.from({ length: SKETCH_COLUMNS }, (_, index) => (
                  // oxlint-disable-next-line react/no-array-index-key -- shimmer slots are positional
                  <th key={index} className={index === 0 ? "dk-txt" : undefined}>
                    <LineBlank className="w-10" />
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {Array.from({ length: count }, (_, row) => (
                // oxlint-disable-next-line react/no-array-index-key -- shimmer rows are positional
                <tr key={row}>
                  {Array.from({ length: SKETCH_COLUMNS }, (_, index) => (
                    // oxlint-disable-next-line react/no-array-index-key -- shimmer slots are positional
                    <td key={index} className={index === 0 ? "dk-txt" : undefined}>
                      <LineBlank className={index === 0 ? "w-24" : "w-8"} />
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      );
    case "log":
      return (
        <ol className="dk-bd-log dk-bd-mlog" data-sketch={form} aria-hidden="true">
          {Array.from({ length: count }, (_, index) => (
            // oxlint-disable-next-line react/no-array-index-key -- shimmer rows are positional
            <li key={index}>
              <time>
                <LineBlank className="w-10" />
              </time>
              <i />
              <div>
                <span>
                  <LineBlank className="w-3/4" />
                </span>
                <em>
                  <LineBlank className="w-1/2" />
                </em>
              </div>
            </li>
          ))}
        </ol>
      );
    case "split":
      return (
        <div className="dk-bd-whole" data-sketch={form} aria-hidden="true">
          <b className="dk-bd-big">
            <LineBlank className="w-40" />
          </b>
          <i className="dk-bd-wait-strip ui-shimmer" />
          <div
            className={(rows || DEFAULT_PARTS) > 3 ? "dk-bd-keys dk-bd-keys-many" : "dk-bd-keys"}
            style={{ "--n": rows || DEFAULT_PARTS } as CSSProperties}
          >
            {Array.from({ length: rows || DEFAULT_PARTS }, (_, index) => (
              // oxlint-disable-next-line react/no-array-index-key -- shimmer slots are positional
              <span key={index}>
                <LineBlank className="w-24" />
              </span>
            ))}
          </div>
        </div>
      );
    default:
      return (
        <div className="dk-bd-wait dk-rows" data-sketch="rows" data-rows={count}>
          <Shimmers count={count} />
        </div>
      );
  }
}
