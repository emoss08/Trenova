import type { ConversationSchedule } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { scheduleLine } from "./desk-schedules";

/**
 * One schedule as a row: the clock, the request, when it runs and next, and
 * pause/resume and delete. `fresh` marks one just made, which pops in.
 */
export function DeskScheduleRow({
  schedule,
  now,
  timezone,
  fresh = false,
  disabled = false,
  onToggle,
  onDelete,
}: {
  schedule: ConversationSchedule;
  now: number;
  timezone: string;
  fresh?: boolean;
  disabled?: boolean;
  onToggle: (schedule: ConversationSchedule) => void;
  onDelete: (schedule: ConversationSchedule) => void;
}) {
  const t = useT();

  return (
    <div className={cn("dk-sch-r", !schedule.enabled && "dk-off", fresh && "dk-fresh")}>
      <span className="dk-sch-i">
        <DeskIcon name="clock" size={14} />
      </span>
      <span className="dk-sch-t">
        <b>{schedule.prompt}</b>
        <em>{scheduleLine(schedule, now, timezone, t)}</em>
      </span>
      <button
        type="button"
        className="dk-ib"
        title={schedule.enabled ? t("Pause") : t("Resume")}
        aria-label={schedule.enabled ? t("Pause") : t("Resume")}
        disabled={disabled}
        onClick={() => onToggle(schedule)}
      >
        <DeskIcon name={schedule.enabled ? "pause" : "play"} size={13} />
      </button>
      <button
        type="button"
        className="dk-ib"
        title={t("Delete")}
        aria-label={t("Delete")}
        disabled={disabled}
        onClick={() => onDelete(schedule)}
      >
        <DeskIcon name="trash" size={13} />
      </button>
    </div>
  );
}

/**
 * The card a scheduled request leaves in its conversation: "Scheduled ·
 * Results will post into this conversation" over the schedule's row, or
 * "Schedule deleted" once it is gone. `schedule` is undefined while the
 * conversation's schedules are still being read, and null when this one is
 * not among them.
 */
export function DeskScheduleCard({
  schedule,
  now,
  timezone,
  disabled,
  onToggle,
  onDelete,
}: {
  schedule: ConversationSchedule | null | undefined;
  now: number;
  timezone: string;
  disabled?: boolean;
  onToggle: (schedule: ConversationSchedule) => void;
  onDelete: (schedule: ConversationSchedule) => void;
}) {
  const t = useT();
  if (schedule === undefined) {
    return null;
  }
  if (schedule === null) {
    return (
      <div className="dk-schc dk-gone">
        <DeskIcon name="clock" size={13} />
        {t("Schedule deleted")}
      </div>
    );
  }

  return (
    <div className="dk-schc">
      <div className="dk-schc-h">
        <b>{t("Scheduled")}</b>
        <span>{t("Results will post into this conversation")}</span>
      </div>
      <DeskScheduleRow
        schedule={schedule}
        now={now}
        timezone={timezone}
        disabled={disabled}
        onToggle={onToggle}
        onDelete={onDelete}
      />
    </div>
  );
}
