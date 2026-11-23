import { RowActionsMenu, type RowAction } from "@/components/row-actions-menu";
import type {
  WorkerTrainingRecordRow,
  WorkerTrainingSummaryItem,
} from "@/lib/graphql/worker-training";
import {
  FileCheck02Icon,
  PlusIcon,
  RefreshCw02Icon,
  SkipForwardIcon,
  XCircleIcon,
} from "@trenova/shared/components/icons";
import { TrainingHealthBadge } from "@trenova/shared/components/training-health-badge";
import { TableCell, TableRow } from "@trenova/shared/components/ui/table";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { describeTrainingTiming } from "@trenova/shared/lib/training";
import { cn } from "@trenova/shared/lib/utils";
import {
  TRAINING_CATEGORY_LABELS,
  TRAINING_DELIVERY_LABELS,
  type TrainingCategory,
  type TrainingDelivery,
} from "@trenova/shared/types/worker-training";

export type TrainingSlotPermissions = {
  canAssign: boolean;
  canRecord: boolean;
  canWaive: boolean;
};

export type TrainingRowActions = {
  onAssign: (courseId: string) => void;
  onRecord: (item: WorkerTrainingSummaryItem) => void;
  onRead: (record: WorkerTrainingRecordRow) => void;
  onWaive: (record: WorkerTrainingRecordRow) => void;
  onCancel: (record: WorkerTrainingRecordRow) => void;
};

type TrainingSlotRowProps = TrainingRowActions & {
  item: WorkerTrainingSummaryItem;
  permissions: TrainingSlotPermissions;
  busy: boolean;
};

export function isOpenRecord(record: WorkerTrainingRecordRow | null | undefined): boolean {
  return record?.status === "Assigned" || record?.status === "InProgress";
}

/**
 * One course on the worker's file as a table row: what it is, when it was assigned, the
 * result, how long it stays valid, its health and a menu. Clicking the row opens what it
 * needs next: an open assignment takes its result, a closed record reads in the pane, a
 * course nobody assigned is assigned.
 */
