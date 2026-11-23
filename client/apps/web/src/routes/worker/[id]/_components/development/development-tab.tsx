import { Kept } from "@/components/kept";
import { useWorkPane } from "@/components/work-pane";
import { usePermission } from "@/hooks/use-permission";
import type { PerformanceReviewRow } from "@/lib/graphql/performance-review";
import {
  fetchWorkerTrainingRecord,
  WORKER_TRAINING_KEY,
  type WorkerTrainingRecordRow,
  type WorkerTrainingSummaryItem,
} from "@/lib/graphql/worker-training";
import { lazyComponent } from "@/lib/lazy-component";
import type { WorkerTabProps, WorkerTarget, WorkerViewOf } from "@/routes/worker/use-worker-state";
import { useQuery } from "@tanstack/react-query";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useRecordById } from "../use-record-by-id";
import { useWorkerFormGate } from "../use-worker-form-gate";
import { useWorkerList } from "../use-worker-list";
import { WORKER_VIEW_MODULES } from "../worker-view-modules";
import {
  AssignTrainingPane,
  CoursePane,
  EditReviewPane,
  RecordTrainingResultPane,
  ReviewReadPane,
  StartReviewPane,
} from "./development-panes";
import { trainingRecordsList, trainingSummaryQuery, workerReviewsQuery } from "./development-reads";
import { ReviewsSkeletonBody, TrainingSkeletonBody } from "./development-skeleton";
import { isOpenRecord, slotRecord, trainingRecordsById } from "./training/training-model";

const Training = lazyComponent(WORKER_VIEW_MODULES.training, (module) => module.default);
const Reviews = lazyComponent(WORKER_VIEW_MODULES.reviews, (module) => module.default);

/**
 * Development: Training or Reviews, and the panes the address opens: `assign-training`,
 * `record-training-result` (with `training` for an open assignment), `training` alone to read
 * a course, `start-review`, `edit-review` with `review`, and `review` alone to read a past one.
 */
