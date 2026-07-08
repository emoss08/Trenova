import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { GripVerticalIcon, MinusIcon } from "lucide-react";
import type { ReactNode } from "react";
import { useRightStackStore, type RightStackModuleId } from "./right-stack-store";

type Props = {
  id: RightStackModuleId;
  title: string;
  count?: number;
  countTone?: "muted" | "danger" | "warning" | "brand" | "success";
  rightSlot?: ReactNode;
  children: ReactNode;
};

const COUNT_CLASS: Record<NonNullable<Props["countTone"]>, string> = {
  muted: "bg-muted text-muted-foreground",
  danger: "bg-destructive/12 text-destructive",
  warning: "bg-warning/15 text-warning",
  brand: "bg-brand/15 text-brand",
  success: "bg-success/15 text-success",
};

export function ModuleCard({ id, title, count, countTone = "muted", rightSlot, children }: Props) {
  const hide = useRightStackStore.use.hide();
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id,
  });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
    zIndex: isDragging ? 20 : "auto",
  } as React.CSSProperties;

  return (
    <section ref={setNodeRef} style={style} className="cc-module-card flex min-h-0 flex-1 flex-col">
      <header className="flex items-center justify-between gap-2 border-b border-border px-2 py-1">
        <div className="flex min-w-0 items-center gap-1">
          <button
            type="button"
            aria-label={`Drag ${title}`}
            className="flex size-4 shrink-0 cursor-grab items-center justify-center rounded text-muted-foreground hover:bg-muted hover:text-foreground active:cursor-grabbing"
            {...attributes}
            {...listeners}
          >
            <GripVerticalIcon className="size-3" />
          </button>
          <h3 className="cc-label truncate text-foreground">{title}</h3>
          {typeof count === "number" && (
            <span
              className={cn(
                "inline-flex min-w-4.5 justify-center rounded px-1 font-table text-[9px] tabular-nums",
                COUNT_CLASS[countTone],
              )}
            >
              {count}
            </span>
          )}
        </div>

        <div className="flex items-center gap-1">
          {rightSlot}
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  variant="ghost"
                  size="icon-xxs"
                  aria-label={`Hide ${title}`}
                  onClick={() => hide(id)}
                >
                  <MinusIcon className="size-2.5" />
                </Button>
              }
            />
            <TooltipContent side="left">Hide Panel</TooltipContent>
          </Tooltip>
        </div>
      </header>
      <ScrollArea className="min-h-0 flex-1" maskVariant="card">
        <div className="flex flex-col px-3 py-1">{children}</div>
      </ScrollArea>
    </section>
  );
}
