import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  closestCenter,
  DndContext,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import { restrictToVerticalAxis } from "@dnd-kit/modifiers";
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { PlusIcon } from "lucide-react";
import { ExceptionsInbox } from "./exceptions-inbox";
import { HosWatchPlaceholder } from "./hos-watch-placeholder";
import {
  ALL_MODULES,
  useRightStackStore,
  type RightStackModuleId,
} from "./right-stack-store";
import { UnassignedQueue } from "./unassigned-queue";

const MODULE_LABEL: Record<RightStackModuleId, string> = {
  unassigned: "Unassigned",
  exceptions: "Exceptions",
  hos: "HOS watch",
};

const RENDERERS: Record<RightStackModuleId, () => React.ReactElement> = {
  unassigned: UnassignedQueue,
  exceptions: ExceptionsInbox,
  hos: HosWatchPlaceholder,
};

export function RightStack() {
  const order = useRightStackStore.use.order();
  const hidden = useRightStackStore.use.hidden();
  const show = useRightStackStore.use.show();
  const reorder = useRightStackStore.use.reorder();

  const visible = order.filter((id) => !hidden.includes(id));

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;
    const oldIndex = order.indexOf(active.id as RightStackModuleId);
    const newIndex = order.indexOf(over.id as RightStackModuleId);
    if (oldIndex < 0 || newIndex < 0) return;
    reorder(arrayMove(order, oldIndex, newIndex));
  };

  if (visible.length === 0) {
    return (
      <aside className="flex h-full min-h-0 flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-border p-6">
        <p className="text-[11.5px] font-medium">All panels hidden</p>
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button variant="outline" size="sm">
                <PlusIcon className="size-3" />
                Add panel
              </Button>
            }
          />
          <DropdownMenuContent align="center" className="min-w-[180px]">
            {ALL_MODULES.map((id) => (
              <DropdownMenuItem
                key={id}
                title={MODULE_LABEL[id]}
                onClick={() => show(id)}
              />
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      </aside>
    );
  }

  return (
    <aside className="relative flex h-full min-h-0 flex-col gap-2">
      {hidden.length > 0 && (
        <div className="absolute -top-7 right-0 z-10">
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button variant="outline" size="xxs">
                  <PlusIcon className="size-2.5" />
                  Add panel ({hidden.length})
                </Button>
              }
            />
            <DropdownMenuContent align="end" className="min-w-[160px]">
              {hidden.map((id) => (
                <DropdownMenuItem
                  key={id}
                  title={MODULE_LABEL[id]}
                  onClick={() => show(id)}
                />
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      )}
      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        modifiers={[restrictToVerticalAxis]}
        onDragEnd={handleDragEnd}
      >
        <SortableContext items={visible} strategy={verticalListSortingStrategy}>
          <div className="flex min-h-0 flex-1 flex-col gap-2">
            {visible.map((id) => {
              const Renderer = RENDERERS[id];
              return <Renderer key={id} />;
            })}
          </div>
        </SortableContext>
      </DndContext>
    </aside>
  );
}