export default function DevelopmentTab({ workerId, worker, view, open, onOpen }: WorkerTabProps) {
  const selected: WorkerViewOf<"development"> = view === "reviews" ? "reviews" : "training";
  const { closeThen } = useWorkPane();
  const { allowed: canAssign } = usePermission(Resource.WorkerTraining, Operation.Assign);
  const { allowed: canRecord } = usePermission(Resource.WorkerTraining, Operation.Update);
  const { allowed: canCreateReview } = usePermission(Resource.PerformanceReview, Operation.Create);
  const { allowed: canUpdateReview } = usePermission(Resource.PerformanceReview, Operation.Update);
  // A course a row chose for the pane it opens rides in the address, so a reload or a shared
  // link keeps it.
  const presetCourseId = open.course;

  // Only a record the address names needs the training reads here; the view reads its own.
  const summaryQuery = useQuery({
    ...trainingSummaryQuery(workerId),
    enabled: open.training !== null,
  });
  const items = summaryQuery.data?.items ?? [];
  const inSlots =
    open.training !== null && items.some((item) => slotRecord(item)?.id === open.training);
  const records = useWorkerList({
    ...trainingRecordsList(workerId),
    enabled: open.training !== null && summaryQuery.isSuccess && !inSlots,
  });
  const reviewsQuery = useQuery({
    ...workerReviewsQuery(workerId),
    enabled: open.review !== null,
  });

  const go = (
    target: Omit<Extract<WorkerTarget, { tab: "development" }>, "tab">,
    courseId: string | null = null,
  ) =>
    closeThen(() =>
      onOpen({ tab: "development", ...target, ...(courseId ? { course: courseId } : {}) }),
    );
  const closePane = () => onOpen({ tab: "development", view: selected });

  // The record the address names: in a slot, in the pages read, or past them, read by id.
  const trainingRecord = useRecordById({
    root: WORKER_TRAINING_KEY,
    workerId,
    id: open.training,
    rows: [...trainingRecordsById(items, records.rows).values()],
    rowsSettled:
      summaryQuery.isError ||
      (summaryQuery.isSuccess && (inSlots || records.isError || !records.isPending)),
    fetchRecord: fetchWorkerTrainingRecord,
  });
  const requested = trainingRecord.record;
  // Still looking for the record the address names: the result pane waits, so it opens once on
  // the assignment rather than on a bare course and then again.
  const lookingUp = trainingRecord.isPending;

  const reviewById = open.review
    ? ((reviewsQuery.data ?? []).find((review) => review.id === open.review) ?? null)
    : null;
  const opens = useWorkerFormGate(open.form, {
    "assign-training": canAssign,
    // A named record must be found; without one the result is recorded against a course.
    "record-training-result": lookingUp
      ? "waiting"
      : canRecord && (open.training === null || requested !== null),
    "start-review": canCreateReview,
    "edit-review": reviewsQuery.isPending
      ? "waiting"
      : canUpdateReview && reviewById?.status === "Draft",
  });
  const resultOpen = opens("record-training-result");
  const resultRecord = resultOpen && requested && isOpenRecord(requested) ? requested : null;
  const resultCourse = resultOpen && !resultRecord ? (requested?.courseId ?? presetCourseId) : null;
  const readRecord = open.form === null ? requested : null;

  const editing = opens("edit-review") ? reviewById : null;
  const readingReview = open.form === null && reviewById?.status === "Closed" ? reviewById : null;

  const assign = (courseId: string | null) =>
    go({ view: "training", form: "assign-training" }, courseId);
  const record = (item: WorkerTrainingSummaryItem) => {
    const slot = slotRecord(item);
    go(
      {
        view: "training",
        form: "record-training-result",
        ...(slot && isOpenRecord(slot) ? { training: slot.id } : {}),
      },
      item.course.id,
    );
  };
  const readCourse = (target: WorkerTrainingRecordRow) =>
    go({ view: "training", training: target.id });
  const retake = (target: WorkerTrainingRecordRow) =>
    go({ view: "training", form: "record-training-result" }, target.courseId);

  return (
    <>
      <Kept shown={selected === "training"}>
        <LazyComponent fallback={<TrainingSkeletonBody />}>
          <Training.Component
            workerId={workerId}
            onAssign={assign}
            onRecord={record}
            onRead={readCourse}
          />
        </LazyComponent>
      </Kept>
      <Kept shown={selected === "reviews"}>
        <LazyComponent fallback={<ReviewsSkeletonBody />}>
          <Reviews.Component
            workerId={workerId}
            workerFirstName={worker.firstName}
            focusId={open.form === null ? open.review : null}
            onStart={() => go({ view: "reviews", form: "start-review" })}
            onEdit={(review: PerformanceReviewRow) =>
              go({ view: "reviews", form: "edit-review", review: review.id })
            }
            onRead={(review: PerformanceReviewRow) => go({ view: "reviews", review: review.id })}
          />
        </LazyComponent>
      </Kept>

      <AssignTrainingPane.Component
        open={opens("assign-training")}
        presetCourseId={presetCourseId}
        workerId={workerId}
        onClose={closePane}
      />
      <RecordTrainingResultPane.Component
        open={resultOpen}
        record={resultRecord}
        presetCourseId={resultCourse}
        workerId={workerId}
        onClose={closePane}
      />
      <CoursePane.Component
        record={readRecord}
        required={
          readRecord
            ? items.some((item) => item.required && item.course.id === readRecord.courseId)
            : false
        }
        onRetake={canRecord ? retake : null}
        onClose={closePane}
      />
      <StartReviewPane.Component
        open={opens("start-review")}
        workerId={workerId}
        workerName={worker.wholeName}
        onStarted={(review) =>
          onOpen(
            canUpdateReview
              ? { tab: "development", view: "reviews", form: "edit-review", review: review.id }
              : { tab: "development", view: "reviews" },
          )
        }
        onClose={closePane}
      />
      <EditReviewPane.Component review={editing} workerId={workerId} onClose={closePane} />
      <ReviewReadPane.Component review={readingReview} onClose={closePane} />
    </>
  );
}
