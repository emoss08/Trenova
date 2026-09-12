import { useT } from "@trenova/shared/i18n/use-t";
import { usePermission } from "@/hooks/use-permission";
import {
  assignRequiredWorkerTraining,
  cancelWorkerTraining,
  fetchWorkerTrainingRecords,
  fetchWorkerTrainingSummary,
  WORKER_TRAINING_KEY,
  WORKER_TRAINING_SUMMARY_KEY,
  type WorkerTrainingRecordRow,
  type WorkerTrainingSummaryItem,
} from "@/lib/graphql/worker-training";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { trainingHealthMeta } from "@trenova/shared/lib/training";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { AssignTrainingDialog } from "./training/assign-training-dialog";
import { CompleteTrainingDialog } from "./training/complete-training-dialog";
import { TrainingHistory } from "./training/training-history";
import { TrainingOverview } from "./training/training-overview";
import { TrainingSlotRow, type TrainingSlotPermissions } from "./training/training-slot-card";
import { useTrainingInvalidation } from "./training/use-training-invalidation";
import { WaiveTrainingDialog } from "./training/waive-training-dialog";

type DialogState =
  | { kind: "assign"; courseId?: string | null }
  | { kind: "complete"; record?: WorkerTrainingRecordRow | null; courseId?: string | null }
  | { kind: "waive"; record: WorkerTrainingRecordRow };

const CLOSED_STATUSES = new Set(["Completed", "Failed", "Expired", "Waived", "Cancelled"]);

