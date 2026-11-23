import { useApiMutation } from "@/hooks/use-api-mutation";
import { useSingleFlight } from "@/hooks/use-single-flight";
import { usePermission } from "@/hooks/use-permission";
import {
  assignRequiredWorkerTraining,
  cancelWorkerTraining,
  type WorkerTrainingRecordRow,
  type WorkerTrainingSummaryItem,
} from "@/lib/graphql/worker-training";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet } from "@trenova/shared/components/ui/empty-sheet";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { toast } from "@trenova/shared/components/ui/toast";
import { useT } from "@trenova/shared/i18n/use-t";
import { trainingHealthMeta } from "@trenova/shared/lib/training";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { ListSketch } from "../../employment/list-sketch";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { TrainingSkeletonBody } from "../development-skeleton";
import { trainingRecordsQuery, trainingSummaryQuery } from "../development-reads";
import { TrainingHistory } from "./training-history";
import { TrainingOverview } from "./training-overview";
import {
  isOpenRecord,
  TrainingSlotRow,
  type TrainingRowActions,
  type TrainingSlotPermissions,
} from "./training-slot-card";
import { useTrainingInvalidation } from "./use-training-invalidation";
import { WaiveTrainingDialog } from "./waive-training-dialog";

const CLOSED_STATUSES = new Set(["Completed", "Failed", "Expired", "Waived", "Cancelled"]);

/** The courses with an open assignment, which cannot be assigned again. */
export function openCourseIdsOf(items: readonly WorkerTrainingSummaryItem[]): Set<string> {
  return new Set(
    items
      .filter((item) => isOpenRecord(item.record as WorkerTrainingRecordRow | null))
      .map((item) => item.course.id),
  );
}

export type TrainingViewProps = {
  workerId: string;
  onAssign: (courseId: string | null) => void;
  onRecord: (item: WorkerTrainingSummaryItem) => void;
  onRead: (record: WorkerTrainingRecordRow) => void;
};

/**
 * Development › Training: the standing in one row, every course in one bounded table grouped
 * Required and Other, and earlier records folded under it.
 */
