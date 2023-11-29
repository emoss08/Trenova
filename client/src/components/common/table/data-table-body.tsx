/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

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
              data-state={row.getIsSelected() && "selected"}
              className={row.getIsExpanded() ? "bg-muted/40" : ""}
            >
              {row.getVisibleCells().map((cell) => (
                <TableCell
                  key={cell.id}
                  className={cn("cursor-pointer")}
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
