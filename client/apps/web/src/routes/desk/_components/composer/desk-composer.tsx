import {
  activeMentions,
  readyAttachments,
  type ComposerPayload,
} from "@/components/assistant/composer";
import type { Suggestion } from "@/components/assistant/suggestions";
import { useComposerDictation } from "@/components/assistant/use-composer-dictation";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { useDeskSettingsStore } from "@/stores/desk-settings-store";
import { useDeskStore } from "@/stores/desk-store";
import type { AssistantEntityRef } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useRef, useState, type KeyboardEvent, type ReactNode } from "react";
import { DeskIcon } from "../desk-icons";
import { DeskAgentPicker } from "./desk-agent-picker";
import { MAX_ATTACHMENTS, type DeskAttachments } from "./desk-attachments";
import { DeskCapturePanel, type DeskScans } from "./desk-capture";
import { DeskDictate } from "./desk-dictate";
import { DeskMentionMirror, DeskMentionPicker, useDeskMentions } from "./desk-mentions";
import { DeskSlashMenu, DeskSlashMirror, useDeskSlash } from "./desk-slash";
import { DeskAttachMenu, DeskAttachRow, DeskDropGhosts, type DragState } from "./desk-uploads";
import { useTypewriter } from "./use-typewriter";
import { DeskCountdown } from "../desk-countdown";

/** What the composer says it is doing while an agent works. */
export type DeskComposerStatus = {
  text: string;
  /** The kind of step: checking the question, retrying a model, or ordinary work. */
  pose: "check" | "retry" | "work";
  /** A second, quieter fact beside the status, such as "Attempt 2 of 3". */
  extra?: string;
  /** Seconds to count down before the next try, drawn as a ring. */
  countdown?: number;
  /** A way out of the wait, such as switching model. */
  action?: { label: string; onClick: () => void };
};

const NO_MENTIONS: AssistantEntityRef[] = [];
const NO_SUGGESTIONS: Suggestion[] = [];

export type DeskComposerProps = {
  value: string;
  onChange: (value: string) => void;
  onSend: (content: string, payload: ComposerPayload) => void;
  onStop?: () => void;
  agent: AgentChoice | null;
  onAgentChange?: (agent: AgentChoice) => void;
  recentAgentIds?: readonly string[];
  agentLastUsedAt?: ReadonlyMap<string, number>;
  /** An agent is working on a reply: the border lights and send becomes stop. */
  busy: boolean;
  status?: DeskComposerStatus | null;
  disabled?: boolean;
  /** The front page's composer: a slow ring and the typed-out starter questions. */
  home?: boolean;
  presets?: readonly string[];
  placeholder?: string;
  /** Replaces the text box with a read-only line saying why nothing can be sent. */
  lock?: ReactNode;
  /** A quiet line under the composer, such as how much of an allowance is used. */
  note?: ReactNode;
  /** Seconds before the next message may be sent; the send button counts them down. */
  wait?: number;
  /** Opens the agent list each time it changes. */
  agentPickerSignal?: number;
  /** Opens the file picker each time it changes. */
  filePickerSignal?: number;
  /** Controls after the agent: the page Desk can see, for one. */
  extras?: ReactNode;
  /** The model picker, on the right before dictation and send. */
  model?: ReactNode;
  /** Files on the message. Without them there is no attach button. */
  attachments?: DeskAttachments;
  /** Scans into the message from Capture, when there is a conversation to scan into. */
  scans?: DeskScans | null;
  /** Records named with @. */
  mentions?: readonly AssistantEntityRef[];
  onMentionsChange?: (mentions: AssistantEntityRef[]) => void;
  /** The agent's starter questions, listed after the commands when a slash opens. */
  suggestions?: readonly Suggestion[];
  /** Files dragged over the Desk. */
  drag?: DragState;
};

/**
 * Where a person talks to the Desk.
 *
 * While an agent works, a light in Trenova's colours runs round the whole
 * border and a line inside the top says what it is doing; nothing about the
 * work appears in the conversation until the reply starts. On the front page
 * the ring turns slowly at rest and the placeholder types out the agent's own
 * starter questions: Tab drops the one on screen into the box and ⌘1–⌘3 asks
 * it outright. A slash lists commands, an @ names a record, the plus attaches
 * files or scans paper, and the microphone writes what is said into the box.
 * Enter sends, Shift+Enter breaks a line.
 */
