import { ConfirmDialog } from "@/components/confirm-dialog";
import { ReasonDialog } from "@/components/reason-dialog";
import { preloadOn } from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { useSingleFlight } from "@/hooks/use-single-flight";
import {
  assignRequiredWorkerTraining,
  cancelWorkerTraining,
  waiveWorkerTraining,
  type WorkerTrainingRecordRow,
  type WorkerTrainingSummaryItem,
} from "@/lib/graphql/worker-training";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { PlusIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet } from "@trenova/shared/components/ui/empty-sheet";
import { toast } from "@trenova/shared/components/ui/toast";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { trainingHealthMeta } from "@trenova/shared/lib/training";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import {
  BOUNDED_CELL_CLASS,
  BoundedGroupRow,
  BoundedTable,
  type BoundedColumn,
  type BoundedRow,
} from "../../bounded-table";
import { ListSketch } from "../../list-sketch";
import { useWorkerInvalidation } from "../../use-worker-invalidation";
import { useWorkerList } from "../../use-worker-list";
import { AssignTrainingPane } from "../development-panes";
import { trainingRecordsList, trainingSummaryQuery } from "../development-reads";
import { TrainingSkeletonBody } from "../development-skeleton";
import { TrainingHistory } from "./training-history";
import { earlierRecords, openCourseIdsOf } from "./training-model";
import { TrainingOverview } from "./training-overview";
import {
  TrainingSlotRow,
  type TrainingRowActions,
  type TrainingSlotPermissions,
} from "./training-slot-row";

export type TrainingViewProps = {
  workerId: string;
  onAssign: (courseId: string | null) => void;
  onRecord: (item: WorkerTrainingSummaryItem) => void;
  onRead: (record: WorkerTrainingRecordRow) => void;
};