export default function WorkerTrainingTab({
  workerId,
  onAssign,
  onRecord,
  onRead,
}: TrainingViewProps) {
  const t = useT();
  const { allowed: canAssign } = usePermission(Resource.WorkerTraining, Operation.Assign);
  const { allowed: canRecord } = usePermission(Resource.WorkerTraining, Operation.Update);
  const { allowed: canWaive } = usePermission(Resource.WorkerTraining, Operation.Cancel);
  const invalidate = useTrainingInvalidation(workerId);
  const [waiving, setWaiving] = useState<WorkerTrainingRecordRow | null>(null);
  const [cancelling, setCancelling] = useState<WorkerTrainingRecordRow | null>(null);

  const summaryQuery = useQuery(trainingSummaryQuery(workerId));
  const recordsQuery = useQuery(trainingRecordsQuery(workerId));

  const assignRequired = useApiMutation({
    resourceName: "Training",
    scope: { id: `worker-training-assign-required:${workerId}` },
    mutationFn: () => assignRequiredWorkerTraining(workerId),
    onSuccess: (records: WorkerTrainingRecordRow[]) => {
      // The rows appear in the table; only an answer of nothing assigned needs saying.
      if (records.length === 0) {
        toast.info(t("Nothing to assign"), {
          description: t("Every required course is already open or current."),
        });
      }
      void invalidate();
    },
  });
  const cancel = useApiMutation({
    resourceName: "Training",
    scope: { id: `worker-training-cancel:${workerId}` },
    mutationFn: (record: WorkerTrainingRecordRow) =>
      cancelWorkerTraining({ id: record.id, version: record.version }),
    onSuccess: () => {
      setCancelling(null);
      void invalidate();
    },
  });

  const assignOnce = useSingleFlight(() => assignRequired.mutateAsync(undefined));
  const cancelOnce = useSingleFlight((record: WorkerTrainingRecordRow) =>
    cancel.mutateAsync(record),
  );

  if (summaryQuery.isPending || recordsQuery.isPending) return <TrainingSkeletonBody />;

  const failed = summaryQuery.isError ? summaryQuery : recordsQuery.isError ? recordsQuery : null;
  if (failed) {
    return (
      <ErrorState
        layout="sheet"
        error={failed.error}
        onRetry={() => {
          void summaryQuery.refetch();
          void recordsQuery.refetch();
        }}
        title={t("Couldn't load the training")}
        sketch={<ListSketch />}
      />
    );
  }

  const summary = summaryQuery.data;
  if (!summary) return <TrainingSkeletonBody />;
  const items = summary.items;
  const required = items.filter((item) => item.required);
  const other = items.filter((item) => !item.required);
  const openCourseIds = openCourseIdsOf(items);
  const gapCount = items.filter(
    (item) =>
      item.required && trainingHealthMeta(item.health).blocks && !openCourseIds.has(item.course.id),
  ).length;
  const history = (recordsQuery.data ?? []).filter(
    (record) =>
      CLOSED_STATUSES.has(record.status) &&
      !items.some((item) => (item.record as WorkerTrainingRecordRow | null)?.id === record.id),
  );

  const permissions: TrainingSlotPermissions = { canAssign, canRecord, canWaive };
  const actions: TrainingRowActions = {
    onAssign: (courseId) => onAssign(courseId),
    onRecord,
    onRead,
    onWaive: setWaiving,
    onCancel: setCancelling,
  };
  const busy = cancel.isPending || assignRequired.isPending;

  return (
    <div className="flex flex-col gap-4">
      <TrainingOverview
        summary={summary}
        canAssign={canAssign}
        gapCount={gapCount}
        assigningRequired={assignOnce.pending}
        onAssign={() => onAssign(null)}
        onAssignRequired={() => void assignOnce.run()}
      />

      <section aria-label={t("Courses")} className="overflow-hidden rounded-lg border bg-card">
        {items.length === 0 ? (
          <EmptySheet
            title={t("No courses on file")}
            description={t(
              "No course is required for this driver type and none has been assigned. Assigned and required courses are listed here.",
            )}
            sketch={<ListSketch />}
            action={
              canAssign ? (
                <Button size="sm" onClick={() => onAssign(null)}>
                  {t("Assign course")}
                </Button>
              ) : undefined
            }
          />
        ) : (
          <Table containerClassName="max-h-[30rem]">
            <TableHeader className="sticky top-0 z-10">
              <TableRow className="hover:bg-transparent">
                <TableHead>{t("Course")}</TableHead>
                <TableHead>{t("Assigned")}</TableHead>
                <TableHead>{t("Completed")}</TableHead>
                <TableHead className="text-right">{t("Valid until")}</TableHead>
                <TableHead>{t("Health")}</TableHead>
                <TableHead className="w-9">
                  <span className="sr-only">{t("Actions")}</span>
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <GroupRow label={t("Required")} count={required.length} />
              {required.length === 0 ? (
                <TableRow className="hover:bg-transparent">
                  <TableCell colSpan={6} className="text-xs text-muted-foreground">
                    {t("No courses are required for this driver type.")}
                  </TableCell>
                </TableRow>
              ) : (
                required.map((item) => (
                  <TrainingSlotRow
                    key={item.course.id}
                    item={item}
                    permissions={permissions}
                    busy={busy}
                    {...actions}
                  />
                ))
              )}
              {other.length > 0 ? (
                <>
                  <GroupRow label={t("Other")} count={other.length} />
                  {other.map((item) => (
                    <TrainingSlotRow
                      key={item.course.id}
                      item={item}
                      permissions={permissions}
                      busy={busy}
                      {...actions}
                    />
                  ))}
                </>
              ) : null}
            </TableBody>
          </Table>
        )}
        <TrainingHistory records={history} onRead={onRead} />
      </section>

      <WaiveTrainingDialog
        open={waiving !== null}
        onOpenChange={(open) => {
          if (!open) setWaiving(null);
        }}
        workerId={workerId}
        record={waiving}
      />
      <ConfirmDialog
        open={cancelling !== null}
        title={
          cancelling?.course?.name
            ? t("Cancel the {0} assignment?", cancelling.course.name)
            : t("Cancel this assignment?")
        }
        description={
          cancelling?.course?.isRequired
            ? t(
                "It leaves the driver's list in Dash and this table. It is required, so the training shows a gap until it is assigned again.",
              )
            : t("It leaves the driver's list in Dash and this table.")
        }
        confirmLabel={t("Cancel assignment")}
        cancelLabel={t("Keep it")}
        pending={cancelOnce.pending}
        onConfirm={() => {
          if (cancelling) void cancelOnce.run(cancelling);
        }}
        onOpenChange={(open) => {
          if (!open) setCancelling(null);
        }}
      />
    </div>
  );
}

function GroupRow({ label, count }: { label: string; count: number }) {
  return (
    <TableRow className="h-7 bg-sunken/60 hover:bg-sunken/60">
      <TableCell colSpan={6} className="py-1 text-xs font-medium text-foreground-subtle">
        {label}
        <span className="ml-1.5 font-normal tabular-nums">{count}</span>
      </TableCell>
    </TableRow>
  );
}
