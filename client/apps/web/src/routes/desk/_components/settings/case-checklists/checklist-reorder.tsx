"use no memo";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { builtinStepLabel } from "@/lib/case-checklist-labels";
import { formatCasePosition } from "@/lib/case-checklist-steps";
import type { ChecklistItemMode, TemplateItem } from "@/types/case-checklist";
import {
  closestCenter,
  DndContext,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import { restrictToVerticalAxis } from "@dnd-kit/modifiers";
import { SortableContext, useSortable, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useLayoutEffect, useRef, type KeyboardEvent } from "react";

function modeLabel(mode: ChecklistItemMode, t: TranslateFn): string {
  switch (mode) {
    case "Required":
      return t("Required");
    case "Optional":
      return t("Optional");
    case "Off":
      return t("Off");
  }
}

/**
 * Every step in the order the case shows them, locked ones too, to drag
 * into the order a team works them. A handle moves its step one place with
 * the arrow keys and keeps focus as it goes; a step that is off is dimmed
 * but can still be moved.
 */
export function ChecklistReorder({
  items,
  sortIds,
  positions,
  locked,
  onMove,
}: {
  items: readonly TemplateItem[];
  sortIds: readonly string[];
  positions: ReadonlyMap<string, number>;
  locked: readonly string[];
  onMove: (from: number, to: number) => void;
}) {
  const t = useT();
  const handles = useRef(new Map<string, HTMLButtonElement>());
  const focusKey = useRef<string | null>(null);
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 4 } }));

  useLayoutEffect(() => {
    if (focusKey.current) {
      handles.current.get(focusKey.current)?.focus();
      focusKey.current = null;
    }
  }, [items]);

  const onDragEnd = ({ active, over }: DragEndEvent) => {
    if (!over || active.id === over.id) {
      return;
    }
    const from = sortIds.indexOf(String(active.id));
    const to = sortIds.indexOf(String(over.id));
    if (from >= 0 && to >= 0) {
      onMove(from, to);
    }
  };

  const onHandleKey = (event: KeyboardEvent<HTMLButtonElement>, index: number) => {
    if (event.key !== "ArrowUp" && event.key !== "ArrowDown") {
      return;
    }
    event.preventDefault();
    const to = index + (event.key === "ArrowUp" ? -1 : 1);
    if (to < 0 || to >= items.length) {
      return;
    }
    focusKey.current = items[index].key;
    onMove(index, to);
  };

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCenter}
      modifiers={[restrictToVerticalAxis]}
      onDragEnd={onDragEnd}
    >
      <SortableContext items={[...sortIds]} strategy={verticalListSortingStrategy}>
        <ol className="dk-ck-ord" aria-label={t("Order on the case")}>
          {items.map((item, index) => {
            const title = item.custom
              ? item.custom.label.trim() || t("New step")
              : builtinStepLabel(item.key, t);
            return (
              <ReorderCard
                key={sortIds[index] ?? item.key}
                sortId={sortIds[index] ?? item.key}
                title={title}
                position={formatCasePosition(positions.get(item.key))}
                off={item.mode === "Off"}
                tag={
                  locked.includes(item.key) ? (
                    <>
                      <DeskIcon name="lock" size={12} />
                      {t("Always required")}
                    </>
                  ) : (
                    modeLabel(item.mode, t)
                  )
                }
                handleRef={(node) => {
                  if (node) handles.current.set(item.key, node);
                  else handles.current.delete(item.key);
                }}
                onHandleKey={(event) => onHandleKey(event, index)}
                moveLabel={t("Move {0}. Use the arrow keys.", title)}
              />
            );
          })}
        </ol>
      </SortableContext>
    </DndContext>
  );
}

function ReorderCard({
  sortId,
  title,
  position,
  off,
  tag,
  handleRef,
  onHandleKey,
  moveLabel,
}: {
  sortId: string;
  title: string;
  position: string;
  off: boolean;
  tag: React.ReactNode;
  handleRef: (node: HTMLButtonElement | null) => void;
  onHandleKey: (event: KeyboardEvent<HTMLButtonElement>) => void;
  moveLabel: string;
}) {
  const { attributes, listeners, setNodeRef, setActivatorNodeRef, transform, transition, isDragging } =
    useSortable({ id: sortId });

  return (
    <li
      ref={setNodeRef}
      style={{ transform: CSS.Transform.toString(transform), transition }}
      className={cn("dk-ck-o", isDragging && "dk-drag", off && "dk-off")}
    >
      <button
        type="button"
        className="dk-ck-grip"
        aria-label={moveLabel}
        {...attributes}
        {...listeners}
        ref={(node) => {
          setActivatorNodeRef(node);
          handleRef(node);
        }}
        onKeyDown={onHandleKey}
      >
        <i />
      </button>
      <span className="dk-ck-n">{position}</span>
      <b>{title}</b>
      <span className="dk-ck-otag">{tag}</span>
    </li>
  );
}
