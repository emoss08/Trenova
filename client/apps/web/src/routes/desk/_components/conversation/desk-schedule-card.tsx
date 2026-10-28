import type { ConversationSchedule } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { useState } from "react";
import { DeskScheduleEditDialog } from "./desk-schedule-edit-dialog";
import { ScheduleActionsMenu } from "./schedule-actions-menu";
import { scheduleLine } from "@/lib/schedule-labels";

/**
 * One schedule as a row: the clock, the request, when it runs and next, and
 * a menu to edit, pause or resume, and delete it. `fresh` marks one just
 * made, which pops in.
 */
export function DeskScheduleRow({
  schedule,
  now,
  timezone,
  fresh = false,
  disabled = false,
  onEdit,
  onToggle,
  onDelete,
}: {
  schedule: ConversationSchedule;
  now: number;
  timezone: string;
  fresh?: boolean;
  disabled?: boolean;
  onEdit: (schedule: ConversationSchedule) => void;
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
      <ScheduleActionsMenu
        schedule={schedule}
        disabled={disabled}
        onEdit={onEdit}
        onToggle={onToggle}
        onDelete={onDelete}
      />
    </div>
  );
}

/**
 * The card a scheduled request leaves in its conversation: "Scheduled ·
 * Results will post into this conversation" over the schedule's row, or
 * "Schedule deleted" once it is gone. `schedule` is undefined while the
 * conversation's schedules are still being read, and null when this one is
 * not among them. A `proposed` card is one the person approved from an
 * agent's proposal, with no request of theirs above it, so it says so.
 */
export function DeskScheduleCard({
  schedule,
  now,
  timezone,
  proposed = false,
  disabled,
  onToggle,
  onDelete,
}: {
  schedule: ConversationSchedule | null | undefined;
  now: number;
  timezone: string;
  proposed?: boolean;
  disabled?: boolean;
  onToggle: (schedule: ConversationSchedule) => void;
  onDelete: (schedule: ConversationSchedule) => void;
}) {
  const t = useT();
  const [editing, setEditing] = useState<ConversationSchedule | null>(null);
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
        <span>
          {proposed
            ? t("From a proposal you approved · results will post into this conversation")
            : t("Results will post into this conversation")}
        </span>
      </div>
      <DeskScheduleRow
        schedule={schedule}
        now={now}
        timezone={timezone}
        disabled={disabled}
        onEdit={setEditing}
        onToggle={onToggle}
        onDelete={onDelete}
      />
      <DeskScheduleEditDialog schedule={editing} onClose={() => setEditing(null)} />
    </div>
  );
}
