"use no memo";
import { HoverGlideGroup } from "@/components/hover-glide";
import { useDeferredMeasure } from "@/hooks/use-deferred-measure";
import { useInfiniteScrollSentinel } from "@/hooks/use-infinite-scroll-sentinel";
import { defaultRangeExtractor, useVirtualizer } from "@tanstack/react-virtual";
import { rowNavigationKeyDown, useFocusedRow, withIndex } from "@/components/virtual-focus";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { cn } from "@trenova/shared/lib/utils";
import { ListCapLine, ListMoreLine, type ListMore } from "./list-more";
import { Fragment, useImperativeHandle, useRef, useState, type ReactNode, type Ref } from "react";

export type BoundedColumn = {
  key: string;
  label: ReactNode;
  align?: "start" | "end" | "center";
  /** Takes only the width its content needs (a menu, a mark). */
  shrink?: boolean;
};

/** What a row's `tr` spreads: its key for the table, and the measuring ref when virtualised. */
export type BoundedRowProps = {
  ref?: (element: HTMLTableRowElement | null) => void;
  "data-index"?: number;
  "data-row-key": string;
};

/** One row of a growing table: a stable key and the `tr` it draws, with `props` spread on it. */
export type BoundedRow = {
  key: string;
  render: (props: BoundedRowProps) => ReactNode;
};

/** The list behind a growing table (`useWorkerList` is one): more pages and how to read them. */
export type BoundedMore = ListMore;

/** Moves a table to a row from outside, such as a row the address names. */
export type BoundedTableHandle = {
  scrollToKey: (key: string) => void;
};

/** From this many rows the body mounts only the rows near the view; under it, all of them. */
export const BOUNDED_VIRTUAL_FROM = 30;

/** The pinned heads' height (`h-8`), where the rows start inside the scrolled region. */
const HEAD_HEIGHT = 32;

const BODY_CLASS = "[&>tr:last-child>td]:border-b-0";

const ALIGN = { start: "text-left", end: "text-right", center: "text-center" } as const;

type BoundedTableProps = {
  columns: readonly BoundedColumn[];
  label: string;
  /** The region's height cap, as a Tailwind class (`max-h-152`). */
  maxHeightClassName: string;
  /** A guess at a row's height before it is measured, for a long table; 37 (one line). */
  estimateRowHeight?: number;
  handleRef?: Ref<BoundedTableHandle>;
} & (
  | {
      /** A growing table's rows; long ones are virtualised. */
      rows: readonly BoundedRow[];
      /** Reads more at the end of the region; the table draws the "loading more" row. */
      more?: BoundedMore;
      /**
       * A list the server reads whole and caps (`SERVER_LIST_CAP`): at the cap the table ends with
       * "Showing the newest 100".
       */
      cap?: number;
      children?: never;
    }
  | {
      /** A table the server keeps small (a capped, fixed list): `tr`s drawn as they are. */
      children: ReactNode;
      rows?: never;
      more?: never;
      cap?: never;
    }
);

/**
 * A table in a bounded region of its own: the column heads stay pinned while the rows scroll
 * under them, and one hover plate glides between rows. A row is a `tr` with `data-glide-target`
 * whose cells wear `BOUNDED_CELL_CLASS`.
 *
 * A growing table passes `rows` (each draws its `tr` with the props it is given spread on it) and
 * `more` from `useWorkerList`. From `BOUNDED_VIRTUAL_FROM` rows only the rows near the view are
 * mounted, measured as they mount; a shorter table is drawn whole, exactly as `children` are, and
 * the region hugs its rows up to its cap either way.
 */
export function BoundedTable(props: BoundedTableProps) {
  const { columns, label, maxHeightClassName, estimateRowHeight = 37, handleRef } = props;
  const [viewport, setViewport] = useState<HTMLDivElement | null>(null);
  // Read through a ref, not the viewport state, so a deep link's first commit can already
  // scroll to its row (the state is still null until the commit after the ref attaches).
  const tableRef = useRef<HTMLTableElement>(null);
  const rows = props.rows;
  const virtual = rows !== undefined && rows.length >= BOUNDED_VIRTUAL_FROM;

  useImperativeHandle(
    virtual ? undefined : handleRef,
    () => ({
      scrollToKey: (key: string) => {
        const row = tableRef.current?.querySelector(`tr[data-row-key="${CSS.escape(key)}"]`);
        if (row && typeof row.scrollIntoView === "function") {
          row.scrollIntoView({ block: "nearest", behavior: "smooth" });
        }
      },
    }),
    [],
  );

  return (
    <ScrollArea
      className={cn("min-h-0", maxHeightClassName)}
      maskVariant="card"
      maskClassName="top-8"
      viewportRef={setViewport}
    >
      <HoverGlideGroup className="bg-card" glideClassName="rounded-none">
        <table
          ref={tableRef}
          aria-label={label}
          className="w-full border-separate border-spacing-0 text-sm"
        >
          <thead className="sticky top-0 z-10">
            <tr>
              {columns.map((column) => (
                <th
                  key={column.key}
                  scope="col"
                  className={cn(
                    "h-8 border-b border-border-subtle bg-sunken px-3 text-xs font-medium whitespace-nowrap text-foreground-subtle first:pl-4",
                    ALIGN[column.align ?? "start"],
                    column.shrink && "w-px",
                  )}
                >
                  {column.label}
                </th>
              ))}
            </tr>
          </thead>
          {virtual ? (
            <VirtualBody
              rows={rows}
              span={columns.length}
              viewport={viewport}
              estimate={estimateRowHeight}
              handleRef={handleRef}
            />
          ) : (
            <tbody className={BODY_CLASS}>
              {rows === undefined
                ? props.children
                : rows.map((row) => (
                    <Fragment key={row.key}>{row.render({ "data-row-key": row.key })}</Fragment>
                  ))}
            </tbody>
          )}
          {rows !== undefined && props.cap !== undefined && rows.length >= props.cap ? (
            <tbody>
              <tr>
                <td colSpan={columns.length} className="p-0">
                  <ListCapLine count={rows.length} cap={props.cap} />
                </td>
              </tr>
            </tbody>
          ) : null}
          {rows !== undefined && (props.more?.hasMore || props.more?.loadMoreFailed) ? (
            <tbody>
              <MoreRow more={props.more} span={columns.length} />
            </tbody>
          ) : null}
        </table>
      </HoverGlideGroup>
    </ScrollArea>
  );
}

