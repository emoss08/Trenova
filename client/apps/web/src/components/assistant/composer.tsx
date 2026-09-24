import { useT } from "@trenova/shared/i18n/use-t";
import { EASE_SETTLE, EASE_SWIFT } from "@/lib/motion";
import { AgentTile } from "@/components/agent-identity/agent-tile";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { Button } from "@trenova/shared/components/ui/button";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { cn, formatFileSize } from "@trenova/shared/lib/utils";
import type {
  AssistantEntityRef,
  AssistantMessageAttachment,
  AssistantPageContext,
  AssistantProviderOption,
} from "@/types/assistant";
import {
  ArrowUpIcon,
  AtSignIcon,
  CornerDownLeftIcon,
  FileIcon,
  MapPinIcon,
  MapPinOffIcon,
  PaperclipIcon,
  SquareIcon,
  XIcon,
} from "lucide-react";
import { AnimatePresence, m, useReducedMotion } from "motion/react";
import { useCallback, useEffect, useId, useMemo, useRef, useState } from "react";
import {
  commandEntries,
  fillCommand,
  parseSlashCommand,
  slashQuery,
  slotHint,
  type CommandEntry,
} from "./composer-commands";
import { composerHint } from "./composer-hint";
import { ComposerHints } from "./composer-hints";
import { DictationControl } from "./dictation-control";
import { ModelPicker } from "./model-picker";
import type { Suggestion } from "./suggestions";
import { useComposerDictation } from "./use-composer-dictation";
import type { DictationScope } from "./use-dictation";

/** A file on its way to the message, or already there. */
export type ComposerAttachment = {
  id: string;
  name: string;
  size: number;
  status: "uploading" | "ready" | "error";
  progress: number;
  documentId?: string;
  contentType?: string;
  error?: string;
};

/** A record the mention search offers. */
export type MentionCandidate = AssistantEntityRef & {
  subtitle?: string;
};

/** What leaves with the words. */
export type ComposerPayload = {
  attachments: AssistantMessageAttachment[];
  mentions: AssistantEntityRef[];
};

export type ComposerProps = {
  onSend: (content: string, payload: ComposerPayload) => void;
  onStop: () => void;
  active: boolean;
  disabled?: boolean;
  disabledReason?: string;
  placeholder: string;
  /** The draft is owned by the thread, so switching conversations keeps it. */
  draft: string;
  onDraftChange: (draft: string) => void;
  /** The agent the message goes to, shown so it is never a guess. */
  agent?: AgentChoice | null;
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
  /** Files on the message. Offered only when the thread can take them. */
  attachments?: readonly ComposerAttachment[];
  onAttachFiles?: (files: File[]) => void;
  onRemoveAttachment?: (id: string) => void;
  /** Records named with @. Offered only when there is a search to name them from. */
  mentions?: readonly AssistantEntityRef[];
  onMentionsChange?: (mentions: AssistantEntityRef[]) => void;
  onSearchMentions?: (query: string) => Promise<MentionCandidate[]>;
  compact?: boolean;
  /** A line above the box: how long the conversation has grown, for one. */
  notice?: React.ReactNode;
  /** Measured by the thread so the last message never hides behind the box. */
  ref?: React.Ref<HTMLDivElement>;
  /** For tests: the page's dictation support, instead of reading it from window. */
  dictationScope?: DictationScope;
};

/** How long the send control holds its confirmation after a message leaves. */
const CONFIRM_MS = 320;

/** How long a mention query waits for typing to settle before searching. */
const MENTION_DEBOUNCE_MS = 150;

/** The most files one message may carry; the server refuses more. */
export const MAX_ATTACHMENTS = 5;

/**
 * The @ token being typed at the caret: the text after an @ that opens a
 * word, up to the caret, with no space inside it. Null when the caret is
 * not inside one.
 */
export function mentionQueryAt(draft: string, caret: number): string | null {
  const before = draft.slice(0, caret);
  const match = /(?:^|\s)@([^\s@]*)$/.exec(before);

  return match ? match[1] : null;
}

/** The mentions still named in the text; one whose @label was deleted is dropped. */
export function activeMentions(
  draft: string,
  mentions: readonly AssistantEntityRef[],
): AssistantEntityRef[] {
  return mentions.filter((mention) => mention.label !== "" && draft.includes("@" + mention.label));
}

