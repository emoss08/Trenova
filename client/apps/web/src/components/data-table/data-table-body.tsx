import { useT } from "@trenova/shared/i18n/use-t";
import { TableBody, TableCell, TableRow } from "@trenova/shared/components/ui/table";
import { useDataTable } from "@/contexts/data-table-context";
import {
  DataTableRowActionsContext,
  DataTableRowStateContext,
} from "@/contexts/data-table-row-context";
import {
  columnCellStyle,
  columnHeaderLabel,
  isChangedSince,
  pinnedCellClass,
  type CompiledFormatRules,
} from "@/lib/data-table";
import { cn } from "@trenova/shared/lib/utils";
import type {
  DataTableBodyProps,
  DataTableExpansion,
  DataTableGroupKey,
  DataTableGrouping,
  Row,
} from "@trenova/shared/types/data-table";
import type { RowData, RowSelectionState } from "@tanstack/react-table";
import { flexRender } from "@tanstack/react-table";
import { Edit02Icon } from "@trenova/shared/components/icons";
import { Fragment, memo, useCallback, useMemo, useRef } from "react";
import { buildGroupedBody } from "@/lib/data-table-grouping";
import { gridCellMove } from "@/lib/data-table-grid";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useTableAtom } from "@trenova/shared/hooks/use-table-atom";
import type { DataTableRowChanges } from "@/hooks/data-table/use-data-table-live-refresh";
import { DataTableCellEditor } from "./data-table-cell-editor";
import { DataTableContextMenu } from "./_components/data-table-context-menu";
import { DataTableExpandedRow } from "./data-table-expanded-row";
import { DataTableVirtualRows, type VirtualRowSlot } from "./data-table-virtual-rows";
import { DataTableGroupHeader } from "./data-table-group-header";
import { DataTablePinnedHeader } from "./data-table-pinned-header";

const ROW_STAGGER_MS = 18;
const NO_ROW_ACTIONS: never[] = [];
const NO_ROWS: never[] = [];

type RowCellSelection = { selected: ReadonlySet<string>; focused: string };

const NO_CELLS: RowCellSelection = { selected: new Set(), focused: "" };

/**
 * The cells of one row a person has selected, as a short key: the row redraws only
 * when its own cells change, however many other rows a drag passes through.
 */
function rowCellSelectionKey<TData extends RowData>(row: Row<TData>): string {
  const selected: string[] = [];
  let focused = "";
  for (const cell of row.getVisibleCells()) {
    if (cell.getIsSelected()) selected.push(cell.column.id);
    if (cell.getIsFocused()) focused = cell.column.id;
  }
  return selected.length === 0 && focused === "" ? "" : `${selected.join("\u0000")}|${focused}`;
}

function parseRowCellSelection(key: string): RowCellSelection {
  const [selected, focused] = key.split("|");
  return { selected: new Set(selected ? selected.split("\u0000") : []), focused: focused ?? "" };
}
const ROW_STAGGER_CAP = 14;

type VirtualEntry<TData extends RowData> = {
  kind: "row" | "expanded";
  row: Row<TData>;
  index: number;
};
/** How many rows Page Up and Page Down move the focused cell. */
const GRID_PAGE_ROWS = 10;

const INTERACTIVE_SELECTOR =
  'button, a, input, select, textarea, [role="button"], [role="checkbox"], [role="switch"]';

