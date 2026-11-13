import { useT } from "@trenova/shared/i18n/use-t";
import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { cn } from "@trenova/shared/lib/utils";
import { GripVerticalIcon } from "@trenova/shared/components/icons";
import type { CSSProperties, ReactNode } from "react";
import { useTileGridColumns } from "./tile-grid";

export type TileDragHandleProps = {
  attributes: ReturnType<typeof useSortable>["attributes"];
  listeners: ReturnType<typeof useSortable>["listeners"];
};

export type SortableTileProps = {
  id: string;
  w: number;
  h: number;
  editing: boolean;
  className?: string;
  /**
   * Receives the drag props so the tile's own header can decide where the grab
   * affordance sits. They come from a single useSortable call — registering the
   * same id twice would leave dnd-kit holding the wrong node and silently break
   * dragging.
   */
  children: (drag: TileDragHandleProps) => ReactNode;
};

/**
 * The span a tile takes where the grid is too narrow for the width it was given: half
 * the grid when it was at most half, otherwise all of it, so two small tiles still sit
 * side by side on a tablet and nothing is squeezed into a sliver.
 */
export function narrowTileSpan(w: number, columns: number): number {
  const half = Math.floor(columns / 2);
  return w <= half ? half : columns;
}

/**
 * One cell of a TileGrid. It owns the grid spans and the drag transform so
 * every tile on either canvas moves identically; the caller owns everything
 * inside. The authored width holds from `lg` up; a tablet gets `narrowTileSpan`
 * and a phone the whole row.
 */
export function SortableTile({ id, w, h, editing, className, children }: SortableTileProps) {
  const columns = useTileGridColumns();
  const { setNodeRef, transform, transition, isDragging, attributes, listeners } = useSortable({
    id,
    disabled: !editing,
  });

  return (
    <div
      ref={setNodeRef}
      style={
        {
          "--tile-w": w,
          "--tile-w-narrow": narrowTileSpan(w, columns),
          gridRow: `span ${h} / span ${h}`,
          transform: CSS.Transform.toString(transform),
          transition,
        } as CSSProperties
      }
      className={cn(
        "group/tile border-border/80 bg-card hover:border-border flex min-h-0 flex-col overflow-hidden rounded-lg border transition-[border-color,box-shadow] duration-200 dark:shadow-[inset_0_1px_0_0_rgb(255_255_255/0.04)]",
        "col-span-full md:[grid-column:span_var(--tile-w-narrow)_/_span_var(--tile-w-narrow)] lg:[grid-column:span_var(--tile-w)_/_span_var(--tile-w)]",
        isDragging && "z-10 opacity-80",
        className,
      )}
    >
      {children({ attributes, listeners })}
    </div>
  );
}

/** The grab affordance, so every canvas presents the same target. */
export function TileDragHandle({ attributes, listeners }: TileDragHandleProps) {
  const t = useT();

  return (
    <button
      type="button"
      className="text-muted-foreground hover:text-foreground shrink-0 cursor-grab"
      aria-label={t("Move tile")}
      {...attributes}
      {...listeners}
    >
      <GripVerticalIcon className="size-3.5" />
    </button>
  );
}
