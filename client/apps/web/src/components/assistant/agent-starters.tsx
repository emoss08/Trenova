import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { ArrowUpRightIcon } from "lucide-react";
import { AnimatePresence, m, useReducedMotion } from "motion/react";
import type { Suggestion } from "./suggestions";

export type AgentStartersProps = {
  /** Changes when the agent does, so the questions trade places rather than blink. */
  agentKey: string;
  suggestions: readonly Suggestion[];
  disabled?: boolean;
  size?: "comfortable" | "compact";
  onPick: (suggestion: Suggestion) => void;
  className?: string;
};

/**
 * The questions an agent is good for, under the box you ask it in.
 *
 * They are the agent's own — its template's, or drawn from the tools it
 * holds — so the first click teaches what this agent is for. When the agent
 * changes the old questions step out and the new ones step in one after
 * another, which is most of what makes a change of agent legible.
 */
export function AgentStarters({
  agentKey,
  suggestions,
  disabled = false,
  size = "comfortable",
  onPick,
  className,
}: AgentStartersProps) {
  const t = useT();
  const reduceMotion = useReducedMotion();

  return (
    <ul aria-label={t("Starter questions")} className={cn("flex flex-wrap gap-1.5", className)}>
      <AnimatePresence mode="popLayout" initial={false}>
        {suggestions.map((suggestion, index) => (
          <m.li
            key={`${agentKey}:${suggestion.prompt}`}
            layout={!reduceMotion}
            initial={reduceMotion ? false : { opacity: 0, y: 4 }}
            animate={{ opacity: 1, y: 0 }}
            exit={reduceMotion ? { opacity: 0 } : { opacity: 0, y: -2 }}
            transition={{
              duration: 0.18,
              ease: [0.16, 1, 0.3, 1],
              delay: reduceMotion ? 0 : index * 0.035,
            }}
          >
            <button
              type="button"
              disabled={disabled}
              title={suggestion.prompt}
              onClick={() => onPick(suggestion)}
              className={cn(
                "group ui-focus-ring ui-press text-muted-foreground inline-flex items-center gap-1 rounded-full",
                "ring-foreground/10 hover:bg-surface-hover hover:text-foreground ring-1 transition-colors",
                "disabled:pointer-events-none disabled:opacity-50",
                size === "compact" ? "h-6 px-2.5 text-xs" : "h-7 px-3 text-sm",
              )}
            >
              <span className="max-w-72 truncate">{t(suggestion.label)}</span>
              <ArrowUpRightIcon
                aria-hidden
                className={cn(
                  "-mr-0.5 size-3 shrink-0 opacity-0 transition-[opacity,translate]",
                  "-translate-x-0.5 translate-y-0.5 group-hover:translate-x-0 group-hover:translate-y-0",
                  "group-hover:opacity-100 group-focus-visible:translate-x-0",
                  "group-focus-visible:translate-y-0 group-focus-visible:opacity-100",
                )}
              />
            </button>
          </m.li>
        ))}
      </AnimatePresence>
    </ul>
  );
}
