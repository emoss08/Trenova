"use no memo";
import { cn } from "@trenova/shared/lib/utils";
import type { Header } from "@trenova/shared/types/data-table";
import type { RowData } from "@tanstack/react-table";

export function DataTableColumnResizeHandle<TData extends RowData>({
  header,
}: {
  header: Header<TData, unknown>;
}) {
  const { column } = header;
  const { minSize, maxSize } = column.columnDef;
  const hasFixedSize = minSize !== undefined && minSize === maxSize;
  if (!column.getCanResize() || hasFixedSize) return null;

  const isResizing = column.getIsResizing();

  return (
    <div
      role="separator"
      aria-orientation="vertical"
      aria-label={`Resize ${column.id} column`}
      title="Drag to resize, double-click to reset"
      data-resizing={isResizing || undefined}
      onMouseDown={header.getResizeHandler()}
      onTouchStart={header.getResizeHandler()}
      onDoubleClick={() => column.resetSize()}
      onPointerDown={(e) => e.stopPropagation()}
      className="group/resize absolute inset-y-0 right-0 z-10 flex w-2 cursor-col-resize touch-none items-center justify-end select-none"
    >
      <span
        data-slot="column-resize-grip"
        aria-hidden="true"
        className={cn(
          "bg-border h-4 w-px rounded-full transition-[height,width,background-color] duration-150",
          "group-hover/head:bg-muted-foreground/40 group-hover/head:h-full",
          "group-hover/resize:bg-primary group-hover/resize:h-full group-hover/resize:w-0.5",
          isResizing && "bg-primary h-full w-0.5",
        )}
      />
    </div>
  );
}
