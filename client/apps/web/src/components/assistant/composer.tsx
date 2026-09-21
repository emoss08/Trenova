import { useT } from "@trenova/shared/i18n/use-t";
import { AgentTile } from "@/components/agent-identity/agent-tile";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { cn } from "@trenova/shared/lib/utils";
import type { AssistantPageContext, AssistantProviderOption } from "@/types/assistant";
import { ArrowUpIcon, CornerDownLeftIcon, MapPinIcon, SquareIcon } from "lucide-react";
import { AnimatePresence, m, useReducedMotion } from "motion/react";
import { useCallback, useEffect, useId, useMemo, useRef, useState } from "react";
import { matchSuggestions, slashQuery } from "./composer-commands";
import { ModelPicker } from "./model-picker";
import type { Suggestion } from "./suggestions";

export type ComposerProps = {
  onSend: (content: string) => void;
  onStop: () => void;
  active: boolean;
  disabled?: boolean;
  disabledReason?: string;
  placeholder: string;
  /** The draft is owned by the thread, so switching conversations keeps it. */
  draft: string;
  onDraftChange: (draft: string) => void;
  /** The agent the message goes to, shown so it is never a guess. */
  agent?: AgentDefinitionRow | null;
  onPickAgent?: () => void;
  /** What the person is looking at, offered as context they can drop. */
  pageContext?: AssistantPageContext | null;
  contextIncluded?: boolean;
  onToggleContext?: () => void;
  /** The models this organization offers, and the one this thread is set to. */
  providers?: readonly AssistantProviderOption[];
  providerId?: string;
  onPickProvider?: (providerId: string) => void;
  /** Opening questions, listed when the draft opens with a slash. */
  suggestions?: readonly Suggestion[];
  compact?: boolean;
  /** A line above the box: how long the conversation has grown, for one. */
  notice?: React.ReactNode;
  /** Measured by the thread so the last message never hides behind the box. */
  ref?: React.Ref<HTMLDivElement>;
};

/** How long the send control holds its confirmation after a message leaves. */
const CONFIRM_MS = 320;

/**
 * Where a person types.
 *
 * Enter sends and Shift+Enter breaks a line; Cmd+Enter sends too, for people
 * who expect it. A draft that opens with a slash lists the starter questions
 * and narrows them as it grows, so the questions an agent is good at are one
 * keystroke away without living under the box on every turn. The row under
 * the text says who is being asked, which model answers and what they can
 * see, because all three are things a person would otherwise have to assume.
 * One control on the right does send and stop, so the action that makes
 * sense is always the one on screen.
 */
