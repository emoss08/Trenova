"use no memo";
import { TableBody, TableCell, TableRow } from "@/components/ui/table";
import { cn } from "@/lib/utils";
import { DataTableBodyProps } from "@/types/data-table";
import { flexRender, type Row, type Table } from "@tanstack/react-table";
import React from "react";

function DataTableRow<TData>({
  row,
  selected,
  isLastRow = false,
  table,
  // We don't actually use columnVisibility in the component,
  // but we need it for the memo comparison
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  columnVisibility,
}: {
  row: Row<TData>;
  selected?: boolean;
  isLastRow?: boolean;
  columnVisibility: Record<string, boolean>;
  table: Table<TData>;
}) {
  return (
    <TableRow
      id={row.id}
      tabIndex={0}
      data-state={selected && "selected"}
      onClick={() => row.toggleSelected()}
      onKeyDown={(event) => {
        if (event.key === "Enter") {
          event.preventDefault();
          row.toggleSelected();
        }
      }}
      className={cn(
        "[&>:not(:last-child)]:border-r border-border",
        "-outline-offset-1 outline-primary transition-colors focus-visible:bg-muted/50 focus-visible:outline data-[state=selected]:outline",
        table.options.meta?.getRowClassName?.(row),
      )}
    >
      {row.getVisibleCells().map((cell, index) => {
        const isFirstCell = index === 0;
        const isLastCell = index === row.getVisibleCells().length - 1;

        return (
          <TableCell
            key={cell.id}
            role="cell"
            aria-label={`${cell.column.id} cell`}
            className={cn(
              "border-b border-border bg-transparent",
              cell.column.columnDef.meta?.cellClassName,
              isLastRow && isFirstCell && "rounded-bl-md",
              isLastRow && isLastCell && "rounded-br-md",
            )}
          >
            {flexRender(cell.column.columnDef.cell, cell.getContext())}
          </TableCell>
        );
      })}
    </TableRow>
  );
}

const MemoizedRow = React.memo(DataTableRow, (prev, next) => {
  // Check ID and selection state first (fast checks)
  if (prev.row.id !== next.row.id || prev.selected !== next.selected) {
    return false;
  }

  // Check for column visibility changes
  const prevKeys = Object.keys(prev.columnVisibility);
  const nextKeys = Object.keys(next.columnVisibility);

  if (prevKeys.length !== nextKeys.length) {
    return false;
  }

  for (const key of prevKeys) {
    if (prev.columnVisibility[key] !== next.columnVisibility[key]) {
      return false;
    }
  }

  const prevOriginal = prev.row.original as Record<string, any>;
  const nextOriginal = next.row.original as Record<string, any>;

  // Compare updatedAt timestamps for data changes
  return prevOriginal.updatedAt === nextOriginal.updatedAt;
}) as typeof DataTableRow;

export function DataTableBody<TData extends Record<string, any>>({
  table,
  columns,
}: DataTableBodyProps<TData>) {
  return (
    <TableBody id="content" tabIndex={-1}>
      {table.getRowModel().rows?.length ? (
        table.getRowModel().rows.map((row, index) => {
          const isLastRow = index === table.getRowModel().rows.length - 1;
          return (
            <MemoizedRow
              key={row.id}
              row={row}
              selected={row.getIsSelected()}
              isLastRow={isLastRow}
              columnVisibility={table.getState().columnVisibility}
              table={table}
            />
          );
        })
      ) : (
        <TableRow>
          <TableCell
            colSpan={columns.length}
            className="h-24 text-center border-b rounded-b-md"
          >
            No results.
          </TableCell>
        </TableRow>
      )}
    </TableBody>
  );
}