/** The documents that are on the message, as the server takes them. */
export function readyAttachments(
  attachments: readonly ComposerAttachment[],
): AssistantMessageAttachment[] {
  return attachments.flatMap((attachment) =>
    attachment.status === "ready" && attachment.documentId
      ? [
          {
            documentId: attachment.documentId,
            fileName: attachment.name,
            contentType: attachment.contentType ?? "",
            fileSize: attachment.size,
          },
        ]
      : [],
  );
}

/** A message on its way out of the box, drawn where the words were. */
type SentGhost = { id: number; text: string; top: number };

/**
 * Where a person types.
 *
 * Enter sends and Shift+Enter breaks a line; Cmd+Enter sends too, for people
 * who expect it. A draft that opens with a slash lists the commands and the
 * starter questions and narrows them as it grows; a command with slots is
 * filled in the box, its empty slots shown as a hint, and sent as the full
 * question. An @ opens a search over the organization's records, and the
 * record picked rides with the message as something to look up. A file
 * dropped, pasted or attached becomes a document on the thread before the
 * message leaves. Dictation writes into the draft as it is heard.
 *
 * One row of controls sits under the text and stays one row at the corner
 * panel's width: what goes with the message on the left (a file, a voice,
 * the page), who answers it on the right (the model, then send). Under the
 * box, one line says the one thing worth knowing right now.
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
  attachments = [],
  onAttachFiles,
  onRemoveAttachment,
  mentions = [],
  onMentionsChange,
  onSearchMentions,
  compact = false,
  notice,
  ref,
  dictationScope,
}: ComposerProps) {
  const t = useT();
  const reduceMotion = useReducedMotion();
  const [ghost, setGhost] = useState<SentGhost | null>(null);
  const [confirming, setConfirming] = useState(false);
  const [focused, setFocused] = useState(false);
  const [highlighted, setHighlighted] = useState(0);
  const [dragging, setDragging] = useState(false);
  const [caret, setCaret] = useState(0);
  // Results are kept with the query that produced them, so a list from an
  // earlier query never shows under a later one and closing is a matter of
  // the query moving on rather than a state to clear.
  const [mentionSearch, setMentionSearch] = useState<{
    query: string;
    results: MentionCandidate[];
  } | null>(null);
  const [mentionHighlighted, setMentionHighlighted] = useState(0);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLUListElement>(null);
  const sentCountRef = useRef(0);
  const listId = useId();
  const mentionListId = useId();

  const uploading = attachments.some((attachment) => attachment.status !== "ready");
  const canSend = draft.trim() !== "" && !active && !disabled && !uploading;

  const parsedCommand = useMemo(() => parseSlashCommand(draft), [draft]);
  const query = slashQuery(draft);
  const commands = useMemo<CommandEntry[]>(
    () => (query === null || active ? [] : commandEntries(query, suggestions)),
    [active, query, suggestions],
  );
  // Once a command's slots are being filled, the list is the hint, not a
  // menu: the row stays so the person sees what they are filling.
  const fillingSlots =
    parsedCommand !== null && parsedCommand.command.slots.length > 0 && /\s/.test(draft);
  const commandsOpen = commands.length > 0 && !fillingSlots;
  const hint = parsedCommand && fillingSlots ? slotHint(parsedCommand) : "";

  useEffect(() => {
    setHighlighted((index) => (index < commands.length ? index : 0));
  }, [commands.length]);

  const mentionQuery = onSearchMentions ? mentionQueryAt(draft, caret) : null;
  const mentionResults = useMemo(
    () =>
      mentionQuery !== null && mentionSearch?.query === mentionQuery ? mentionSearch.results : [],
    [mentionQuery, mentionSearch],
  );
  const mentionsOpen = mentionResults.length > 0 && !active;

  useEffect(() => {
    if (mentionQuery === null || !onSearchMentions || mentionQuery.length < 2) {
      return;
    }
    let cancelled = false;
    const timer = window.setTimeout(() => {
      onSearchMentions(mentionQuery)
        .then((results) => {
          if (!cancelled) {
            setMentionSearch({ query: mentionQuery, results });
            setMentionHighlighted(0);
          }
        })
        .catch(() => {
          if (!cancelled) {
            setMentionSearch({ query: mentionQuery, results: [] });
          }
        });
    }, MENTION_DEBOUNCE_MS);

    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [mentionQuery, onSearchMentions]);

  // The highlighted row stays in view as the arrows walk a long list.
  const activeIndex = mentionsOpen ? mentionHighlighted : commandsOpen ? highlighted : -1;
  useEffect(() => {
    if (activeIndex < 0) {
      return;
    }
    listRef.current
      ?.querySelector<HTMLElement>('[aria-selected="true"]')
      ?.scrollIntoView?.({ block: "nearest" });
  }, [activeIndex]);

  const dictation = useComposerDictation({ draft, onDraftChange, scope: dictationScope });
  const { phase: dictationPhase, release: releaseDictation, stop: stopDictation } = dictation;

  const deliver = useCallback(
    (content: string) => {
      releaseDictation();
      onSend(content, {
        attachments: readyAttachments(attachments),
        mentions: activeMentions(content, mentions),
      });
      onDraftChange("");
      onMentionsChange?.([]);
      sentCountRef.current += 1;
      // A list open over the box closes as the message leaves, so the words
      // lift from the top of the box rather than from below the list.
      setGhost({
        id: sentCountRef.current,
        text: content,
        top: commandsOpen || mentionsOpen ? 0 : (textareaRef.current?.offsetTop ?? 0),
      });
      setConfirming(true);
      window.setTimeout(() => setConfirming(false), CONFIRM_MS);
    },
    [
      attachments,
      commandsOpen,
      mentions,
      mentionsOpen,
      onDraftChange,
      onMentionsChange,
      onSend,
      releaseDictation,
    ],
  );

  const chooseEntry = useCallback(
    (entry: CommandEntry) => {
      if (entry.kind === "question") {
        if (!uploading) {
          deliver(entry.prompt);
        }
        return;
      }
      if (entry.command.slots.length === 0) {
        if (!uploading) {
          deliver(fillCommand(entry.command, []));
        }
        return;
      }
      onDraftChange("/" + entry.command.name + " ");
      textareaRef.current?.focus();
    },
    [deliver, onDraftChange, uploading],
  );

  const pickMention = useCallback(
    (candidate: MentionCandidate) => {
      if (mentionQuery === null) {
        return;
      }
      const before = draft.slice(0, caret);
      const after = draft.slice(caret);
      const start = before.length - mentionQuery.length - 1;
      const next = before.slice(0, start) + "@" + candidate.label + " " + after;
      onDraftChange(next);
      const known = mentions.some(
        (mention) => mention.type === candidate.type && mention.id === candidate.id,
      );
      if (!known) {
        onMentionsChange?.([
          ...mentions,
          { type: candidate.type, id: candidate.id, label: candidate.label },
        ]);
      }
      setMentionSearch(null);
      const position = start + candidate.label.length + 2;
      window.requestAnimationFrame(() => {
        const element = textareaRef.current;
        if (element) {
          element.focus();
          element.setSelectionRange(position, position);
          setCaret(position);
        }
      });
    },
    [caret, draft, mentionQuery, mentions, onDraftChange, onMentionsChange],
  );

  const submit = useCallback(() => {
    if (mentionsOpen) {
      pickMention(mentionResults[mentionHighlighted]);
      return;
    }
    if (commandsOpen) {
      chooseEntry(commands[highlighted]);
      return;
    }
    if (active || disabled || uploading) {
      return;
    }
    if (parsedCommand) {
      if (parsedCommand.complete) {
        deliver(fillCommand(parsedCommand.command, parsedCommand.args));
      }
      return;
    }
    const content = draft.trim();
    if (content === "") {
      return;
    }
    deliver(content);
  }, [
    active,
    chooseEntry,
    commands,
    commandsOpen,
    deliver,
    disabled,
    draft,
    highlighted,
    mentionHighlighted,
    mentionResults,
    mentionsOpen,
    parsedCommand,
    pickMention,
    uploading,
  ]);

  const onKeyDown = useCallback(
    (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (mentionsOpen) {
        if (event.key === "ArrowDown") {
          event.preventDefault();
          setMentionHighlighted((index) => (index + 1) % mentionResults.length);
          return;
        }
        if (event.key === "ArrowUp") {
          event.preventDefault();
          setMentionHighlighted(
            (index) => (index - 1 + mentionResults.length) % mentionResults.length,
          );
          return;
        }
        if (event.key === "Escape") {
          event.preventDefault();
          setMentionSearch(null);
          return;
        }
      }
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
    [commands.length, commandsOpen, mentionResults.length, mentionsOpen, onDraftChange, submit],
  );

  // Esc stops dictation wherever focus is in the box — the text or the mic —
  // and goes no further, so it does not also close the panel around it.
  const onKeyDownCapture = useCallback(
    (event: React.KeyboardEvent<HTMLDivElement>) => {
      if (event.key === "Escape" && dictationPhase !== "idle") {
        event.preventDefault();
        event.stopPropagation();
        stopDictation();
      }
    },
    [dictationPhase, stopDictation],
  );

  const takeFiles = useCallback(
    (list: FileList | File[] | null) => {
      if (!onAttachFiles || !list) {
        return;
      }
      const room = Math.max(0, MAX_ATTACHMENTS - attachments.length);
      const files = Array.from(list).slice(0, room);
      if (files.length > 0) {
        onAttachFiles(files);
      }
    },
    [attachments.length, onAttachFiles],
  );

  const onPaste = useCallback(
    (event: React.ClipboardEvent<HTMLTextAreaElement>) => {
      const files = Array.from(event.clipboardData.files);
      if (files.length > 0 && onAttachFiles) {
        event.preventDefault();
        takeFiles(files);
      }
    },
    [onAttachFiles, takeFiles],
  );

  const syncCaret = useCallback(() => {
    setCaret(textareaRef.current?.selectionStart ?? 0);
  }, []);

  const canAttach =
    onAttachFiles !== undefined && !disabled && attachments.length < MAX_ATTACHMENTS;

  const hintKind = composerHint({
    disabled,
    issue: dictation.issue,
    dictation: dictation.phase,
    menuOpen: commandsOpen || mentionsOpen,
    active,
    focused,
    draftEmpty: draft.trim() === "",
  });

  const optionId = (list: string, index: number) => `${list}-option-${index}`;

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
          compact ? "px-3 pt-0.5 pb-1.5" : "px-4 pt-0.5 pb-2.5",
        )}
      >
        <div className={cn("mx-auto flex flex-col gap-1", !compact && "max-w-3xl")}>
          {notice}
          <div className="relative">
            {/* The words lifting out of the box as it is sent: the one moment
                the composer moves on its own account, so sending reads as the
                message leaving rather than the text being deleted. */}
            <AnimatePresence>
              {ghost !== null && !reduceMotion && (
                <m.p
                  key={ghost.id}
                  aria-hidden
                  initial={{ opacity: 1, y: 0 }}
                  animate={{ opacity: 0, y: -28 }}
                  transition={{ duration: 0.34, ease: EASE_SETTLE }}
                  onAnimationComplete={() =>
                    setGhost((current) => (current?.id === ghost.id ? null : current))
                  }
                  style={{ top: ghost.top }}
                  className="text-foreground pointer-events-none absolute inset-x-0 z-10 truncate px-3 py-2.5 text-sm"
                >
                  {ghost.text}
                </m.p>
              )}
            </AnimatePresence>

            <div
              data-slot="composer"
              data-disabled={disabled || undefined}
              data-dragging={dragging || undefined}
              onKeyDownCapture={onKeyDownCapture}
              onDragOver={(event) => {
                if (!canAttach) return;
                event.preventDefault();
                setDragging(true);
              }}
              onDragLeave={(event) => {
                const next = event.relatedTarget;
                if (next instanceof Node && event.currentTarget.contains(next)) return;
                setDragging(false);
              }}
              onDrop={(event) => {
                if (!canAttach) return;
                event.preventDefault();
                setDragging(false);
                takeFiles(event.dataTransfer.files);
              }}
              className={cn(
                "ui-field ui-container-focus-ring group/composer relative flex flex-col overflow-hidden rounded-lg",
                "data-disabled:opacity-60",
                "data-dragging:border-brand data-dragging:border-dashed",
              )}
            >
              <AnimatePresence initial={false}>
                {commandsOpen && (
                  <m.div
                    key="commands"
                    initial={reduceMotion ? false : { opacity: 0, height: 0 }}
                    animate={{ opacity: 1, height: "auto" }}
                    exit={{ opacity: 0, height: 0 }}
                    transition={{ duration: 0.18, ease: EASE_SETTLE }}
                    className="border-border-subtle overflow-hidden border-b"
                  >
                    <ul
                      ref={listRef}
                      id={listId}
                      role="listbox"
                      aria-label={t("Starter questions")}
                      className="scrollbar-overlay max-h-60 overflow-y-auto py-1"
                    >
                      {commands.map((entry, index) => (
                        <CommandRow
                          key={entry.kind === "command" ? entry.label : entry.prompt}
                          id={optionId(listId, index)}
                          entry={entry}
                          heading={
                            index === 0 || commands[index - 1].kind !== entry.kind
                              ? entry.kind === "command"
                                ? t("Commands")
                                : t("Starter questions")
                              : null
                          }
                          selected={index === highlighted}
                          onHover={() => setHighlighted(index)}
                          onChoose={() => chooseEntry(entry)}
                        />
                      ))}
                    </ul>
                  </m.div>
                )}
                {mentionsOpen && (
                  <m.div
                    key="mentions"
                    initial={reduceMotion ? false : { opacity: 0, height: 0 }}
                    animate={{ opacity: 1, height: "auto" }}
                    exit={{ opacity: 0, height: 0 }}
                    transition={{ duration: 0.18, ease: EASE_SETTLE }}
                    className="border-border-subtle overflow-hidden border-b"
                  >
                    <ul
                      ref={listRef}
                      id={mentionListId}
                      role="listbox"
                      aria-label={t("Records")}
                      className="scrollbar-overlay max-h-60 overflow-y-auto py-1"
                    >
                      {mentionResults.map((candidate, index) => (
                        <li
                          key={`${candidate.type}:${candidate.id}`}
                          id={optionId(mentionListId, index)}
                          role="option"
                          aria-selected={index === mentionHighlighted}
                          onMouseEnter={() => setMentionHighlighted(index)}
                          onMouseDown={(event) => event.preventDefault()}
                          onClick={() => pickMention(candidate)}
                          className={cn(
                            "mx-1 flex h-8 cursor-pointer items-center gap-2 rounded-md px-2 text-sm transition-colors",
                            index === mentionHighlighted && "bg-surface-hover",
                          )}
                        >
                          <AtSignIcon className="text-foreground-subtle size-3.5 shrink-0" />
                          <span className="min-w-0 truncate">{candidate.label}</span>
                          <span className="text-muted-foreground min-w-0 flex-1 truncate text-xs">
                            {candidate.type.replaceAll("_", " ")}
                            {candidate.subtitle ? ` · ${candidate.subtitle}` : ""}
                          </span>
                          {index === mentionHighlighted && (
                            <CornerDownLeftIcon className="text-foreground-subtle ml-auto size-3 shrink-0" />
                          )}
                        </li>
                      ))}
                    </ul>
                  </m.div>
                )}
              </AnimatePresence>

              {attachments.length > 0 && (
                <ul
                  aria-label={t("Attached files")}
                  className="flex flex-wrap gap-1.5 px-2.5 pt-2.5"
                >
                  <AnimatePresence initial={false}>
                    {attachments.map((attachment) => (
                      <AttachmentChip
                        key={attachment.id}
                        attachment={attachment}
                        onRemove={
                          onRemoveAttachment ? () => onRemoveAttachment(attachment.id) : undefined
                        }
                      />
                    ))}
                  </AnimatePresence>
                </ul>
              )}

              <Textarea
                ref={textareaRef}
                value={draft}
                onChange={(event) => {
                  // Typing takes the box back from the microphone: what is on
                  // screen stays, and the recogniser stops writing into it.
                  releaseDictation();
                  if (dictation.issue !== null) {
                    dictation.dismissIssue();
                  }
                  onDraftChange(event.target.value);
                  setCaret(event.target.selectionStart ?? event.target.value.length);
                }}
                onSelect={syncCaret}
                onClick={syncCaret}
                onKeyUp={syncCaret}
                onFocus={() => setFocused(true)}
                onBlur={() => setFocused(false)}
                onKeyDown={onKeyDown}
                onPaste={onPaste}
                placeholder={
                  disabled
                    ? (disabledReason ?? placeholder)
                    : dictation.phase === "listening" && draft === ""
                      ? t("Listening…")
                      : placeholder
                }
                disabled={disabled}
                minRows={1}
                maxRows={compact ? 5 : 8}
                aria-label={t("Message the assistant")}
                aria-controls={commandsOpen ? listId : mentionsOpen ? mentionListId : undefined}
                aria-expanded={commandsOpen || mentionsOpen || undefined}
                aria-activedescendant={
                  mentionsOpen
                    ? optionId(mentionListId, mentionHighlighted)
                    : commandsOpen
                      ? optionId(listId, highlighted)
                      : undefined
                }
                // The box around it carries the one focus ring and the one
                // disabled treatment, so the field's own are switched off.
                className="resize-none border-0 bg-transparent px-3 pt-2.5 pb-1 text-sm [--ring-width:0px] disabled:bg-transparent disabled:opacity-100 md:text-sm"
              />

              {hint !== "" && (
                <p className="text-foreground-subtle animate-rise px-3 pb-1 font-mono text-xs">
                  {hint}
                </p>
              )}

              <div className="flex items-center gap-1 px-1.5 pb-1.5">
                <div className="flex min-w-0 flex-1 items-center gap-0.5">
                  {onAttachFiles && (
                    <>
                      <input
                        ref={fileInputRef}
                        type="file"
                        multiple
                        className="sr-only"
                        tabIndex={-1}
                        aria-hidden
                        onChange={(event) => {
                          takeFiles(event.target.files);
                          event.target.value = "";
                        }}
                      />
                      <Tooltip>
                        <TooltipTrigger
                          render={
                            <Button
                              size="icon-sm"
                              variant="ghost"
                              className="text-foreground-muted hover:text-foreground rounded-full"
                              disabled={!canAttach}
                              onClick={() => fileInputRef.current?.click()}
                              aria-label={t("Attach a file")}
                            />
                          }
                        >
                          <PaperclipIcon className="size-4" />
                        </TooltipTrigger>
                        <TooltipContent>
                          {attachments.length >= MAX_ATTACHMENTS
                            ? t("Up to {0} files per message", MAX_ATTACHMENTS)
                            : t("Attach a file")}
                        </TooltipContent>
                      </Tooltip>
                    </>
                  )}

                  <DictationControl
                    availability={dictation.availability}
                    phase={dictation.phase}
                    issue={dictation.issue}
                    stream={dictation.stream}
                    disabled={disabled}
                    onToggle={dictation.toggleFromDraft}
                  />

                  {pageContext && onToggleContext && (
                    <ContextChip
                      context={pageContext}
                      included={contextIncluded}
                      compact={compact}
                      onToggle={onToggleContext}
                    />
                  )}

                  {agent && onPickAgent && (
                    <Tooltip>
                      <TooltipTrigger
                        render={
                          <button
                            type="button"
                            onClick={onPickAgent}
                            aria-label={t("Ask a different agent")}
                            className={cn(
                              "ui-focus-ring ui-press text-foreground-muted hover:text-foreground hover:bg-surface-hover inline-flex h-7 min-w-0 shrink items-center gap-1.5 rounded-full px-1 text-xs",
                              !compact && "max-w-44 pr-2",
                            )}
                          >
                            <AgentTile agent={agent} size="xs" />
                            {!compact && <span className="truncate">{agent.name}</span>}
                          </button>
                        }
                      />
                      <TooltipContent>
                        {compact
                          ? t("Asking {0}. Click to ask a different agent.", agent.name)
                          : t("Ask a different agent")}
                      </TooltipContent>
                    </Tooltip>
                  )}
                </div>

                <div className="flex shrink-0 items-center gap-1">
                  {onPickProvider && (
                    <ModelPicker
                      options={providers}
                      value={providerId}
                      onChange={onPickProvider}
                      disabled={disabled || active}
                      compact={compact}
                    />
                  )}

                  <SendControl
                    active={active}
                    armed={canSend || commandsOpen}
                    confirming={confirming}
                    uploading={uploading}
                    onSend={submit}
                    onStop={onStop}
                  />
                </div>
              </div>

              <AnimatePresence>
                {dragging && (
                  <m.div
                    key="drop"
                    aria-hidden
                    initial={reduceMotion ? false : { opacity: 0 }}
                    animate={{ opacity: 1 }}
                    exit={{ opacity: 0 }}
                    transition={{ duration: 0.12, ease: EASE_SWIFT }}
                    className="bg-field text-foreground-muted pointer-events-none absolute inset-0 flex items-center justify-center gap-2 text-sm"
                  >
                    <PaperclipIcon className="size-4" />
                    {t("Drop to attach")}
                  </m.div>
                )}
              </AnimatePresence>
            </div>
          </div>

          <ComposerHints
            kind={hintKind}
            issue={dictation.issue}
            onDismissIssue={dictation.dismissIssue}
            canMention={onSearchMentions !== undefined}
            canAttach={onAttachFiles !== undefined}
            canDictate={dictation.supported}
          />
        </div>
      </div>
    </div>
  );
}

