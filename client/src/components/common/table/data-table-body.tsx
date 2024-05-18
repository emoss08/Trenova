import { TableBody, TableCell, TableRow } from "@/components/ui/table";
import { cn } from "@/lib/utils";
import {
  ColumnDef,
  Row,
  Table as TableType,
  flexRender,
} from "@tanstack/react-table";
import { Fragment } from "react";

export function DataTableBody<K extends Record<string, any>>({
  table,
  setCurrentRecord,
  setEditDrawerOpen,
  columns,
  renderSubComponent,
}: {
  table: TableType<K>;
  setCurrentRecord: (currentRecord: K | null) => void;
  setEditDrawerOpen: (editDrawerOpen: boolean) => void;
  columns: ColumnDef<K>[];
  renderSubComponent?: (props: { row: Row<K> }) => React.ReactElement;
}) {
  return (
    <TableBody>
      {table.getRowModel().rows?.length ? (
        table.getRowModel().rows.map((row) => (
          <Fragment key={row.id}>
            <TableRow
              data-state={row.getIsSelected() ? "selected" : undefined}
              className={cn(row.getIsExpanded() ? "bg-accent" : "")}
            >
              {row.getVisibleCells().map((cell) => (
                <TableCell
                  key={cell.id}
                  className="cursor-pointer"
                  onDoubleClick={() => {
                    setCurrentRecord(row.original);
                    setEditDrawerOpen(true);
                  }}
                >
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </TableCell>
              ))}
            </TableRow>
            {/* Expanded row */}
            {row.getIsExpanded() && (
              <tr>
                <td colSpan={row.getVisibleCells().length}>
                  {renderSubComponent && renderSubComponent({ row })}
                </td>
              </tr>
            )}
          </Fragment>
        ))
      ) : (
        <TableRow>
          <TableCell colSpan={columns.length} className="h-24 text-center">
            No data available to display.
          </TableCell>
        </TableRow>
      )}
    </TableBody>
  );
}
