"use no memo";
import { TableBody, TableCell, TableRow } from "@/components/ui/table";
import { DataTableBodyProps } from "@/types/data-table";
import { flexRender, type Cell, type Row } from "@tanstack/react-table";
import { memo } from "react";

const DataTableCell = memo(function DataTableCell<
  TData extends Record<string, any>,
  TValue,
>({ cell }: { cell: Cell<TData, TValue> }) {
  return (
    <TableCell
      key={cell.id}
      role="cell"
      aria-label={`${cell.column.id} cell`}
      tabIndex={0}
    >
      {flexRender(cell.column.columnDef.cell, cell.getContext())}
    </TableCell>
  );
});

function DataTableRow<TData extends Record<string, any>>({
  row,
}: {
  row: Row<TData>;
}) {
  return (
    <TableRow
      key={row.id}
      data-state={row.getIsSelected() ? "selected" : undefined}
      className="hover:bg-muted transition-colors duration-200"
      role="row"
      aria-selected={row.getIsSelected()}
    >
      {row.getVisibleCells().map((cell) => (
        <DataTableCell key={cell.id} cell={cell as any} />
      ))}
    </TableRow>
  );
}

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

export function DataTableBody<TData extends Record<string, any>>({
  table,
}: DataTableBodyProps<TData>) {
  if (!table.getRowModel().rows?.length) {
    return <EmptyTableBody />;
  }

  return (
    <TableBody>
      {table.getRowModel().rows.map((row) => (
        <DataTableRow key={row.id} row={row} />
      ))}
    </TableBody>
  );
}
