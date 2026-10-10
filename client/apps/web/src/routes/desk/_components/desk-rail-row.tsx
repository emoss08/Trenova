"use no memo";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { deskThreadState } from "@/components/desk-chat/rail/desk-thread-state";
import { RailDot, railTime, stateLabel } from "@/components/desk-chat/rail/rail-parts";
import type { AssistantThread } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { memo, type KeyboardEvent } from "react";
import { caseRailLabel, caseStateLabel } from "./case/case-labels";

const ROW_ACTION_CLASS =
  "dk-sb-a relative size-5.5 justify-center rounded-md text-dsk-subtle transition-colors duration-150 hover:bg-dsk-fg/8 hover:text-dsk-fg";

/**
 * What a row can ask of the rail. Built once by the rail and handed to every
 * row, so a row re-renders only when its own conversation or state changes.
 */
export type DeskRailRowActions = {
  open: (thread: AssistantThread) => void;
  togglePin: (thread: AssistantThread) => void;
  /** Asks once more before deleting: the row turns its actions into a Delete button. */
  confirmDelete: (threadId: string) => void;
  cancelDelete: () => void;
  remove: (thread: AssistantThread) => void;
};

type DeskRailThreadRowProps = {
  thread: AssistantThread;
  /** The agent's name, or null when the agent is no longer available. */
  agentName: string | null;
  live: boolean;
  active: boolean;
  confirming: boolean;
  leaving: boolean;
  now: number;
  timezone: string;
  actions: DeskRailRowActions;
};

/**
 * One conversation on the rail: a dot that says what it waits on, its name,
 * the case it is about, and pin and delete on hover. Delete asks once more,
 * inline, before it acts. It is renamed from its name in the top bar.
 *
 * Memoized by hand: it is drawn inside a virtual list, which is kept out of
 * the React Compiler, and the list re-renders on every scroll frame.
 */
export const DeskRailThreadRow = memo(function DeskRailThreadRow({
  thread,
  agentName,
  live,
  active,
  confirming,
  leaving,
  now,
  timezone,
  actions,
}: DeskRailThreadRowProps) {
  const t = useT();
  const state = deskThreadState(thread, { live, active });
  const title = thread.title || t("Untitled conversation");
  const caseLabel = thread.case ? caseRailLabel(thread.case, now, timezone, t) : "";
  const openOnKey = (event: KeyboardEvent) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      actions.open(thread);
    }
  };

  return (
    <div
      role="button"
      tabIndex={0}
      data-k={`c:${thread.id}`}
      className={cn(
        "dk-sb-i dk-sb-c",
        active && "dk-on",
        leaving && "dk-out",
        confirming && "dk-cf",
      )}
      title={[
        agentName ?? t("Agent unavailable"),
        stateLabel(t, state),
        thread.case ? caseStateLabel(thread.case, now, timezone, t) : "",
        railTime(thread, now, timezone),
      ]
        .filter(Boolean)
        .join(" · ")}
      onClick={() => actions.open(thread)}
      onKeyDown={openOnKey}
      onMouseLeave={() => confirming && actions.cancelDelete()}
    >
      <RailDot state={state} />
      <span className="dk-sb-t">{title}</span>
      {caseLabel !== "" && <span className="dk-sb-cs">{caseLabel}</span>}
      <span
        className="dk-sb-acts"
        onClick={(event) => event.stopPropagation()}
        onKeyDown={(event) => event.stopPropagation()}
      >
        {confirming ? (
          <Button
            variant="bare"
            size="bare"
            className="h-5.5 animate-[dk-pop_180ms_var(--dk-spring)_both] rounded-md bg-danger px-2 text-xs font-medium text-dsk-on-solid transition-colors duration-150 hover:bg-danger-hover"
            onClick={() => actions.remove(thread)}
          >
            {t("Delete")}
          </Button>
        ) : (
          <>
            <Button
              variant="bare"
              size="bare"
              className={cn(ROW_ACTION_CLASS, thread.pinned && "text-dsk-fg [&_svg_path]:fill-current")}
              data-tip={thread.pinned ? t("Unpin") : t("Pin")}
              aria-label={thread.pinned ? t("Unpin conversation") : t("Pin conversation")}
              aria-pressed={thread.pinned}
              onClick={() => actions.togglePin(thread)}
            >
              <DeskIcon name="pin" size={13} />
            </Button>
            <Button
              variant="bare"
              size="bare"
              className={cn(ROW_ACTION_CLASS, "hover:text-dsk-error")}
              data-tip={t("Delete")}
              aria-label={t("Delete conversation")}
              onClick={() => actions.confirmDelete(thread.id)}
            >
              <DeskIcon name="trash" size={13} />
            </Button>
          </>
        )}
      </span>
    </div>
  );
});