export default function WorkerTrainingTab({ workerId }: { workerId: string }) {
  const t = useT();

  const { allowed: canAssign } = usePermission(Resource.WorkerTraining, Operation.Assign);
  const { allowed: canRecord } = usePermission(Resource.WorkerTraining, Operation.Update);
  const { allowed: canWaive } = usePermission(Resource.WorkerTraining, Operation.Cancel);
  const invalidate = useTrainingInvalidation(workerId);
  const [dialog, setDialog] = useState<DialogState | null>(null);

  const summaryQuery = useQuery({
    queryKey: [WORKER_TRAINING_SUMMARY_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerTrainingSummary(workerId, { signal }),
  });
  const recordsQuery = useQuery({
    queryKey: [WORKER_TRAINING_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerTrainingRecords(workerId, true, { signal }),
  });

  const summary = summaryQuery.data;
  const items = useMemo(() => summary?.items ?? [], [summary]);
  const { required, optional } = useMemo(
    () => ({
      required: items.filter((item) => item.required),
      optional: items.filter((item) => !item.required),
    }),
    [items],
  );
  const courseNames = useMemo(
    () => new Map(items.map((item) => [item.course.id, item.course.name])),
    [items],
  );
  const openCourseIds = useMemo(
    () =>
      new Set(
        items
          .filter(
            (item) => item.record?.status === "Assigned" || item.record?.status === "InProgress",
          )
          .map((item) => item.course.id),
      ),
    [items],
  );
  const gapCount = useMemo(
    () =>
      items.filter(
        (item) =>
          item.required &&
          trainingHealthMeta(item.health).blocks &&
          !openCourseIds.has(item.course.id),
      ).length,
    [items, openCourseIds],
  );
  const history = useMemo(
    () => (recordsQuery.data ?? []).filter((record) => CLOSED_STATUSES.has(record.status)),
    [recordsQuery.data],
  );

  const assignRequired = useMutation({
    mutationFn: () => assignRequiredWorkerTraining(workerId),
    onSuccess: (records) => {
      if (records.length === 0) {
        toast.info(t("Nothing to assign"), {
          description: t("Every required course is already open or current."),
        });
      } else {
        const names = records
          .map((record) => record.course?.name ?? courseNames.get(record.courseId) ?? "Course")
          .join(", ");
        toast.success(`${records.length} course${records.length === 1 ? "" : "s"} assigned`, {
          description: names,
        });
      }
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error(t("Could not assign training"), { description: error.message }),
  });

  const cancel = useMutation({
    mutationFn: (record: WorkerTrainingRecordRow) =>
      cancelWorkerTraining({ id: record.id, version: record.version }),
    onSuccess: (record) => {
      toast.success(`${record.course?.name ?? "Assignment"} cancelled`);
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error(t("Could not cancel assignment"), { description: error.message }),
  });

  const permissions = useMemo<TrainingSlotPermissions>(
    () => ({ canAssign, canRecord, canWaive }),
    [canAssign, canRecord, canWaive],
  );

  const openRecord = useCallback((item: WorkerTrainingSummaryItem) => {
    const record = item.record as WorkerTrainingRecordRow | null | undefined;
    const isOpen = record?.status === "Assigned" || record?.status === "InProgress";
    setDialog({
      kind: "complete",
      record: isOpen ? record : null,
      courseId: item.course.id,
    });
  }, []);

  if (summaryQuery.isLoading || recordsQuery.isLoading) {
    return (
      <div className="flex flex-col gap-4">
        <Skeleton className="h-24 w-full rounded-lg" />
        <Skeleton className="h-40 w-full rounded-lg" />
      </div>
    );
  }

  if (!summary) {
    return (
      <p className="text-muted-foreground text-sm">
        {t("Training could not be loaded. Try again in a moment.")}
      </p>
    );
  }

  const busy = cancel.isPending || assignRequired.isPending;
  const cardProps = {
    permissions,
    busy,
    onAssign: (courseId: string) => setDialog({ kind: "assign", courseId }),
    onRecord: openRecord,
    onWaive: (record: WorkerTrainingRecordRow) => setDialog({ kind: "waive", record }),
    onCancel: (record: WorkerTrainingRecordRow) => cancel.mutate(record),
  };

  return (
    <div className="flex flex-col gap-5">
      <TrainingOverview
        summary={summary}
        canAssign={canAssign}
        gapCount={gapCount}
        assigningRequired={assignRequired.isPending}
        onAssign={() => setDialog({ kind: "assign" })}
        onAssignRequired={() => assignRequired.mutate()}
      />

      <TrainingSection
        title={t("Required")}
        hint={t("Every course this worker must complete for their driver type.")}
        items={required}
        empty={t("No courses are required for this driver type.")}
        {...cardProps}
      />

      {optional.length > 0 ? (
        <TrainingSection
          title={t("Other courses")}
          hint={t("Optional training assigned to or completed by this worker.")}
          items={optional}
          {...cardProps}
        />
      ) : null}

      <TrainingHistory records={history} />

      <AssignTrainingDialog
        open={dialog?.kind === "assign"}
        onOpenChange={(open) => {
          if (!open) setDialog(null);
        }}
        workerId={workerId}
        courseId={dialog?.kind === "assign" ? dialog.courseId : null}
        openCourseIds={openCourseIds}
      />
      <CompleteTrainingDialog
        open={dialog?.kind === "complete"}
        onOpenChange={(open) => {
          if (!open) setDialog(null);
        }}
        workerId={workerId}
        record={dialog?.kind === "complete" ? dialog.record : null}
        courseId={dialog?.kind === "complete" ? dialog.courseId : null}
      />
      <WaiveTrainingDialog
        open={dialog?.kind === "waive"}
        onOpenChange={(open) => {
          if (!open) setDialog(null);
        }}
        workerId={workerId}
        record={dialog?.kind === "waive" ? dialog.record : null}
      />
    </div>
  );
}

type TrainingSectionProps = {
  title: string;
  hint: string;
  items: WorkerTrainingSummaryItem[];
  empty?: string;
  permissions: TrainingSlotPermissions;
  busy: boolean;
  onAssign: (courseId: string) => void;
  onRecord: (item: WorkerTrainingSummaryItem) => void;
  onWaive: (record: WorkerTrainingRecordRow) => void;
  onCancel: (record: WorkerTrainingRecordRow) => void;
};

function TrainingSection({ title, hint, items, empty, ...cardProps }: TrainingSectionProps) {
  const t = useT();

  return (
    <section className="flex flex-col gap-2">
      <div className="flex items-baseline justify-between gap-3">
        <h4 className="text-muted-foreground text-[11px] font-semibold uppercase">{title}</h4>
        <p className="text-muted-foreground truncate text-xs">{hint}</p>
      </div>
      {items.length === 0 ? (
        <p className="text-muted-foreground rounded-lg border border-dashed px-3 py-4 text-center text-xs">
          {empty ?? t("Nothing on file.")}
        </p>
      ) : (
        <div className="divide-border divide-y rounded-lg border">
          {items.map((item) => (
            <TrainingSlotRow key={item.course.id} item={item} {...cardProps} />
          ))}
        </div>
      )}
    </section>
  );
}
