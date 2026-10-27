import { useRules } from "@trenova/shared/hooks/use-rules";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { formatReportColumns, reportCellText } from "@trenova/shared/rules/report";
import type { CSSProperties, ReactNode } from "react";
import { useMemo } from "react";
import { ArtIcon } from "../../desk-art-kinds";
import {
  numberAt,
  rankedTotal,
  shareOf,
  shownColumns,
  trendPoints,
  type Delta,
  type Kpi,
  type Picture,
  type SheetSource,
} from "./report-sheet";

/** Every value a sheet writes, formatted once per source. */
export type SheetText = {
  cell: (row: number, column: number) => string;
  total: (column: number) => string;
};

/**
 * Formats a sheet's rows and totals through the rules module in one call, the
 * way the Reports grid writes them: a run's sample by each column's resolved
 * display, a preview's by its catalog type and hint. Dates are written in the
 * organization's time zone.
 */
export function useSheetText(source: SheetSource, timeZone?: string): SheetText {
  const rules = useRules();

  return useMemo(() => {
    const height = source.rows.length;
    const formatted = formatReportColumns(
      rules,
      source.columns.map((column, at) => ({
        display: column.display,
        resolve:
          column.display || (column.dated && column.type !== "epoch")
            ? undefined
            : { type: column.type, hint: column.hint },
        valueType: column.type,
        values: [...source.rows.map((row) => row[at]), source.totals?.[at] ?? null],
      })),
    );
    const write = (column: number, row: number) => {
      const cell = formatted[column]?.[row];
      return cell ? reportCellText(cell, source.columns[column]?.display, timeZone) : "";
    };

    return {
      cell: (row, column) => write(column, row),
      total: (column) => (source.totals?.[column] == null ? "" : write(column, height)),
    };
  }, [rules, source, timeZone]);
}

const percent = new Intl.NumberFormat(undefined, { style: "percent", maximumFractionDigits: 1 });

/** A share or a change as a percentage: 0.082 as "8.2%". */
export function percentText(value: number): string {
  return percent.format(value);
}

/** The headline figures, each beside the same figure in the previous run. */
export function ReportKpis({
  source,
  kpis,
  text,
  since,
}: {
  source: SheetSource;
  kpis: readonly Kpi[];
  text: SheetText;
  /** When the previous run finished, as the deltas name it. */
  since: string;
}) {
  const t = useT();
  if (kpis.length === 0) {
    return null;
  }

  return (
    <div className="dk-rr-kpis" style={{ "--dk-rr-k": kpis.length } as CSSProperties}>
      {kpis.map((kpi) => {
        if (kpi.kind === "peak") {
          return (
            <div key="peak" className="dk-rr-kpi">
              <span>{t("Highest {0}", source.columns[kpi.measure]?.label ?? "")}</span>
              <b>{text.cell(kpi.row, kpi.period)}</b>
              <span className="dk-rr-delta">{text.cell(kpi.row, kpi.measure)}</span>
            </div>
          );
        }
        const label =
          kpi.kind === "rows"
            ? source.truncated
              ? t("Rows kept")
              : t("Rows")
            : (source.columns[kpi.column]?.label ?? "");
        const value =
          kpi.kind === "rows" ? source.rowCount.toLocaleString() : text.total(kpi.column);
        return (
          <div
            key={kpi.kind === "rows" ? "rows" : source.columns[kpi.column]?.id}
            className="dk-rr-kpi"
          >
            <span>{label}</span>
            <b title={value}>{value}</b>
            <KpiNote delta={kpi.delta} since={since} truncated={source.truncated} />
          </div>
        );
      })}
    </div>
  );
}

function KpiNote({
  delta,
  since,
  truncated,
}: {
  delta: Delta | null;
  since: string;
  truncated: boolean;
}) {
  const t = useT();
  if (truncated) {
    return <span className="dk-rr-delta">{t("Kept rows only")}</span>;
  }
  if (!delta || since === "") {
    return null;
  }
  const change = percentText(Math.abs(delta.change));

  return (
    <span className="dk-rr-delta" data-direction={delta.direction}>
      <em>
        {delta.direction !== "flat" && <ArtIcon name={delta.direction} size={10} stroke={2.4} />}
        {delta.direction === "flat" ? t("No change") : change}
      </em>
      {t("vs {0}", since)}
    </span>
  );
}

/**
 * The first page of the result as the file holds it: the picture the result
 * reads best as, its first rows, the totals under them, and how much more the
 * file holds.
 */
export function ReportSheet({
  source,
  picture,
  text,
  title,
  size,
  rows,
  footer,
}: {
  source: SheetSource;
  picture: Picture;
  text: SheetText;
  title: string;
  size: string;
  /** How many rows to show. */
  rows: number;
  footer: ReactNode;
}) {
  return (
    <div className="dk-rr-sheet">
      <div className="dk-rr-sheet-h">
        <b>{title}</b>
        <span>{size}</span>
      </div>
      {picture.kind === "trend" && <ReportTrend source={source} picture={picture} text={text} />}
      <SheetTable source={source} picture={picture} text={text} rows={rows} />
      {footer}
    </div>
  );
}