export function DeskComposer({
  value,
  onChange,
  onSend,
  onStop,
  agent,
  onAgentChange,
  recentAgentIds = [],
  agentLastUsedAt,
  busy,
  status,
  disabled = false,
  home = false,
  presets = [],
  placeholder,
  lock,
  note,
  wait = 0,
  agentPickerSignal = 0,
  filePickerSignal = 0,
  extras,
  model,
  attachments,
  scans,
  mentions = NO_MENTIONS,
  onMentionsChange,
  suggestions = NO_SUGGESTIONS,
  drag,
}: DeskComposerProps) {
  const t = useT();
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [menu, setMenu] = useState<"menu" | "capture" | null>(null);
  const markTermsSeen = useDeskStore((state) => state.markTermsSeen);
  const settings = useDeskSettingsStore((state) => state.settings);
  const empty = value === "";
  const typing = home && settings.presets === "on" && presets.length > 0 && empty && !disabled;
  const typed = useTypewriter(presets, typing);
  const dictation = useComposerDictation({ draft: value, onDraftChange: onChange });

  const uploading = attachments?.uploading ?? false;
  const ready = attachments?.ready ?? [];
  const blocked = (attachments?.failed ?? 0) > 0;
  const canSend =
    (value.trim() !== "" || ready.length > 0) &&
    !uploading &&
    !blocked &&
    !busy &&
    !disabled &&
    !lock;

  const send = (content: string) => {
    let text = content.trim();
    if (text === "" && ready.length > 0) {
      text = ready.length > 1 ? t("What's in these?") : t("What's in this?");
    }
    if (text === "" || busy || disabled || lock || uploading || blocked || wait > 0) {
      return;
    }
    dictation.release();
    markTermsSeen();
    onSend(text, {
      attachments: readyAttachments(ready),
      mentions: activeMentions(text, mentions),
    });
    onChange("");
    onMentionsChange?.([]);
  };

  const slash = useDeskSlash({
    value,
    onChange,
    textareaRef,
    onSendText: send,
    suggestions,
    enabled: !typing && settings.slash === "on",
  });
  const mention = useDeskMentions({
    value,
    onChange,
    textareaRef,
    mentions,
    onMentionsChange: onMentionsChange ?? (() => undefined),
    enabled: onMentionsChange !== undefined && settings.mentions === "on",
  });

  useEffect(() => {
    if (filePickerSignal > 0) {
      fileInputRef.current?.click();
    }
  }, [filePickerSignal]);

  useEffect(() => {
    if (!attachments || lock) {
      return;
    }
    const onKey = (event: globalThis.KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "u") {
        event.preventDefault();
        fileInputRef.current?.click();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [attachments, lock]);

  const onKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (mention.onKeyDown(event) || slash.onKeyDown(event)) {
      return;
    }
    if (typing && event.key === "Tab" && !event.shiftKey) {
      event.preventDefault();
      onChange(typed.full);
      return;
    }
    if (
      typing &&
      (event.metaKey || event.ctrlKey) &&
      /^[1-9]$/.test(event.key) &&
      presets[Number(event.key) - 1]
    ) {
      event.preventDefault();
      send(presets[Number(event.key) - 1]);
      return;
    }
    if (event.key === "Enter" && !event.shiftKey && !event.nativeEvent.isComposing) {
      const withModifier = event.metaKey || event.ctrlKey;
      if ((settings.send === "mod") === withModifier) {
        event.preventDefault();
        send(value);
      }
    }
  };

  const dropping = Boolean(drag?.on) && !lock && attachments !== undefined;
  const full = attachments?.full ?? false;

  return (
    <div
      className={cn(
        "dk-cmp",
        busy && "dk-busy",
        home && "dk-hm",
        lock && "dk-ec dk-off",
        busy && status?.pose === "check" && "dk-ec dk-r-check",
        busy && status?.pose === "retry" && "dk-ec dk-r-retry",
        dropping && "dk-drop-on",
        dropping && drag?.hot && "dk-drop-hot",
      )}
    >
      {dropping && (
        <span className={cn("dk-drop-tag", full && "dk-full")}>
          {full ? t("Message full") : drag?.hot ? t("Release to attach") : t("Drop here")}
        </span>
      )}
      <span className="dk-cmp-ring" aria-hidden />
      {(busy || wait > 0) && status && (
        <div className="dk-cmp-st dk-ec-st" key={status.text} role="status">
          {status.pose === "check" ? (
            <span className="dk-ec-shield">
              <DeskIcon name="shield" size={13} stroke={2} />
            </span>
          ) : status.countdown ? (
            <DeskCountdown from={status.countdown} key={status.extra ?? status.text} />
          ) : (
            <span className="dk-cmp-st-d" />
          )}
          <span className={status.pose === "retry" ? undefined : "dk-shim"}>{status.text}</span>
          {(status.extra || status.action) && <span className="dk-ec-sp" />}
          {status.extra && <span className="dk-ec-att">{status.extra}</span>}
          {status.action && (
            <button type="button" className="dk-ec-link" onClick={status.action.onClick}>
              {status.action.label}
            </button>
          )}
        </div>
      )}
      {attachments && !lock && <DeskAttachRow attachments={attachments} />}
      {dropping && drag && (
        <DeskDropGhosts count={drag.count} taken={attachments?.items.length ?? 0} />
      )}
      {lock ? (
        <div className="dk-ec-offmsg">{lock}</div>
      ) : (
        <div className="dk-cmp-ta">
          <DeskMentionPicker mentions={mention} />
          <DeskSlashMenu slash={slash} agentName={agent?.name ?? t("the agent")} />
          {slash.parsed && slash.started ? (
            <DeskSlashMirror slash={slash} />
          ) : (
            <DeskMentionMirror value={value} mentions={mentions} />
          )}
          <textarea
            ref={textareaRef}
            rows={2}
            value={value}
            disabled={disabled}
            className={slash.parsed && slash.started ? "dk-sl-on" : undefined}
            placeholder={
              typing
                ? ""
                : (placeholder ??
                  (agent ? t("Reply to {0}…", agent.name) : t("Ask the Desk anything…")))
            }
            aria-label={agent ? t("Message {0}", agent.name) : t("Message the Desk")}
            onChange={(event) => {
              if (dictation.phase !== "idle") {
                dictation.release();
              }
              onChange(event.target.value);
              mention.detect(event.target.value, event.target.selectionStart);
            }}
            onSelect={(event) =>
              mention.detect(event.currentTarget.value, event.currentTarget.selectionStart)
            }
            onKeyDown={onKeyDown}
          />
          {typing && (
            <div className="dk-tw" aria-hidden>
              <span>{typed.text}</span>
              <i className="dk-tw-c" />
              {typed.done && (
                <span className="dk-tw-tab">
                  <span className="dk-kbd">Tab</span> {t("to use")} ·{" "}
                  <span className="dk-kbd">⌘{typed.index + 1}</span> {t("to ask")}
                </span>
              )}
            </div>
          )}
        </div>
      )}
      <div className="dk-cmp-b">
        {attachments && (
          <span className="relative">
            <button
              type="button"
              className={cn("dk-ib", menu && "dk-on")}
              title={full ? t("Up to {0} files per message", MAX_ATTACHMENTS) : t("Attach files")}
              aria-label={t("Attach files")}
              data-attach-toggle
              disabled={Boolean(lock) || full}
              onClick={() => setMenu((current) => (current ? null : "menu"))}
            >
              <DeskIcon name="plus" size={16} />
            </button>
            <input
              ref={fileInputRef}
              type="file"
              multiple
              hidden
              onChange={(event) => {
                if (event.target.files) {
                  attachments.add(event.target.files);
                }
                event.target.value = "";
              }}
            />
            {menu && (
              <DeskAttachMenu
                onPickFiles={attachments.add}
                onScan={scans ? () => setMenu("capture") : undefined}
                onClose={() => setMenu(null)}
              >
                {menu === "capture" && scans ? (
                  <DeskCapturePanel
                    scans={scans}
                    onBack={() => setMenu("menu")}
                    onStarted={() => setMenu(null)}
                  />
                ) : undefined}
              </DeskAttachMenu>
            )}
          </span>
        )}
        {agent && onAgentChange && (
          <DeskAgentPicker
            agent={agent}
            onSelect={onAgentChange}
            recentIds={recentAgentIds}
            lastUsedAt={agentLastUsedAt}
            disabled={busy}
            openSignal={agentPickerSignal}
          />
        )}
        {extras}
        <span className="flex-1" />
        {model}
        {!lock && settings.mic === "on" && (
          <DeskDictate dictation={dictation} disabled={busy || disabled} />
        )}
        {busy ? (
          <button
            type="button"
            className="dk-send dk-stop"
            title={t("Stop")}
            aria-label={t("Stop the reply")}
            onClick={onStop}
          >
            <span className="dk-sq" />
          </button>
        ) : (
          <button
            type="button"
            className="dk-send"
            disabled={!canSend || wait > 0}
            title={
              wait > 0
                ? t("You can send again in a moment")
                : uploading
                  ? t("Waiting for the files to finish uploading")
                  : t("Send")
            }
            aria-label={t("Send")}
            onClick={() => send(value)}
          >
            {wait > 0 ? (
              <DeskCountdown from={wait} size={18} tone="ink" />
            ) : uploading ? (
              <span className="dk-send-wait" />
            ) : (
              <DeskIcon name="up" size={15} stroke={2.2} />
            )}
          </button>
        )}
      </div>
      {note && <div className="dk-ec-cnote">{note}</div>}
    </div>
  );
}
