import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { BorderBeam } from "@trenova/shared/components/ui/border-beam";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { useAutoResizeTextarea } from "@trenova/shared/hooks/use-auto-resize-textarea";
import { useTypewriter } from "@trenova/shared/hooks/use-typewriter";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { ArrowUpIcon } from "lucide-react";
import { m, useReducedMotion } from "motion/react";
import { useId, useImperativeHandle, useMemo, useState, type FormEvent, type Ref } from "react";
import { AgentPicker } from "./agent-picker";
import { agentSuggestions } from "./suggestions";

/**
 * The light that runs round the hero's ask box: violet at its head, through
 * indigo and the brand cobalt, out to rose at its tail, so the one moving
 * thing on the page carries a little spectrum rather than one flat hue.
 */
const HERO_BEAM = [
  "var(--accent-violet)",
  "var(--accent-indigo)",
  "var(--brand)",
  "color-mix(in oklch, var(--accent-rose) 70%, transparent)",
  "color-mix(in oklch, var(--accent-rose) 10%, transparent)",
] as const;

const VARIANTS = {
  hero: { minHeight: 56, maxHeight: 220, starters: 4 },
  compact: { minHeight: 40, maxHeight: 128, starters: 3 },
} as const;

/** The key that takes the question being written in the empty box as the person's own. */
export const STARTER_ACCEPT_KEY = "Tab";

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
 * sixty. The questions the agent is good for are written into the empty box
 * itself, one at a time, as if the agent were suggesting them: each is typed
 * out, held, taken back, and the next written, and Tab takes the one on
 * screen as the person's own so it can be sent or changed. They are the
 * agent's own questions, so a change of agent is a change of handwriting.
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
  const prompts = useMemo(() => suggestions.map((item) => item.prompt), [suggestions]);

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
  const reduceMotion = useReducedMotion();
  const placeholder = t("Ask {0}…", agent.name);
  const ready = question.trim() !== "" && !disabled;
  const empty = question === "";
  const typed = useTypewriter(prompts, {
    instant: reduceMotion === true,
    paused: !empty || disabled,
  });
  const offered = empty && typed.text !== "";
  const acceptable = offered && typed.complete;
  const suggestionId = useId();

  const acceptStarter = () => {
    const prompt = prompts[typed.index];
    if (!acceptable || !prompt) {
      return false;
    }
    setQuestion(prompt);
    requestAnimationFrame(() => {
      adjustHeight();
      const field = textareaRef.current;
      if (field) {
        field.focus();
        field.setSelectionRange(prompt.length, prompt.length);
      }
    });
    return true;
  };

  return (
    <div className={cn("flex flex-col", hero ? "gap-3" : "gap-2.5", className)}>
      <m.form
        onSubmit={onSubmit}
        aria-busy={disabled || undefined}
        initial={reduceMotion || !hero ? false : { opacity: 0, y: 12, scale: 0.985 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        transition={{ type: "spring", stiffness: 260, damping: 26, mass: 0.9, delay: 0.18 }}
        className={cn(
          "ui-field ui-lift-whisper ui-container-focus-ring rounded-surface relative flex flex-col",
          hero ? "gap-1 p-2" : "gap-1 p-1.5",
          disabled && "opacity-70",
        )}
      >
        {hero && !reduceMotion && (
          <BorderBeam duration={8} borderWidth={1.5} colors={HERO_BEAM} glow />
        )}
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
              return;
            }
            if (
              event.key === STARTER_ACCEPT_KEY &&
              !event.shiftKey &&
              !event.altKey &&
              !event.ctrlKey &&
              !event.metaKey &&
              acceptStarter()
            ) {
              event.preventDefault();
            }
          }}
          placeholder={offered ? undefined : placeholder}
          aria-label={placeholder}
          aria-describedby={acceptable ? suggestionId : undefined}
          disabled={disabled}
          className={cn(
            "placeholder:text-muted-foreground relative z-10 w-full resize-none bg-transparent outline-none",
            hero ? "px-2 py-1.5 text-base" : "px-1.5 py-1 text-sm",
          )}
        />
        {offered && (
          <StarterGhost text={typed.text} writing={typed.phase !== "holding"} hero={hero} />
        )}
        {acceptable && (
          <span id={suggestionId} className="sr-only">
            {t("Press Tab to ask: {0}", typed.text)}
          </span>
        )}

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
          {hero && acceptable && (
            <span
              aria-hidden
              className="text-foreground-subtle animate-rise hidden items-center gap-1 text-xs sm:inline-flex"
            >
              <Kbd className="h-4 min-w-4 px-1">Tab</Kbd>
              {t("to ask this")}
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
      </m.form>
    </div>
  );
}

/**
 * The question being written into the empty box. It sits exactly where the
 * person's own words will, in the placeholder's colour, with a caret that
 * blinks while the line is held and stands still while it is being written,
 * as a hand does. The textarea stays on top of it, so a click lands in the
 * field and typing a first letter puts the ghost away.
 */
function StarterGhost({ text, writing, hero }: { text: string; writing: boolean; hero: boolean }) {
  return (
    <div
      aria-hidden
      data-slot="starter-ghost"
      className={cn(
        "text-muted-foreground pointer-events-none absolute inset-x-0 top-0 truncate whitespace-pre",
        hero ? "px-4 pt-3.5 text-base" : "px-3 pt-2.5 text-sm",
      )}
    >
      {text}
      <span
        className={cn(
          "ml-px inline-block h-[1.05em] w-[1.5px] translate-y-[0.15em] rounded-full bg-current",
          !writing && "animate-caret",
        )}
      />
    </div>
  );
}
