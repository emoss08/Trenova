import { useT } from "@trenova/shared/i18n/use-t";
import { AgentTile } from "@/components/agent-identity/agent-tile";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { cn } from "@trenova/shared/lib/utils";
import type { AssistantPageContext } from "@/types/assistant";
import { ArrowUpIcon, MapPinIcon, SquareIcon, XIcon } from "lucide-react";
import { AnimatePresence, m, useReducedMotion } from "motion/react";
import { useCallback, useRef, useState } from "react";
import type { Suggestion } from "./suggestions";

export type ComposerProps = {
  onSend: (content: string) => void;
  onStop: () => void;
  active: boolean;
  disabled?: boolean;
  disabledReason?: string;
  placeholder: string;
  /** The agent the message goes to, shown so it is never a guess. */
  agent?: AgentDefinitionRow | null;
  onPickAgent?: () => void;
  /** What the person is looking at, offered as context they can drop. */
  pageContext?: AssistantPageContext | null;
  contextIncluded?: boolean;
  onToggleContext?: () => void;
  /** Opening questions, shown in the toolbar until one is used or dismissed. */
  suggestions?: Suggestion[];
  onDismissSuggestion?: (prompt: string) => void;
  compact?: boolean;
};

/**
 * Where a person types.
 *
 * Enter sends and Shift+Enter breaks a line; Cmd+Enter sends too, for people who
 * expect it. The toolbar under the text says who is being asked and what they
 * can see, because both are things a person would otherwise have to assume. One
 * control on the right does send and stop, so the action that makes sense is
 * always the one on screen.
 */
