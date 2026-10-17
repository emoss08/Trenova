import { conversationPath } from "@/lib/conversation-path";
import type { ConversationSchedule } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Link } from "react-router";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { nextRunLabel, scheduleLine } from "@/lib/schedule-labels";
import { ScheduleActionsMenu } from "../conversation/schedule-actions-menu";

const TEXT_BUTTON_CLASS =
  "h-7 rounded-lg px-2.5 text-sm font-medium text-dsk-muted transition-colors duration-140 hover:bg-dsk-hover hover:text-dsk-fg disabled:opacity-40";

export type DeskScheduledRowActions = {
  edit: (schedule: ConversationSchedule) => void;
  toggle: (schedule: ConversationSchedule) => void;
  run: (schedule: ConversationSchedule) => void;
  confirmDelete: (id: string) => void;
  cancelDelete: () => void;
  remove: (schedule: ConversationSchedule) => void;
};

/**
 * One schedule on the Scheduled page: the request, when it runs and next,
 * the conversation its answers post into and when it last ran, with run now
 * and a menu to edit, pause or resume, and delete. Delete asks once more, in
 * the row, before it acts. `canRun` is whether the person may ask the agent
 * now; without it the schedule can still be edited, paused and deleted.
 */
export function DeskScheduledRow({
  schedule,
  now,
  timezone,
  canRun,
  changing,
  confirming,
  actions,
}: {
  schedule: ConversationSchedule;
  now: number;
  timezone: string;
  canRun: boolean;
  changing: boolean;
  confirming: boolean;
  actions: DeskScheduledRowActions;
}) {
  const t = useT();
  const title = schedule.threadTitle || t("Untitled conversation");

  return (
    <li
      className={cn("dk-sd-r", !schedule.enabled && "dk-off", confirming && "dk-cf")}
      aria-busy={changing || undefined}
    >
      <span className="dk-sch-i">
        <DeskIcon name="clock" size={14} />
      </span>
      <div className="dk-sd-t">
        <b title={schedule.prompt}>{schedule.prompt}</b>
        <em>{scheduleLine(schedule, now, timezone, t)}</em>
        <span className="dk-sd-m">
          <Link
            to={conversationPath(schedule.threadId)}
            className="dk-sd-c ui-focus-ring"
            aria-label={t("Open {0}", title)}
          >
            <DeskIcon name="chat" size={12} />
            <span className="truncate">{title}</span>
          </Link>
          {schedule.lastRunAt ? (
            <span className="dk-sd-last">
              {t("Last ran {0}", nextRunLabel(schedule.lastRunAt, now, timezone, t))}
            </span>
          ) : (
            <span className="dk-sd-last">{t("Hasn't run yet")}</span>
          )}
        </span>
      </div>
      <div className="dk-sd-acts">
        {confirming ? (
          <>
            <span className="dk-sd-q">{t("Delete this schedule?")}</span>
            <Button
              variant="bare"
              size="bare"
              className={TEXT_BUTTON_CLASS}
              onClick={actions.cancelDelete}
            >
              {t("Cancel")}
            </Button>
            <Button
              variant="bare"
              size="bare"
              className="h-7 animate-[dk-pop_180ms_var(--dk-spring)_both] rounded-lg bg-danger px-2.5 text-sm font-medium text-dsk-on-solid transition-colors duration-150 hover:bg-danger-hover"
              onClick={() => actions.remove(schedule)}
            >
              {t("Delete")}
            </Button>
          </>
        ) : (
          <>
            {canRun && (
              <Button
                variant="bare"
                size="bare"
                className={TEXT_BUTTON_CLASS}
                title={t("Ask it now, without waiting for its next run")}
                disabled={changing}
                onClick={() => actions.run(schedule)}
              >
                {t("Run now")}
              </Button>
            )}
            <ScheduleActionsMenu
              schedule={schedule}
              disabled={changing}
              onEdit={actions.edit}
              onToggle={actions.toggle}
              onDelete={(item) => actions.confirmDelete(item.id)}
            />
          </>
        )}
      </div>
    </li>
  );
}
