/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

"use no memo";
import { TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { flexRender, type Table } from "@tanstack/react-table";

export function DataTableHeader<TData>({
  table,
}: {
  table: Table<TData>;
}) {
  return (
    <TableHeader>
      {table.getHeaderGroups().map((headerGroup) => (
        <TableRow key={headerGroup.id}>
          {headerGroup.headers.map((header) => {
            return (
              <TableHead
                key={header.id}
                className="relative select-none truncate"
                style={{
                  width: `var(--header-${header.id.replace(".", "-")}-size)`,
                }}
                aria-sort={
                  header.column.getIsSorted() === "asc"
                    ? "ascending"
                    : header.column.getIsSorted() === "desc"
                      ? "descending"
                      : "none"
                }
              >
                {header.isPlaceholder
                  ? null
                  : flexRender(
                      header.column.columnDef.header,
                      header.getContext(),
                    )}
                {header.column.getCanResize() && (
                  <div
                    onDoubleClick={() => header.column.resetSize()}
                    onMouseDown={header.getResizeHandler()}
                    onTouchStart={header.getResizeHandler()}
                    className="absolute -right-2 top-0 z-10 flex h-full w-4 cursor-col-resize touch-none justify-center user-select-none before:absolute before:inset-y-0 before:w-px before:translate-x-px before:bg-border"
                  />
                )}
              </TableHead>
            );
          })}
        </TableRow>
      ))}
    </TableHeader>
  );
}
