"use no memo";
import { useT } from "@trenova/shared/i18n/use-t";
import { TableBody, TableCell, TableRow } from "@trenova/shared/components/ui/table";
import { useDataTable } from "@/contexts/data-table-context";
import {
  DataTableRowActionsContext,
  DataTableRowStateContext,
} from "@/contexts/data-table-row-context";
import {
  columnSizeVar,
  pinnedCellClass,
  pinnedCellStyle,
  type CompiledFormatRules,
} from "@/lib/data-table";
import { cn } from "@trenova/shared/lib/utils";
import type {
  ColumnDef,
  DataTableBodyProps,
  DataTableExpansion,
  DataTableGroupKey,
  DataTableGrouping,
  Row,
} from "@trenova/shared/types/data-table";
import type { ColumnPinningState, RowData, RowSelectionState } from "@tanstack/react-table";
import { flexRender } from "@tanstack/react-table";
import { Edit02Icon } from "@trenova/shared/components/icons";
import { Fragment, memo, useCallback, useMemo, useRef } from "react";
import { buildGroupedBody } from "@/lib/data-table-grouping";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { DataTableCellEditor } from "./data-table-cell-editor";
import { DataTableContextMenu } from "./_components/data-table-context-menu";
import { DataTableExpandedRow } from "./data-table-expanded-row";
import { DataTableGroupHeader } from "./data-table-group-header";

const ROW_STAGGER_MS = 18;
const NO_ROW_ACTIONS: never[] = [];
const ROW_STAGGER_CAP = 14;

const INTERACTIVE_SELECTOR =
  'button, a, input, select, textarea, [role="button"], [role="checkbox"], [role="switch"]';

type DataTableRowProps<TData extends RowData> = {
  row: Row<TData>;
  rowIndex: number;
  selected: boolean;
  isLastRow: boolean;
  columns: ColumnDef<TData>[];
  columnVisibility: Record<string, boolean>;
  columnOrder: string[];
  columnPinning: ColumnPinningState;
  canEditCells: boolean;
  formatClass?: string;
  rowClassName?: string;
  hasRowActions: boolean;
  onRowClick?: (row: Row<TData>) => void;
  openPanelEdit: (row: Row<TData>) => void;
  hasPanel: boolean;
  canOpenPanel: boolean;
  canUpdate: boolean;
  editingColumnId: string | null;
  isCursor: boolean;
  isExpanded: boolean;
};

/**
 * One body row. It is memoized and takes only what it draws with, so a change
 * that touches one row (the cursor, the open row, a checkbox) redraws that row
 * alone. The table instance is deliberately not a prop: TanStack hands back a
 * new wrapper every render, which would redraw every row every time.
 */
