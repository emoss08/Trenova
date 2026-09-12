import { useT } from "@trenova/shared/i18n/use-t";
import type {
  WorkerTrainingRecordRow,
  WorkerTrainingSummaryItem,
} from "@/lib/graphql/worker-training";
import { RowActionsMenu, type RowAction } from "@/components/row-actions-menu";
import { TrainingHealthBadge } from "@trenova/shared/components/training-health-badge";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { describeTrainingTiming } from "@trenova/shared/lib/training";
import { cn } from "@trenova/shared/lib/utils";
import {
  TRAINING_CATEGORY_LABELS,
  TRAINING_DELIVERY_LABELS,
  WORKER_TRAINING_STATUS_LABELS,
  type TrainingCategory,
  type TrainingDelivery,
  type WorkerTrainingStatus,
} from "@trenova/shared/types/worker-training";
import {
  BanIcon,
  CheckCheckIcon,
  ClipboardCheckIcon,
  FileCheckIcon,
  PlusIcon,
  RefreshCwIcon,
} from "lucide-react";

export type TrainingSlotPermissions = {
  canAssign: boolean;
  canRecord: boolean;
  canWaive: boolean;
};

type TrainingSlotRowProps = {
  item: WorkerTrainingSummaryItem;
  permissions: TrainingSlotPermissions;
  busy: boolean;
  onAssign: (courseId: string) => void;
  onRecord: (item: WorkerTrainingSummaryItem) => void;
  onWaive: (record: WorkerTrainingRecordRow) => void;
  onCancel: (record: WorkerTrainingRecordRow) => void;
};

/**
 * One course in the matrix as a row: what it is, where it stands in time,
 * what the record says, and what can be done about it. The health badge is
 * the only colour; a course that was never assigned reads as an empty line.
 */
export function TrainingSlotRow({
  item,
  permissions,
  busy,
  onAssign,
  onRecord,
  onWaive,
  onCancel,
}: TrainingSlotRowProps) {
  const t = useT();

  const { course, health } = item;
  const record = item.record as WorkerTrainingRecordRow | null | undefined;
  const isOpen = record?.status === "Assigned" || record?.status === "InProgress";
  const needsRenewal = health === "Expired" || health === "Failed" || health === "ExpiringSoon";
  const timing = describeTrainingTiming({
    health,
    daysUntilDue: item.daysUntilDue,
    daysUntilExpiry: item.daysUntilExpiry,
  });
  const caption = [
    TRAINING_CATEGORY_LABELS[course.category as TrainingCategory],
    TRAINING_DELIVERY_LABELS[course.delivery as TrainingDelivery],
    course.durationMinutes > 0 ? `${course.durationMinutes} min` : null,
  ]
    .filter(Boolean)
    .join(" · ");

  const actions: RowAction[] = [];
  if (record && isOpen) {
    if (permissions.canRecord) {
      actions.push({
        id: "record",
        label: t("Record result"),
        icon: ClipboardCheckIcon,
        disabled: busy,
        onSelect: () => onRecord(item),
      });
    }
    if (permissions.canWaive) {
      actions.push({
        id: "waive",
        label: t("Waive"),
        icon: CheckCheckIcon,
        disabled: busy,
        onSelect: () => onWaive(record),
      });
      actions.push({
        id: "cancel",
        label: `Cancel ${course.name} assignment`,
        icon: BanIcon,
        disabled: busy,
        destructive: true,
        onSelect: () => onCancel(record),
      });
    }
  } else {
    if (permissions.canAssign) {
      actions.push({
        id: "assign",
        label: needsRenewal ? "Assign renewal" : "Assign",
        icon: needsRenewal ? RefreshCwIcon : PlusIcon,
        disabled: busy,
        onSelect: () => onAssign(course.id),
      });
    }
    if (permissions.canRecord && health !== "Current") {
      actions.push({
        id: "record-completion",
        label: t("Record completion"),
        icon: ClipboardCheckIcon,
        disabled: busy,
        onSelect: () => onRecord(item),
      });
    }
  }

  return (
    <div
      data-testid={`training-slot-${course.id}`}
      data-health={health}
      className="group hover:bg-muted/30 grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-4 gap-y-1 px-3 py-2.5 transition-colors sm:grid-cols-[minmax(0,1fr)_minmax(0,14rem)_auto]"
    >
      <div className="min-w-0">
        <p className={cn("truncate text-sm font-medium", !record && "text-muted-foreground")}>
          {course.name}
        </p>
        <p className="text-muted-foreground truncate text-xs">{caption}</p>
      </div>

      <div className="col-span-2 min-w-0 text-xs sm:col-span-1">
        <p className="font-medium">{timing}</p>
        <RecordFacts record={record} />
      </div>

      <div className="col-start-2 row-start-1 flex items-center justify-end gap-2 sm:col-start-3">
        <TrainingHealthBadge health={health} />
        <RowActionsMenu label={`Actions for ${course.name}`} actions={actions} />
      </div>
    </div>
  );
}

function RecordFacts({ record }: { record: WorkerTrainingRecordRow | null | undefined }) {
  if (!record) return null;
  const facts: string[] = [];
  const status = WORKER_TRAINING_STATUS_LABELS[record.status as WorkerTrainingStatus];
  if (record.status === "Assigned" || record.status === "InProgress") {
    facts.push(`${status} ${formatUnixDate(record.assignedAt)}`);
    if (record.acknowledgedAt) {
      facts.push(`Acknowledged by the driver ${formatUnixDate(record.acknowledgedAt)}`);
    } else if (record.startedAt) {
      facts.push(`Opened ${formatUnixDate(record.startedAt)}`);
    }
  } else if (record.completedAt) {
    facts.push(`${status} ${formatUnixDate(record.completedAt)}`);
    if (record.score) facts.push(`Score ${Number(record.score).toFixed(0)}%`);
    if (record.expiresAt) facts.push(`Valid until ${formatUnixDate(record.expiresAt)}`);
  } else if (record.status === "Waived") {
    facts.push(`Waived${record.waivedReason ? ` — ${record.waivedReason}` : ""}`);
  }
  if (record.document) {
    facts.push("Certificate on file");
  }
  return (
    <p className="text-muted-foreground flex flex-wrap items-center gap-x-1.5 truncate">
      {facts.map((fact, index) => (
        <span key={fact} className="flex items-center gap-1">
          {index > 0 ? <span aria-hidden>·</span> : null}
          {fact === "Certificate on file" ? <FileCheckIcon className="size-3" /> : null}
          {fact}
        </span>
      ))}
    </p>
  );
}
