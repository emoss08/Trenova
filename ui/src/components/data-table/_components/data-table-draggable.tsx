/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

"use no memo";
import { TableCell, TableHead } from "@/components/ui/table";
import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { flexRender, type Cell, type Header } from "@tanstack/react-table";
import { GripVertical } from "lucide-react";
import type { CSSProperties } from "react";

export function DraggableTableHeader<TData>({
  header,
}: {
  header: Header<TData, unknown>;
}) {
  const { attributes, isDragging, listeners, setNodeRef, transform } =
    useSortable({
      id: header.column.id,
    });

  const style: CSSProperties = {
    opacity: isDragging ? 0.8 : 1,
    position: "relative",
    transform: CSS.Translate.toString(transform),
    transition: "width transform 0.2s ease-in-out",
    whiteSpace: "nowrap",
    width: `var(--header-${header.id.replace(".", "-")}-size)`,
    zIndex: isDragging ? 1 : 0,
  };

  return (
    <TableHead
      colSpan={header.colSpan}
      ref={setNodeRef}
      style={style}
      aria-sort={
        header.column.getIsSorted() === "asc"
          ? "ascending"
          : header.column.getIsSorted() === "desc"
            ? "descending"
            : "none"
      }
      className="group relative select-none truncate"
    >
      <div className="flex items-center gap-1">
        {header.isPlaceholder
          ? null
          : flexRender(header.column.columnDef.header, header.getContext())}
        <button
          {...attributes}
          {...listeners}
          className="cursor-move p-0.5 hover:bg-muted rounded touch-none opacity-0 group-hover:opacity-100 transition-opacity focus:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
          aria-label={`Reorder ${header.column.columnDef.header} column`}
          tabIndex={0}
        >
          <GripVertical className="h-3 w-3 text-muted-foreground hover:text-foreground transition-colors" />
        </button>
      </div>
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
}

export function DragAlongCell<TData>({ cell }: { cell: Cell<TData, unknown> }) {
  const { isDragging, setNodeRef, transform } = useSortable({
    id: cell.column.id,
  });

  const style: CSSProperties = {
    opacity: isDragging ? 0.8 : 1,
    position: "relative",
    transform: CSS.Translate.toString(transform),
    transition: "width transform 0.2s ease-in-out",
    width: `var(--col-${cell.column.id.replace(".", "-")}-size)`,
    maxWidth: `var(--col-${cell.column.id.replace(".", "-")}-size)`,
    zIndex: isDragging ? 1 : 0,
  };

  return (
    <TableCell style={style} ref={setNodeRef} className="font-table truncate">
      {flexRender(cell.column.columnDef.cell, cell.getContext())}
    </TableCell>
  );
}