/** One row of the slash list, with the group's name above the first of its kind. */
function CommandRow({
  id,
  entry,
  heading,
  selected,
  onHover,
  onChoose,
}: {
  id: string;
  entry: CommandEntry;
  heading: string | null;
  selected: boolean;
  onHover: () => void;
  onChoose: () => void;
}) {
  const t = useT();

  return (
    <>
      {heading && (
        <li role="presentation" className="text-foreground-subtle px-3 pt-1.5 pb-1 text-2xs">
          {heading}
        </li>
      )}
      <li
        id={id}
        role="option"
        aria-selected={selected}
        onMouseEnter={onHover}
        onMouseDown={(event) => event.preventDefault()}
        onClick={onChoose}
        className={cn(
          "mx-1 flex h-8 cursor-pointer items-center gap-2 rounded-md px-2 text-sm transition-colors",
          selected && "bg-surface-hover",
        )}
      >
        <span
          className={cn(
            "min-w-0 shrink-0 truncate",
            entry.kind === "command" ? "font-mono text-xs" : "max-w-full",
          )}
        >
          {entry.kind === "command" ? entry.label : t(entry.label)}
        </span>
        {entry.kind === "command" && (
          <span className="text-muted-foreground min-w-0 flex-1 truncate text-xs">
            {entry.command.slots.map((slot) => `{${t(slot.hint)}}`).join(" ")}
            {entry.command.slots.length > 0 ? " · " : ""}
            {t(entry.description)}
          </span>
        )}
        {selected && (
          <CornerDownLeftIcon className="text-foreground-subtle ml-auto size-3 shrink-0" />
        )}
      </li>
    </>
  );
}

