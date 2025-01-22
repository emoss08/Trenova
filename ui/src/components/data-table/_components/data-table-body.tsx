"use no memo";
import { TableBody, TableCell, TableRow } from "@/components/ui/table";
import { DataTableBodyProps } from "@/types/data-table";
import { flexRender } from "@tanstack/react-table";

export function DataTableBody<TData extends Record<string, any>>({
  table,
  setCurrentRecord,
  setEditModalOpen,
  isLoading,
}: DataTableBodyProps<TData>) {
  return (
    <TableBody>
      {table.getRowModel().rows?.length ? (
        table.getRowModel().rows.map((row) => (
          <TableRow
            key={row.id}
            data-state={row.getIsSelected() && "selected"}
            className="hover:cursor-pointer hover:bg-muted/40"
          >
            {row.getVisibleCells().map((cell) => (
              <TableCell
                key={cell.id}
                onDoubleClick={() => {
                  if (!isLoading) {
                    setCurrentRecord(row.original);
                    setEditModalOpen(true);
                  }
                }}
              >
                {flexRender(cell.column.columnDef.cell, cell.getContext())}
              </TableCell>
            ))}
          </TableRow>
        ))
      ) : (
        <TableRow>
          <TableCell
            colSpan={table.getAllColumns().length}
            className="h-24 text-center"
          >
            No results.
          </TableCell>
        </TableRow>
      )}
    </TableBody>
  );
}
