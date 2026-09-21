import { useT } from "@trenova/shared/i18n/use-t";
import { AgentTile } from "@/components/agent-identity/agent-tile";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd } from "@trenova/shared/components/ui/kbd";
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
  MicIcon,
  MicOffIcon,
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
import { ModelPicker } from "./model-picker";
import type { Suggestion } from "./suggestions";
import { useDictation } from "./use-dictation";

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

/**
 * Where a person types.
 *
 * Enter sends and Shift+Enter breaks a line; Cmd+Enter sends too, for people
 * who expect it. A draft that opens with a slash lists the commands and the
 * starter questions and narrows them as it grows; a command with slots is
 * filled in the box, its empty slots shown as a hint, and sent as the full
 * question. An @ opens a search over the organization's records, and the
 * record picked rides with the message as something to look up. A file
 * dropped or attached becomes a document on the thread before the message
 * leaves. The row under the text says who is being asked, which model
 * answers and what they can see, because all three are things a person
 * would otherwise have to assume.
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
}: ComposerProps) {
  const t = useT();
  const reduceMotion = useReducedMotion();
  const [sent, setSent] = useState(0);
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
  const [interim, setInterim] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
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

  const dictation = useDictation({
    onTranscript: (text) => {
      if (text === "") {
        return;
      }
      const base = draft.trimEnd();
      onDraftChange(base === "" ? text : base + " " + text);
    },
    onInterim: setInterim,
  });

  const deliver = useCallback(
    (content: string) => {
      onSend(content, {
        attachments: readyAttachments(attachments),
        mentions: activeMentions(content, mentions),
      });
      onDraftChange("");
      onMentionsChange?.([]);
      setSent((count) => count + 1);
      setConfirming(true);
      window.setTimeout(() => setConfirming(false), CONFIRM_MS);
    },
    [attachments, mentions, onDraftChange, onMentionsChange, onSend],
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
              onDragOver={(event) => {
                if (!canAttach) return;
                event.preventDefault();
                setDragging(true);
              }}
              onDragLeave={() => setDragging(false)}
              onDrop={(event) => {
                if (!canAttach) return;
                event.preventDefault();
                setDragging(false);
                takeFiles(event.dataTransfer.files);
              }}
              className={cn(
                "ui-container-focus-ring bg-field border-input hover:border-border-strong relative flex flex-col overflow-hidden rounded-lg border transition-colors",
                disabled && "bg-sunken opacity-60",
                dragging && "border-brand border-dashed",
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
                    {commands.map((entry, index) => (
                      <li
                        key={entry.kind === "command" ? entry.label : entry.prompt}
                        role="option"
                        aria-selected={index === highlighted}
                        onMouseEnter={() => setHighlighted(index)}
                        onMouseDown={(event) => event.preventDefault()}
                        onClick={() => chooseEntry(entry)}
                        className={cn(
                          "flex cursor-pointer items-center gap-2 px-3 py-1.5 text-sm",
                          index === highlighted ? "bg-surface-hover" : "",
                        )}
                      >
                        <span
                          className={cn(
                            "min-w-0 shrink-0 truncate",
                            entry.kind === "command" && "font-mono text-xs",
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
                        {index === highlighted && (
                          <CornerDownLeftIcon className="text-muted-foreground ml-auto size-3 shrink-0" />
                        )}
                      </li>
                    ))}
                  </m.ul>
                )}
                {mentionsOpen && (
                  <m.ul
                    key="mentions"
                    id={mentionListId}
                    role="listbox"
                    aria-label={t("Records")}
                    initial={reduceMotion ? false : { opacity: 0, height: 0 }}
                    animate={{ opacity: 1, height: "auto" }}
                    exit={{ opacity: 0, height: 0 }}
                    transition={{ duration: 0.16, ease: [0.16, 1, 0.3, 1] }}
                    className="border-border overflow-hidden border-b"
                  >
                    {mentionResults.map((candidate, index) => (
                      <li
                        key={`${candidate.type}:${candidate.id}`}
                        role="option"
                        aria-selected={index === mentionHighlighted}
                        onMouseEnter={() => setMentionHighlighted(index)}
                        onMouseDown={(event) => event.preventDefault()}
                        onClick={() => pickMention(candidate)}
                        className={cn(
                          "flex cursor-pointer items-center gap-2 px-3 py-1.5 text-sm",
                          index === mentionHighlighted ? "bg-surface-hover" : "",
                        )}
                      >
                        <AtSignIcon className="text-muted-foreground size-3 shrink-0" />
                        <span className="min-w-0 truncate">{candidate.label}</span>
                        <span className="text-muted-foreground min-w-0 flex-1 truncate text-xs">
                          {candidate.type.replaceAll("_", " ")}
                          {candidate.subtitle ? ` · ${candidate.subtitle}` : ""}
                        </span>
                        {index === mentionHighlighted && (
                          <CornerDownLeftIcon className="text-muted-foreground ml-auto size-3 shrink-0" />
                        )}
                      </li>
                    ))}
                  </m.ul>
                )}
              </AnimatePresence>

              {attachments.length > 0 && (
                <ul
                  aria-label={t("Attached files")}
                  className="flex flex-wrap gap-1.5 px-2.5 pt-2.5"
                >
                  {attachments.map((attachment) => (
                    <AttachmentChip
                      key={attachment.id}
                      attachment={attachment}
                      onRemove={
                        onRemoveAttachment ? () => onRemoveAttachment(attachment.id) : undefined
                      }
                    />
                  ))}
                </ul>
              )}

              <Textarea
                ref={textareaRef}
                value={draft}
                onChange={(event) => {
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
                placeholder={disabled ? (disabledReason ?? placeholder) : placeholder}
                disabled={disabled}
                minRows={1}
                maxRows={compact ? 5 : 8}
                aria-label={t("Message the assistant")}
                aria-controls={commandsOpen ? listId : mentionsOpen ? mentionListId : undefined}
                aria-expanded={commandsOpen || mentionsOpen || undefined}
                // The box around it carries the one focus ring, so the field's
                // own is switched off by repointing its width, not rebuilt.
                className="resize-none border-0 bg-transparent px-3 py-2.5 text-sm [--ring-width:0px] md:text-sm"
              />

              {(hint !== "" || interim !== "") && (
                <div className="text-muted-foreground px-3 pb-1 text-xs">
                  {interim !== "" ? (
                    <span className="italic">{interim}</span>
                  ) : (
                    <span className="font-mono">{hint}</span>
                  )}
                </div>
              )}

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
                            {contextIncluded && pageContext.view && (
                              <span className="text-muted-foreground shrink-0">
                                {pageViewSummary(pageContext, t)}
                              </span>
                            )}
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

                <div className="flex shrink-0 items-center gap-0.5">
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
                              className="text-muted-foreground hover:text-foreground rounded-full"
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

                  {dictation.supported && (
                    <Tooltip>
                      <TooltipTrigger
                        render={
                          <Button
                            size="icon-sm"
                            variant="ghost"
                            className={cn(
                              "rounded-full",
                              dictation.listening
                                ? "text-danger-foreground"
                                : "text-muted-foreground hover:text-foreground",
                            )}
                            disabled={disabled}
                            aria-pressed={dictation.listening}
                            onClick={dictation.toggle}
                            aria-label={dictation.listening ? t("Stop dictating") : t("Dictate")}
                          />
                        }
                      >
                        {dictation.listening ? (
                          <MicOffIcon className="size-4" />
                        ) : (
                          <MicIcon className="size-4" />
                        )}
                      </TooltipTrigger>
                      <TooltipContent>
                        {dictation.listening ? t("Stop dictating") : t("Dictate")}
                      </TooltipContent>
                    </Tooltip>
                  )}

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
                    <TooltipContent>
                      {active
                        ? t("Stop")
                        : uploading
                          ? t("Waiting for the files to finish uploading")
                          : t("Send")}
                    </TooltipContent>
                  </Tooltip>
                </div>
              </div>
            </div>
          </div>

          {/* The keyboard hint appears while someone is typing and goes away
              again. It is the one place the slash and the @ are taught. */}
          <AnimatePresence initial={false}>
            {focused && !commandsOpen && !mentionsOpen && (
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
                  {" · "}
                  <Kbd>/</Kbd> {t("for commands")}
                  {onSearchMentions && (
                    <>
                      {" · "}
                      <Kbd>@</Kbd> {t("to name a record")}
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

/** One file on the message: its name, how far it has got, and a way off. */
function AttachmentChip({
  attachment,
  onRemove,
}: {
  attachment: ComposerAttachment;
  onRemove?: () => void;
}) {
  const t = useT();
  const failed = attachment.status === "error";
  const busy = attachment.status === "uploading";

  return (
    <li
      className={cn(
        "border-border bg-surface inline-flex h-7 max-w-[16rem] items-center gap-1.5 rounded-full border pr-1 pl-2 text-xs",
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
          className="text-muted-foreground hover:text-foreground ui-focus-ring inline-flex size-5 shrink-0 items-center justify-center rounded-full"
        >
          <XIcon className="size-3" />
        </button>
      )}
    </li>
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
