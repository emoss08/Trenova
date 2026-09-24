import {
  dockPositionClass,
  isLeftDock,
  nearestDock,
  type AssistantDock,
} from "@/lib/assistant-dock";
import { Kbd, KbdGroup } from "@trenova/shared/components/ui/kbd";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { EyeOffIcon } from "lucide-react";
import { m, useMotionValue, useReducedMotion, type PanInfo } from "motion/react";
import { useRef, useState } from "react";
import { AssistantMark } from "./assistant-mark";
import { ASSISTANT_SURFACE_ID } from "./assistant-surface";

type AssistantLauncherProps = {
  pendingCount: number;
  /** Replies still being written with the panel closed, across every conversation. */
  writingCount?: number;
  dock?: AssistantDock;
  onClick: () => void;
  /** Moves the launcher, and the panel it opens, to another corner. */
  onMove?: (dock: AssistantDock) => void;
  /** Tucks the launcher into a tab at the edge of the screen. */
  onHide?: () => void;
};

/** The launcher's accessible name: everything the pill says, uncapped. */
export function launcherLabel(t: TranslateFn, pendingCount: number, writingCount: number): string {
  if (pendingCount > 0 && writingCount > 0) {
    return t(
      "Open the assistant, {0} changes await your decision, {1, plural, one {# reply is} other {# replies are}} being written",
      pendingCount,
      writingCount,
    );
  }
  if (pendingCount > 0) {
    return t("Open the assistant, {0} changes await your decision", pendingCount);
  }
  if (writingCount > 0) {
    return t(
      "Open the assistant, {0, plural, one {# reply is} other {# replies are}} being written",
      writingCount,
    );
  }

  return t("Open the assistant");
}

function capped(count: number): string {
  return count > 99 ? "99+" : String(count);
}

/**
 * The corner mark. At rest it does nothing at all: it is on every page in the
 * product, so anything that moves would be movement a person cannot escape.
 *
 * When decisions are waiting it grows into a pill and says how many.
 *
 * It used to say the same thing with a beam travelling its border and a
 * numbered dot in the corner, and that was two devices for one fact, one of
 * them a loop running on every screen in the product for as long as anything
 * was pending. A shape that changes is a stronger signal than a shape that
 * moves, and it can carry a word: "3 waiting" is read at a glance, where a
 * beam has to be interpreted and a bare 3 could be anything.
 *
 * A reply still being written with the panel closed says so the same way,
 * and just as still: "Writing", or "2 writing" for several. It is not a claim
 * on anyone's time, so it stays in the launcher's own ink, and when decisions
 * are waiting too they lead, because only they need the person.
 *
 * It sits over whatever page is open, so it can be moved out of the way:
 * dragged to any corner, or tucked into a tab at the edge of the screen.
 */
