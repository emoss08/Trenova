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
import type { BillingQueueSummary } from "@trenova/shared/types/billing-queue";
import { useMemo, useState, type Dispatch, type ReactNode, type SetStateAction } from "react";
import { Link, useNavigate } from "react-router";
import { reportPreviewFrom, tableViewFrom } from "../../artifact-payloads";
import {
  selectionState,
  toggleAll,
  toggleOne,
} from "../../cards/billing-queue-item/billing-item-checks";
import { ArtIcon } from "../../desk-art-kinds";
import {
  BillingBulkBar,
  BillingCheckbox,
  BillingRowPill,
  useBillingSummaries,
  useOpenBillingItem,
} from "../../cards/billing-queue-item/desk-billing-table";
import {
  InPlaceRecord,
  recordCardFor,
  usePreloadRecordCard,
  type RecordCardEntry,
} from "../../cards/registry";

/** A table the workspace draws: columns, rows by key, and where each row opens. */
export type GridRow = {
  key: string;
  values: Record<string, unknown>;
  path: string;
  recordId?: string;
};
export type Grid = {
  /** The record kind of the rows, when they are records; empty otherwise. */
  entity?: string;
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
    entity: view.recordEntity,
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

/** One cell as the table draws it; a first column that names its row reads as an id. */
export function Cell({
  column,
  value,
  first,
}: {
  column: DisplayColumn;
  value: unknown;
  first: boolean;
}) {
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

/**
 * The totals a table's footer shows: the source's own when it sent them, or
 * else the sum of each money column over the rows showing, so a filtered
 * table totals what is left.
 */
export function footTotals(grid: Grid, rows: GridRow[]): Record<string, unknown> | null {
  if (grid.totals) {
    return grid.totals;
  }
  const money = grid.columns.filter((column) => column.type === "money");
  if (money.length === 0 || rows.length === 0) {
    return null;
  }
  return Object.fromEntries(
    money.map((column) => [
      column.key,
      rows.reduce((sum, row) => {
        const figure = Number(row.values[column.key]);
        return Number.isFinite(figure) ? sum + figure : sum;
      }, 0),
    ]),
  );
}

/** What a version changed from the one before it, in a few words. */
export function versionNote(
  current: AssistantArtifact,
  previous: AssistantArtifact | null,
  t: TranslateFn,
): string {
  if (!previous) {
    return t("First read");
  }
  const tabular = (artifact: AssistantArtifact) =>
    (artifact.kind === "table_view" && !("path" in artifact.payload)) ||
    artifact.kind === "report_preview";
  if (!tabular(current) || !tabular(previous)) {
    return t("Read again");
  }
  const now = gridOf(current);
  const before = gridOf(previous);
  const had = new Set(before.rows.map((row) => row.key));
  const has = new Set(now.rows.map((row) => row.key));
  const added = now.rows.filter((row) => !had.has(row.key)).length;
  const removed = before.rows.filter((row) => !has.has(row.key)).length;
  const changed = changedCells(now, before).size;
  const parts = [
    added > 0 ? t("{0, plural, one {# row added} other {# rows added}}", added) : "",
    removed > 0 ? t("{0, plural, one {# row gone} other {# rows gone}}", removed) : "",
    changed > 0 ? t("{0, plural, one {# value changed} other {# values changed}}", changed) : "",
  ].filter((part) => part !== "");
  return parts.length > 0 ? parts.join(" · ") : t("Nothing changed");
}

type Sort = { key: string; direction: 1 | -1 };

/**
 * A table read whole: a filter over every cell as it reads, sortable columns,
 * figures on the right, statuses as coloured dots, and each row opening its
 * record — in place, as the record itself, for the kinds that have a card
 * here, and on its page otherwise. A cell that changed since the version
 * before is marked.
 */
export function DeskTableBody(props: TableBodyProps) {
  const listed = props.artifact.kind === "table_view" && !("path" in props.artifact.payload);
  const entity = listed ? tableViewFrom(props.artifact).recordEntity : "";
  const card = recordCardFor(entity);
  if (card && entity === "billing_queue_item") {
    return <BillingTableBody {...props} card={card} />;
  }

  return card ? (
    <InPlaceTableBody {...props} card={card} />
  ) : (
    <TableBodyView {...props} kit={null} records={null} />
  );
}

type TableBodyProps = {
  artifact: AssistantArtifact;
  /** The version before this one, for marking what changed. */
  previous: AssistantArtifact | null;
  /** The versions picker, when the table has more than one. */
  versions?: ReactNode;
};

/** What a billing queue table adds: live row state, picked rows, and an item opened in place. */
type BillingKit = {
  summaries: ReadonlyMap<string, BillingQueueSummary>;
  selected: string[];
  setSelected: Dispatch<SetStateAction<string[]>>;
  inline: string | null;
  setInline: (itemId: string | null) => void;
  openItem: (itemId: string) => void;
  card: RecordCardEntry;
};

/** What a table whose rows open in place adds: the row open now, and its card. */
type RecordKit = {
  inline: string | null;
  setInline: (recordId: string | null) => void;
  card: RecordCardEntry;
};

// A table whose rows open in place keeps the open row beside its own filter
// and sort, so going back finds the table as it was left. The card's code is
// read while the browser is idle, so the first row opened draws at once.
function InPlaceTableBody({ card, ...props }: TableBodyProps & { card: RecordCardEntry }) {
  const [inline, setInline] = useState<string | null>(null);
  usePreloadRecordCard(card);

  return <TableBodyView {...props} kit={null} records={{ inline, setInline, card }} />;
}

// A table of billing queue items is worked from: rows are picked, opened and
// approved in bulk, and their status is read live rather than as the agent
// saw it.
function BillingTableBody({ card, ...props }: TableBodyProps & { card: RecordCardEntry }) {
  const grid = useMemo(() => gridOf(props.artifact), [props.artifact]);
  const rowIds = useMemo(
    () => grid.rows.map((row) => row.recordId ?? "").filter((id) => id !== ""),
    [grid],
  );
  const summaries = useBillingSummaries(rowIds, true);
  const [selected, setSelected] = useState<string[]>([]);
  const [inline, setInline] = useState<string | null>(null);
  const openItem = useOpenBillingItem(props.artifact.threadId, setInline);
  usePreloadRecordCard(card);

  return (
    <TableBodyView
      {...props}
      kit={{ summaries, selected, setSelected, inline, setInline, openItem, card }}
      records={null}
    />
  );
}

function TableBodyView({
  artifact,
  previous,
  versions,
  kit,
  records,
}: TableBodyProps & { kit: BillingKit | null; records: RecordKit | null }) {
  const t = useT();
  const navigate = useNavigate();
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

  const totals = useMemo(() => footTotals(grid, rows), [grid, rows]);

  const billing = kit !== null;
  const opensRecords = billing || records !== null;
  const shownIds = useMemo(
    () => (opensRecords ? rows.map((row) => row.recordId ?? "").filter((id) => id !== "") : []),
    [opensRecords, rows],
  );

  const cycle = (key: string) =>
    setSort((current) =>
      current?.key === key
        ? current.direction === 1
          ? { key, direction: -1 }
          : null
        : { key, direction: 1 },
    );

  if (kit?.inline) {
    return (
      <InPlaceRecord
        entry={kit.card}
        recordId={kit.inline}
        order={shownIds}
        onSelect={kit.setInline}
        onBack={() => kit.setInline(null)}
      />
    );
  }
  if (records?.inline) {
    return (
      <InPlaceRecord
        entry={records.card}
        recordId={records.inline}
        order={shownIds}
        onSelect={records.setInline}
        onBack={() => records.setInline(null)}
      />
    );
  }

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
        <table className={billing ? "dk-sel" : undefined}>
          <thead>
            <tr>
              {kit && (
                <th className="dk-cb">
                  <BillingCheckbox
                    state={selectionState(shownIds, kit.selected)}
                    label={t("Select all")}
                    onClick={() => kit.setSelected(toggleAll(shownIds, kit.selected))}
                  />
                </th>
              )}
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
            {rows.map((row, index) => {
              const id = opensRecords ? (row.recordId ?? "") : "";
              const summary = id ? kit?.summaries.get(id) : undefined;
              const picked = id !== "" && Boolean(kit?.selected.includes(id));
              return (
                <tr
                  key={`${row.key}:${artifact.id}`}
                  className={cn(picked && "dk-on")}
                  style={{
                    animationDelay: `${Math.min(index, 14) * 18}ms`,
                    cursor: id || row.path !== "" ? "pointer" : undefined,
                  }}
                  onClick={
                    id && kit
                      ? () => kit.openItem(id)
                      : id && records
                        ? () => records.setInline(id)
                        : row.path !== ""
                          ? () => void navigate(row.path)
                          : undefined
                  }
                >
                  {kit && (
                    <td
                      className="dk-cb"
                      onClick={(event) => {
                        event.stopPropagation();
                        if (id) kit.setSelected((current) => toggleOne(current, id));
                      }}
                    >
                      <BillingCheckbox state={picked} label={t("Select")} />
                    </td>
                  )}
                  {grid.columns.map((column, at) => (
                    <td
                      key={column.key}
                      className={cn(
                        isFigureType(column.type) && "dk-ar",
                        changed.has(`${row.key}:${column.key}`) && "dk-chg",
                      )}
                    >
                      {summary && column.type === "status" ? (
                        <BillingRowPill summary={summary} />
                      ) : (
                        <Cell column={column} value={row.values[column.key]} first={at === 0} />
                      )}
                    </td>
                  ))}
                  <td className="dk-go">
                    {row.path !== "" && (
                      <Link
                        to={row.path}
                        aria-label={t("Open this record")}
                        onClick={(event) => event.stopPropagation()}
                      >
                        <ArtIcon name="ext" size={12} />
                      </Link>
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
          {totals && (
            <tfoot>
              <tr>
                {billing && <td />}
                {grid.columns.map((column, at) => (
                  <td key={column.key} className={isFigureType(column.type) ? "dk-ar" : undefined}>
                    {at === 0 ? (
                      t("{0, plural, one {# item} other {# items}}", rows.length)
                    ) : totals[column.key] != null && isFigureType(column.type) ? (
                      <span className="dk-ax-num">
                        {formatDisplayValue(column.type, totals[column.key], t)}
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
      {kit && (
        <BillingBulkBar
          selected={kit.selected}
          summaries={kit.summaries}
          onClear={() => kit.setSelected([])}
          onReview={kit.openItem}
        />
      )}
    </div>
  );
}
