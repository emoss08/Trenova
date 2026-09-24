import { dockPositionClass, isLeftDock, type AssistantDock } from "@/lib/assistant-dock";
import { Kbd, KbdGroup } from "@trenova/shared/components/ui/kbd";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { m, useReducedMotion } from "motion/react";
import { launcherLabel } from "./assistant-launcher";
import { AssistantMark } from "./assistant-mark";
import { ASSISTANT_SURFACE_ID } from "./assistant-surface";

type AssistantEdgeTabProps = {
  dock: AssistantDock;
  pendingCount: number;
  writingCount?: number;
  onClick: () => void;
};

/**
 * What is left of the launcher once someone has hidden it: a narrow tab on
 * the edge of the screen, in the same corner, that covers nothing a page
 * puts there.
 *
 * It still carries the one signal the launcher exists for. A decision waiting
 * on the person turns the tab warm, because hiding the button is a choice
 * about space, not about being told nothing.
 */
export function AssistantEdgeTab({
  dock,
  pendingCount,
  writingCount = 0,
  onClick,
}: AssistantEdgeTabProps) {
  const t = useT();
  const reduceMotion = useReducedMotion();
  const left = isLeftDock(dock);
  const hasPending = pendingCount > 0;

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <m.button
            type="button"
            onClick={onClick}
            layoutId={ASSISTANT_SURFACE_ID}
            style={{
              borderTopLeftRadius: left ? 0 : 8,
              borderBottomLeftRadius: left ? 0 : 8,
              borderTopRightRadius: left ? 8 : 0,
              borderBottomRightRadius: left ? 8 : 0,
            }}
            transition={
              reduceMotion ? { duration: 0 } : { type: "spring", stiffness: 420, damping: 34 }
            }
            aria-label={launcherLabel(t, pendingCount, writingCount)}
            data-pending={hasPending || undefined}
            className={cn(
              "group ui-focus-ring fixed z-50 flex h-10 w-2 items-center justify-center overflow-hidden outline-none",
              "transition-[width] duration-150 hover:w-7 focus-visible:w-7",
              dockPositionClass(dock, "tab"),
              hasPending ? "bg-warning text-warning-on-solid" : "bg-foreground/70 text-background",
            )}
          />
        }
      >
        <AssistantMark className="size-4 shrink-0 opacity-0 transition-opacity group-hover:opacity-100 group-focus-visible:opacity-100" />
      </TooltipTrigger>
      <TooltipContent
        side={left ? "right" : "left"}
        sideOffset={8}
        className="flex items-center gap-2"
      >
        {hasPending ? t("{0} waiting", pendingCount) : t("Assistant")}
        <KbdGroup>
          <Kbd>⌘</Kbd>
          <Kbd>J</Kbd>
        </KbdGroup>
      </TooltipContent>
    </Tooltip>
  );
}
