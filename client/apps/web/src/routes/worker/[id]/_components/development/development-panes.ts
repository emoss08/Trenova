import { lazyWorkPane } from "@/components/work-pane";
import { WORKER_FORM_PANES } from "../worker-form-panes";

/**
 * The Development tab's panes, each read the first time it opens. The tab mounts them; the views
 * spread `preloadOn(pane)` on the controls that open them, so this module holds only loaders and
 * never a pane's code.
 */
export const AssignTrainingPane = lazyWorkPane(
  WORKER_FORM_PANES["assign-training"],
  (module) => module.AssignTrainingPane,
  { title: (_, t) => t("Assign course"), fields: 3 },
);

export const RecordTrainingResultPane = lazyWorkPane(
  WORKER_FORM_PANES["record-training-result"],
  (module) => module.RecordTrainingResultPane,
  {
    openKey: (props) => `${props.record?.id ?? ""}:${props.presetCourseId ?? ""}`,
    title: (props, t) =>
      props.record?.course?.name
        ? t("Record result: {0}", props.record.course.name)
        : t("Record result"),
  },
);

export const CoursePane = lazyWorkPane(
  () => import("./training/course-pane"),
  (module) => module.CoursePane,
  {
    isOpen: (props) => props.record !== null,
    openKey: (props) => props.record?.id,
    title: (props, t) => props.record?.course?.name ?? t("Course"),
  },
);

export const StartReviewPane = lazyWorkPane(
  WORKER_FORM_PANES["start-review"],
  (module) => module.StartReviewPane,
  { title: (_, t) => t("Start a review"), fields: 4 },
);

export const EditReviewPane = lazyWorkPane(
  WORKER_FORM_PANES["edit-review"],
  (module) => module.EditReviewPane,
  {
    isOpen: (props) => props.review !== null,
    openKey: (props) => props.review?.id,
    title: (props, t) => (props.review ? t("Edit {0}", props.review.title) : t("Edit review")),
  },
);

export const ReviewReadPane = lazyWorkPane(
  () => import("./reviews/review-read-pane"),
  (module) => module.ReviewReadPane,
  {
    isOpen: (props) => props.review !== null,
    openKey: (props) => props.review?.id,
    title: (props, t) => props.review?.title ?? t("Review"),
  },
);