type DataTableRowProps<TData extends RowData> = {
  row: Row<TData>;
  rowIndex: number;
  isLastRow: boolean;
  canEditCells: boolean;
  formatClass?: string;
  rowClassName?: string;
  onRowClick?: (row: Row<TData>) => void;
  openPanelEdit: (row: Row<TData>) => void;
  hasPanel: boolean;
  canOpenPanel: boolean;
  canUpdate: boolean;
  isCursor: boolean;
  isExpanded: boolean;
  /** Which of the two glow animations to play, or none: alternating replays it. */
  changed: RowChangeMark;
  /** Kept at the top of the table by the person, whatever page or filter is showing. */
  isPinned: boolean;
  /** Whether the person can pin this table's rows, which puts pinning in the row's menu. */
  canPin: boolean;
  /** Changed since the person last had the table open, and not opened since. */
  unseen: boolean;
  /** Where the row sits among every row the filters match, counting the header as 1. */
  ariaRowIndex?: number;
  /** Whether rows can be ticked, so the row says whether it is. */
  selectable: boolean;
  /** Set when the table draws only the rows in view, so the window can measure this one. */
  measureRef?: (element: Element | null) => void;
  virtualIndex?: number;
};

type RowChangeMark = "a" | "b" | undefined;

/**
 * One body row. It is memoized and takes only what it draws with, so a change
 * that touches one row (the cursor, the open row, a checkbox) redraws that row
 * alone. The table instance is deliberately not a prop: TanStack hands back a
 * new wrapper every render, which would redraw every row every time. Whether the
 * row is ticked or has a cell open for editing it reads from the table's state
 * itself, so neither redraws the body or the rows around it.
 */