function DataTableRowInner<TData extends RowData>({
  row,
  rowIndex,
  selected,
  isLastRow,
  formatClass,
  rowClassName,
  hasRowActions,
  onRowClick,
  openPanelEdit,
  hasPanel,
  canOpenPanel,
  canUpdate,
  editingColumnId,
  isCursor,
  isExpanded,
}: DataTableRowProps<TData>) {
  const isClickable = !!(onRowClick || (hasPanel && canOpenPanel));
  const hasContextMenu = hasRowActions || (hasPanel && canOpenPanel);
  const rowState = useMemo(
    () => ({ isExpanded, isCursor, isSelected: selected }),
    [isExpanded, isCursor, selected],
  );

  const handleRowClick = useCallback(
    (e: React.MouseEvent<HTMLTableRowElement>) => {
      if (e.shiftKey) return;
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
      id={row.id}
      data-row-index={rowIndex}
      tabIndex={-1}
      data-state={selected ? "selected" : undefined}
      data-cursor={isCursor || undefined}
      data-expanded={isExpanded || undefined}
      aria-expanded={isExpanded ? true : undefined}
      onClick={isClickable ? handleRowClick : undefined}
      style={{ animationDelay: `${Math.min(rowIndex, ROW_STAGGER_CAP) * ROW_STAGGER_MS}ms` }}
      className={cn(
        "ui-inset-focus-ring group/row outline-brand animate-row-rise -outline-offset-2 transition-colors data-[state=selected]:outline",
        "hover:bg-field data-cursor:bg-field data-expanded:bg-field data-expanded:[&>td:first-child]:shadow-[inset_2px_0_0_var(--border-strong)]",
        isClickable && "cursor-pointer",
        formatClass,
        rowClassName,
      )}
    >
      <DataTableRowStateContext value={rowState}>
        {row.getVisibleCells().map((cell) => {
          const pinned = cell.column.getIsPinned();
          const canEdit = cell.getCanEdit();
          const isEditing = editingColumnId === cell.column.id;
          return (
            <TableCell
              className={cn(
                "border-border truncate border-b font-sans",
                isLastRow && "border-b-0",
                pinned && pinnedCellClass(cell.column),
                pinned &&
                  "group-hover/row:bg-field group-data-cursor/row:bg-field group-data-expanded/row:bg-field",
                canEdit && "group/cell relative",
                isEditing && "overflow-visible py-1",
              )}
              key={cell.id}
              role="cell"
              data-column-id={cell.column.id}
              aria-label={`${cell.column.id} cell`}
              onDoubleClick={
                canEdit && !isEditing && !isClickable
                  ? (e) => {
                      e.preventDefault();
                      cell.startEditing();
                    }
                  : undefined
              }
              style={{
                width: `var(${columnSizeVar(cell.column.id)})`,
                maxWidth: `var(${columnSizeVar(cell.column.id)})`,
                ...pinnedCellStyle(cell.column),
              }}
            >
              {isEditing ? (
                <DataTableCellEditor cell={cell} />
              ) : (
                <>
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  {canEdit && (
                    <button
                      type="button"
                      aria-label={`Edit ${cell.column.id}`}
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

  if (hasContextMenu) {
    return (
      <DataTableContextMenu
        row={row}
        openPanelEdit={openPanelEdit}
        hasPanel={hasPanel}
        canOpenPanel={canOpenPanel}
        canUpdate={canUpdate}
      >
        {tableRow}
      </DataTableContextMenu>
    );
  }

  return tableRow;
}

const DataTableRow = memo(DataTableRowInner) as typeof DataTableRowInner;

export function DataTableBody<TData extends Record<string, any>>({
  table,
  columns,
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
}: DataTableBodyProps<TData> & {
  isLoading?: boolean;
  getFormatClass?: CompiledFormatRules<TData> | null;
  grouping?: DataTableGrouping<TData>;
  loadingGroupKeys?: readonly DataTableGroupKey[];
  expansion?: DataTableExpansion<TData>;
  cursorRowId?: string | null;
  isFirstPage?: boolean;
  isLastPage?: boolean;
}) {
  const t = useT();

  const { openPanelEdit, hasPanel, canOpenPanel, canUpdate } = useDataTable<TData, unknown>();
  const rows = table.getRowModel().rows;
  const { columnVisibility, columnOrder, columnPinning, cellEditing } = table.state;
  const canEditCells = !!table.options.enableCellEditing;
  const rowActions = contextMenuActions ?? NO_ROW_ACTIONS;
  const hasRowActions = rowActions.length > 0;
  const getRowClassName = table.options.meta?.getRowClassName;
  const enableSelection = table.options.enableRowSelection === true;
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

      const focusedRow = target.closest<HTMLTableRowElement>("tr[data-row-index]");
      const rowCount = table.getRowModel().rows.length;
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
          const row = table.getRowModel().rows[currentIndex];
          row?.toggleSelected();
          selectionAnchorRef.current = currentIndex;
          break;
        }
        default:
          break;
      }
    },
    [table, enableSelection, focusRowAt],
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
        const pageRows = table.getRowModel().rows;
        const rangeSelection: RowSelectionState = {};
        for (let i = start; i <= end; i++) {
          const row = pageRows[i];
          if (row?.getCanSelect()) rangeSelection[row.id] = true;
        }
        table.setRowSelection((current) => ({ ...current, ...rangeSelection }));
        return;
      }

      if (target.closest('[role="checkbox"]')) {
        selectionAnchorRef.current = rowIndex;
      }
    },
    [table, enableSelection],
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
      {rows.length || (grouping && grouping.collapsedKeys.length > 0 && !isLoading) ? (
        <DataTableRowActionsContext value={rowActions}>
          {bodyItems.map((item) =>
            item.kind === "group" ? (
              <DataTableGroupHeader
                key={`group:${item.group.key}`}
                group={item.group}
                collapsed={item.collapsed}
                colSpan={colSpan}
                onToggle={grouping!.onToggleGroup}
              />
            ) : (
              <Fragment key={item.row.id}>
                <DataTableRow
                  row={item.row}
                  rowIndex={item.index}
                  selected={item.row.getIsSelected()}
                  isLastRow={item.index === rows.length - 1 && expandedRowId !== item.row.id}
                  columns={columns}
                  columnVisibility={columnVisibility}
                  columnOrder={columnOrder}
                  columnPinning={columnPinning}
                  canEditCells={canEditCells}
                  formatClass={getFormatClass?.(item.row)}
                  rowClassName={getRowClassName?.(item.row)}
                  hasRowActions={hasRowActions}
                  onRowClick={onRowClick}
                  openPanelEdit={openPanelEdit}
                  hasPanel={hasPanel}
                  canOpenPanel={canOpenPanel}
                  canUpdate={canUpdate}
                  editingColumnId={cellEditing?.rowId === item.row.id ? cellEditing.columnId : null}
                  isCursor={cursorRowId === item.row.id}
                  isExpanded={expandedRowId === item.row.id}
                />
                {expansion && expandedRowId === item.row.id ? (
                  <DataTableExpandedRow rowId={item.row.id} colSpan={colSpan}>
                    {expansion.renderExpandedRow(item.row, { collapse: collapseExpanded })}
                  </DataTableExpandedRow>
                ) : null}
              </Fragment>
            ),
          )}
        </DataTableRowActionsContext>
      ) : isLoading ? (
        <TableRow>
          <TableCell colSpan={columns.length} className="h-24 rounded-b-md border-b text-center">
            <div className="border-border bg-muted-foreground/10 text-foreground mx-auto flex w-fit flex-row items-center justify-center rounded-md border p-2 text-sm font-medium">
              <Spinner className="size-4" />
              <p className="text-foreground text-xs">{t("Loading data...")}</p>
            </div>
          </TableCell>
        </TableRow>
      ) : null}
    </TableBody>
  );
}