function SheetTable({
  source,
  picture,
  text,
  rows,
}: {
  source: SheetSource;
  picture: Picture;
  text: SheetText;
  rows: number;
}) {
  const t = useT();
  const columns = shownColumns(source, picture);
  const shown = source.rows.slice(0, rows);
  const ranked = picture.kind === "ranked" ? picture : null;
  const total = rankedTotal(source, picture);
  const max = ranked ? Math.max(0, ...shown.map((row) => numberAt(row, ranked.measure) ?? 0)) : 0;
  const widest = ranked
    ? Math.max(
        text.total(ranked.measure).length,
        ...shown.map((_, at) => text.cell(at, ranked.measure).length),
      )
    : 0;
  const fades = source.more || source.rowCount > shown.length;
  const hasTotals = source.totals !== null && source.totals.some((value) => value != null);

  const cellClass = (column: number) =>
    cn(
      source.columns[column]?.numeric && "dk-rr-num",
      columns.indexOf(column) >= 4 && "dk-rr-wide",
    );

  return (
    <table className={cn("dk-rr-tbl", fades && "dk-rr-fade")}>
      <thead>
        <tr>
          {columns.map((column) => (
            <th
              key={source.columns[column]?.id}
              className={cn(cellClass(column), ranked?.measure === column && "dk-rr-rank")}
            >
              {source.columns[column]?.label}
            </th>
          ))}
          {total !== null && <th className="dk-rr-num">{t("Share")}</th>}
        </tr>
      </thead>
      <tbody>
        {shown.map((row, at) => (
          <tr key={at}>
            {columns.map((column) =>
              ranked?.measure === column ? (
                <td key={column}>
                  <span
                    className="dk-rr-cbar"
                    style={{ "--dk-rr-figw": `${widest + 1}ch` } as CSSProperties}
                  >
                    <span className="dk-rr-tr">
                      <i
                        style={{
                          width: `${max > 0 ? ((numberAt(row, column) ?? 0) / max) * 100 : 0}%`,
                        }}
                      />
                    </span>
                    <span className="dk-rr-num">{text.cell(at, column)}</span>
                  </span>
                </td>
              ) : (
                <td key={column} className={cellClass(column)}>
                  {text.cell(at, column)}
                </td>
              ),
            )}
            {total !== null && ranked && (
              <td className="dk-rr-num dk-rr-mut">
                {shareText(shareOf(numberAt(row, ranked.measure), total))}
              </td>
            )}
          </tr>
        ))}
      </tbody>
      {hasTotals && (
        <tfoot>
          <tr>
            {columns.map((column, place) =>
              ranked?.measure === column ? (
                <td key={column}>
                  <span
                    className="dk-rr-cbar"
                    style={{ "--dk-rr-figw": `${widest + 1}ch` } as CSSProperties}
                  >
                    <span />
                    <span className="dk-rr-num">{text.total(column)}</span>
                  </span>
                </td>
              ) : (
                <td key={column} className={cellClass(column)}>
                  {place === 0 && !source.columns[column]?.numeric
                    ? source.truncated
                      ? t("The {0} rows kept", source.rowCount.toLocaleString())
                      : t("All {0} rows", source.rowCount.toLocaleString())
                    : text.total(column)}
                </td>
              ),
            )}
            {total !== null && <td className="dk-rr-num">{percentText(1)}</td>}
          </tr>
        </tfoot>
      )}
    </table>
  );
}

function shareText(share: number | null): string {
  return share === null ? "" : percentText(share);
}

const TREND_W = 1000;
const TREND_H = 168;

/** Periods along the bottom: one measure as columns, another as a line over them. */
function ReportTrend({
  source,
  picture,
  text,
}: {
  source: SheetSource;
  picture: Extract<Picture, { kind: "trend" }>;
  text: SheetText;
}) {
  const t = useT();
  const points = trendPoints(source, picture);
  const slot = TREND_W / Math.max(points.length, 1);
  const scale = (values: (number | null)[]) => {
    const top = Math.max(0, ...values.map((value) => value ?? 0));
    return (value: number) => TREND_H - 14 - (top > 0 ? (value / top) * (TREND_H - 28) : 0);
  };
  const lineY = scale(points.map((point) => point.line));
  const columnY = scale(points.map((point) => point.column));
  const line = points
    .filter((point) => point.line !== null)
    .map((point) => `${points.indexOf(point) * slot + slot / 2},${lineY(point.line ?? 0)}`)
    .join(" ");
  const lineLabel = source.columns[picture.line]?.label ?? "";
  const columnLabel =
    picture.columns === null ? "" : (source.columns[picture.columns]?.label ?? "");

  return (
    <div className="dk-rr-pic">
      <div className="dk-rr-pic-h">
        <span>
          {columnLabel
            ? t(
                "{0} and {1} by {2}",
                lineLabel,
                columnLabel,
                source.columns[picture.period]?.label ?? "",
              )
            : t("{0} by {1}", lineLabel, source.columns[picture.period]?.label ?? "")}
        </span>
        <span className="dk-rr-legend">
          {columnLabel && (
            <span>
              <i />
              {columnLabel}
            </span>
          )}
          <span>
            <i className="dk-ln" />
            {lineLabel}
          </span>
        </span>
      </div>
      <div className="dk-rr-trend">
        <svg
          viewBox={`0 0 ${TREND_W} ${TREND_H}`}
          preserveAspectRatio="none"
          role="img"
          aria-label={lineLabel}
        >
          {points.map((point, at) =>
            point.column === null ? null : (
              <rect
                key={point.row}
                className={cn("dk-col", at === points.length - 1 && "dk-last")}
                x={at * slot + slot * 0.22}
                y={columnY(point.column)}
                width={slot * 0.56}
                height={TREND_H - 14 - columnY(point.column)}
                rx={3}
              />
            ),
          )}
          <polyline className="dk-line" points={line} vectorEffect="non-scaling-stroke" />
        </svg>
      </div>
      <div
        className="dk-rr-ticks"
        style={{ gridTemplateColumns: `repeat(${points.length}, minmax(0, 1fr))` }}
      >
        {points.map((point) => (
          <span key={point.row}>{text.cell(point.row, picture.period)}</span>
        ))}
      </div>
    </div>
  );
}
