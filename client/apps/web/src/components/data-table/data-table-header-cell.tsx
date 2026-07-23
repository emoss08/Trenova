"use no memo";
import { TableHead } from "@trenova/shared/components/ui/table";
import { columnSizeVar, pinnedCellClass, pinnedCellStyle } from "@/lib/data-table";
import { cn } from "@trenova/shared/lib/utils";
import type { SortDirection, SortField } from "@trenova/shared/types/data-table";
import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { flexRender, type Header } from "@tanstack/react-table";
import { DataTableColumnHeader } from "./data-table-column-header";

type DataTableHeaderCellProps<TData> = {
  header: Header<TData, unknown>;
  sort: SortField[];
  onSort: (field: string, direction: SortDirection | null) => void;
};

export function DataTableHeaderCell<TData>({
  header,
  sort,
  onSort,
}: DataTableHeaderCellProps<TData>) {
  const { column } = header;
  const meta = column.columnDef.meta;
  const isSortable = meta?.sortable !== false;
  const isPinned = column.getIsPinned();
  const canReorder = column.id !== "select" && !isPinned;

  const { listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: column.id,
    disabled: !canReorder,
  });

  return (
    <TableHead
      ref={setNodeRef}
      className={cn(
        "group/head relative border-b border-border",
        pinnedCellClass(column) ?? undefined,
        isPinned && "bg-sidebar",
        isDragging && "z-20 opacity-80",
      )}
      style={{
        width: `var(${columnSizeVar(column.id)})`,
        ...pinnedCellStyle(column),
        transform: canReorder ? CSS.Translate.toString(transform) : undefined,
        transition: canReorder ? transition : undefined,
      }}
      {...(canReorder ? listeners : {})}
    >
      {header.isPlaceholder ? null : isSortable ? (
        <DataTableColumnHeader
          column={column}
          title={
            typeof column.columnDef.header === "string"
              ? column.columnDef.header
              : meta?.label || column.id
          }
          currentSort={sort}
          onSort={onSort}
        />
      ) : (
        flexRender(column.columnDef.header, header.getContext())
      )}
      {column.getCanResize() && (
        <div
          role="separator"
          aria-orientation="vertical"
          aria-label={`Resize ${column.id} column`}
          onMouseDown={header.getResizeHandler()}
          onTouchStart={header.getResizeHandler()}
          onDoubleClick={() => column.resetSize()}
          onPointerDown={(e) => e.stopPropagation()}
          className={cn(
            "absolute inset-y-0 right-0 z-10 w-1 cursor-col-resize touch-none transition-colors select-none hover:bg-border",
            column.getIsResizing() && "bg-primary hover:bg-primary",
          )}
        />
      )}
    </TableHead>
  );
}
