"use no memo";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { deskThreadState } from "@/components/desk-chat/rail/desk-thread-state";
import { RailDot, railTime, stateLabel } from "@/components/desk-chat/rail/rail-parts";
import { conversationPath } from "@/lib/conversation-path";
import type { AssistantThread } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { usePendingValue } from "@trenova/shared/hooks/use-pending-value";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { memo } from "react";
import { Link } from "react-router";

const ROW_ACTION_CLASS =
  "dk-sb-a relative size-5.5 justify-center rounded-md text-dsk-subtle transition-colors duration-150 hover:bg-dsk-fg/8 hover:text-dsk-fg";

/**
 * What a row can ask of the rail. Built once by the rail and handed to every
 * row, so a row re-renders only when its own conversation or state changes.
 */
export type DeskRailRowActions = {
  setPinned: (thread: AssistantThread, pinned: boolean) => void;
  /** Deletes the conversation, asking first when its scheduled requests would go with it. */
  askDelete: (thread: AssistantThread) => void;
};

type DeskRailThreadRowProps = {
  thread: AssistantThread;
  /** The agent's name, or null when the agent is no longer available. */
  agentName: string | null;
  live: boolean;
  active: boolean;
  leaving: boolean;
  /** When the reader's day began, so the row's time says the hour today and the date before. */
  today: number;
  timezone: string;
  actions: DeskRailRowActions;
};

/**
 * One conversation on the rail, as a link to it: a dot that says what it waits
 * on, its name and, while something needs the person or is under way, a line
 * that says what in words with the agent's name. Pin and delete sit beside the
 * link rather than inside it. It is renamed from its name in the top bar.
 *
 * Memoized by hand: it is drawn inside a virtual list, which is kept out of
 * the React Compiler, and the list re-renders on every scroll frame.
 */
export const DeskRailThreadRow = memo(function DeskRailThreadRow({
  thread,
  agentName,
  live,
  active,
  leaving,
  today,
  timezone,
  actions,
}: DeskRailThreadRowProps) {
  const t = useT();
  // The pin shows its new state from the click, not the old one until the
  // conversation carries it, and a second click undoes what the first asked.
  const [pinned, setShownPinned] = usePendingValue(thread.pinned);
  const togglePin = () => {
    setShownPinned(!pinned);
    actions.setPinned(thread, !pinned);
  };
  const state = deskThreadState(thread, { live, active });
  const title = thread.title || t("Untitled conversation");
  const note = railNote(t, state, agentName);

  return (
    <div
      data-k={`c:${thread.id}`}
      className={cn("dk-sb-i dk-sb-c", note && "dk-sb-2", active && "dk-on", leaving && "dk-out")}
      title={[
        agentName ?? t("Agent unavailable"),
        stateLabel(t, state),
        railTime(thread, today, timezone),
      ]
        .filter(Boolean)
        .join(" · ")}
    >
      <Link
        to={conversationPath(thread.id)}
        className="dk-sb-l"
        aria-current={active ? "page" : undefined}
      >
        <RailDot state={state} />
        <span className="dk-sb-tx">
          <span className="dk-sb-t">{title}</span>
          {note && <span className={cn("dk-sb-n", state && `dk-s-${state}`)}>{note}</span>}
        </span>
      </Link>
      <span className="dk-sb-acts">
        <Button
          variant="bare"
          size="bare"
          className={cn(ROW_ACTION_CLASS, pinned && "text-dsk-fg [&_svg_path]:fill-current")}
          data-tip={pinned ? t("Unpin") : t("Pin")}
          aria-label={pinned ? t("Unpin conversation") : t("Pin conversation")}
          aria-pressed={pinned}
          onClick={togglePin}
        >
          <DeskIcon name="pin" size={13} />
        </Button>
        <Button
          variant="bare"
          size="bare"
          className={cn(ROW_ACTION_CLASS, "hover:text-dsk-error")}
          data-tip={t("Delete")}
          aria-label={t("Delete conversation")}
          onClick={() => actions.askDelete(thread)}
        >
          <DeskIcon name="trash" size={13} />
        </Button>
      </span>
    </div>
  );
});

/**
 * The row's second line, while the conversation needs the person or an agent is
 * at work in it: "EDI · Needs your approval", "Billing · Writing a reply".
 * Nothing for a conversation at rest, so a quiet rail stays one line a row.
 */
function railNote(
  t: ReturnType<typeof useT>,
  state: ReturnType<typeof deskThreadState>,
  agentName: string | null,
): string | null {
  if (state !== "wait" && state !== "work" && state !== "error") {
    return null;
  }
  const what = stateLabel(t, state);
  return agentName ? t("{0} · {1}", agentName, what) : what;
}
