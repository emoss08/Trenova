import { TableBody, TableCell, TableRow } from "@/components/ui/table";
import { DataTableBodyProps } from "@/types/data-table";
import { flexRender } from "@tanstack/react-table";

export function DataTableBody<TData extends Record<string, any>>({
  table,
}: DataTableBodyProps<TData>) {
  // Render empty state
  if (!table.getRowModel().rows?.length) {
    return (
      <TableBody>
        <TableRow>
          <TableCell
            colSpan={table.getAllColumns().length}
            className="h-24 text-center"
            role="cell"
            aria-label="No results available"
          >
            No results.
          </TableCell>
        </TableRow>
      </TableBody>
    );
  }

  return (
    <TableBody>
      {table.getRowModel().rows.map((row) => (
        <TableRow
          key={row.id}
          data-state={row.getIsSelected() ? "selected" : undefined}
          className="hover:bg-muted/40 transition-colors duration-200"
          role="row"
          aria-selected={row.getIsSelected()}
        >
          {row.getVisibleCells().map((cell) => (
            <TableCell
              key={cell.id}
              role="cell"
              aria-label={`${cell.column.id} cell`}
              tabIndex={0}
            >
              {flexRender(cell.column.columnDef.cell, cell.getContext())}
            </TableCell>
          ))}
        </TableRow>
      ))}
    </TableBody>
  );
}