export function TrainingSlotRow({
  item,
  permissions,
  busy,
  onAssign,
  onRecord,
  onRead,
  onWaive,
  onCancel,
}: TrainingSlotRowProps) {
  const t = useT();
  const { course, health } = item;
  const record = item.record as WorkerTrainingRecordRow | null | undefined;
  const open = isOpenRecord(record);
  const needsRenewal = health === "Expired" || health === "Failed" || health === "ExpiringSoon";
  const caption = [
    TRAINING_CATEGORY_LABELS[course.category as TrainingCategory],
    TRAINING_DELIVERY_LABELS[course.delivery as TrainingDelivery],
    course.durationMinutes > 0 ? t("{0} min", course.durationMinutes) : null,
  ]
    .filter(Boolean)
    .join(" · ");
  const timing = describeTrainingTiming({
    health,
    daysUntilDue: item.daysUntilDue,
    daysUntilExpiry: item.daysUntilExpiry,
  });

  const actions: RowAction[] = [];
  if (record && open) {
    if (permissions.canRecord) {
      actions.push({
        id: "record",
        label: t("Record result"),
        icon: FileCheck02Icon,
        disabled: busy,
        onSelect: () => onRecord(item),
      });
    }
    if (permissions.canWaive) {
      actions.push({
        id: "waive",
        label: t("Waive"),
        icon: SkipForwardIcon,
        disabled: busy,
        onSelect: () => onWaive(record),
      });
      actions.push({
        id: "cancel",
        label: t("Cancel assignment"),
        icon: XCircleIcon,
        disabled: busy,
        destructive: true,
        onSelect: () => onCancel(record),
      });
    }
  } else {
    if (permissions.canAssign) {
      actions.push({
        id: "assign",
        label: needsRenewal ? t("Assign renewal") : t("Assign"),
        icon: needsRenewal ? RefreshCw02Icon : PlusIcon,
        disabled: busy,
        onSelect: () => onAssign(course.id),
      });
    }
    if (permissions.canRecord && health !== "Current") {
      actions.push({
        id: "record-completion",
        label: t("Record result"),
        icon: FileCheck02Icon,
        disabled: busy,
        onSelect: () => onRecord(item),
      });
    }
  }

  const activate = () => {
    if (open && permissions.canRecord) onRecord(item);
    else if (record) onRead(record);
    else if (permissions.canAssign) onAssign(course.id);
  };
  const interactive = (open && permissions.canRecord) || Boolean(record) || permissions.canAssign;

  return (
    <TableRow
      data-testid={`training-slot-${course.id}`}
      data-health={health}
      tabIndex={interactive ? 0 : undefined}
      aria-label={interactive ? course.name : undefined}
      className={cn("align-top", interactive && "cursor-pointer ui-inset-focus-ring")}
      onClick={(event) => {
        if (!interactive || (event.target as Element).closest("button, a, [role=menu]")) return;
        activate();
      }}
      onKeyDown={(event) => {
        if (!interactive || event.target !== event.currentTarget) return;
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          activate();
        }
      }}
    >
      <TableCell className="py-2 whitespace-normal">
        <span className={cn("block text-sm font-medium", !record && "text-muted-foreground")}>
          {course.name}
        </span>
        <span className="block text-xs text-muted-foreground">{caption}</span>
      </TableCell>
      <TableCell className="py-2">
        {record ? (
          <>
            <span className="block font-mono text-xs tabular-nums">
              {formatUnixDateMedium(record.assignedAt)}
            </span>
            {record.assignedBy?.name ? (
              <span className="block text-xs text-muted-foreground">
                {t("by {0}", record.assignedBy.name)}
              </span>
            ) : null}
          </>
        ) : (
          <span className="text-xs text-foreground-subtle">{t("Not assigned")}</span>
        )}
      </TableCell>
      <TableCell className="py-2">
        <CompletedCell record={record} />
      </TableCell>
      <TableCell className="py-2 text-right">
        {open ? (
          <>
            <span className="block font-mono text-xs tabular-nums">
              {record?.dueAt ? t("Due {0}", formatUnixDateMedium(record.dueAt)) : t("No due date")}
            </span>
            <span
              className={cn(
                "block text-xs",
                health === "Overdue"
                  ? "text-danger-foreground"
                  : health === "DueSoon"
                    ? "text-warning-foreground"
                    : "text-muted-foreground",
              )}
            >
              {timing}
            </span>
          </>
        ) : record?.expiresAt ? (
          <>
            <span className="block font-mono text-xs tabular-nums">
              {formatUnixDateMedium(record.expiresAt)}
            </span>
            <span
              className={cn(
                "block text-xs",
                health === "Expired"
                  ? "text-danger-foreground"
                  : health === "ExpiringSoon"
                    ? "text-warning-foreground"
                    : "text-muted-foreground",
              )}
            >
              {timing}
            </span>
          </>
        ) : record?.completedAt ? (
          <span className="text-xs text-muted-foreground">{t("No expiry")}</span>
        ) : (
          <span className="text-xs text-foreground-subtle">—</span>
        )}
      </TableCell>
      <TableCell className="py-2 align-middle">
        <TrainingHealthBadge health={health} />
      </TableCell>
      <TableCell className="w-9 py-2 pr-1.5 align-middle">
        <RowActionsMenu label={t("Actions for {0}", course.name)} actions={actions} />
      </TableCell>
    </TableRow>
  );
}

function CompletedCell({ record }: { record: WorkerTrainingRecordRow | null | undefined }) {
  const t = useT();
  if (!record) return <span className="text-xs text-foreground-subtle">—</span>;
  if (record.status === "Waived") {
    return (
      <>
        <span className="block text-xs">{t("Waived")}</span>
        {record.recordedBy?.name ? (
          <span className="block text-xs text-muted-foreground">
            {t("by {0}", record.recordedBy.name)}
          </span>
        ) : null}
      </>
    );
  }
  if (!record.completedAt) {
    return <span className="text-xs text-foreground-subtle">{t("Not started")}</span>;
  }
  const failed = record.status === "Failed";
  return (
    <>
      <span className="block font-mono text-xs tabular-nums">
        {formatUnixDateMedium(record.completedAt)}
      </span>
      <span
        className={cn("block text-xs", failed ? "text-danger-foreground" : "text-muted-foreground")}
      >
        {record.score
          ? failed
            ? t("Score {0}% · failed", Number(Number(record.score).toFixed(0)))
            : t("Score {0}%", Number(Number(record.score).toFixed(0)))
          : t("Passed")}
      </span>
    </>
  );
}
