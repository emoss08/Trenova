import { Kbd, KbdGroup } from "@trenova/shared/components/ui/kbd";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { m, useReducedMotion } from "motion/react";
import { useState } from "react";
import { AssistantMark } from "./assistant-mark";
import { ASSISTANT_SURFACE_ID } from "./assistant-surface";

type AssistantLauncherProps = {
  pendingCount: number;
  /** Replies still being written with the panel closed, across every conversation. */
  writingCount?: number;
  onClick: () => void;
};

/** The launcher's accessible name: everything the pill says, uncapped. */
function launcherLabel(t: TranslateFn, pendingCount: number, writingCount: number): string {
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
 */
export function AssistantLauncher({
  pendingCount,
  writingCount = 0,
  onClick,
}: AssistantLauncherProps) {
  const t = useT();
  const reduceMotion = useReducedMotion();
  const [hovered, setHovered] = useState(false);
  const hasPending = pendingCount > 0;
  const isWriting = writingCount > 0;
  const expanded = hasPending || isWriting;

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
            aria-label={launcherLabel(t, pendingCount, writingCount)}
            data-writing={isWriting || undefined}
            className={cn(
              "ui-focus-ring bg-foreground text-background ring-foreground/10",
              "fixed right-5 bottom-5 z-50 flex h-12 items-center justify-center ring-1 outline-none",
              expanded ? "gap-2 pr-3.5 pl-3" : "w-12",
            )}
          />
        }
      >
        <AssistantMark className="size-5 shrink-0" animated={hovered && !reduceMotion} />
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