export function AssistantLauncher({
  pendingCount,
  writingCount = 0,
  dock = "bottom-right",
  onClick,
  onMove,
  onHide,
}: AssistantLauncherProps) {
  const t = useT();
  const reduceMotion = useReducedMotion();
  const [hovered, setHovered] = useState(false);
  const dragged = useRef(false);
  const x = useMotionValue(0);
  const y = useMotionValue(0);
  const hasPending = pendingCount > 0;
  const isWriting = writingCount > 0;
  const expanded = hasPending || isWriting;
  const movable = onMove !== undefined;

  const handleDragEnd = (_event: MouseEvent | TouchEvent | PointerEvent, info: PanInfo) => {
    const next = nearestDock(
      { x: info.point.x - window.scrollX, y: info.point.y - window.scrollY },
      { width: window.innerWidth, height: window.innerHeight },
    );
    x.set(0);
    y.set(0);
    if (next !== dock) {
      onMove?.(next);
    }
  };

  return (
    <m.div
      layout
      drag={movable}
      dragMomentum={false}
      style={{ x, y }}
      onPointerDown={() => {
        dragged.current = false;
      }}
      onDragStart={() => {
        dragged.current = true;
      }}
      onDragEnd={handleDragEnd}
      transition={reduceMotion ? { duration: 0 } : { type: "spring", stiffness: 420, damping: 34 }}
      className={cn(
        "group fixed z-50",
        dockPositionClass(dock, "launcher"),
        movable && "cursor-grab active:cursor-grabbing",
      )}
    >
      <Tooltip>
        <TooltipTrigger
          render={
            <m.button
              type="button"
              onClick={() => {
                // A drag ends with the pointer let go over the launcher, which
                // the browser reports as a click. Moving it is not asking for it.
                if (dragged.current) {
                  dragged.current = false;
                  return;
                }
                onClick();
              }}
              onHoverStart={() => setHovered(true)}
              onHoverEnd={() => setHovered(false)}
              layoutId={ASSISTANT_SURFACE_ID}
              style={{ borderRadius: 12 }}
              whileHover={reduceMotion ? undefined : { y: -2 }}
              whileTap={reduceMotion ? undefined : { scale: 0.96 }}
              transition={
                reduceMotion ? { duration: 0 } : { type: "spring", stiffness: 420, damping: 30 }
              }
              aria-label={launcherLabel(t, pendingCount, writingCount)}
              data-writing={isWriting || undefined}
              className={cn(
                "ui-focus-ring bg-foreground text-background ring-foreground/10",
                "flex h-10 items-center justify-center ring-1 outline-none",
                movable && "cursor-[inherit]",
                expanded ? "gap-2 pr-3 pl-2.5" : "w-10",
              )}
            />
          }
        >
          <AssistantMark className="size-4.5 shrink-0" animated={hovered && !reduceMotion} />
          {expanded && (
            <span className="flex items-baseline gap-1 text-sm whitespace-nowrap">
              {hasPending && (
                <>
                  {/* The count is the only warm thing on the mark, and it is
                    warm because it is the only part that is a claim on
                    someone's time. */}
                  <span className="text-warning font-semibold tabular-nums">
                    {capped(pendingCount)}
                  </span>
                  <span className="text-background/70">{t("waiting")}</span>
                </>
              )}
              {hasPending && isWriting && (
                <span aria-hidden className="text-background/40">
                  ·
                </span>
              )}
              {isWriting &&
                (writingCount > 1 ? (
                  <>
                    <span className="text-background font-medium tabular-nums">
                      {capped(writingCount)}
                    </span>
                    <span className="text-background/70">{t("writing")}</span>
                  </>
                ) : (
                  <span className="text-background/70">
                    {hasPending ? t("writing") : t("Writing")}
                  </span>
                ))}
            </span>
          )}
        </TooltipTrigger>
        <TooltipContent
          side={isLeftDock(dock) ? "right" : "left"}
          sideOffset={8}
          className="flex flex-col items-start gap-1"
        >
          <span className="flex items-center gap-2">
            {t("Assistant")}
            <KbdGroup>
              <Kbd>⌘</Kbd>
              <Kbd>J</Kbd>
            </KbdGroup>
          </span>
          {movable && <span className="text-muted-foreground">{t("Drag to move it")}</span>}
        </TooltipContent>
      </Tooltip>
      {onHide && (
        <Tooltip>
          <TooltipTrigger
            render={
              <button
                type="button"
                aria-label={t("Hide the assistant button")}
                onPointerDown={(event) => event.stopPropagation()}
                onClick={onHide}
                className={cn(
                  "ui-focus-ring bg-popover text-muted-foreground hover:text-foreground ring-foreground/10",
                  "absolute -top-2 flex size-5 items-center justify-center rounded-full ring-1 outline-none",
                  "opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100",
                  isLeftDock(dock) ? "-right-2" : "-left-2",
                )}
              />
            }
          >
            <EyeOffIcon className="size-3" />
          </TooltipTrigger>
          <TooltipContent side="top">
            {t("Hide the button. Open the assistant from the edge tab or with ⌘J.")}
          </TooltipContent>
        </Tooltip>
      )}
    </m.div>
  );
}
