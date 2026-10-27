import { DisplayValue } from "@/components/assistant/display-value";
import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
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
import { describeError } from "@trenova/shared/lib/error-presentation";
import { useMemo, useState, type Dispatch, type ReactNode, type SetStateAction } from "react";
import { Link, useNavigate } from "react-router";
import {
  reportPreviewFrom,
  tableViewFrom,
  type TablePaging,
  type ArtifactReader,
} from "../../artifact-payloads";
import {
  selectionState,
  toggleAll,
  toggleOne,
} from "../../cards/billing-queue-item/billing-item-checks";
import { ArtIcon } from "../../desk-art-kinds";
import { cardTime } from "../../record-card/record-card-values";
import { agentShare } from "./table-pages";
import { TablePager } from "./table-pager";
import { useTablePages, type TablePages, type TableSort } from "./use-table-pages";
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
  /** What the agent asked the list for, as it was read back. */
  searchedFor: string[];
  /** Where the list goes on from the rows held; null for a table that is all there is. */
  paging: TablePaging | null;
  /** The agent that read the rows the table holds; null when the table does not say. */
  readBy: ArtifactReader | null;
  /** Whether the person can read the list again, live, from its start. */
  rereadable: boolean;
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
      searchedFor: [],
      paging: null,
      readBy: null,
      rereadable: false,
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
    searchedFor: view.searchedFor,
    paging: view.paging.hasMore ? view.paging : null,
    readBy: view.readBy,
    rereadable: view.rereadable,
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
  if (current.kind === "report_run") {
    return previous === null
      ? t("First run")
      : previous.kind === "report_preview"
        ? t("Full report")
        : t("Run again");
  }
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