/**
 * The page the person is on, offered with the message. Included, it is a
 * quiet filled chip that says what it carries; left out, it is an outline
 * with the pin struck through, so the difference is a shape, not a colour.
 */
function ContextChip({
  context,
  included,
  compact,
  onToggle,
}: {
  context: AssistantPageContext;
  included: boolean;
  compact: boolean;
  onToggle: () => void;
}) {
  const t = useT();
  const reduceMotion = useReducedMotion();
  const summary = included && context.view ? pageViewSummary(context, t) : "";

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <button
            type="button"
            onClick={onToggle}
            aria-pressed={included}
            className={cn(
              "ui-focus-ring ui-press inline-flex h-7 min-w-0 shrink items-center gap-1 rounded-full border px-2 text-xs",
              compact ? "max-w-36" : "max-w-56",
              included
                ? "bg-surface-hover text-foreground-muted hover:text-foreground hover:bg-surface-active border-transparent"
                : "border-border text-foreground-subtle hover:text-foreground-muted border-dashed",
            )}
          >
            <span className="relative flex size-3 shrink-0 items-center justify-center">
              <AnimatePresence mode="popLayout" initial={false}>
                <m.span
                  key={included ? "in" : "out"}
                  initial={reduceMotion ? false : { opacity: 0, scale: 0.5 }}
                  animate={{ opacity: 1, scale: 1 }}
                  exit={{ opacity: 0, scale: 0.5 }}
                  transition={{ duration: 0.16, ease: EASE_SWIFT }}
                  className="flex"
                >
                  {included ? (
                    <MapPinIcon className="size-3" />
                  ) : (
                    <MapPinOffIcon className="size-3" />
                  )}
                </m.span>
              </AnimatePresence>
            </span>
            <span className="truncate">{context.title || t("This page")}</span>
            {summary !== "" && (
              <span className="text-foreground-subtle shrink-0 tabular-nums">{summary}</span>
            )}
          </button>
        }
      />
      <TooltipContent>
        {included
          ? t("The assistant can see this page. Click to leave it out.")
          : t("Include what you are looking at")}
      </TooltipContent>
    </Tooltip>
  );
}

