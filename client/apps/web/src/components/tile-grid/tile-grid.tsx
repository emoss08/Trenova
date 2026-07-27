import {
  closestCenter,
  DndContext,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import { arrayMove, rectSortingStrategy, SortableContext } from "@dnd-kit/sortable";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import type { GridItem } from "./pack-tiles";

export type TileGridProps<T extends GridItem> = {
  items: T[];
  columns: number;
  /** Height of one grid row in pixels; a tile's h is a multiple of this. */
  rowHeight?: number;
  className?: string;
  /**
   * Receives the reordered list. Callers that persist x/y should pack it before
   * saving; callers that persist order alone can store it as-is.
   */
  onReorder: (items: T[]) => void;
  children: (item: T, index: number) => ReactNode;
};

/**
 * The shared canvas behind the report dashboard and the home screen: an
 * ordered list of tiles flowed into a fixed-column CSS grid, reordered by
 * dragging. Both surfaces use the same component so a change to how tiles move
 * cannot land on one and miss the other.
 */
export function TileGrid<T extends GridItem>({
  items,
  columns,
  rowHeight = 64,
  className,
  onReorder,
  children,
}: TileGridProps<T>) {
  const sensors = useSensors(useSensor(PointerSensor), useSensor(KeyboardSensor));

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;

    const from = items.findIndex((item) => item.id === active.id);
    const to = items.findIndex((item) => item.id === over.id);
    if (from < 0 || to < 0) return;

    onReorder(arrayMove(items, from, to));
  };

  return (
    <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
      <SortableContext items={items.map((item) => item.id)} strategy={rectSortingStrategy}>
        <div
          className={cn("grid gap-3", className)}
          style={{
            gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`,
            gridAutoRows: `${rowHeight}px`,
          }}
        >
          {items.map((item, index) => children(item, index))}
        </div>
      </SortableContext>
    </DndContext>
  );
}