function DataTableRowInner<TData extends RowData>({
  row,
  rowIndex,
  isLastRow,
  canEditCells,
  formatClass,
  rowClassName,
  onRowClick,
  openPanelEdit,
  hasPanel,
  canOpenPanel,
  canUpdate,
  isCursor,
  isExpanded,
  changed,
  isPinned,
  canPin,
  unseen,
  ariaRowIndex,
  selectable,
  measureRef,
  virtualIndex,
}: DataTableRowProps<TData>) {
  const t = useT();
  // The row object outlives a column being shown, hidden, moved or pinned, so its
  // cells are followed from the table's state; the table caches the list, so the
  // row redraws only when its columns actually change.
  const cells = useTableAtom(row.table.store, () => row.getVisibleCells());
  const selected = useTableAtom(row.table.atoms.rowSelection, (selection) => !!selection[row.id]);
  const cellSelection = useTableAtom(row.table.atoms.cellSelection, (ranges) =>
    ranges.length === 0 ? "" : rowCellSelectionKey(row),
  );
  const selectedCells = cellSelection === "" ? NO_CELLS : parseRowCellSelection(cellSelection);
  const editingColumnId = useTableAtom(row.table.atoms.cellEditing, (editing) =>
    editing?.rowId === row.id ? editing.columnId : null,
  );
  const isClickable = !!(onRowClick || (hasPanel && canOpenPanel));
  const rowState = useMemo(
    () => ({ isExpanded, isCursor, isSelected: selected }),
    [isExpanded, isCursor, selected],
  );

  const handleRowClick = useCallback(
    (e: React.MouseEvent<HTMLTableRowElement>) => {
      if (e.shiftKey || e.altKey) return;
      const target = e.target as HTMLElement;

      if (target.closest(INTERACTIVE_SELECTOR)) return;
      if (target.closest<HTMLElement>("td")?.dataset.columnId === "select") return;

      const selection = window.getSelection();
      if (selection && selection.toString().length > 0) return;

      if (onRowClick) {
        onRowClick(row);
      } else if (hasPanel && canOpenPanel) {
        openPanelEdit(row);
      }
    },
    [row, onRowClick, hasPanel, canOpenPanel, openPanelEdit],
  );

  const tableRow = (
    <TableRow
      ref={measureRef}
      data-index={virtualIndex}
      id={row.id}
      data-row-index={rowIndex}
      tabIndex={-1}
      data-state={selected ? "selected" : undefined}
      data-cursor={isCursor || undefined}
      data-expanded={isExpanded || undefined}
      data-changed={changed}
      data-pinned={isPinned || undefined}
      data-unseen={unseen || undefined}
      aria-expanded={isExpanded ? true : undefined}
      aria-rowindex={ariaRowIndex}
      aria-selected={selectable ? selected : undefined}
      onClick={isClickable ? handleRowClick : undefined}
      style={{ animationDelay: `${Math.min(rowIndex, ROW_STAGGER_CAP) * ROW_STAGGER_MS}ms` }}
      className={cn(
        "ui-inset-focus-ring group/row outline-brand -outline-offset-2 transition-colors data-[state=selected]:outline",
        // A row drawn as the table scrolls is not arriving, so it does not rise in.
        virtualIndex === undefined && "animate-row-rise",
        "hover:bg-field data-cursor:bg-field data-expanded:bg-field data-expanded:[&>td:first-child]:shadow-[inset_2px_0_0_var(--border-strong)]",
        // The glow is on the cells: the row's own animation is its entrance, which
        // must not replay when the glow is taken away.
        "data-[changed=a]:[&>td]:animate-row-changed-a data-[changed=b]:[&>td]:animate-row-changed-b",
        "data-unseen:[&>td:first-child]:shadow-[inset_2px_0_0_var(--info)]",
        isClickable && "cursor-pointer",
        formatClass,
        rowClassName,
      )}
    >
      <DataTableRowStateContext value={rowState}>
        {cells.map((cell, cellIndex) => {
          const pinned = cell.column.getIsPinned();
          const canEdit = cell.getCanEdit();
          const isEditing = editingColumnId === cell.column.id;
          const cellSelected = selectedCells.selected.has(cell.column.id);
          const cellFocused = selectedCells.focused === cell.column.id;
          return (
            <TableCell
              className={cn(
                "border-border truncate border-b font-sans",
                isLastRow && "border-b-0",
                pinned && pinnedCellClass(cell.column),
                pinned &&
                  "group-hover/row:bg-field group-data-cursor/row:bg-field group-data-expanded/row:bg-field",
                canEdit && "group/cell relative",
                "data-cell-selected:bg-brand-subtle data-cell-focused:shadow-[inset_0_0_0_1px_var(--brand)]",
                isEditing && "overflow-visible py-1",
              )}
              key={cell.id}
              role="gridcell"
              aria-colindex={cellIndex + 1}
              aria-selected={cellSelected}
              aria-readonly={canEditCells ? !canEdit : undefined}
              tabIndex={
                cellFocused || (rowIndex === 0 && cellSelection === "" && cellIndex === 0) ? 0 : -1
              }
              data-column-id={cell.column.id}
              data-cell-selected={cellSelected || undefined}
              data-cell-focused={cellFocused || undefined}
              onMouseDown={(event) => {
                if (!event.altKey) return;
                event.preventDefault();
                cell.getSelectionStartHandler()(event.nativeEvent);
              }}
              onMouseEnter={cell.getSelectionExtendHandler()}
              onDoubleClick={
                canEdit && !isEditing && !isClickable
                  ? (e) => {
                      e.preventDefault();
                      cell.startEditing();
                    }
                  : undefined
              }
              style={columnCellStyle(cell.column)}
            >
              {isEditing ? (
                <DataTableCellEditor cell={cell} />
              ) : (
                <>
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  {canEdit && (
                    <button
                      type="button"
                      aria-label={t("Edit {0}", columnHeaderLabel(cell.column))}
                      onClick={(e) => {
                        e.stopPropagation();
                        cell.startEditing();
                      }}
                      className="absolute top-1/2 right-1 -translate-y-1/2 rounded-sm border border-border bg-background p-1 text-muted-foreground opacity-0 transition-opacity hover:text-foreground focus-visible:opacity-100 group-hover/cell:opacity-100"
                    >
                      <Edit02Icon className="size-3" />
                    </button>
                  )}
                </>
              )}
            </TableCell>
          );
        })}
      </DataTableRowStateContext>
    </TableRow>
  );

  // Every row has a menu: copying a row or a cell is always offered.
  return (
    <DataTableContextMenu
      row={row}
      openPanelEdit={openPanelEdit}
      hasPanel={hasPanel}
      canOpenPanel={canOpenPanel}
      canUpdate={canUpdate}
      canPin={canPin}
    >
      {tableRow}
    </DataTableContextMenu>
  );
}