export function Composer({
  onSend,
  onStop,
  active,
  disabled = false,
  disabledReason,
  placeholder,
  agent,
  onPickAgent,
  pageContext,
  contextIncluded = true,
  onToggleContext,
  suggestions = [],
  onDismissSuggestion,
  compact = false,
}: ComposerProps) {
  const t = useT();
  const reduceMotion = useReducedMotion();
  const [draft, setDraft] = useState("");
  const [sent, setSent] = useState(0);
  const [focused, setFocused] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const canSend = draft.trim() !== "" && !active && !disabled;
  const showSuggestions = suggestions.length > 0 && draft === "" && !active;

  const submit = useCallback(() => {
    const content = draft.trim();
    if (content === "" || active || disabled) {
      return;
    }
    onSend(content);
    setDraft("");
    setSent((count) => count + 1);
  }, [active, disabled, draft, onSend]);

  return (
    <div
      className={cn(
        "border-border/70 bg-background border-t",
        compact ? "px-3 py-2.5" : "px-4 py-3",
      )}
    >
      <div className={cn("mx-auto flex flex-col gap-1.5", !compact && "max-w-3xl")}>
        <div className="relative">
          {/* The draft lifting out of the box as it is sent: the one moment the
              composer moves, so sending reads as the message leaving. */}
          <AnimatePresence>
            {sent > 0 && !reduceMotion && (
              <m.span
                key={sent}
                aria-hidden
                initial={{ opacity: 0.5, y: 0, scaleY: 1 }}
                animate={{ opacity: 0, y: -18, scaleY: 0.9 }}
                exit={{ opacity: 0 }}
                transition={{ duration: 0.28, ease: [0.2, 0.8, 0.2, 1] }}
                className="bg-secondary pointer-events-none absolute inset-x-0 top-0 h-8 rounded-2xl"
              />
            )}
          </AnimatePresence>

          <div className="assistant-focus-ring border-input bg-card focus-within:border-brand relative flex flex-col rounded-2xl border transition-colors duration-150">
            <Textarea
              ref={textareaRef}
              value={draft}
              onChange={(event) => setDraft(event.target.value)}
              onFocus={() => setFocused(true)}
              onBlur={() => setFocused(false)}
              onKeyDown={(event) => {
                if (event.key === "Enter" && (event.metaKey || event.ctrlKey || !event.shiftKey)) {
                  event.preventDefault();
                  submit();
                }
              }}
              placeholder={disabled ? (disabledReason ?? placeholder) : placeholder}
              disabled={disabled}
              minRows={1}
              maxRows={compact ? 5 : 8}
              aria-label={t("Message the assistant")}
              className="resize-none border-0 bg-transparent px-3 py-2.5 text-sm shadow-none focus-visible:border-transparent focus-visible:ring-0 md:text-sm"
            />

            <div className="flex items-end justify-between gap-2 px-2 pb-2">
              <div className="flex min-w-0 flex-wrap items-center gap-1">
                {agent && onPickAgent && (
                  <Tooltip>
                    <TooltipTrigger
                      render={
                        <button
                          type="button"
                          onClick={onPickAgent}
                          className="text-muted-foreground hover:text-foreground hover:bg-muted inline-flex h-6 max-w-[11rem] items-center gap-1.5 rounded-full px-1 pr-2 text-xs transition-colors"
                        >
                          <AgentTile agent={agent} size="xs" />
                          <span className="truncate">{agent.name}</span>
                        </button>
                      }
                    />
                    <TooltipContent>{t("Ask a different agent")}</TooltipContent>
                  </Tooltip>
                )}

                {pageContext && onToggleContext && (
                  <Tooltip>
                    <TooltipTrigger
                      render={
                        <button
                          type="button"
                          onClick={onToggleContext}
                          aria-pressed={contextIncluded}
                          className={cn(
                            "inline-flex h-6 max-w-[12rem] items-center gap-1 rounded-full border px-2 text-xs transition-colors",
                            contextIncluded
                              ? "border-border/70 bg-muted text-foreground"
                              : "border-dashed border-border/70 text-muted-foreground hover:text-foreground",
                          )}
                        >
                          <MapPinIcon className="size-3 shrink-0" />
                          <span className="truncate">{pageContext.title || t("This page")}</span>
                        </button>
                      }
                    />
                    <TooltipContent>
                      {contextIncluded
                        ? t("The assistant can see this page. Click to leave it out.")
                        : t("Include what you are looking at")}
                    </TooltipContent>
                  </Tooltip>
                )}

                <AnimatePresence initial={false}>
                  {showSuggestions &&
                    suggestions.map((suggestion) => (
                      <m.span
                        key={suggestion.prompt}
                        initial={reduceMotion ? false : { opacity: 0, scale: 0.94 }}
                        animate={{ opacity: 1, scale: 1 }}
                        exit={{ opacity: 0, scale: 0.94 }}
                        transition={{ duration: 0.12 }}
                        className="text-muted-foreground hover:text-foreground border-border/70 group inline-flex h-6 items-center rounded-full border text-xs transition-colors"
                      >
                        <button
                          type="button"
                          onClick={() => onSend(suggestion.prompt)}
                          className="hover:bg-muted h-full rounded-l-full pr-1 pl-2.5"
                        >
                          {t(suggestion.label)}
                        </button>
                        {onDismissSuggestion && (
                          <button
                            type="button"
                            aria-label={t("Dismiss suggestion")}
                            onClick={() => onDismissSuggestion(suggestion.prompt)}
                            className="hover:bg-muted h-full rounded-r-full pr-1.5 pl-0.5 opacity-60 hover:opacity-100"
                          >
                            <XIcon className="size-3" />
                          </button>
                        )}
                      </m.span>
                    ))}
                </AnimatePresence>
              </div>

              <Tooltip>
                <TooltipTrigger
                  render={
                    <Button
                      size="icon-sm"
                      className={cn("shrink-0 rounded-full", active && "assistant-stop")}
                      onClick={active ? onStop : submit}
                      disabled={!active && !canSend}
                      aria-label={active ? t("Stop") : t("Send")}
                    />
                  }
                >
                  <AnimatePresence mode="popLayout" initial={false}>
                    {active ? (
                      <m.span
                        key="stop"
                        initial={reduceMotion ? false : { opacity: 0, scale: 0.6 }}
                        animate={{ opacity: 1, scale: 1 }}
                        exit={{ opacity: 0, scale: 0.6 }}
                        transition={{ duration: 0.14, ease: [0.2, 0.8, 0.2, 1] }}
                      >
                        <SquareIcon className="size-3 fill-current" />
                      </m.span>
                    ) : (
                      <m.span
                        key="send"
                        initial={reduceMotion ? false : { opacity: 0, scale: 0.6 }}
                        animate={{ opacity: 1, scale: 1 }}
                        exit={{ opacity: 0, scale: 0.6 }}
                        transition={{ duration: 0.14, ease: [0.2, 0.8, 0.2, 1] }}
                      >
                        <ArrowUpIcon className="size-4" />
                      </m.span>
                    )}
                  </AnimatePresence>
                </TooltipTrigger>
                <TooltipContent>{active ? t("Stop") : t("Send")}</TooltipContent>
              </Tooltip>
            </div>
          </div>
        </div>

        {/* Two permanent lines of chrome under the box said the same thing on
            every page for the life of the session. The keyboard hint appears
            while someone is actually typing and goes away again; what the
            assistant may do is stated once, on the launch pad. */}
        <AnimatePresence initial={false}>
          {focused && (
            <m.div
              key="hint"
              initial={reduceMotion ? false : { opacity: 0, height: 0 }}
              animate={{ opacity: 1, height: "auto" }}
              exit={{ opacity: 0, height: 0 }}
              transition={{ duration: 0.14, ease: [0.2, 0.8, 0.2, 1] }}
              className="text-muted-foreground overflow-hidden px-1 text-xs"
            >
              <span className="hidden items-center gap-1 pt-1 sm:flex">
                <Kbd>Enter</Kbd> {t("to send")} · <Kbd>Shift</Kbd>+<Kbd>Enter</Kbd>{" "}
                {t("for a new line")}
              </span>
            </m.div>
          )}
        </AnimatePresence>
      </div>
    </div>
  );
}