/** Rows filtered and sorted where they are: the whole table, or the rows read of a list. */
function arrangeRows(
  columns: readonly DisplayColumn[],
  rows: readonly GridRow[],
  needle: string,
  sort: TableSort | null,
  t: TranslateFn,
): GridRow[] {
  const lowered = needle.trim().toLowerCase();
  let shown = lowered
    ? rows.filter((row) =>
        columns.some((column) =>
          `${cellText(column, row.values[column.key], t)} ${rawText(row.values[column.key])}`
            .toLowerCase()
            .includes(lowered),
        ),
      )
    : [...rows];
  const column = sort ? columns.find((candidate) => candidate.key === sort.key) : undefined;
  if (sort && column) {
    shown = shown.sort((a, b) => {
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
  return shown;
}

/**
 * A table beside the conversation: how many rows its list holds and what the
 * agent asked it for, a search, columns that sort, figures on the right,
 * statuses as coloured dots, each row opening its record — in place, as the
 * record itself, for the kinds that have a card here, and on its page
 * otherwise — and, when the list is longer than a page, a pager.
 *
 * A list the agent read in part pages on past its rows, read again with the
 * person's own access; the rows the agent read are marked as its, and a cell
 * that changed since the version before is marked too.
 */
export function DeskTableBody(props: TableBodyProps) {
  const t = useT();
  const { artifact } = props;
  const grid = useMemo(() => gridOf(artifact), [artifact]);
  const pages = useTablePages({
    threadId: artifact.threadId,
    artifactId: artifact.id,
    stamp: artifact.updatedAt,
    rows: grid.rows,
    columns: grid.columns,
    entity: grid.entity ?? "",
    paging: grid.paging,
    rereadable: grid.rereadable,
    arrange: (rows, needle, sort) => arrangeRows(grid.columns, rows, needle, sort, t),
  });
  const listed = artifact.kind === "table_view" && !("path" in artifact.payload);
  const entity = listed ? (grid.entity ?? "") : "";
  const card = recordCardFor(entity);
  const body = { ...props, grid, pages };

  if (card && entity === "billing_queue_item") {
    return <BillingTableBody {...body} card={card} />;
  }

  return card ? (
    <InPlaceTableBody {...body} card={card} />
  ) : (
    <TableBodyView {...body} kit={null} records={null} />
  );
}

type TableBodyProps = {
  artifact: AssistantArtifact;
  /** The version before this one, for marking what changed. */
  previous: AssistantArtifact | null;
  /** The versions picker, when the table has more than one. */
  versions?: ReactNode;
};

type PagedBodyProps = TableBodyProps & { grid: Grid; pages: TablePages };

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

// A table whose rows open in place keeps the open row beside its own page,
// search and sort, so going back finds the table as it was left. The card's
// code is read while the browser is idle, so the first row opened draws at
// once.
function InPlaceTableBody({ card, ...props }: PagedBodyProps & { card: RecordCardEntry }) {
  const [inline, setInline] = useState<string | null>(null);
  usePreloadRecordCard(card);

  return <TableBodyView {...props} kit={null} records={{ inline, setInline, card }} />;
}

// A table of billing queue items is worked from: rows are picked, opened and
// approved in bulk, and the status of the rows on the page is read live
// rather than as the agent saw it.
function BillingTableBody({ card, ...props }: PagedBodyProps & { card: RecordCardEntry }) {
  const rowIds = useMemo(
    () => props.pages.rows.map((row) => row.recordId ?? "").filter((id) => id !== ""),
    [props.pages.rows],
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
  grid,
  pages,
  kit,
  records,
}: PagedBodyProps & { kit: BillingKit | null; records: RecordKit | null }) {
  const t = useT();
  const navigate = useNavigate();
  const changed = useMemo(
    () => changedCells(grid, previous ? gridOf(previous) : null),
    [grid, previous],
  );
  const rows = pages.rows;
  const totals = useMemo(() => footTotals(grid, rows), [grid, rows]);
  const billing = kit !== null;
  const opensRecords = billing || records !== null;
  const shownIds = useMemo(
    () => (opensRecords ? rows.map((row) => row.recordId ?? "").filter((id) => id !== "") : []),
    [opensRecords, rows],
  );
  const showsPager = pages.pages > 1;
  const withheld = grid.columns.filter((column) => pages.withheld.includes(column.key));
  const hidden = new Set(withheld.map((column) => column.key));

  const cycle = (key: string) => {
    const current = pages.sort;
    pages.setSort(
      current?.key === key
        ? current.direction === 1
          ? { key, direction: -1 }
          : null
        : { key, direction: 1 },
    );
  };

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
    <div className="dk-ax-tbl dk-tv">
      <TableHead grid={grid} pages={pages} />
      <div className="dk-tv-tools">
        <label className="dk-ax-q dk-tv-q">
          <ArtIcon name="search" size={13} />
          <input
            value={pages.search}
            onChange={(event) => pages.setSearch(event.target.value)}
            placeholder={searchPlaceholder(pages, t)}
            aria-label={pages.searchesList ? t("Search the list") : t("Filter rows")}
          />
          {pages.search !== "" && (
            <button
              type="button"
              className="dk-tv-clr"
              aria-label={t("Clear the search")}
              onClick={() => pages.setSearch("")}
            >
              <ArtIcon name="x" size={10} stroke={2.6} />
            </button>
          )}
        </label>
        <span className="dk-tv-sp" />
        <ReadMark artifact={artifact} reader={grid.readBy} pages={pages} />
        <ReadAgain pages={pages} />
        {versions}
      </div>
      <KeptRead reader={grid.readBy} pages={pages} />
      {withheld.length > 0 && (
        <div className="dk-tv-note">
          <span className="dk-tv-wh">
            <ArtIcon name="lock" size={11} stroke={2} />
            {t("Not shown at your access: {0}", withheld.map((column) => column.label).join(", "))}
          </span>
        </div>
      )}
      {pages.locked && (
        <div className="dk-tv-note">
          <span className="dk-tv-wh">
            <ArtIcon name="lock" size={11} stroke={2} />
            {grid.readBy
              ? t(
                  "You can no longer read this list, so only the rows {0} read are shown",
                  grid.readBy.name,
                )
              : t(
                  "You can no longer read this list, so only the rows read in this conversation are shown",
                )}
          </span>
        </div>
      )}
      {pages.failure !== null && (
        <div className="dk-tv-fail" role="alert">
          <ArtIcon name="warn" size={13} stroke={2} />
          <div>
            <b>{t("Page {0} couldn't be read", pages.page + 1)}</b>
            <span>
              {describeError(pages.failure).detail ??
                t("The list did not answer. The pages read so far are still here.")}
            </span>
          </div>
          <button type="button" className="dk-ax-btn" onClick={pages.retry}>
            <ArtIcon name="retry" size={12} stroke={2} />
            {t("Try again")}
          </button>
        </div>
      )}
      <div className="dk-ax-scroll">
        <table className={billing ? "dk-sel" : undefined} aria-busy={pages.loading || undefined}>
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
              {grid.columns.map((column) => {
                const sortable = pages.sortable(column.key);
                const sorted = pages.sort?.key === column.key ? pages.sort : null;
                return (
                  <th
                    key={column.key}
                    className={cn(isFigureType(column.type) && "dk-ar", !sortable && "dk-tv-fixed")}
                    onClick={sortable ? () => cycle(column.key) : undefined}
                    aria-sort={
                      sorted ? (sorted.direction === 1 ? "ascending" : "descending") : undefined
                    }
                  >
                    <span>
                      {hidden.has(column.key) && <ArtIcon name="lock" size={10} stroke={2.2} />}
                      {column.label}
                      {sorted && (
                        <ArtIcon
                          name={sorted.direction === 1 ? "up" : "down"}
                          size={10}
                          stroke={2.4}
                        />
                      )}
                    </span>
                  </th>
                );
              })}
              <th className="dk-go" />
            </tr>
          </thead>
          <tbody>
            {pages.loading
              ? Array.from({ length: loadingRows(pages) }, (_, index) => (
                  // oxlint-disable-next-line react/no-array-index-key -- placeholders for rows not read yet
                  <tr key={index} className="dk-tv-skel" aria-hidden>
                    {kit && <td className="dk-cb" />}
                    {grid.columns.map((column, at) => (
                      <td
                        key={column.key}
                        className={isFigureType(column.type) ? "dk-ar" : undefined}
                      >
                        <i
                          className="ui-shimmer"
                          style={{
                            width: `${SKELETON_WIDTHS[(at + index) % SKELETON_WIDTHS.length]}%`,
                          }}
                        />
                      </td>
                    ))}
                    <td className="dk-go" />
                  </tr>
                ))
              : rows.map((row, index) => {
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
                            hidden.has(column.key) && "dk-tv-hidden",
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
          {totals && !pages.loading && rows.length > 0 && (
            <tfoot>
              <tr>
                {billing && <td />}
                {grid.columns.map((column, at) => (
                  <td key={column.key} className={isFigureType(column.type) ? "dk-ar" : undefined}>
                    {at === 0 ? (
                      showsPager ? (
                        t("This page")
                      ) : (
                        t("{0, plural, one {# item} other {# items}}", rows.length)
                      )
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
        {!pages.loading && pages.failure === null && rows.length === 0 && (
          <div className="dk-ax-trunc">
            {pages.search.trim() !== ""
              ? t("No rows match “{0}”", pages.search.trim())
              : t("No rows")}
          </div>
        )}
      </div>
      {showsPager && <TablePager pages={pages} reader={grid.readBy} />}
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

const SKELETON_WIDTHS = [62, 78, 54, 70, 46, 66] as const;

/** How many placeholder rows stand in for a page while it is read. */
function loadingRows(pages: TablePages): number {
  if (pages.total === null) {
    return pages.size;
  }
  return Math.max(1, Math.min(pages.size, pages.total - pages.page * pages.size));
}

function searchPlaceholder(pages: TablePages, t: TranslateFn): string {
  if (pages.searchesList) {
    return pages.total !== null
      ? t("{0, plural, one {Search # row} other {Search all # rows}}", pages.total)
      : t("Search the list");
  }
  return pages.remote
    ? t("{0, plural, one {Filter the # row read} other {Filter the # rows read}}", pages.readRows)
    : t("Filter {0, plural, one {# row} other {# rows}}", pages.readRows);
}

/** How many rows the list holds, and what the agent asked it for. */
function TableHead({ grid, pages }: { grid: Grid; pages: TablePages }) {
  const t = useT();
  const unknown = pages.total === null;
  const figure = unknown ? pages.shownRows : (pages.total ?? 0);

  return (
    <header className="dk-tv-head">
      <div className="dk-tv-fig">
        <b>{unknown && pages.remote ? t("{0}+", figure) : figure}</b>
        <span>
          {pages.filtering
            ? t("{0, plural, one {of # row read} other {of # rows read}}", pages.readRows)
            : pages.searching
              ? t("{0, plural, one {row matches} other {rows match}}", figure)
              : t("{0, plural, one {row} other {rows}}", figure)}
        </span>
      </div>
      {grid.searchedFor.length > 0 && (
        <span
          className="dk-tv-terms"
          aria-label={
            grid.readBy ? t("What {0} asked for", grid.readBy.name) : t("What was asked for")
          }
        >
          {grid.searchedFor.map((term) => (
            <span key={term}>{term}</span>
          ))}
        </span>
      )}
    </header>
  );
}

/**
 * Reads the list again from its start, live, with the person's access, in
 * place of the rows the agent read. The table the conversation keeps is never
 * changed: it is what the agent's answer read.
 */
function ReadAgain({ pages }: { pages: TablePages }) {
  const t = useT();
  if (!pages.rereadable) {
    return null;
  }

  return (
    <button
      type="button"
      className="dk-ax-btn dk-ghost"
      disabled={pages.live && pages.fetching}
      onClick={pages.readAgain}
    >
      <ArtIcon name="retry" size={12} stroke={2} />
      {t("Read again now")}
    </button>
  );
}

/** While the list is read again: the agent's rows are kept, and one click away. */
function KeptRead({ reader, pages }: { reader: ArtifactReader | null; pages: TablePages }) {
  const t = useT();
  if (!pages.live) {
    return null;
  }

  return (
    <div className="dk-tv-note">
      <span className="dk-tv-wh">
        {reader && <DeskAgentTile agent={reader} size="xs" />}
        {reader
          ? t("What {0} read is kept in the conversation", reader.name)
          : t("What was read in this conversation is kept")}
      </span>
      <button type="button" className="dk-ax-btn dk-ghost" onClick={pages.backToAgent}>
        {t("Show it")}
      </button>
    </div>
  );
}

function readShare(
  t: TranslateFn,
  whole: boolean,
  reader: ArtifactReader | null,
  when: string,
): string {
  if (reader) {
    return whole
      ? t("{0} read this page · {1}", reader.name, when)
      : t("{0} read part of this page · {1}", reader.name, when);
  }

  return whole
    ? t("Read in this conversation · {0}", when)
    : t("Partly read in this conversation · {0}", when);
}

/**
 * Whose read the page is: rows the conversation's agent read, wearing its
 * mark, or the list read just now with the person's access.
 */
function ReadMark({
  artifact,
  reader,
  pages,
}: {
  artifact: AssistantArtifact;
  reader: ArtifactReader | null;
  pages: TablePages;
}) {
  const t = useT();
  const when = cardTime(artifact.updatedAt);
  if (!pages.remote) {
    return artifact.kind === "table_view" ? (
      <span className="dk-tv-agent">
        {reader && <DeskAgentTile agent={reader} size="xs" />}
        {reader
          ? t("{0} read every row · {1}", reader.name, when)
          : t("Every row read in this conversation · {0}", when)}
      </span>
    ) : null;
  }
  if (pages.failure !== null && pages.rows.length === 0) {
    return null;
  }
  if (pages.loading) {
    return (
      <span className="dk-tv-live">
        <i />
        {t("Reading page {0}…", pages.page + 1)}
      </span>
    );
  }
  const share = agentShare(pages.page, pages.size, pages.agentRows, pages.rows.length);
  if (share !== "none") {
    return (
      <span className="dk-tv-agent">
        {reader && <DeskAgentTile agent={reader} size="xs" />}
        {readShare(t, share === "all", reader, when)}
      </span>
    );
  }

  return (
    <span className="dk-tv-live">
      <i />
      {pages.searching ? t("Searched just now") : t("Read just now, with your access")}
    </span>
  );
}
