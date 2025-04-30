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
  return (
    <TableRow
      id={row.id}
      tabIndex={0}
      data-state={selected && "selected"}
      className={cn(
        "[&>:not(:last-child)]:border-r",
        "outline-1 -outline-offset-1 outline-primary transition-colors focus-visible:bg-muted/50 focus-visible:outline data-[state=selected]:outline",
      )}
      onClick={() => row.toggleSelected()}
      onKeyDown={(event) => {
        if (event.key === "Enter") {
          event.preventDefault();
          row.toggleSelected();
        }
      }}
    >
      {row.getVisibleCells().map((cell) => (
        <TableCell
          key={cell.id}
          role="cell"
          aria-label={`${cell.column.id} cell`}
          className="truncate border-b border-border"
        >
          {flexRender(cell.column.columnDef.cell, cell.getContext())}
        </TableCell>
      ))}
    </TableRow>
  );
}

const MemoizedRow = React.memo(
  DataTableRow,
  (prev, next) =>
    prev.row.id === next.row.id && prev.selected === next.selected,
) as typeof DataTableRow;

export function DataTableBody<TData extends Record<string, any>>({
  table,
  columns,
}: DataTableBodyProps<TData>) {
  return (
    <TableBody
      id="content"
      tabIndex={-1}
      className="outline-1 -outline-offset-1 outline-primary transition-colors focus-visible:outline"
    >
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
          <TableCell colSpan={columns.length} className="h-24 text-center">
            No results.
          </TableCell>
        </TableRow>
      )}
    </TableBody>
  );
}
