import type { AssistantThread } from "@/types/assistant";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useLayoutEffect, useState, type CSSProperties, type RefObject } from "react";
import { DeskIcon } from "../desk-icons";
import type { DeskThreadState } from "./desk-thread-state";
import type { DeskShelfKey } from "./desk-threads";

/** Where the raised card behind the current row sits, in the list's own coordinates. */
export type RailKnob = { y: number; h: number };

/**
 * The raised card that slides to whichever row is current. The row is found
 * by its `data-k`, so a list only marks its rows; the card follows when the
 * current key changes or the rows move under it.
 */
export function useRailKnob(
  listRef: RefObject<HTMLElement | null>,
  activeKey: string | null,
  layoutKey: unknown,
): RailKnob | null {
  const [knob, setKnob] = useState<RailKnob | null>(null);

  useLayoutEffect(() => {
    const root = listRef.current;
    if (!root) {
      return;
    }
    const element =
      activeKey === null
        ? null
        : root.querySelector<HTMLElement>(`[data-k="${CSS.escape(activeKey)}"]`);
    if (!element) {
      setKnob(null);
      return;
    }
    const rootBox = root.getBoundingClientRect();
    const box = element.getBoundingClientRect();
    setKnob({ y: box.top - rootBox.top + root.scrollTop, h: box.height });
  }, [activeKey, layoutKey, listRef]);

  return knob;
}

/** The card itself, drawn once at the top of the list it moves through. */
export function RailKnobCard({ knob }: { knob: RailKnob | null }) {
  return knob ? (
    <span className="dk-sb-knob" style={{ transform: `translateY(${knob.y}px)`, height: knob.h }} />
  ) : null;
}

/**
 * A conversation's dot: what it waits on when something does (an agent at
 * work, a change for the person to approve, a reply that failed, one not yet
 * seen), and otherwise a quiet dot, in the agent's accent when one is given.
 */
export function RailDot({ state, accent }: { state: DeskThreadState; accent?: string }) {
  return (
    <span
      className={cn("dk-sb-dot", state && `dk-s-${state}`)}
      style={accent && !state ? ({ "--dk-dot": accent } as CSSProperties) : undefined}
    >
      {state === "wait" ? (
        <DeskIcon name="info" size={13} stroke={2} />
      ) : state === "error" ? (
        <DeskIcon name="alert" size={13} stroke={2} />
      ) : (
        <i />
      )}
    </span>
  );
}

/** When a conversation was last touched, as its row's tooltip ends: the time today, the day otherwise. */
export function railTime(thread: AssistantThread, now: number, timezone: string): string {
  const at = thread.lastMessageAt > 0 ? thread.lastMessageAt : thread.createdAt;
  if (!at) return "";
  const moment = new Date(at * 1000);
  const sameDay =
    new Date(now * 1000).toLocaleDateString(undefined, { timeZone: timezone }) ===
    moment.toLocaleDateString(undefined, { timeZone: timezone });
  return sameDay
    ? moment.toLocaleTimeString(undefined, {
        hour: "numeric",
        minute: "2-digit",
        timeZone: timezone,
      })
    : moment.toLocaleDateString(undefined, { month: "short", day: "numeric", timeZone: timezone });
}

export function shelfHeading(t: TranslateFn, key: DeskShelfKey): string {
  switch (key) {
    case "pinned":
      return t("Pinned");
    case "today":
      return t("Today");
    case "yesterday":
      return t("Yesterday");
    case "week":
      return t("Previous 7 days");
    case "month":
      return t("Previous 30 days");
    case "snoozed":
      return t("Snoozed");
    case "settled":
      return t("Settled");
    default:
      return t("Older");
  }
}

export function stateLabel(t: TranslateFn, state: DeskThreadState): string {
  switch (state) {
    case "work":
      return t("Working");
    case "wait":
      return t("Needs your approval");
    case "error":
      return t("Last reply failed");
    case "new":
      return t("New reply");
    default:
      return "";
  }
}