/**
 * Send, and Stop while a reply is being written. It is one ink circle that
 * changes what it holds rather than two buttons, so the busy state reads as
 * the same control doing its other job. Empty, it rests unfilled; the first
 * character arms it and the arrow settles in with a small spring.
 */
function SendControl({
  active,
  armed,
  confirming,
  uploading,
  onSend,
  onStop,
}: {
  active: boolean;
  armed: boolean;
  confirming: boolean;
  uploading: boolean;
  onSend: () => void;
  onStop: () => void;
}) {
  const t = useT();
  const reduceMotion = useReducedMotion();
  const ready = active || armed;

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            size="icon-sm"
            variant="default"
            className={cn(
              "shrink-0 rounded-full",
              !ready && "bg-surface-active text-foreground-subtle disabled:opacity-100",
              confirming && "animate-confirm",
            )}
            onClick={active ? onStop : onSend}
            disabled={!ready}
            aria-label={active ? t("Stop the reply") : t("Send")}
          />
        }
      >
        <AnimatePresence mode="popLayout" initial={false}>
          {active ? (
            <m.span
              key="stop"
              initial={reduceMotion ? false : { opacity: 0, scale: 0.4, rotate: -45 }}
              animate={{ opacity: 1, scale: 1, rotate: 0 }}
              exit={{ opacity: 0, scale: 0.4 }}
              transition={{ duration: 0.18, ease: EASE_SETTLE }}
              className="flex"
            >
              <SquareIcon className="size-3 fill-current" />
            </m.span>
          ) : (
            <m.span
              key="send"
              initial={reduceMotion ? false : { opacity: 0, y: 6 }}
              animate={{ opacity: 1, y: 0, scale: armed ? 1 : 0.9 }}
              exit={{ opacity: 0, y: -10 }}
              transition={
                reduceMotion
                  ? { duration: 0 }
                  : { type: "spring", stiffness: 520, damping: 26, mass: 0.6 }
              }
              className="flex"
            >
              <ArrowUpIcon className="size-4" />
            </m.span>
          )}
        </AnimatePresence>
      </TooltipTrigger>
      <TooltipContent>
        {active
          ? t("Stop the reply")
          : uploading
            ? t("Waiting for the files to finish uploading")
            : t("Send")}
      </TooltipContent>
    </Tooltip>
  );
}

