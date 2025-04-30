import { TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { cn } from "@/lib/utils";
import { flexRender, type Table } from "@tanstack/react-table";

export function DataTableHeader<TData>({ table }: { table: Table<TData> }) {
  return (
    <TableHeader className={cn("sticky top-0 rounded-t-md z-20 bg-background")}>
      {table.getHeaderGroups().map((headerGroup) => (
        <TableRow
          key={headerGroup.id}
          className={cn(
            "bg-muted/50 hover:bg-muted/50",
            "[&>:not(:last-child)]:border-r",
          )}
        >
          {headerGroup.headers.map((header) => {
            return (
              <TableHead
                key={header.id}
                className={cn(
                  "relative select-none truncate border-t border-b border-border [&>.cursor-col-resize]:last:opacity-0",
                  header.index === 0 ? "rounded-tl-md" : "",
                  header.index === headerGroup.headers.length - 1
                    ? "rounded-tr-md"
                    : "",
                )}
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
                    className={cn(
                      "user-select-none absolute -right-2 top-0 z-10 flex h-full w-4 cursor-col-resize touch-none justify-center",
                      "before:absolute before:inset-y-0 before:w-px before:translate-x-px before:bg-border",
                    )}
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
