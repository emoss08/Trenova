"use no memo";
import { deskThreadState } from "@/components/desk-chat/rail/desk-thread-state";
import { railTime, stateLabel } from "@/components/desk-chat/rail/rail-parts";
import { conversationPath } from "@/lib/conversation-path";
import type { AssistantThread } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { memo, type MouseEvent, type ReactNode } from "react";
import { Link } from "react-router";

/** Whether a click is a plain one, which the rail may take over; any other follows the link. */
export function isPlainClick(event: MouseEvent): boolean {
  return event.button === 0 && !event.metaKey && !event.ctrlKey && !event.shiftKey && !event.altKey;
}

/**
 * One two-line row of the rail, as the design draws it: a dot, a title, and a
 * quieter line under it. The dot turns blue while the row wants the person.
 */
export function RailRow({
  to,
  title,
  sub,
  on = false,
  tone,
  current = false,
  hint,
  onPlainClick,
}: {
  to: string;
  title: string;
  sub: ReactNode;
  on?: boolean;
  tone?: "warn" | "bad";
  current?: boolean;
  hint?: string;
  onPlainClick?: () => void;
}) {
  return (
    <Link
      to={to}
      className="dk-rl-th ui-focus-ring"
      aria-current={current ? "page" : undefined}
      title={hint}
      onClick={(event) => {
        if (onPlainClick && isPlainClick(event)) {
          event.preventDefault();
          onPlainClick();
        }
      }}
    >
      <span className={cn("dk-rl-d", on && "dk-on")} />
      <span className="dk-rl-tx">
        <span className="dk-rl-t">{title}</span>
        <span className={cn("dk-rl-s", tone && `dk-${tone}`)}>{sub}</span>
      </span>
    </Link>
  );
}

/**
 * One conversation on the rail: its title, and under it its agent, or what it
 * needs from the person and what an agent is doing in it ("Billing · Needs
 * your approval"). Memoized by hand: it is drawn inside a virtual list, which
 * is kept out of the React Compiler.
 */
export const RailThreadRow = memo(function RailThreadRow({
  thread,
  agentName,
  live,
  current,
  today,
  timezone,
  onOpenBeside,
}: {
  thread: AssistantThread;
  agentName: string | null;
  live: boolean;
  current: boolean;
  today: number;
  timezone: string;
  /** Opens it in Nova's margin; left out where a conversation fills the room. */
  onOpenBeside?: (thread: AssistantThread) => void;
}) {
  const t = useT();
  const state = deskThreadState(thread, { live, active: current });
  const agent = agentName ?? t("Agent unavailable");
  const busy = state === "wait" || state === "work" || state === "error";

  return (
    <RailRow
      to={conversationPath(thread.id)}
      title={thread.title || t("Untitled conversation")}
      sub={busy ? t("{0} · {1}", agent, stateLabel(t, state)) : agent}
      on={state === "wait" || state === "new"}
      tone={state === "wait" ? "warn" : state === "error" ? "bad" : undefined}
      current={current}
      hint={[agent, stateLabel(t, state), railTime(thread, today, timezone)]
        .filter(Boolean)
        .join(" · ")}
      onPlainClick={onOpenBeside ? () => onOpenBeside(thread) : undefined}
    />
  );
});