function courseColumns(t: TranslateFn): BoundedColumn[] {
  return [
    { key: "course", label: t("Course") },
    { key: "assigned", label: t("Assigned") },
    { key: "completed", label: t("Completed") },
    { key: "valid", label: t("Valid until"), align: "end" },
    { key: "health", label: t("Health") },
    { key: "actions", label: <span className="sr-only">{t("Actions")}</span>, shrink: true },
  ];
}

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
  const invalidate = useWorkerInvalidation(workerId, "training");
  const [waiving, setWaiving] = useState<WorkerTrainingRecordRow | null>(null);
  const [cancelling, setCancelling] = useState<WorkerTrainingRecordRow | null>(null);

  const summaryQuery = useQuery(trainingSummaryQuery(workerId));
  const records = useWorkerList(trainingRecordsList(workerId));

  const assignRequired = useApiMutation({
    resourceName: "Training",
    scope: { id: `worker-training-assign-required:${workerId}` },
    mutationFn: () => assignRequiredWorkerTraining(workerId),
    onSuccess: async (assigned: WorkerTrainingRecordRow[]) => {
      // The rows appear in the table; only an answer of nothing assigned needs saying.
      if (assigned.length === 0) {
        toast.info(t("Nothing to assign"), {
          description: t("Every required course is already open or current."),
        });
      }
      await invalidate();
    },
  });
  const cancel = useApiMutation({
    resourceName: "Training",
    scope: { id: `worker-training-cancel:${workerId}` },
    mutationFn: (record: WorkerTrainingRecordRow) =>
      cancelWorkerTraining({ id: record.id, version: record.version }),
    onSuccess: async () => {
      await invalidate();
      setCancelling(null);
    },
  });
  const waive = useApiMutation({
    resourceName: "Training",
    scope: { id: `worker-training-waive:${workerId}` },
    mutationFn: ({ record, reason }: { record: WorkerTrainingRecordRow; reason: string }) =>
      waiveWorkerTraining({ id: record.id, reason, version: record.version }),
    onSuccess: async () => {
      await invalidate();
      setWaiving(null);
    },
  });

  const assignOnce = useSingleFlight(() => assignRequired.mutateAsync(undefined));
  const cancelOnce = useSingleFlight((record: WorkerTrainingRecordRow) =>
    cancel.mutateAsync(record),
  );
  const waiveOnce = useSingleFlight((input: { record: WorkerTrainingRecordRow; reason: string }) =>
    waive.mutateAsync(input),
  );

  if (summaryQuery.isPending || records.isPending) return <TrainingSkeletonBody />;
  if (summaryQuery.isError) {
    return (
      <ErrorState
        layout="sheet"
        error={summaryQuery.error}
        onRetry={() => void summaryQuery.refetch()}
        title={t("The training could not be loaded")}
        sketch={<ListSketch />}
      />
    );
  }

  const summary = summaryQuery.data;
  const items = summary.items;
  const required = items.filter((item) => item.required);
  const other = items.filter((item) => !item.required);
  const openCourseIds = openCourseIdsOf(items);
  const gapCount = required.filter(
    (item) => trainingHealthMeta(item.health).blocks && !openCourseIds.has(item.course.id),
  ).length;

  const permissions: TrainingSlotPermissions = { canAssign, canRecord, canWaive };
  const actions: TrainingRowActions = {
    onAssign,
    onRecord,
    onRead,
    onWaive: setWaiving,
    onCancel: setCancelling,
  };
  const busy = cancel.isPending || assignRequired.isPending || waive.isPending;
  const columns = courseColumns(t);
  const slotRow = (item: WorkerTrainingSummaryItem): BoundedRow => ({
    key: item.course.id,
    render: (props) => (
      <TrainingSlotRow
        props={props}
        item={item}
        permissions={permissions}
        busy={busy}
        {...actions}
      />
    ),
  });
  const rows: BoundedRow[] = [
    {
      key: "group:required",
      render: (props) => (
        <BoundedGroupRow
          props={props}
          span={columns.length}
          label={t("Required")}
          count={required.length}
        />
      ),
    },
    ...(required.length === 0
      ? [
          {
            key: "none:required",
            render: (props) => (
              <tr {...props}>
                <td colSpan={columns.length} className={BOUNDED_CELL_CLASS}>
                  <span className="text-xs text-muted-foreground">
                    {t("No courses are required for this driver type.")}
                  </span>
                </td>
              </tr>
            ),
          } satisfies BoundedRow,
        ]
      : required.map(slotRow)),
    ...(other.length > 0
      ? [
          {
            key: "group:other",
            render: (props) => (
              <BoundedGroupRow
                props={props}
                span={columns.length}
                label={t("Other")}
                count={other.length}
              />
            ),
          } satisfies BoundedRow,
          ...other.map(slotRow),
        ]
      : []),
  ];

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

      {/* The courses panel has no title of its own (the standing above names it): the table,
          then the history folded under it, as the mockup draws it. */}
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
                <Button size="sm" {...preloadOn(AssignTrainingPane)} onClick={() => onAssign(null)}>
                  <PlusIcon className="size-3.5" />
                  {t("Assign course")}
                </Button>
              ) : undefined
            }
          />
        ) : (
          <BoundedTable
            label={t("Courses")}
            columns={columns}
            maxHeightClassName="max-h-120"
            estimateRowHeight={52}
            rows={rows}
          />
        )}
        <TrainingHistory
          records={earlierRecords(records.rows, items)}
          list={records}
          onRead={onRead}
        />
      </section>

      <ReasonDialog
        open={waiving !== null}
        onOpenChange={(open) => {
          if (!open) setWaiving(null);
        }}
        title={t("Waive {0}", waiving?.course?.name ?? t("this course"))}
        description={t(
          "Use this when the worker already meets the requirement another way — prior experience, an equivalent certificate, a grandfathered rule. The waiver satisfies the matrix and is kept in the audit log with your reason.",
        )}
        placeholder={t("e.g. Completed at previous carrier; certificate on file")}
        fieldDescription={t("Kept on the record and in the audit log.")}
        maxLength={255}
        confirmLabel={t("Waive course")}
        cancelLabel={t("Cancel")}
        pending={waiveOnce.pending}
        onConfirm={(reason) => {
          if (waiving) void waiveOnce.run({ record: waiving, reason });
        }}
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
