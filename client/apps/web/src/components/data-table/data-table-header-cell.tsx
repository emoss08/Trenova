"use no memo";
import { TableHead } from "@trenova/shared/components/ui/table";
import {
  columnHeaderLabel,
  columnSizeVar,
  pinnedCellClass,
  pinnedCellStyle,
} from "@/lib/data-table";
import { cn } from "@trenova/shared/lib/utils";
import type { SortDirection, SortField, Header } from "@trenova/shared/types/data-table";
import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { flexRender, type RowData } from "@tanstack/react-table";
import { DataTableColumnHeader } from "./data-table-column-header";
import { DataTableColumnResizeHandle } from "./data-table-column-resize-handle";

type DataTableHeaderCellProps<TData extends RowData> = {
  header: Header<TData, unknown>;
  sort: SortField[];
  onSort: (field: string, direction: SortDirection | null) => void;
};

export function DataTableHeaderCell<TData extends RowData>({
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
        "group/head border-border relative border-b",
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
          title={columnHeaderLabel(column)}
          currentSort={sort}
          onSort={onSort}
        />
      ) : (
        flexRender(column.columnDef.header, header.getContext())
      )}
      <DataTableColumnResizeHandle header={header} />
    </TableHead>
  );
}
