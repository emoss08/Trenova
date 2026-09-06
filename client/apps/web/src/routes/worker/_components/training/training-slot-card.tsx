import type {
  WorkerTrainingRecordRow,
  WorkerTrainingSummaryItem,
} from "@/lib/graphql/worker-training";
import { TrainingHealthBadge } from "@trenova/shared/components/training-health-badge";
import { Button } from "@trenova/shared/components/ui/button";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { describeTrainingTiming, trainingHealthMeta } from "@trenova/shared/lib/training";
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

type TrainingSlotCardProps = {
  item: WorkerTrainingSummaryItem;
  permissions: TrainingSlotPermissions;
  busy: boolean;
  onAssign: (courseId: string) => void;
  onRecord: (item: WorkerTrainingSummaryItem) => void;
  onWaive: (record: WorkerTrainingRecordRow) => void;
  onCancel: (record: WorkerTrainingRecordRow) => void;
};

export function TrainingSlotCard({
  item,
  permissions,
  busy,
  onAssign,
  onRecord,
  onWaive,
  onCancel,
}: TrainingSlotCardProps) {
  const { course, health } = item;
  const record = item.record as WorkerTrainingRecordRow | null | undefined;
  const meta = trainingHealthMeta(health);
  const isOpen = record?.status === "Assigned" || record?.status === "InProgress";
  const needsRenewal = health === "Expired" || health === "Failed" || health === "ExpiringSoon";
  const timing = describeTrainingTiming({
    health,
    daysUntilDue: item.daysUntilDue,
    daysUntilExpiry: item.daysUntilExpiry,
  });

  return (
    <div
      data-testid={`training-slot-${course.id}`}
      className="bg-card flex flex-col gap-3 rounded-xl border p-4 transition-colors"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="truncate text-sm font-semibold">{course.name}</p>
          <p className="text-muted-foreground flex flex-wrap items-center gap-x-1.5 text-xs w-full">
            <span>{TRAINING_CATEGORY_LABELS[course.category as TrainingCategory]}</span>
            <span aria-hidden>·</span>
            <span>{TRAINING_DELIVERY_LABELS[course.delivery as TrainingDelivery]}</span>
            {course.durationMinutes > 0 ? (
              <>
                <span aria-hidden>·</span>
                <span>{course.durationMinutes} min</span>
              </>
            ) : null}
          </p>
        </div>
        <TrainingHealthBadge health={health} />
      </div>

      <div className="flex flex-col gap-0.5 text-xs">
        <p className={cn("font-medium", meta.textClass)}>{timing}</p>
        <RecordFacts record={record} />
      </div>

      <div className="mt-auto flex flex-row items-center gap-2">
        {record && isOpen ? (
          <>
            {permissions.canRecord ? (
              <Button size="sm" disabled={busy} onClick={() => onRecord(item)}>
                <ClipboardCheckIcon className="size-3.5" />
                Record result
              </Button>
            ) : null}
            {permissions.canWaive ? (
              <>
                <Button size="sm" variant="outline" disabled={busy} onClick={() => onWaive(record)}>
                  <CheckCheckIcon className="size-3.5" />
                  Waive
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  className="text-muted-foreground"
                  disabled={busy}
                  aria-label={`Cancel ${course.name} assignment`}
                  onClick={() => onCancel(record)}
                >
                  <BanIcon className="size-3.5" />
                  Cancel
                </Button>
              </>
            ) : null}
          </>
        ) : null}
        {!isOpen && permissions.canAssign ? (
          <Button
            size="sm"
            variant={needsRenewal ? "default" : "outline"}
            disabled={busy}
            onClick={() => onAssign(course.id)}
          >
            {needsRenewal ? (
              <RefreshCwIcon className="size-3.5" />
            ) : (
              <PlusIcon className="size-3.5" />
            )}
            {needsRenewal ? "Assign renewal" : "Assign"}
          </Button>
        ) : null}
        {!isOpen && permissions.canRecord && health !== "Current" ? (
          <Button size="sm" variant="ghost" disabled={busy} onClick={() => onRecord(item)}>
            <ClipboardCheckIcon className="size-3.5" />
            Record completion
          </Button>
        ) : null}
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
    <p className="text-muted-foreground flex flex-wrap items-center gap-x-1.5">
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
