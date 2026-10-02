import { ArtifactKindIcon } from "@/components/assistant/voice/artifact-chrome";
import { EASE_SETTLE } from "@/lib/motion";
import type { AssistantArtifact } from "@/types/assistant";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { PinIcon } from "lucide-react";
import { m, useReducedMotion } from "motion/react";
import { useEffect, useRef } from "react";

/**
 * One row of marks under the header, one per artifact in the order the
 * switcher lists them, for the eye: it says how much the conversation has
 * made and where the open one sits in it, and a click jumps there. It never
 * grows taller than one row — a long conversation scrolls it sideways, and
 * the open mark is kept in view — and with one artifact there is nothing
 * to jump to, so it is not drawn.
 */
export function ArtifactFilmstrip({
  artifacts,
  activeId,
  threadId,
  onOpen,
}: {
  artifacts: readonly AssistantArtifact[];
  activeId: string;
  threadId: string;
  onOpen: (id: string) => void;
}) {
  const t = useT();
  const reduceMotion = useReducedMotion();
  const chips = useRef(new Map<string, HTMLButtonElement>());

  useEffect(() => {
    chips.current.get(activeId)?.scrollIntoView?.({
      inline: "nearest",
      block: "nearest",
      behavior: reduceMotion ? "auto" : "smooth",
    });
  }, [activeId, reduceMotion]);

  if (artifacts.length < 2) {
    return null;
  }

  return (
    <div
      role="group"
      aria-label={t("Jump to an artifact")}
      data-slot="artifact-filmstrip"
      className={cn(
        "border-border-subtle bg-card flex h-10 shrink-0 items-center gap-1 overflow-x-auto border-b px-2",
        "[scrollbar-width:none] [&::-webkit-scrollbar]:hidden",
      )}
    >
      {artifacts.map((artifact, index) => {
        const active = artifact.id === activeId;

        return (
          <Tooltip key={artifact.id}>
            <TooltipTrigger
              render={
                <button
                  type="button"
                  ref={(element) => {
                    if (element) {
                      chips.current.set(artifact.id, element);
                    } else {
                      chips.current.delete(artifact.id);
                    }
                  }}
                  aria-label={t("Open {0}", artifact.title)}
                  aria-current={active ? "true" : undefined}
                  onClick={() => onOpen(artifact.id)}
                  // Dealt one after another when the row first appears, so a
                  // conversation's output reads as a hand rather than a block.
                  style={{ animationDelay: `${Math.min(index, 10) * 30}ms` }}
                  className={cn(
                    "ui-focus-ring animate-land relative flex size-8 shrink-0 items-center justify-center rounded-md transition-colors",
                    active
                      ? "text-foreground"
                      : "text-foreground-subtle hover:text-foreground hover:bg-surface-hover",
                  )}
                />
              }
            >
              {/* One indicator that slides to the mark picked, so the eye
                  follows the choice instead of hunting for it. */}
              {active && (
                <m.span
                  layoutId={`artifact-chip-${threadId}`}
                  aria-hidden
                  transition={
                    reduceMotion ? { duration: 0 } : { duration: 0.24, ease: EASE_SETTLE }
                  }
                  className="bg-sunken ring-foreground/10 absolute inset-0 rounded-md ring-1"
                />
              )}
              <ArtifactKindIcon kind={artifact.kind} className="relative size-3.5" />
              {artifact.pinned && (
                <PinIcon
                  aria-hidden
                  className="bg-card text-foreground-muted absolute -top-0.5 -right-0.5 size-3 rounded-full p-px"
                />
              )}
            </TooltipTrigger>
            <TooltipContent>{artifact.title}</TooltipContent>
          </Tooltip>
        );
      })}
    </div>
  );
}
