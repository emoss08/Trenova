import { cn } from "@trenova/shared/lib/utils";
import type { Header } from "@trenova/shared/types/data-table";
import type { RowData } from "@tanstack/react-table";
import { useT } from "@trenova/shared/i18n/use-t";
import { useTableAtom } from "@trenova/shared/hooks/use-table-atom";
import { useOptionalDataTable } from "@/contexts/data-table-context";
import { isColumnResizable } from "@/lib/data-table";

export function DataTableColumnResizeHandle<TData extends RowData>({
  header,
}: {
  header: Header<TData, unknown>;
}) {
  if (!isColumnResizable(header.column)) return null;

  return <ResizeGrip header={header} />;
}

/**
 * The handle follows only whether its own column is the one being dragged, so a
 * drag redraws this one handle when it starts and when it ends, and nothing else.
 */
function ResizeGrip<TData extends RowData>({ header }: { header: Header<TData, unknown> }) {
  const t = useT();
  const fitColumns = useOptionalDataTable()?.fitColumns;
  const { column } = header;
  const isResizing = useTableAtom(
    column.table.atoms.columnResizing,
    (resizing) => resizing.isResizingColumn === column.id,
  );

  return (
    <div
      role="separator"
      aria-orientation="vertical"
      aria-label={t("Resize {0} column", column.id)}
      title={t("Drag to resize, double-click to fit, Alt + double-click to reset")}
      data-resizing={isResizing || undefined}
      onMouseDown={header.getResizeHandler()}
      onTouchStart={header.getResizeHandler()}
      onDoubleClick={(event) => {
        if (event.altKey || !fitColumns) column.resetSize();
        else fitColumns([column.id]);
      }}
      onPointerDown={(e) => e.stopPropagation()}
      className="group/resize absolute inset-y-0 right-0 z-10 flex w-2 cursor-col-resize touch-none items-center justify-end select-none"
    >
      <span
        data-slot="column-resize-grip"
        aria-hidden="true"
        className={cn(
          "bg-muted-foreground/40 h-4 w-px rounded-full transition-[height,width,background-color] duration-150",
          "group-hover/head:bg-muted-foreground/70 group-hover/head:h-full",
          "group-hover/resize:bg-primary group-hover/resize:h-full group-hover/resize:w-0.5",
          isResizing && "bg-primary h-full w-0.5",
        )}
      />
    </div>
  );
}
