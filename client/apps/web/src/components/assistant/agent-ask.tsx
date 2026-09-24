import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { useAutoResizeTextarea } from "@trenova/shared/hooks/use-auto-resize-textarea";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { ArrowUpIcon } from "lucide-react";
import { useImperativeHandle, useMemo, useState, type FormEvent, type Ref } from "react";
import { AgentPicker } from "./agent-picker";
import { AgentStarters } from "./agent-starters";
import { agentSuggestions } from "./suggestions";

const VARIANTS = {
  hero: { minHeight: 56, maxHeight: 220, starters: 4 },
  compact: { minHeight: 40, maxHeight: 128, starters: 2 },
} as const;

export type AgentAskHandle = {
  /** Puts the caret in the box, for a surface that just chose the agent for the person. */
  focus: () => void;
};

export type AgentAskProps = {
  /** Who the question goes to. Owned by the surface, so another control can choose too. */
  agent: AgentChoice;
  onAgentChange: (agent: AgentChoice) => void;
  /** The person's agents, most recent first, which lead the picker. */
  recentIds: readonly string[];
  lastUsedAt?: ReadonlyMap<string, number>;
  disabled?: boolean;
  /** The Desk's front page asks in a large box; the corner panel in a small one. */
  variant?: keyof typeof VARIANTS;
  onAsk: (agentId: string, question: string) => void;
  ref?: Ref<AgentAskHandle>;
  className?: string;
};

/**
 * The box you talk into, who it goes to, and what that agent is good for.
 *
 * It is the same control on the Desk's front page and in the corner panel,
 * because they are the same act. Who is being asked is one control inside
 * the box — a picker that searches every agent the person can ask — rather
 * than a chip per agent, which read well at six agents and became a wall at
 * sixty. The questions under the box are the chosen agent's own, from the
 * server, and they trade places when the agent changes.
 */
export function AgentAsk({
  agent,
  onAgentChange,
  recentIds,
  lastUsedAt,
  disabled = false,
  variant = "hero",
  onAsk,
  ref,
  className,
}: AgentAskProps) {
  const t = useT();
  const sizing = VARIANTS[variant];
  const [question, setQuestion] = useState("");
  const { textareaRef, adjustHeight } = useAutoResizeTextarea({
    minHeight: sizing.minHeight,
    maxHeight: sizing.maxHeight,
  });

  useImperativeHandle(ref, () => ({ focus: () => textareaRef.current?.focus() }), [textareaRef]);

  const suggestions = useMemo(
    () => agentSuggestions(agent).slice(0, sizing.starters),
    [agent, sizing.starters],
  );

  const ask = (text: string) => {
    const trimmed = text.trim();
    if (trimmed === "" || disabled) {
      return;
    }
    onAsk(agent.id, trimmed);
    setQuestion("");
    adjustHeight(true);
  };

  const onSubmit = (event: FormEvent) => {
    event.preventDefault();
    ask(question);
  };

  const hero = variant === "hero";
  const placeholder = t("Ask {0} anything about your operation", agent.name);
  const ready = question.trim() !== "" && !disabled;

  return (
    <div className={cn("flex flex-col", hero ? "gap-3" : "gap-2.5", className)}>
      <form
        onSubmit={onSubmit}
        aria-busy={disabled || undefined}
        className={cn(
          "ui-field ui-container-focus-ring rounded-surface flex flex-col",
          hero ? "gap-1 p-2" : "gap-1 p-1.5",
          disabled && "opacity-70",
        )}
      >
        <textarea
          ref={textareaRef}
          value={question}
          onChange={(event) => {
            setQuestion(event.target.value);
            adjustHeight();
          }}
          onKeyDown={(event) => {
            if (event.key === "Enter" && !event.shiftKey && !event.nativeEvent.isComposing) {
              event.preventDefault();
              ask(question);
            }
          }}
          placeholder={placeholder}
          aria-label={placeholder}
          disabled={disabled}
          className={cn(
            "placeholder:text-muted-foreground w-full resize-none bg-transparent outline-none",
            hero ? "px-2 py-1.5 text-base" : "px-1.5 py-1 text-sm",
          )}
        />

        <div className="flex items-center gap-2">
          <AgentPicker
            agent={agent}
            onSelect={(picked) => {
              onAgentChange(picked);
              textareaRef.current?.focus();
            }}
            recentIds={recentIds}
            lastUsedAt={lastUsedAt}
            disabled={disabled}
            side={hero ? "bottom" : "top"}
            className={
              hero ? "bg-card hover:bg-surface-hover ring-foreground/10 ring-1" : undefined
            }
          />
          <span className="flex-1" />
          {/* The key that sends, said only once there is something to send:
              the hint answers the typing rather than sitting in an empty box. */}
          {hero && ready && (
            <span
              aria-hidden
              className="text-foreground-subtle animate-rise hidden items-center gap-1 text-xs sm:inline-flex"
            >
              <Kbd className="h-4 min-w-4 px-1">Enter</Kbd>
              {t("to send")}
            </span>
          )}
          <Button
            type="submit"
            size="icon-sm"
            disabled={!ready}
            aria-label={t("Ask")}
            className="shrink-0 rounded-full"
          >
            {/* Re-keyed as the question becomes sendable, so the arrow lands
                with the confirm spring at the moment the box can be sent. */}
            <ArrowUpIcon
              key={ready ? "ready" : "empty"}
              className={cn("size-4", ready && "animate-confirm")}
            />
          </Button>
        </div>
      </form>

      {suggestions.length > 0 && (
        <AgentStarters
          agentKey={agent.id}
          suggestions={suggestions}
          disabled={disabled}
          size={hero ? "comfortable" : "compact"}
          onPick={(suggestion) => ask(suggestion.prompt)}
        />
      )}
    </div>
  );
}
