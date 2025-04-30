"use no memo";
import { TableBody, TableCell, TableRow } from "@/components/ui/table";
import { cn } from "@/lib/utils";
import { DataTableBodyProps } from "@/types/data-table";
import { flexRender, type Row } from "@tanstack/react-table";
import React from "react";

function DataTableRow<TData>({
  row,
  selected,
}: {
  row: Row<TData>;
  selected?: boolean;
}) {
  console.info("DataTableRow debug info", {
    row,
    selected,
  });
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
      )}
    >
      {row.getVisibleCells().map((cell) => (
        <TableCell
          key={cell.id}
          role="cell"
          aria-label={`${cell.column.id} cell`}
          className={cn("border-b border-border")}
        >
          {flexRender(cell.column.columnDef.cell, cell.getContext())}
        </TableCell>
      ))}
    </TableRow>
  );
}

const MemoizedRow = React.memo(DataTableRow, (prev, next) => {
  // Check ID and selection state first (fast checks)
  if (prev.row.id !== next.row.id || prev.selected !== next.selected) {
    return false;
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
        table
          .getRowModel()
          .rows.map((row) => (
            <MemoizedRow
              key={row.id}
              row={row}
              selected={row.getIsSelected()}
            />
          ))
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
