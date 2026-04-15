"use no memo";
import { TableBody, TableCell, TableRow } from "@/components/ui/table";
import { useDataTable } from "@/contexts/data-table-context";
import { cn } from "@/lib/utils";
import type { DataTableBodyProps, RowAction } from "@/types/data-table";
import { flexRender, type Row, type Table } from "@tanstack/react-table";
import { useCallback } from "react";
import { Spinner } from "../ui/spinner";
import { DataTableContextMenu } from "./_components/data-table-context-menu";
import { DataTableEmptyState } from "./data-table-empty-state";

const INTERACTIVE_SELECTOR =
  'button, a, input, select, textarea, [role="button"], [role="checkbox"], [role="switch"]';

function DataTableRow<TData>({
  row,
  selected,
  table,
  // We don't actually use columnVisibility in the component,
  // but we need it for the memo comparison
  // @ts-expect-error - This is a temporary solution to avoid the memo comparison
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  columnVisibility,
  contextMenuActions,
  onRowClick,
}: {
  row: Row<TData>;
  selected?: boolean;
  columnVisibility: Record<string, boolean>;
  table: Table<TData>;
  contextMenuActions?: RowAction<TData>[];
  onRowClick?: (row: Row<TData>) => void;
}) {
  const { openPanelEdit, hasPanel, canUpdate } = useDataTable<TData, unknown>();

  const isClickable = !!(onRowClick || (hasPanel && canUpdate));
  const hasContextMenu =
    (contextMenuActions && contextMenuActions.length > 0) || (hasPanel && canUpdate);

  const handleRowClick = useCallback(
    (e: React.MouseEvent<HTMLTableRowElement>) => {
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
      } else if (hasPanel && canUpdate) {
        openPanelEdit(row);
      }
    },
    [row, onRowClick, hasPanel, canUpdate, openPanelEdit, table],
  );

  const tableRow = (
    <TableRow
      id={row.id}
      data-state={selected && "selected"}
      onClick={isClickable ? handleRowClick : undefined}
      className={cn(
        "-outline-offset-2 outline-brand transition-colors data-[state=selected]:outline",
        isClickable && "cursor-pointer",
        table.options.meta?.getRowClassName?.(row),
      )}
    >
      {row.getVisibleCells().map((cell) => (
        <TableCell
          className={cn("truncate border-b border-border font-sans", {
            "border-b-0": row.index === table.getRowModel().rows.length - 1,
          })}
          key={cell.id}
          role="cell"
          aria-label={`${cell.column.id} cell`}
          style={{
            width: `var(--col-${cell.column.id.replace(".", "-")}-size)`,
            maxWidth: `var(--col-${cell.column.id.replace(".", "-")}-size)`,
          }}
        >
          {flexRender(cell.column.columnDef.cell, cell.getContext())}
        </TableCell>
      ))}
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

export function DataTableBody<TData extends Record<string, any>>({
  table,
  columns,
  isLoading,
  contextMenuActions,
  onRowClick,
}: DataTableBodyProps<TData> & {
  isLoading?: boolean;
}) {
  return (
    <TableBody
      id="content"
      tabIndex={-1}
      // REMINDER: avoids scroll (skipping the table header) when using skip to content
      style={{
        scrollMarginTop: "calc(var(--top-bar-height) + 40px)",
      }}
    >
      {table.getRowModel().rows?.length ? (
        table.getRowModel().rows.map((row) => {
          return (
            <DataTableRow
              key={row.id}
              row={row}
              selected={row.getIsSelected()}
              columnVisibility={table.getState().columnVisibility}
              table={table}
              contextMenuActions={contextMenuActions}
              onRowClick={onRowClick}
            />
          );
        })
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
            <DataTableEmptyState />
          </TableCell>
        </TableRow>
      )}
    </TableBody>
  );
}
