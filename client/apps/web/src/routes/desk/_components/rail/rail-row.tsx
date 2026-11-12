"use no memo";
import { deskThreadState } from "@/components/desk-chat/rail/desk-thread-state";
import { railTime, stateLabel } from "@/components/desk-chat/rail/rail-parts";
import { conversationPath } from "@/lib/conversation-path";
import { casePath } from "../attention/attention-record";
import { caseOfThread } from "../record-case/case-subject";
import type { RailCaseRow } from "./open-work";
import type { AssistantThread } from "@/types/assistant";
import { useContextMenuTrigger } from "@trenova/shared/components/context-menu/context-menu-host";
import type { ContextMenuSpec } from "@trenova/shared/components/context-menu/menu-spec";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { memo, type ReactNode } from "react";
import { Link } from "react-router";

/**
 * One two-line row of the rail, as the design draws it: a dot, a title, and a
 * quieter line under it. The dot turns blue while the row wants the person.
 * A row is a link and a click follows it, wherever the person is: a
 * conversation opens as its own page. Opening one beside the page is asked
 * for from the row's menu.
 */
export function RailRow({
  to,
  title,
  sub,
  on = false,
  tone,
  current = false,
  hint,
  menu,
}: {
  to: string;
  title: string;
  sub: ReactNode;
  on?: boolean;
  tone?: "warn" | "bad";
  current?: boolean;
  hint?: string;
  /** The row's menu, built when a right-click or a long press opens it. */
  menu?: () => ContextMenuSpec | null;
}) {
  const trigger = useContextMenuTrigger(() => menu?.() ?? null);

  return (
    <Link
      to={to}
      viewTransition
      data-glide-target
      data-slot="rail-row"
      className="relative flex w-full gap-2.5 rounded-[8px] ui-focus-ring px-2.5 py-1.5 text-left aria-[current=page]:bg-dsk-canvas aria-[current=page]:shadow-[0_0_0_1px_var(--dsk-b)]"
      aria-current={current ? "page" : undefined}
      title={hint}
      {...trigger}
    >
      <span
        data-slot="rail-dot"
        data-on={on || undefined}
        className={cn(
          "mt-1.5 size-1.5 flex-none rounded-full",
          on ? "bg-dsk-blue" : "bg-dsk-b-strong",
        )}
      />
      <span className="min-w-0 flex-1">
        <span className="block truncate text-dsk-base text-dsk-fg2">{title}</span>
        <span
          data-slot="rail-sub"
          className={cn(
            "block truncate text-xs leading-[inherit]",
            tone === "warn"
              ? "text-dsk-warn-fg"
              : tone === "bad"
                ? "text-danger-subtle-foreground"
                : "text-dsk-subtle",
          )}
        >
          {sub}
        </span>
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
  menu,
}: {
  thread: AssistantThread;
  agentName: string | null;
  live: boolean;
  current: boolean;
  today: number;
  timezone: string;
  menu?: (thread: AssistantThread) => ContextMenuSpec | null;
}) {
  const t = useT();
  const state = deskThreadState(thread, { live, active: current });
  const agent = agentName ?? t("Agent unavailable");
  const busy = state === "wait" || state === "work" || state === "error";
  // A conversation asked on a record's case is that case: its row opens the
  // case, where the conversation sits beside the record, rather than the
  // conversation alone in the margin.
  const record = caseOfThread(thread);

  return (
    <RailRow
      to={record ? casePath(record.kind, record.id) : conversationPath(thread.id)}
      title={thread.title || t("Untitled conversation")}
      sub={
        busy ? t("{0} · {1}", agent, stateLabel(t, state)) : record ? t("{0} · Case", agent) : agent
      }
      on={state === "wait" || state === "new"}
      tone={state === "wait" ? "warn" : state === "error" ? "bad" : undefined}
      current={current}
      hint={[agent, stateLabel(t, state), railTime(thread, today, timezone)]
        .filter(Boolean)
        .join(" · ")}
      menu={menu ? () => menu(thread) : undefined}
    />
  );
});

/**
 * One case on the rail: the record's name, and under it what it needs from
 * the person, or the agent the person has asked on it and what that agent is
 * doing. It opens the case, never a conversation on its own.
 */
export function RailCaseRowItem({
  row,
  agentName,
  live,
  current,
  menu,
}: {
  row: RailCaseRow;
  agentName: string | null;
  live: boolean;
  current: boolean;
  menu: () => ContextMenuSpec | null;
}) {
  const t = useT();
  const state = row.thread ? deskThreadState(row.thread, { live, active: current }) : null;
  const busy = state === "wait" || state === "work" || state === "error";
  const waiting = row.waiting || state === "wait";
  const sub = row.waiting
    ? t("Needs your approval")
    : row.thread && busy && state
      ? t("{0} · {1}", agentName ?? t("Agent unavailable"), stateLabel(t, state))
      : row.thread
        ? (agentName ?? t("Agent unavailable"))
        : row.view
          ? t("View")
          : t("Case");

  return (
    <RailRow
      to={row.to}
      title={row.title || t("Untitled conversation")}
      sub={sub}
      on={waiting || state === "new"}
      tone={waiting ? "warn" : state === "error" ? "bad" : undefined}
      current={current}
      menu={menu}
    />
  );
}
