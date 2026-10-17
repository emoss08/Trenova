import type { ConversationSchedule } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { useT } from "@trenova/shared/i18n/use-t";
import { DeskIcon } from "@/components/desk-chat/desk-icons";

const TRIGGER_CLASS =
  "size-6.5 rounded-lg text-dsk-subtle transition-colors duration-140 hover:bg-dsk-hover hover:text-dsk-fg data-popup-open:bg-dsk-hover data-popup-open:text-dsk-fg";

/**
 * A schedule's edit, pause or resume, and delete, behind one button, for its
 * card in a conversation and its row on the Scheduled page.
 */
export function ScheduleActionsMenu({
  schedule,
  disabled = false,
  onEdit,
  onToggle,
  onDelete,
}: {
  schedule: ConversationSchedule;
  disabled?: boolean;
  onEdit: (schedule: ConversationSchedule) => void;
  onToggle: (schedule: ConversationSchedule) => void;
  onDelete: (schedule: ConversationSchedule) => void;
}) {
  const t = useT();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        disabled={disabled}
        render={
          <Button
            variant="quiet"
            size="icon-sm"
            className={TRIGGER_CLASS}
            title={t("Schedule actions")}
            aria-label={t("Schedule actions")}
          />
        }
      >
        <DeskIcon name="moreV" size={14} />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" sideOffset={6} className="w-44">
        <DropdownMenuItem
          title={t("Edit")}
          startContent={<DeskIcon name="edit" size={14} />}
          onClick={() => onEdit(schedule)}
        />
        <DropdownMenuItem
          title={schedule.enabled ? t("Pause") : t("Resume")}
          startContent={<DeskIcon name={schedule.enabled ? "pause" : "play"} size={14} />}
          onClick={() => onToggle(schedule)}
        />
        <DropdownMenuSeparator />
        <DropdownMenuItem
          title={t("Delete")}
          color="danger"
          startContent={<DeskIcon name="trash" size={14} />}
          onClick={() => onDelete(schedule)}
        />
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
