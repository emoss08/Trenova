import { Kept } from "@/components/kept";
import { useWorkPane } from "@/components/work-pane";
import { usePermission } from "@/hooks/use-permission";
import type { PerformanceReviewRow } from "@/lib/graphql/performance-review";
import type {
  WorkerTrainingRecordRow,
  WorkerTrainingSummaryItem,
} from "@/lib/graphql/worker-training";
import type { WorkerTabProps, WorkerTarget, WorkerViewOf } from "@/routes/worker/use-worker-state";
import { useQuery } from "@tanstack/react-query";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { lazy, useState } from "react";
import {
  trainingRecordsQuery,
  trainingSummaryQuery,
  workerReviewsQuery,
} from "./development-reads";
import { ReviewsSkeletonBody, TrainingSkeletonBody } from "./development-skeleton";
import { EditReviewPane } from "./reviews/edit-review-pane";
import { ReviewReadPane } from "./reviews/review-read-pane";
import { StartReviewPane } from "./reviews/start-review-pane";
import { AssignTrainingPane } from "./training/assign-training-pane";
import { CoursePane } from "./training/course-pane";
import { RecordTrainingResultPane } from "./training/record-training-result-pane";
import { isOpenRecord } from "./training/training-slot-card";

const Training = lazy(() => import("./training/worker-training-tab"));
const Reviews = lazy(() => import("./reviews/worker-reviews-tab"));

/** Every training record the tab has read, by id: the summary's slots and the full list. */
export function trainingRecordsById(
  items: readonly WorkerTrainingSummaryItem[] | undefined,
  records: readonly WorkerTrainingRecordRow[] | undefined,
): Map<string, WorkerTrainingRecordRow> {
  const byId = new Map<string, WorkerTrainingRecordRow>();
  for (const record of records ?? []) byId.set(record.id, record);
  for (const item of items ?? []) {
    const record = item.record as WorkerTrainingRecordRow | null | undefined;
    if (record) byId.set(record.id, record);
  }
  return byId;
}

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
  const [now] = useState(() => Math.floor(Date.now() / 1000));
  // A course a row chose for the pane it opens rides in the address, so a reload or a shared
  // link keeps it.
  const presetCourseId = open.course;

  const summaryQuery = useQuery(trainingSummaryQuery(workerId));
  const recordsQuery = useQuery({
    ...trainingRecordsQuery(workerId),
    enabled: open.training !== null,
  });
  const reviewsQuery = useQuery({
    ...workerReviewsQuery(workerId),
    enabled: open.review !== null || open.form === "edit-review",
  });

  const go = (
    target: Omit<Extract<WorkerTarget, { tab: "development" }>, "tab">,
    courseId: string | null = null,
  ) =>
    closeThen(() =>
      onOpen({ tab: "development", ...target, ...(courseId ? { course: courseId } : {}) }),
    );
  const closePane = () => onOpen({ tab: "development", view: selected });

  const items = summaryQuery.data?.items ?? [];
  const byId = trainingRecordsById(items, recordsQuery.data);
  const requested = open.training ? (byId.get(open.training) ?? null) : null;
  const openCourseIds = new Set(
    items
      .filter((item) => isOpenRecord(item.record as WorkerTrainingRecordRow | null))
      .map((item) => item.course.id),
  );

  const resultOpen = open.form === "record-training-result" && canRecord;
  const resultRecord = resultOpen && requested && isOpenRecord(requested) ? requested : null;
  const resultCourse = resultOpen && !resultRecord ? (requested?.courseId ?? presetCourseId) : null;
  const readRecord = open.form === null ? requested : null;

  const reviews = reviewsQuery.data ?? [];
  const reviewById = open.review
    ? (reviews.find((review) => review.id === open.review) ?? null)
    : null;
  const editing =
    open.form === "edit-review" && canUpdateReview && reviewById?.status === "Draft"
      ? reviewById
      : null;
  const readingReview = open.form === null && reviewById?.status === "Closed" ? reviewById : null;

  const assign = (courseId: string | null) =>
    go({ view: "training", form: "assign-training" }, courseId);
  const record = (item: WorkerTrainingSummaryItem) => {
    const slot = item.record as WorkerTrainingRecordRow | null | undefined;
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
          <Training workerId={workerId} onAssign={assign} onRecord={record} onRead={readCourse} />
        </LazyComponent>
      </Kept>
      <Kept shown={selected === "reviews"}>
        <LazyComponent fallback={<ReviewsSkeletonBody />}>
          <Reviews
            workerId={workerId}
            workerFirstName={worker.firstName}
            focusId={open.form === null ? open.review : null}
            now={now}
            onStart={() => go({ view: "reviews", form: "start-review" })}
            onEdit={(review: PerformanceReviewRow) =>
              go({ view: "reviews", form: "edit-review", review: review.id })
            }
            onRead={(review: PerformanceReviewRow) => go({ view: "reviews", review: review.id })}
          />
        </LazyComponent>
      </Kept>

      <AssignTrainingPane
        open={open.form === "assign-training" && canAssign}
        presetCourseId={presetCourseId}
        workerId={workerId}
        openCourseIds={openCourseIds}
        onClose={closePane}
      />
      <RecordTrainingResultPane
        open={resultOpen}
        record={resultRecord}
        presetCourseId={resultCourse}
        workerId={workerId}
        onClose={closePane}
      />
      <CoursePane
        record={readRecord}
        required={
          readRecord
            ? items.some((item) => item.required && item.course.id === readRecord.courseId)
            : false
        }
        onRetake={canRecord ? retake : null}
        onClose={closePane}
      />
      <StartReviewPane
        open={open.form === "start-review" && canCreateReview}
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
      <EditReviewPane review={editing} workerId={workerId} onClose={closePane} />
      <ReviewReadPane review={readingReview} onClose={closePane} />
    </>
  );
}
