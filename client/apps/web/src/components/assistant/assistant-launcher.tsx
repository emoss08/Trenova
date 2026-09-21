import { Kbd, KbdGroup } from "@trenova/shared/components/ui/kbd";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { m, useReducedMotion } from "motion/react";
import { useState } from "react";
import { AssistantMark } from "./assistant-mark";
import { ASSISTANT_SURFACE_ID } from "./assistant-surface";

type AssistantLauncherProps = {
  pendingCount: number;
  onClick: () => void;
};

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
 */
export function AssistantLauncher({ pendingCount, onClick }: AssistantLauncherProps) {
  const t = useT();
  const reduceMotion = useReducedMotion();
  const [hovered, setHovered] = useState(false);
  const hasPending = pendingCount > 0;
  const shown = pendingCount > 99 ? "99+" : String(pendingCount);

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <m.button
            type="button"
            onClick={onClick}
            onHoverStart={() => setHovered(true)}
            onHoverEnd={() => setHovered(false)}
            layoutId={ASSISTANT_SURFACE_ID}
            style={{ borderRadius: 14 }}
            whileHover={reduceMotion ? undefined : { y: -2 }}
            whileTap={reduceMotion ? undefined : { scale: 0.96 }}
            transition={
              reduceMotion ? { duration: 0 } : { type: "spring", stiffness: 420, damping: 30 }
            }
            aria-label={
              hasPending
                ? t("Open the assistant, {0} changes await your decision", pendingCount)
                : t("Open the assistant")
            }
            className={cn(
              "ui-focus-ring bg-foreground text-background ring-foreground/10",
              "fixed right-5 bottom-5 z-50 flex h-12 items-center justify-center ring-1 outline-none",
              hasPending ? "gap-2 pr-3.5 pl-3" : "w-12",
            )}
          />
        }
      >
        <AssistantMark className="size-5 shrink-0" animated={hovered && !reduceMotion} />
        {hasPending && (
          <span className="flex items-baseline gap-1 text-sm whitespace-nowrap">
            {/* The count is the only warm thing on the mark, and it is warm
                because it is the only part that is a claim on someone's
                time. */}
            <span className="text-warning font-semibold tabular-nums">{shown}</span>
            <span className="text-background/70">{t("waiting")}</span>
          </span>
        )}
      </TooltipTrigger>
      <TooltipContent side="left" sideOffset={8} className="flex items-center gap-2">
        {t("Assistant")}
        <KbdGroup>
          <Kbd>⌘</Kbd>
          <Kbd>J</Kbd>
        </KbdGroup>
      </TooltipContent>
    </Tooltip>
  );
}
