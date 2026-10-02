import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { useDeskStore } from "@/stores/desk-store";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useRef, type KeyboardEvent, type ReactNode } from "react";
import { DeskIcon } from "../desk-icons";
import { DeskAgentPicker } from "./desk-agent-picker";
import { useTypewriter } from "./use-typewriter";

/** What the composer says it is doing while an agent works. */
export type DeskComposerStatus = {
  text: string;
  /** The kind of step: checking the question, retrying a model, or ordinary work. */
  pose: "check" | "retry" | "work";
  /** A second, quieter fact beside the status, such as "Attempt 2 of 3". */
  extra?: string;
};

export type DeskComposerProps = {
  value: string;
  onChange: (value: string) => void;
  onSend: (content: string) => void;
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
  /** Controls between the agent and the model: the page Desk can see, for one. */
  extras?: ReactNode;
  /** The model picker, on the right before dictation and send. */
  model?: ReactNode;
  /** The dictation control, beside send. */
  dictation?: ReactNode;
};

/**
 * Where a person talks to the Desk.
 *
 * While an agent works, a light in Trenova's colours runs round the whole
 * border and a line inside the top says what it is doing; nothing about the
 * work appears in the conversation until the reply starts. On the front page
 * the ring turns slowly at rest and the placeholder types out the agent's own
 * starter questions: Tab drops the one on screen into the box and ⌘1–⌘3 asks
 * it outright. Enter sends, Shift+Enter breaks a line.
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
  extras,
  model,
  dictation,
}: DeskComposerProps) {
  const t = useT();
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const markTermsSeen = useDeskStore((state) => state.markTermsSeen);
  const empty = value === "";
  const typing = home && presets.length > 0 && empty && !disabled;
  const typed = useTypewriter(presets, typing);
  const canSend = value.trim() !== "" && !busy && !disabled && !lock;

  const send = (content: string) => {
    const text = content.trim();
    if (text === "" || busy || disabled || lock) {
      return;
    }
    markTermsSeen();
    onSend(text);
    onChange("");
  };

  const onKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
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
      event.preventDefault();
      send(value);
    }
  };

  return (
    <div className={cn("dk-cmp", busy && "dk-busy", home && "dk-hm", lock && "dk-ec dk-off")}>
      <span className="dk-cmp-ring" aria-hidden />
      {busy && status && (
        <div className="dk-cmp-st dk-ec-st" key={status.text} role="status">
          {status.pose === "check" ? (
            <span className="dk-ec-shield">
              <DeskIcon name="shield" size={13} stroke={2} />
            </span>
          ) : (
            <span className="dk-cmp-st-d" />
          )}
          <span className={status.pose === "retry" ? undefined : "dk-shim"}>{status.text}</span>
          {status.extra && (
            <>
              <span className="dk-ec-sp" />
              <span className="dk-ec-att">{status.extra}</span>
            </>
          )}
        </div>
      )}
      {lock ? (
        <div className="dk-ec-offmsg">{lock}</div>
      ) : (
        <div className="dk-cmp-ta">
          <textarea
            ref={textareaRef}
            rows={2}
            value={value}
            disabled={disabled}
            placeholder={
              typing
                ? ""
                : (placeholder ??
                  (agent ? t("Reply to {0}…", agent.name) : t("Ask the Desk anything…")))
            }
            aria-label={agent ? t("Message {0}", agent.name) : t("Message the Desk")}
            onChange={(event) => onChange(event.target.value)}
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
        {agent && onAgentChange && (
          <DeskAgentPicker
            agent={agent}
            onSelect={onAgentChange}
            recentIds={recentAgentIds}
            lastUsedAt={agentLastUsedAt}
            disabled={busy}
          />
        )}
        {extras}
        <span style={{ flex: 1 }} />
        {model}
        {dictation}
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
            disabled={!canSend}
            title={t("Send")}
            aria-label={t("Send")}
            onClick={() => send(value)}
          >
            <DeskIcon name="up" size={15} stroke={2.2} />
          </button>
        )}
      </div>
    </div>
  );
}
