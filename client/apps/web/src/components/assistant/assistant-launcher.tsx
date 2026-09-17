import { useT } from "@trenova/shared/i18n/use-t";
import { BorderBeam } from "@trenova/shared/components/ui/border-beam";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { Kbd, KbdGroup } from "@trenova/shared/components/ui/kbd";
import { m, useReducedMotion } from "motion/react";
import { useState } from "react";
import { ASSISTANT_SURFACE_ID } from "./assistant-surface";
import { TrenovaSpark } from "./trenova-spark";

type AssistantLauncherProps = {
  pendingCount: number;
  onClick: () => void;
};

/**
 * The corner mark. At rest it does nothing at all: it is on every page in the
 * product, so anything that moves would be movement a person cannot escape.
 *
 * It reacts on hover, and it carries a slow beam only while a change waits on
 * someone's decision, because that is the one thing worth pulling them back for.
 */
export function AssistantLauncher({ pendingCount, onClick }: AssistantLauncherProps) {
  const t = useT();
  const reduceMotion = useReducedMotion();
  const [hovered, setHovered] = useState(false);
  const hasPending = pendingCount > 0;

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
            className="bg-foreground text-background ring-foreground/10 focus-visible:ring-ring/50 fixed right-5 bottom-5 z-50 flex size-12 items-center justify-center shadow-lg shadow-black/15 ring-1 outline-none transition-shadow hover:shadow-xl focus-visible:ring-[3px]"
          />
        }
      >
        <TrenovaSpark className="size-5" animated={hovered && !reduceMotion} />
        {hasPending && (
          <>
            <BorderBeam
              className="assistant-beam"
              duration={9}
              borderWidth={2}
              colorFrom="var(--warning)"
              colorTo="color-mix(in oklch, var(--warning) 8%, transparent)"
            />
            <span className="bg-warning text-warning-foreground ring-background absolute -top-1 -right-1 flex h-5 min-w-5 items-center justify-center rounded-full px-1 text-xs font-semibold tabular-nums ring-2">
              {pendingCount > 99 ? "99+" : pendingCount}
            </span>
          </>
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
