import { formatWorkDuration } from "@/lib/ai-usage-format";
import { ChevronRightIcon } from "@trenova/shared/components/icons";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useRef, useState, type CSSProperties } from "react";

/**
 * The sheen that crosses "Thinking…" while a thought arrives: the muted ink
 * with the full ink passing through it, so the words stay readable in both
 * themes. It moves with the shimmer loop and stops when the thought does.
 */
const THINKING_SHEEN: CSSProperties = {
  backgroundImage:
    "linear-gradient(100deg, var(--foreground-muted) 35%, var(--foreground) 50%, var(--foreground-muted) 65%)",
};

/**
 * What the model thought, shown apart from what it said.
 *
 * Open while the thinking is still arriving, because that is the minute a
 * heavy model would otherwise spend looking hung; the line says "Thinking…"
 * with a sheen moving across the words, the one moment the product shimmers,
 * and it stops when the thought does. Folded away once the answer begins,
 * into how long it took: the reasoning is there for whoever wants to check
 * the answer against it, not in the way of reading the answer. The fold is
 * animated so the answer rising into its place reads as the thought giving
 * way to it.
 */
export function ReasoningDisclosure({
  text,
  streaming = false,
  seconds = null,
}: {
  text: string;
  streaming?: boolean;
  /** How long the thinking took, when it was timed; a live thought times itself. */
  seconds?: number | null;
}) {
  const t = useT();
  const [open, setOpen] = useState(streaming);
  const wasStreaming = useRef(streaming);
  // A live thought is timed from the first word to the last, so the fold can
  // say how long it was; a saved thought keeps whatever it was given.
  const startedAt = useRef<number | null>(null);
  const [timed, setTimed] = useState<number | null>(null);

  useEffect(() => {
    if (streaming && startedAt.current === null) {
      startedAt.current = Date.now();
    }
    if (wasStreaming.current && !streaming) {
      setOpen(false);
      if (startedAt.current !== null) {
        setTimed(Math.round((Date.now() - startedAt.current) / 1000));
      }
    }
    wasStreaming.current = streaming;
  }, [streaming]);

  if (text === "" && !streaming) {
    return null;
  }

  const took = seconds ?? timed;
  const label = streaming
    ? t("Thinking…")
    : took !== null
      ? t("Thought for {0}", formatWorkDuration(took))
      : t("Thought it through");

  return (
    <Collapsible open={open} onOpenChange={setOpen} className="min-w-0">
      <CollapsibleTrigger className="group/reasoning text-foreground-muted hover:text-foreground ui-focus-ring -mx-1.5 flex items-center gap-1.5 rounded-control px-1.5 py-0.5 text-xs transition-colors">
        <ChevronRightIcon
          className={cn("size-3 transition-transform duration-200", open && "rotate-90")}
          aria-hidden
        />
        <span
          key={streaming ? "thinking" : "thought"}
          className={cn(
            streaming
              ? "animate-shimmer bg-[length:200%_100%] bg-clip-text text-transparent"
              : "animate-rise",
          )}
          style={streaming ? THINKING_SHEEN : undefined}
        >
          {label}
        </span>
      </CollapsibleTrigger>
      <CollapsibleContent className="h-(--collapsible-panel-height) overflow-hidden transition-[height] duration-200 ease-settle data-ending-style:h-0 data-starting-style:h-0">
        <div className="text-foreground-muted border-border-subtle mt-1 ml-1.25 border-l pt-0.5 pb-1 pl-3 text-xs leading-relaxed whitespace-pre-wrap">
          {text}
          {streaming && (
            <span
              aria-hidden
              className="assistant-caret bg-foreground-subtle ml-0.5 inline-block h-[1em] w-[2px] rounded-full align-text-bottom"
            />
          )}
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}
