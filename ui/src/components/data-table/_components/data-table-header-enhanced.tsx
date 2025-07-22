"use no memo";
import { TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { flexRender, type Table } from "@tanstack/react-table";
import { SortableContext, horizontalListSortingStrategy } from "@dnd-kit/sortable";
import { DraggableTableHeader } from "./data-table-draggable";

export function DataTableHeaderEnhanced<TData>({ 
  table,
  enableDragging = false,
}: { 
  table: Table<TData>;
  enableDragging?: boolean;
}) {
  const columnOrder = table.getState().columnOrder;
  
  if (!enableDragging) {
    return (
      <TableHeader>
        {table.getHeaderGroups().map((headerGroup) => (
          <TableRow key={headerGroup.id}>
            {headerGroup.headers.map((header) => {
              return (
                <TableHead
                  key={header.id}
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

  return (
    <TableHeader>
      {table.getHeaderGroups().map((headerGroup) => (
        <TableRow key={headerGroup.id}>
          <SortableContext
            items={columnOrder}
            strategy={horizontalListSortingStrategy}
          >
            {headerGroup.headers.map((header) => (
              <DraggableTableHeader key={header.id} header={header} />
            ))}
          </SortableContext>
        </TableRow>
      ))}
    </TableHeader>
  );
}