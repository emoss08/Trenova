import { DisplayValue } from "@/components/assistant/display-value";
import {
  formatDisplayValue,
  isDetailType,
  isFigureType,
  sortKey,
  statusPhase,
  type DisplayColumn,
  type DisplayType,
} from "@/components/assistant/readable-values";
import type { AssistantArtifact } from "@/types/assistant";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo, useState, type ReactNode } from "react";
import { Link } from "react-router";
import { reportPreviewFrom, tableViewFrom } from "./artifact-payloads";
import { ArtIcon } from "./desk-art-kinds";

/** A table the workspace draws: columns, rows by key, and where each row opens. */
type GridRow = { key: string; values: Record<string, unknown>; path: string };
type Grid = {
  columns: DisplayColumn[];
  rows: GridRow[];
  rowCount: number;
  truncated: boolean;
  /** The row each total sits under, when the source sent one. */
  totals: Record<string, unknown> | null;
};

const REPORT_TYPES: Record<string, DisplayType> = {
  currency: "money",
  money: "money",
  number: "number",
  integer: "number",
  decimal: "number",
  percent: "percent",
  date: "date",
  datetime: "datetime",
  timestamp: "datetime",
  boolean: "boolean",
};

/** Either a list result or a report preview, read into one shape. */
export function gridOf(artifact: AssistantArtifact): Grid {
  if (artifact.kind === "report_preview") {
    const preview = reportPreviewFrom(artifact);
    const columns = preview.columns.map<DisplayColumn>((column) => ({
      key: column.id,
      label: column.label,
      type: REPORT_TYPES[column.type] ?? "text",
    }));
    const rows = preview.rows.map((cells, index) => ({
      key: String(index),
      path: "",
      values: Object.fromEntries(columns.map((column, at) => [column.key, cells[at]])),
    }));
    return {
      columns,
      rows,
      rowCount: preview.rowCount,
      truncated: preview.truncated || preview.rowCount > rows.length,
      totals: preview.totals
        ? Object.fromEntries(columns.map((column, at) => [column.key, preview.totals?.[at]]))
        : null,
    };
  }
  const view = tableViewFrom(artifact);
  return {
    columns: view.columns.filter((column) => !isDetailType(column.type)),
    rows: view.rows.map((row) => ({ ...row, key: row.path || row.key })),
    rowCount: view.rowCount,
    truncated: view.truncated,
    totals: null,
  };
}

/** The cells that differ from the version before, by row and column. */
export function changedCells(current: Grid, previous: Grid | null): Set<string> {
  const changed = new Set<string>();
  if (!previous) {
    return changed;
  }
  const before = new Map(previous.rows.map((row) => [row.key, row]));
  for (const row of current.rows) {
    const old = before.get(row.key);
    if (!old) {
      continue;
    }
    for (const column of current.columns) {
      if (
        JSON.stringify(old.values[column.key] ?? null) !==
        JSON.stringify(row.values[column.key] ?? null)
      ) {
        changed.add(`${row.key}:${column.key}`);
      }
    }
  }
  return changed;
}

const PHASE_DOT: Record<string, string> = {
  failed: "dk-warn",
  attention: "dk-warn",
  awaiting: "dk-blue",
  active: "dk-blue",
  queued: "dk-blue",
  complete: "dk-green",
  closed: "dk-ink",
  draft: "",
};

function Cell({ column, value, first }: { column: DisplayColumn; value: unknown; first: boolean }) {
  const t = useT();
  if (value === null || value === undefined || value === "") {
    return <span className="dk-ax-none">—</span>;
  }
  if (column.type === "enum") {
    return <>{formatDisplayValue(column.type, value, t)}</>;
  }
  if (column.type === "status") {
    const text = formatDisplayValue(column.type, value, t);
    const phase = typeof value === "string" ? statusPhase(value) : null;
    return (
      <span className={cn("dk-ax-pill", phase ? PHASE_DOT[phase] : "")}>
        <i />
        {text}
      </span>
    );
  }
  if (isFigureType(column.type)) {
    return <span className="dk-ax-num">{formatDisplayValue(column.type, value, t)}</span>;
  }
  if (first) {
    return <span className="dk-ax-id">{formatDisplayValue(column.type, value, t)}</span>;
  }
  return <DisplayValue type={column.type} value={value} label={column.label} inline />;
}

function cellText(column: DisplayColumn, value: unknown, t: TranslateFn): string {
  return value === null || value === undefined ? "" : formatDisplayValue(column.type, value, t);
}

/** A value's raw form, so a status is found by its code as well as its words. */
function rawText(value: unknown): string {
  return typeof value === "string" || typeof value === "number" ? String(value) : "";
}

type Sort = { key: string; direction: 1 | -1 };