/** One file on the message: its name, how far it has got, and a way off. */
function AttachmentChip({
  attachment,
  onRemove,
}: {
  attachment: ComposerAttachment;
  onRemove?: () => void;
}) {
  const t = useT();
  const reduceMotion = useReducedMotion();
  const failed = attachment.status === "error";
  const busy = attachment.status === "uploading";

  return (
    <m.li
      layout={!reduceMotion}
      initial={reduceMotion ? false : { opacity: 0, scale: 0.9 }}
      animate={{ opacity: 1, scale: 1 }}
      exit={reduceMotion ? { opacity: 0, transition: { duration: 0 } } : { opacity: 0, scale: 0.9 }}
      transition={{ duration: 0.18, ease: EASE_SETTLE }}
      className={cn(
        "border-border bg-card relative inline-flex h-7 max-w-64 items-center gap-1.5 overflow-hidden rounded-full border pr-1 pl-2 text-xs",
        failed && "border-danger-border text-danger-foreground",
      )}
      title={failed ? attachment.error : undefined}
    >
      <FileIcon className="size-3 shrink-0" />
      <span className="min-w-0 truncate">{attachment.name}</span>
      <span className="text-muted-foreground shrink-0 tabular-nums">
        {failed
          ? t("Failed")
          : busy
            ? `${Math.round(attachment.progress)}%`
            : formatFileSize(attachment.size)}
      </span>
      {onRemove && (
        <button
          type="button"
          onClick={onRemove}
          aria-label={t("Remove {0}", attachment.name)}
          className="text-muted-foreground hover:text-foreground hover:bg-surface-hover ui-focus-ring inline-flex size-5 shrink-0 items-center justify-center rounded-full transition-colors"
        >
          <XIcon className="size-3" />
        </button>
      )}
    </m.li>
  );
}

/** A word on what the page chip carries besides its title. */
function pageViewSummary(context: AssistantPageContext, t: ReturnType<typeof useT>): string {
  const view = context.view;
  if (!view) {
    return "";
  }
  const filters = (view.fieldFilters?.length ?? 0) + (view.filterGroups?.length ?? 0);
  const selected = view.selection?.count ?? 0;
  if (selected > 0) {
    return t("· {0, plural, one {# selected} other {# selected}}", selected);
  }
  if (filters > 0) {
    return t("· {0, plural, one {# filter} other {# filters}}", filters);
  }
  if (view.rowCount != null) {
    return t("· {0, plural, one {# row} other {# rows}}", view.rowCount);
  }

  return "";
}