const DataTableRow = memo(DataTableRowInner) as typeof DataTableRowInner;

/** The most placeholder rows drawn while a page loads: enough to fill a tall screen. */
const MAX_SKELETON_ROWS = 25;
/** Bar widths a column can take; each column keeps one all the way down, like real text. */
const SKELETON_WIDTHS = ["w-3/5", "w-2/3", "w-1/2", "w-3/4", "w-2/5"] as const;

function skeletonWidth(columnId: string): string {
  let hash = 0;
  for (const char of columnId) hash = (hash * 31 + char.charCodeAt(0)) | 0;
  return SKELETON_WIDTHS[Math.abs(hash) % SKELETON_WIDTHS.length];
}

/**
 * The table as it will look while its first page loads: placeholder rows in the real
 * columns, at the real widths, pinning and row height, so it reads as this table
 * waiting for its rows and nothing moves when they arrive.
 */
function DataTableSkeletonRows<TData extends Record<string, any>>({
  table,
  label,
}: {
  table: DataTableBodyProps<TData>["table"];
  label: string;
}) {
  const visibleColumns = table.getVisibleLeafColumns();
  const count = Math.min(table.state.pagination?.pageSize ?? MAX_SKELETON_ROWS, MAX_SKELETON_ROWS);

  return (
    <>
      {Array.from({ length: count }, (_, rowIndex) => (
        <TableRow
          key={rowIndex}
          aria-busy
          aria-hidden={rowIndex > 0 || undefined}
          className="hover:bg-transparent"
        >
          {visibleColumns.map((column, columnIndex) => (
            <TableCell
              key={column.id}
              className={cn(
                "border-border border-b",
                rowIndex === count - 1 && "border-b-0",
                pinnedCellClass(column),
              )}
              style={columnCellStyle(column)}
            >
              {rowIndex === 0 && columnIndex === 0 && <span className="sr-only">{label}</span>}
              {column.id === "select" ? (
                <span className="border-border-strong block size-4 rounded-sm border" />
              ) : (
                <Skeleton className={cn("h-2.5", skeletonWidth(column.id))} />
              )}
            </TableCell>
          ))}
        </TableRow>
      ))}
    </>
  );
}