/**
 * A table read whole: a filter over every cell as it reads, sortable columns,
 * figures on the right, statuses as coloured dots, and each row opening its
 * record. A cell that changed since the version before is marked.
 */
export function DeskTableBody({
  artifact,
  previous,
  versions,
}: {
  artifact: AssistantArtifact;
  /** The version before this one, for marking what changed. */
  previous: AssistantArtifact | null;
  /** The versions picker, when the table has more than one. */
  versions?: ReactNode;
}) {
  const t = useT();
  const grid = useMemo(() => gridOf(artifact), [artifact]);
  const changed = useMemo(
    () => changedCells(grid, previous ? gridOf(previous) : null),
    [grid, previous],
  );
  const [query, setQuery] = useState("");
  const [sort, setSort] = useState<Sort | null>(null);

  const rows = useMemo(() => {
    const needle = query.trim().toLowerCase();
    let shown = needle
      ? grid.rows.filter((row) =>
          grid.columns.some((column) =>
            `${cellText(column, row.values[column.key], t)} ${rawText(row.values[column.key])}`
              .toLowerCase()
              .includes(needle),
          ),
        )
      : grid.rows;
    if (sort) {
      const column = grid.columns.find((candidate) => candidate.key === sort.key);
      if (column) {
        shown = [...shown].sort((a, b) => {
          const x = sortKey(column.type, a.values[column.key]);
          const y = sortKey(column.type, b.values[column.key]);
          if (x === null) return 1;
          if (y === null) return -1;
          return (
            (typeof x === "number" && typeof y === "number"
              ? x - y
              : String(x).localeCompare(String(y))) * sort.direction
          );
        });
      }
    }
    return shown;
  }, [grid, query, sort, t]);

  const cycle = (key: string) =>
    setSort((current) =>
      current?.key === key
        ? current.direction === 1
          ? { key, direction: -1 }
          : null
        : { key, direction: 1 },
    );

  return (
    <div className="dk-ax-tbl">
      <div className="dk-ax-tools">
        <label className="dk-ax-q">
          <ArtIcon name="search" size={13} />
          <input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={t("Filter {0, plural, one {# row} other {# rows}}", rows.length)}
            aria-label={t("Filter rows")}
          />
        </label>
        {versions}
      </div>
      <div className="dk-ax-scroll">
        <table>
          <thead>
            <tr>
              {grid.columns.map((column) => (
                <th
                  key={column.key}
                  className={isFigureType(column.type) ? "dk-ar" : undefined}
                  onClick={() => cycle(column.key)}
                  aria-sort={
                    sort?.key === column.key
                      ? sort.direction === 1
                        ? "ascending"
                        : "descending"
                      : undefined
                  }
                >
                  <span>
                    {column.label}
                    {sort?.key === column.key && (
                      <ArtIcon name={sort.direction === 1 ? "up" : "down"} size={10} stroke={2.4} />
                    )}
                  </span>
                </th>
              ))}
              <th className="dk-go" />
            </tr>
          </thead>
          <tbody>
            {rows.map((row, index) => (
              <tr
                key={`${row.key}:${artifact.id}`}
                style={{ animationDelay: `${Math.min(index, 14) * 18}ms` }}
              >
                {grid.columns.map((column, at) => (
                  <td
                    key={column.key}
                    className={cn(
                      isFigureType(column.type) && "dk-ar",
                      changed.has(`${row.key}:${column.key}`) && "dk-chg",
                    )}
                  >
                    <Cell column={column} value={row.values[column.key]} first={at === 0} />
                  </td>
                ))}
                <td className="dk-go">
                  {row.path !== "" && (
                    <Link to={row.path} aria-label={t("Open this record")}>
                      <ArtIcon name="ext" size={12} />
                    </Link>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
          {grid.totals && (
            <tfoot>
              <tr>
                {grid.columns.map((column, at) => (
                  <td key={column.key} className={isFigureType(column.type) ? "dk-ar" : undefined}>
                    {at === 0 ? (
                      t("{0, plural, one {# row} other {# rows}}", rows.length)
                    ) : grid.totals?.[column.key] != null && isFigureType(column.type) ? (
                      <span className="dk-ax-num">
                        {formatDisplayValue(column.type, grid.totals[column.key], t)}
                      </span>
                    ) : null}
                  </td>
                ))}
                <td />
              </tr>
            </tfoot>
          )}
        </table>
        {grid.truncated && query === "" && (
          <div className="dk-ax-trunc">
            {t(
              "Showing {0} of {1}. Ask for the rest, or open it as a view.",
              grid.rows.length,
              grid.rowCount,
            )}
          </div>
        )}
        {rows.length === 0 && <div className="dk-ax-trunc">{t("No rows match “{0}”", query)}</div>}
      </div>
    </div>
  );
}
