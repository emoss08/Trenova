import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { cn } from "@trenova/shared/lib/utils";
import { GripVerticalIcon } from "lucide-react";
import type { ReactNode } from "react";

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
 * One cell of a TileGrid. It owns the grid spans and the drag transform so
 * every tile on either canvas moves identically; the caller owns everything
 * inside.
 */
export function SortableTile({ id, w, h, editing, className, children }: SortableTileProps) {
  const sortable = useSortable({ id, disabled: !editing });

  return (
    <div
      ref={sortable.setNodeRef}
      style={{
        gridColumn: `span ${w} / span ${w}`,
        gridRow: `span ${h} / span ${h}`,
        transform: CSS.Transform.toString(sortable.transform),
        transition: sortable.transition,
      }}
      className={cn(
        "group/tile flex min-h-0 flex-col overflow-hidden rounded-lg border border-border bg-background",
        sortable.isDragging && "z-10 opacity-80 shadow-lg",
        className,
      )}
    >
      {children({ attributes: sortable.attributes, listeners: sortable.listeners })}
    </div>
  );
}

/** The grab affordance, so every canvas presents the same target. */
export function TileDragHandle({ attributes, listeners }: TileDragHandleProps) {
  return (
    <button
      type="button"
      className="shrink-0 cursor-grab text-muted-foreground hover:text-foreground"
      aria-label="Move tile"
      {...attributes}
      {...listeners}
    >
      <GripVerticalIcon className="size-3.5" />
    </button>
  );
}
