import {
  ASSISTANT_DOCKS,
  dockPositionClass,
  isLeftDock,
  isTopDock,
  type AssistantDock,
} from "@/lib/assistant-dock";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { m, useReducedMotion } from "motion/react";
import { createPortal } from "react-dom";
import { dockLabel } from "./dock-label";

/** The quarter of the screen that lands in each corner. */
const QUADRANT_CLASSES: Record<AssistantDock, string> = {
  "bottom-right": "right-0 bottom-0",
  "bottom-left": "bottom-0 left-0",
  "top-right": "top-0 right-0",
  "top-left": "top-0 left-0",
};

type AssistantDockTargetsProps = {
  /** The corner the launcher will land in if it is let go now. */
  active: AssistantDock;
};

/**
 * Shown only while the launcher is being dragged: the four places it can go,
 * and which one it will go to.
 *
 * Without them a drag reads as free placement, and a launcher let go in the
 * middle of the left edge jumping to the bottom left looks like a bug. The
 * whole quarter of the screen that leads to the highlighted corner is tinted,
 * so it is plain that anywhere in it will do.
 */
export function AssistantDockTargets({ active }: AssistantDockTargetsProps) {
  const t = useT();
  const reduceMotion = useReducedMotion();

  return createPortal(
    <m.div
      aria-hidden
      data-testid="assistant-dock-targets"
      initial={reduceMotion ? false : { opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.12 }}
      className="pointer-events-none fixed inset-0 z-40"
    >
      {ASSISTANT_DOCKS.map((dock) => {
        const isActive = dock === active;

        return (
          <div key={dock} data-dock={dock} data-active={isActive || undefined}>
            <div
              className={cn(
                "absolute h-1/2 w-1/2 transition-colors duration-100",
                QUADRANT_CLASSES[dock],
                isActive ? "bg-brand/5" : "bg-transparent",
              )}
            />
            <div
              className={cn(
                "fixed size-10 rounded-xl border-2 border-dashed transition-colors duration-100",
                dockPositionClass(dock, "launcher"),
                isActive ? "border-brand bg-brand/10" : "border-foreground/25 bg-background/60",
              )}
            >
              {isActive && (
                <span
                  className={cn(
                    "bg-foreground text-background absolute rounded-md px-2 py-1 text-xs whitespace-nowrap",
                    isTopDock(dock) ? "top-full mt-2" : "bottom-full mb-2",
                    isLeftDock(dock) ? "left-0" : "right-0",
                  )}
                >
                  {dockLabel(t, dock)}
                </span>
              )}
            </div>
          </div>
        );
      })}
    </m.div>,
    document.body,
  );
}
