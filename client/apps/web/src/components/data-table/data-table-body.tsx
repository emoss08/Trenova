"use no memo";
import { TableBody, TableCell, TableRow } from "@trenova/shared/components/ui/table";
import { useDataTable } from "@/contexts/data-table-context";
import {
  columnSizeVar,
  pinnedCellClass,
  pinnedCellStyle,
  type CompiledFormatRules,
} from "@/lib/data-table";
import { cn } from "@trenova/shared/lib/utils";
import type { DataTableBodyProps, RowAction } from "@trenova/shared/types/data-table";
import type { ColumnPinningState } from "@tanstack/react-table";
import { flexRender, type Row, type Table } from "@tanstack/react-table";
import { memo, useCallback, useRef } from "react";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { DataTableContextMenu } from "./_components/data-table-context-menu";
import { DataTableEmptyState } from "./data-table-empty-state";

const INTERACTIVE_SELECTOR =
  'button, a, input, select, textarea, [role="button"], [role="checkbox"], [role="switch"]';

type DataTableRowProps<TData> = {
  row: Row<TData>;
  rowIndex: number;
  selected?: boolean;
  isLastRow: boolean;
  columnVisibility: Record<string, boolean>;
  columnOrder: string[];
  columnPinning: ColumnPinningState;
  formatClass?: string;
  table: Table<TData>;
  contextMenuActions?: RowAction<TData>[];
  onRowClick?: (row: Row<TData>) => void;
};

function DataTableRowInner<TData>({
  row,
  rowIndex,
  selected,
  isLastRow,
  formatClass,
  table,
  contextMenuActions,
  onRowClick,
}: DataTableRowProps<TData>) {
  const { openPanelEdit, hasPanel, canOpenPanel } = useDataTable<TData, unknown>();

  const isClickable = !!(onRowClick || (hasPanel && canOpenPanel));
  const hasContextMenu =
    (contextMenuActions && contextMenuActions.length > 0) || (hasPanel && canOpenPanel);

  const handleRowClick = useCallback(
    (e: React.MouseEvent<HTMLTableRowElement>) => {
      if (e.shiftKey) return;
      const target = e.target as HTMLElement;

      if (target.closest(INTERACTIVE_SELECTOR)) return;

      const cell = target.closest("td");
      if (cell) {
        const cellIndex = cell.cellIndex;
        const visibleColumns = table.getVisibleFlatColumns();
        if (visibleColumns[cellIndex]?.id === "select") return;
      }

      const selection = window.getSelection();
      if (selection && selection.toString().length > 0) return;

      if (onRowClick) {
        onRowClick(row);
      } else if (hasPanel && canOpenPanel) {
        openPanelEdit(row);
      }
    },
    [row, onRowClick, hasPanel, canOpenPanel, openPanelEdit, table],
  );

  const tableRow = (
    <TableRow
      id={row.id}
      data-row-index={rowIndex}
      tabIndex={-1}
      data-state={selected && "selected"}
      onClick={isClickable ? handleRowClick : undefined}
      className={cn(
        "group/row -outline-offset-2 outline-brand transition-colors focus-visible:outline data-[state=selected]:outline",
        isClickable && "cursor-pointer",
        formatClass,
        table.options.meta?.getRowClassName?.(row),
      )}
    >
      {row.getVisibleCells().map((cell) => {
        const pinned = cell.column.getIsPinned();
        return (
          <TableCell
            className={cn(
              "truncate border-b border-border font-sans",
              isLastRow && "border-b-0",
              pinned && pinnedCellClass(cell.column),
              pinned && "group-hover/row:bg-muted",
            )}
            key={cell.id}
            role="cell"
            aria-label={`${cell.column.id} cell`}
            style={{
              width: `var(${columnSizeVar(cell.column.id)})`,
              maxWidth: `var(${columnSizeVar(cell.column.id)})`,
              ...pinnedCellStyle(cell.column),
            }}
          >
            {flexRender(cell.column.columnDef.cell, cell.getContext())}
          </TableCell>
        );
      })}
    </TableRow>
  );

  if (hasContextMenu) {
    return (
      <DataTableContextMenu row={row} actions={contextMenuActions}>
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
  hasActiveFilters,
  onClearFilters,
}: DataTableBodyProps<TData> & {
  isLoading?: boolean;
  getFormatClass?: CompiledFormatRules<TData> | null;
  hasActiveFilters?: boolean;
  onClearFilters?: () => void;
}) {
  const rows = table.getRowModel().rows;
  const { columnVisibility, columnOrder, columnPinning } = table.getState();
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
        const rangeSelection: Record<string, boolean> = {};
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
      {rows.length ? (
        rows.map((row, index) => (
          <DataTableRow
            key={row.id}
            row={row}
            rowIndex={index}
            selected={row.getIsSelected()}
            isLastRow={index === rows.length - 1}
            columnVisibility={columnVisibility}
            columnOrder={columnOrder}
            columnPinning={columnPinning}
            formatClass={getFormatClass?.(row)}
            table={table}
            contextMenuActions={contextMenuActions}
            onRowClick={onRowClick}
          />
        ))
      ) : isLoading ? (
        <TableRow>
          <TableCell colSpan={columns.length} className="h-24 rounded-b-md border-b text-center">
            <div className="mx-auto flex w-fit flex-row items-center justify-center rounded-md border border-border bg-muted-foreground/10 p-2 text-sm font-medium text-foreground">
              <Spinner className="size-4" />
              <p className="text-xs text-foreground">Loading data...</p>
            </div>
          </TableCell>
        </TableRow>
      ) : (
        <TableRow>
          <TableCell colSpan={columns.length} className="h-[300px] p-0">
            <DataTableEmptyState
              hasActiveFilters={hasActiveFilters}
              onClearFilters={onClearFilters}
            />
          </TableCell>
        </TableRow>
      )}
    </TableBody>
  );
}
