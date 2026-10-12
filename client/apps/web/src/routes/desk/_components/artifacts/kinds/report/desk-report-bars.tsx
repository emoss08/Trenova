import {
  formatDisplayValue,
  isFigureType,
  type DisplayColumn,
} from "@/components/assistant/readable-values";
import type { AssistantArtifact } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo } from "react";
import { Link } from "react-router";
import { artifactCsvUrl } from "@/services/assistant";
import { downloadFromUrl } from "@trenova/shared/lib/utils";
import { reportPreviewFrom } from "../../artifact-payloads";
import { ArtIcon } from "../../desk-art-kinds";
import { translateLabel } from "@trenova/shared/i18n/labels";
import { gridOf, type Grid, type GridRow } from "../table/desk-table-body";

/** A report short enough to read as bars: its label column, and the figures beside it. */
type Bars = { label: DisplayColumn; bar: DisplayColumn; extra: DisplayColumn | null };

/**
 * Whether a report reads best as bars, the way the design draws a summary:
 * one label per row, a figure to measure them by, and a dozen rows or fewer,
 * all of them present and none below zero.
 */
export function barsOf(grid: Grid): Bars | null {
  const [label, ...rest] = grid.columns;
  const figures = rest.filter((column) => isFigureType(column.type));
  if (!label || isFigureType(label.type) || figures.length === 0) {
    return null;
  }
  if (grid.truncated || grid.rows.length < 2 || grid.rows.length > 12) {
    return null;
  }
  const [bar, extra = null] = figures;
  const measured = grid.rows.every((row) => {
    const figure = Number(row.values[bar.key]);
    return row.values[bar.key] != null && Number.isFinite(figure) && figure >= 0;
  });
  return measured ? { label, bar, extra } : null;
}

/** A short report as bars: each row measured against the largest, and the totals under them. */
export function DeskReportBars({ artifact, bars }: { artifact: AssistantArtifact; bars: Bars }) {
  const t = useT();
  const grid = useMemo(() => gridOf(artifact), [artifact]);
  const figure = (row: GridRow, column: DisplayColumn) => Number(row.values[column.key]) || 0;
  const max = Math.max(...grid.rows.map((row) => figure(row, bars.bar)), 0);
  const sum = (column: DisplayColumn) =>
    grid.totals?.[column.key] ?? grid.rows.reduce((total, row) => total + figure(row, column), 0);
  const columns = bars.extra ? [bars.bar, bars.extra] : [bars.bar];

  const preview = reportPreviewFrom(artifact);
  const definitionId =
    typeof artifact.payload.definitionId === "string" ? artifact.payload.definitionId : "";
  const fullReport = definitionId
    ? `/reports/explore/${definitionId}`
    : `/desk/c/${artifact.threadId}/a/${artifact.slug || artifact.lineageId || artifact.id}/page`;

  return (
    <div className="dk-ax-pad dk-ax-rep">
      <div className="dk-ax-repm">
        <span>
          {humanizeDataset(preview.dataset) || columns.map((column) => column.label).join(" · ")}
        </span>
        <span>
          {t(
            "{0, plural, one {# row} other {# rows}} · top {1} by {2}",
            grid.rowCount,
            grid.rows.length,
            bars.bar.label.toLowerCase(),
          )}
        </span>
      </div>
      <div className={cn("dk-ax-bars", !bars.extra && "dk-one")}>
        {grid.rows.map((row, index) => (
          <div key={row.key} className="dk-ax-br" style={{ animationDelay: `${index * 60}ms` }}>
            <span className="dk-ax-bn">
              {formatDisplayValue(bars.label.type, row.values[bars.label.key], t)}
            </span>
            <span className="dk-ax-bt">
              <i
                style={{
                  width: `${max > 0 ? (figure(row, bars.bar) / max) * 100 : 0}%`,
                  animationDelay: `${120 + index * 60}ms`,
                }}
              />
            </span>
            {columns.map((column, at) => (
              <span key={column.key} className={cn("dk-ax-num", at > 0 && "dk-mut")}>
                {formatDisplayValue(column.type, row.values[column.key], t)}
              </span>
            ))}
          </div>
        ))}
        <div className="dk-ax-br dk-tot">
          <span className="dk-ax-bn">{t("Total")}</span>
          <span />
          {columns.map((column) => (
            <span key={column.key} className="dk-ax-num">
              {column.type === "percent" ? "" : formatDisplayValue(column.type, sum(column), t)}
            </span>
          ))}
        </div>
      </div>
      <div className="dk-ax-acts">
        <Button
          variant="quiet"
          onClick={() => downloadFromUrl(artifactCsvUrl(artifact.threadId, artifact.id))}
        >
          <ArtIcon name="dl" size={13} />
          {t("Download CSV")}
        </Button>
        <Link
          className="dk-ax-btn dk-ghost"
          to={fullReport}
          target={definitionId ? undefined : "_blank"}
        >
          <ArtIcon name="ext" size={13} />
          {t("Open full report")}
        </Link>
      </div>
    </div>
  );
}

/** A dataset's key as a reader names it: "shipment_stops" as "Shipment stops". */
function humanizeDataset(dataset: string): string {
  const words = dataset.replace(/[_.-]+/gu, " ").trim();
  return words ? translateLabel(words.charAt(0).toUpperCase() + words.slice(1)) : "";
}