export function DataTableBody<TData extends Record<string, any>>({
  table,
  isLoading,
  contextMenuActions,
  onRowClick,
  getFormatClass,
  grouping,
  loadingGroupKeys,
  expansion,
  cursorRowId = null,
  isFirstPage = true,
  isLastPage = true,
  rowChanges,
  pinnedRowsCollapsed = false,
  onPinnedRowsCollapsedChange,
  changesSince = 0,
  seenRowIds,
  rowOffset = 0,
  virtualized = false,
  tableRef,
  estimatedRowHeight = 40,
}: DataTableBodyProps<TData> & {
  isLoading?: boolean;
  getFormatClass?: CompiledFormatRules<TData> | null;
  grouping?: DataTableGrouping<TData>;
  loadingGroupKeys?: readonly DataTableGroupKey[];
  expansion?: DataTableExpansion<TData>;
  cursorRowId?: string | null;
  isFirstPage?: boolean;
  isLastPage?: boolean;
  rowChanges?: DataTableRowChanges;
  pinnedRowsCollapsed?: boolean;
  onPinnedRowsCollapsedChange?: (collapsed: boolean) => void;
  /** Rows changed after this Unix second are marked; zero marks none. */
  changesSince?: number;
  seenRowIds?: ReadonlySet<string>;
  /** How many matching rows come before this page, so each row can say where it is. */
  rowOffset?: number;
  /**
   * Draw only the page rows near the visible part of the table. Off by default, as
   * the browser's find-in-page sees only drawn rows; ignored while rows are grouped.
   */
  virtualized?: boolean;
  /** The table element, so the window can find what scrolls it. */
  tableRef?: React.RefObject<HTMLTableElement | null>;
  /** A row's height before it is measured. */
  estimatedRowHeight?: number;
}) {
  const t = useT();

  const { openPanelEdit, hasPanel, canOpenPanel, canUpdate } = useDataTable<TData, unknown>();
  // Pinned rows come first, then the page; the keyboard, a shift-click range and
  // the row indices all follow that order, so they match what is on screen.
  const pinnedRows = table.getTopRows();
  const shownPinnedRows = pinnedRowsCollapsed ? NO_ROWS : pinnedRows;
  const rows = table.getCenterRows();
  const displayedRows = useMemo(
    () => (shownPinnedRows.length > 0 ? [...shownPinnedRows, ...rows] : rows),
    [shownPinnedRows, rows],
  );
  const canEditCells = !!table.options.enableCellEditing;
  const rowActions = contextMenuActions ?? NO_ROW_ACTIONS;
  const getRowClassName = table.options.meta?.getRowClassName;
  const enableSelection = table.options.enableRowSelection === true;
  const canPinRows = table.options.enableRowPinning === true;
  const selectionAnchorRef = useRef<number | null>(null);
  const bodyRef = useRef<HTMLTableSectionElement>(null);

  const focusRowAt = useCallback((index: number) => {
    const target = bodyRef.current?.querySelector<HTMLTableRowElement>(
      `tr[data-row-index="${index}"]`,
    );
    target?.focus();
  }, []);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTableSectionElement>) => {
      const target = e.target as HTMLElement;
      if (target.closest("input, textarea, select, [contenteditable=true]")) return;

      const cell = target.closest<HTMLTableCellElement>('td[role="gridcell"]');
      if (cell && cell === target && bodyRef.current) {
        const move = gridCellMove(e.nativeEvent, bodyRef.current, cell, GRID_PAGE_ROWS);
        if (move) {
          e.preventDefault();
          const rowId = move.cell.closest("tr")?.id;
          const columnId = move.cell.dataset.columnId;
          if (rowId && columnId) {
            if (move.extend) {
              const fromRowId = cell.closest("tr")?.id;
              if (table.getSelectedCellCount() === 0 && fromRowId && cell.dataset.columnId) {
                table.setFocusedCell(fromRowId, cell.dataset.columnId);
              }
              table.extendCellSelection(move.extend);
            } else {
              table.setFocusedCell(rowId, columnId);
            }
          }
          move.cell.focus();
          return;
        }

        const rowId = cell.closest("tr")?.id;
        const columnId = cell.dataset.columnId;
        if ((e.key === "Enter" || e.key === "F2") && rowId && columnId) {
          const tableCell = table
            .getRow(rowId, true)
            ?.getVisibleCells()
            .find((entry) => entry.column.id === columnId);
          if (tableCell?.getCanEdit()) {
            e.preventDefault();
            tableCell.startEditing();
            return;
          }
          if (e.key === "Enter") {
            e.preventDefault();
            cell.closest("tr")?.click();
          }
          return;
        }
        if (e.key === " " && enableSelection && rowId) {
          e.preventDefault();
          table.getRow(rowId, true)?.toggleSelected();
          return;
        }
        if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "a") {
          e.preventDefault();
          table.selectAllCells();
          return;
        }
        return;
      }

      const focusedRow = target.closest<HTMLTableRowElement>("tr[data-row-index]");
      const rowCount = displayedRows.length;
      if (rowCount === 0) return;
      const currentIndex = focusedRow ? Number(focusedRow.dataset.rowIndex) : -1;

      switch (e.key) {
        case "ArrowDown":
        case "j":
          e.preventDefault();
          focusRowAt(Math.min(currentIndex + 1, rowCount - 1));
          break;
        case "ArrowUp":
        case "k":
          e.preventDefault();
          focusRowAt(Math.max(currentIndex - 1, 0));
          break;
        case "ArrowRight": {
          const first = focusedRow?.querySelector<HTMLTableCellElement>('td[role="gridcell"]');
          if (!first || !focusedRow) return;
          e.preventDefault();
          if (first.dataset.columnId) table.setFocusedCell(focusedRow.id, first.dataset.columnId);
          first.focus();
          break;
        }
        case "Home":
          e.preventDefault();
          focusRowAt(0);
          break;
        case "End":
          e.preventDefault();
          focusRowAt(rowCount - 1);
          break;
        case "Enter": {
          if (!focusedRow || focusedRow !== target) return;
          e.preventDefault();
          focusedRow.click();
          break;
        }
        case " ": {
          if (!enableSelection || !focusedRow || currentIndex < 0) return;
          e.preventDefault();
          const row = displayedRows[currentIndex];
          row?.toggleSelected();
          selectionAnchorRef.current = currentIndex;
          break;
        }
        default:
          break;
      }
    },
    [displayedRows, enableSelection, focusRowAt, table],
  );

  const handleClickCapture = useCallback(
    (e: React.MouseEvent<HTMLTableSectionElement>) => {
      if (!enableSelection) return;
      const target = e.target as HTMLElement;
      const rowEl = target.closest<HTMLTableRowElement>("tr[data-row-index]");
      if (!rowEl) return;
      const rowIndex = Number(rowEl.dataset.rowIndex);

      if (e.shiftKey && selectionAnchorRef.current !== null) {
        e.preventDefault();
        e.stopPropagation();
        const start = Math.min(selectionAnchorRef.current, rowIndex);
        const end = Math.max(selectionAnchorRef.current, rowIndex);
        const rangeSelection: RowSelectionState = {};
        for (let i = start; i <= end; i++) {
          const row = displayedRows[i];
          if (row?.getCanSelect()) rangeSelection[row.id] = true;
        }
        table.setRowSelection((current) => ({ ...current, ...rangeSelection }));
        return;
      }

      if (target.closest('[role="checkbox"]')) {
        selectionAnchorRef.current = rowIndex;
      }
    },
    [table, displayedRows, enableSelection],
  );

  const expandedRowId = expansion?.expandedRowId ?? null;
  const colSpan = table.getVisibleLeafColumns().length;
  const onExpandedRowIdChange = expansion?.onExpandedRowIdChange;
  const collapseExpanded = useCallback(
    () => onExpandedRowIdChange?.(null),
    [onExpandedRowIdChange],
  );
  const bodyItems = grouping
    ? buildGroupedBody({
        rows,
        groups: grouping.groups,
        getGroupKey: (row) => grouping.getGroupKey(row.original),
        collapsedKeys: grouping.collapsedKeys,
        loadingKeys: loadingGroupKeys,
        isFirstPage,
        isLastPage,
      })
    : rows.map((row, index) => ({ kind: "row" as const, row, index }));
  const firstPageRowIndex = shownPinnedRows.length;
  const lastRowIndex = displayedRows.length - 1;

  const renderRow = (
    row: (typeof displayedRows)[number],
    rowIndex: number,
    pinned: boolean,
    slot?: VirtualRowSlot,
  ) => (
    <DataTableRow
      key={row.id}
      row={row}
      rowIndex={rowIndex}
      isLastRow={rowIndex === lastRowIndex && expandedRowId !== row.id}
      canEditCells={canEditCells}
      formatClass={getFormatClass?.(row)}
      rowClassName={getRowClassName?.(row)}
      onRowClick={onRowClick}
      openPanelEdit={openPanelEdit}
      hasPanel={hasPanel}
      canOpenPanel={canOpenPanel}
      canUpdate={canUpdate}
      isCursor={cursorRowId === row.id}
      isPinned={pinned}
      canPin={canPinRows}
      unseen={
        changesSince > 0 && !seenRowIds?.has(row.id) && isChangedSince(row.original, changesSince)
      }
      changed={rowChanges?.ids.has(row.id) ? (rowChanges.version % 2 === 0 ? "a" : "b") : undefined}
      isExpanded={expandedRowId === row.id}
      selectable={enableSelection}
      ariaRowIndex={
        pinned ? rowIndex + 2 : rowIndex - firstPageRowIndex + rowOffset + shownPinnedRows.length + 2
      }
      measureRef={slot?.measureRef}
      virtualIndex={slot?.index}
    />
  );

  const renderExpanded = (row: (typeof displayedRows)[number], slot?: VirtualRowSlot) =>
    expansion ? (
      <DataTableExpandedRow
        key={`expanded:${row.id}`}
        rowId={row.id}
        colSpan={colSpan}
        measureRef={slot?.measureRef}
        virtualIndex={slot?.index}
      >
        {expansion.renderExpandedRow(row, { collapse: collapseExpanded })}
      </DataTableExpandedRow>
    ) : null;

  const renderWithExpansion = (
    row: (typeof displayedRows)[number],
    rowIndex: number,
    pinned: boolean,
  ) => (
    <Fragment key={row.id}>
      {renderRow(row, rowIndex, pinned)}
      {expandedRowId === row.id ? renderExpanded(row) : null}
    </Fragment>
  );

  const windowed = virtualized && !grouping && !!tableRef;
  const virtualEntries: VirtualEntry<TData>[] = [];
  if (windowed) {
    rows.forEach((row, index) => {
      virtualEntries.push({ kind: "row", row, index });
      if (expandedRowId === row.id) virtualEntries.push({ kind: "expanded", row, index });
    });
  }

  return (
    <TableBody
      ref={bodyRef}
      id="content"
      tabIndex={-1}
      onKeyDown={handleKeyDown}
      onClickCapture={handleClickCapture}
      // REMINDER: avoids scroll (skipping the table header) when using skip to content
      style={{
        scrollMarginTop: "calc(var(--top-bar-height) + 40px)",
      }}
    >
      {pinnedRows.length ||
      rows.length ||
      (grouping && grouping.collapsedKeys.length > 0 && !isLoading) ? (
        <DataTableRowActionsContext value={rowActions}>
          {pinnedRows.length > 0 ? (
            <DataTablePinnedHeader
              count={pinnedRows.length}
              collapsed={!!pinnedRowsCollapsed}
              colSpan={colSpan}
              onToggle={() => onPinnedRowsCollapsedChange?.(!pinnedRowsCollapsed)}
              onUnpinAll={() => table.setRowPinning({ top: [], bottom: [] })}
            />
          ) : null}
          {shownPinnedRows.map((row, index) => renderWithExpansion(row, index, true))}
          {windowed ? (
            <DataTableVirtualRows
              items={virtualEntries}
              getKey={(entry) => `${entry.kind}:${entry.row.id}`}
              tableRef={tableRef}
              estimateSize={estimatedRowHeight}
              colSpan={colSpan}
              renderItem={(entry, slot) =>
                entry.kind === "row"
                  ? renderRow(entry.row, firstPageRowIndex + entry.index, false, slot)
                  : renderExpanded(entry.row, slot)
              }
            />
          ) : null}
          {(windowed ? [] : bodyItems).map((item) =>
            item.kind === "group" ? (
              <DataTableGroupHeader
                key={`group:${item.group.key}`}
                group={item.group}
                collapsed={item.collapsed}
                colSpan={colSpan}
                onToggle={grouping!.onToggleGroup}
              />
            ) : (
              renderWithExpansion(item.row, firstPageRowIndex + item.index, false)
            ),
          )}
        </DataTableRowActionsContext>
      ) : isLoading ? (
        <DataTableSkeletonRows table={table} label={t("Loading data...")} />
      ) : null}
    </TableBody>
  );
}