export function Composer({
  onSend,
  onStop,
  active,
  disabled = false,
  disabledReason,
  placeholder,
  draft,
  onDraftChange,
  agent,
  onPickAgent,
  pageContext,
  contextIncluded = true,
  onToggleContext,
  providers = [],
  providerId = "",
  onPickProvider,
  suggestions = [],
  compact = false,
  notice,
  ref,
}: ComposerProps) {
  const t = useT();
  const reduceMotion = useReducedMotion();
  const [sent, setSent] = useState(0);
  const [confirming, setConfirming] = useState(false);
  const [focused, setFocused] = useState(false);
  const [highlighted, setHighlighted] = useState(0);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const listId = useId();

  const canSend = draft.trim() !== "" && !active && !disabled;

  const query = slashQuery(draft);
  const commands = useMemo(
    () => (query === null || active ? [] : matchSuggestions(query, suggestions)),
    [active, query, suggestions],
  );
  const commandsOpen = commands.length > 0;

  // The highlight follows the list: a narrowed list that no longer reaches
  // the highlighted row starts again at the top.
  useEffect(() => {
    setHighlighted((index) => (index < commands.length ? index : 0));
  }, [commands.length]);

  const deliver = useCallback(
    (content: string) => {
      onSend(content);
      onDraftChange("");
      setSent((count) => count + 1);
      setConfirming(true);
      window.setTimeout(() => setConfirming(false), CONFIRM_MS);
    },
    [onDraftChange, onSend],
  );

  const submit = useCallback(() => {
    if (commandsOpen) {
      deliver(commands[highlighted].prompt);
      return;
    }
    const content = draft.trim();
    if (content === "" || active || disabled) {
      return;
    }
    deliver(content);
  }, [active, commands, commandsOpen, deliver, disabled, draft, highlighted]);

  const onKeyDown = useCallback(
    (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (commandsOpen) {
        if (event.key === "ArrowDown") {
          event.preventDefault();
          setHighlighted((index) => (index + 1) % commands.length);
          return;
        }
        if (event.key === "ArrowUp") {
          event.preventDefault();
          setHighlighted((index) => (index - 1 + commands.length) % commands.length);
          return;
        }
        if (event.key === "Escape") {
          event.preventDefault();
          onDraftChange("");
          return;
        }
      }
      if (event.key === "Enter" && (event.metaKey || event.ctrlKey || !event.shiftKey)) {
        event.preventDefault();
        submit();
      }
    },
    [commands.length, commandsOpen, onDraftChange, submit],
  );

  return (
    // The composer floats on the panel rather than sitting in a bar beneath it,
    // so the thread runs to the bottom and the last lines fade under the box
    // instead of stopping at a rule.
    <div ref={ref} className="pointer-events-none absolute inset-x-0 bottom-0 z-10">
      <div
        aria-hidden
        className={cn(
          "from-popover pointer-events-none bg-gradient-to-t to-transparent",
          compact ? "h-6" : "h-10",
        )}
      />
      <div
        className={cn(
          "bg-popover pointer-events-auto",
          compact ? "px-3 pt-0.5 pb-3" : "px-4 pt-0.5 pb-4",
        )}
      >
        <div className={cn("mx-auto flex flex-col gap-1.5", !compact && "max-w-3xl")}>
          {notice}
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
                  className="bg-sunken pointer-events-none absolute inset-x-0 top-0 h-8 rounded-lg"
                />
              )}
            </AnimatePresence>

            <div
              className={cn(
                "ui-container-focus-ring bg-field border-input hover:border-border-strong relative flex flex-col overflow-hidden rounded-lg border transition-colors",
                disabled && "bg-sunken opacity-60",
              )}
            >
              <AnimatePresence initial={false}>
                {commandsOpen && (
                  <m.ul
                    key="commands"
                    id={listId}
                    role="listbox"
                    aria-label={t("Starter questions")}
                    initial={reduceMotion ? false : { opacity: 0, height: 0 }}
                    animate={{ opacity: 1, height: "auto" }}
                    exit={{ opacity: 0, height: 0 }}
                    transition={{ duration: 0.16, ease: [0.16, 1, 0.3, 1] }}
                    className="border-border overflow-hidden border-b"
                  >
                    {commands.map((suggestion, index) => (
                      <li
                        key={suggestion.prompt}
                        role="option"
                        aria-selected={index === highlighted}
                        onMouseEnter={() => setHighlighted(index)}
                        onMouseDown={(event) => event.preventDefault()}
                        onClick={() => deliver(suggestion.prompt)}
                        className={cn(
                          "flex cursor-pointer items-center gap-2 px-3 py-1.5 text-sm",
                          index === highlighted ? "bg-surface-hover" : "",
                        )}
                      >
                        <span className="min-w-0 flex-1 truncate">{t(suggestion.label)}</span>
                        {index === highlighted && (
                          <CornerDownLeftIcon className="text-muted-foreground size-3 shrink-0" />
                        )}
                      </li>
                    ))}
                  </m.ul>
                )}
              </AnimatePresence>

              <Textarea
                ref={textareaRef}
                value={draft}
                onChange={(event) => onDraftChange(event.target.value)}
                onFocus={() => setFocused(true)}
                onBlur={() => setFocused(false)}
                onKeyDown={onKeyDown}
                placeholder={disabled ? (disabledReason ?? placeholder) : placeholder}
                disabled={disabled}
                minRows={1}
                maxRows={compact ? 5 : 8}
                aria-label={t("Message the assistant")}
                aria-controls={commandsOpen ? listId : undefined}
                aria-expanded={commandsOpen || undefined}
                // The box around it carries the one focus ring, so the field's
                // own is switched off by repointing its width, not rebuilt.
                className="resize-none border-0 bg-transparent px-3 py-2.5 text-sm [--ring-width:0px] md:text-sm"
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
                            className="text-muted-foreground hover:text-foreground hover:bg-surface-hover ui-focus-ring inline-flex h-6 max-w-[11rem] items-center gap-1.5 rounded-full px-1 pr-2 text-xs transition-colors"
                          >
                            <AgentTile agent={agent} size="xs" />
                            <span className="truncate">{agent.name}</span>
                          </button>
                        }
                      />
                      <TooltipContent>{t("Ask a different agent")}</TooltipContent>
                    </Tooltip>
                  )}

                  {onPickProvider && (
                    <ModelPicker
                      options={providers}
                      value={providerId}
                      onChange={onPickProvider}
                      disabled={disabled || active}
                    />
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
                              "ui-focus-ring inline-flex h-6 max-w-[12rem] items-center gap-1 rounded-full border px-2 text-xs transition-colors",
                              contextIncluded
                                ? "border-border bg-surface-selected text-foreground"
                                : "border-border border-dashed text-muted-foreground hover:text-foreground",
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
                </div>

                <Tooltip>
                  <TooltipTrigger
                    render={
                      <Button
                        size="icon-sm"
                        variant={active ? "secondary" : "default"}
                        className={cn("shrink-0 rounded-full", confirming && "animate-confirm")}
                        onClick={active ? onStop : submit}
                        disabled={!active && !canSend && !commandsOpen}
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

          {/* The keyboard hint appears while someone is typing and goes away
              again. It is the one place the slash is taught. */}
          <AnimatePresence initial={false}>
            {focused && !commandsOpen && (
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
                  {suggestions.length > 0 && (
                    <>
                      {" · "}
                      <Kbd>/</Kbd> {t("for starter questions")}
                    </>
                  )}
                </span>
              </m.div>
            )}
          </AnimatePresence>
        </div>
      </div>
    </div>
  );
}
