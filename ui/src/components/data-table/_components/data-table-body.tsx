"use no memo";
import { TableBody, TableCell, TableRow } from "@/components/ui/table";
import { DataTableBodyProps } from "@/types/data-table";
import { flexRender } from "@tanstack/react-table";
import { memo } from "react";

// Memoized table cell component to prevent unnecessary re-renders
const TableCellMemo = memo(({ cell }: { cell: any }) => (
  <TableCell
    key={cell.id}
    role="cell"
    aria-label={`${cell.column.id} cell`}
    tabIndex={0}
  >
    {flexRender(cell.column.columnDef.cell, cell.getContext())}
  </TableCell>
));

TableCellMemo.displayName = "TableCellMemo";

// Memoized row component to prevent unnecessary re-renders
const TableRowMemo = memo(({ row }: { row: any }) => (
  <TableRow
    key={row.id}
    data-state={row.getIsSelected() ? "selected" : undefined}
    className="hover:bg-muted transition-colors duration-200"
    role="row"
    aria-selected={row.getIsSelected()}
  >
    {row.getVisibleCells().map((cell: any) => (
      <TableCellMemo key={cell.id} cell={cell} />
    ))}
  </TableRow>
));

TableRowMemo.displayName = "TableRowMemo";

// Empty state component for tables with no rows
const EmptyTableBody = memo(() => (
  <TableBody>
    <TableRow>
      <TableCell
        colSpan={100}
        className="h-24 text-center"
        role="cell"
        aria-label="No results available"
      >
        No results.
      </TableCell>
    </TableRow>
  </TableBody>
));

EmptyTableBody.displayName = "EmptyTableBody";

// Main DataTableBody component
export function DataTableBody<TData extends Record<string, any>>({
  table,
}: DataTableBodyProps<TData>) {
  // Render empty state
  if (!table.getRowModel().rows?.length) {
    return <EmptyTableBody />;
  }

  return (
    <TableBody>
      {table.getRowModel().rows.map((row) => (
        <TableRowMemo key={row.id} row={row} />
      ))}
    </TableBody>
  );
}