/** A spacer row standing for rows not mounted, `height` tall. */
function Spacer({ span, height }: { span: number; height: number }) {
  return (
    <tr aria-hidden>
      <td colSpan={span} style={{ height, padding: 0, border: 0 }} />
    </tr>
  );
}

/**
 * The rows near the view, with spacer rows standing for the rest. The row holding focus stays
 * mounted wherever it scrolls (with spacers either side of it), and Arrow Up and Down, or Tab out
 * of the window's edge, scroll the next row in before focusing it.
 */
function VirtualBody({
  rows,
  span,
  viewport,
  estimate,
  handleRef,
}: {
  rows: readonly BoundedRow[];
  span: number;
  viewport: HTMLDivElement | null;
  estimate: number;
  handleRef?: Ref<BoundedTableHandle>;
}) {
  const bodyRef = useRef<HTMLTableSectionElement>(null);
  const { focusedKey, handlers } = useFocusedRow((index) => rows[index]?.key);
  // oxlint-disable-next-line react/incompatible-library -- TanStack Virtual is incompatible with the React Compiler; the file is "use no memo"
  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => viewport,
    estimateSize: () => estimate,
    getItemKey: (index) => rows[index].key,
    overscan: 10,
    scrollMargin: HEAD_HEIGHT,
    initialRect: { width: 0, height: 600 },
    rangeExtractor: (range) => {
      const key = focusedKey.current;
      const focused = key === null ? -1 : rows.findIndex((row) => row.key === key);
      return withIndex(defaultRangeExtractor(range), focused === -1 ? null : focused);
    },
  });
  const measureRow = useDeferredMeasure(virtualizer);
  const scrollToIndex = (index: number) => virtualizer.scrollToIndex(index, { align: "auto" });

  useImperativeHandle(
    handleRef,
    () => ({
      scrollToKey: (key: string) => {
        const index = rows.findIndex((row) => row.key === key);
        if (index !== -1) virtualizer.scrollToIndex(index, { align: "auto" });
      },
    }),
    [rows, virtualizer],
  );

  const items = virtualizer.getVirtualItems();
  const drawn: ReactNode[] = [];
  let edge = HEAD_HEIGHT;
  for (const item of items) {
    if (item.start > edge) {
      drawn.push(<Spacer key={`gap:${item.key}`} span={span} height={item.start - edge} />);
    }
    drawn.push(
      <Fragment key={item.key}>
        {rows[item.index].render({
          ref: measureRow,
          "data-index": item.index,
          "data-row-key": rows[item.index].key,
        })}
      </Fragment>,
    );
    edge = item.end;
  }
  const after = virtualizer.getTotalSize() - (edge - HEAD_HEIGHT);

  return (
    <tbody
      ref={bodyRef}
      className={BODY_CLASS}
      {...handlers}
      onKeyDown={rowNavigationKeyDown({ count: rows.length, containerRef: bodyRef, scrollToIndex })}
    >
      {drawn}
      {after > 0 ? <Spacer span={span} height={after} /> : null}
    </tbody>
  );
}

/**
 * The row after the last while more pages exist or the next one failed: reaching it reads the next
 * page (never after a failure, which waits for its retry), and it holds the same height while that
 * page is in flight, so nothing moves.
 */
function MoreRow({ more, span }: { more: BoundedMore; span: number }) {
  const sentinel = useInfiniteScrollSentinel<HTMLTableRowElement>({
    hasNextPage: more.hasMore && !more.loadMoreFailed,
    isFetchingNextPage: more.loadingMore,
    onLoadMore: more.loadMore,
  });
  return (
    <tr ref={sentinel}>
      <td colSpan={span} className="p-0">
        <ListMoreLine more={more} />
      </td>
    </tr>
  );
}

/** A group's head row inside a `BoundedTable` body ("Required 8"); spread `props` in `rows`. */
export function BoundedGroupRow({
  span,
  label,
  count,
  props,
}: {
  span: number;
  label: string;
  count: number;
  props?: BoundedRowProps;
}) {
  return (
    <tr {...props}>
      <td
        colSpan={span}
        className="border-b border-border-subtle bg-card px-4 pt-2.5 pb-1.5 text-xs font-medium"
      >
        {label}
        <span className="ml-1 font-normal text-foreground-subtle tabular-nums">{count}</span>
      </td>
    </tr>
  );
}

/** The cell class every `BoundedTable` cell shares. */
export const BOUNDED_CELL_CLASS =
  "border-b border-border-subtle px-3 py-2 align-middle first:pl-4 last:pr-3";
