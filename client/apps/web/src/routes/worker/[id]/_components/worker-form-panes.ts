import { preloadable } from "@trenova/shared/lib/lazy-component";
import type { WorkerForm } from "@/routes/worker/use-worker-state";

/**
 * Where the code of each form the page's address can open lives, read once and shared. The tab
 * that owns a form builds its lazy pane from the same loader, so a preload from anywhere (the
 * header's Add menu, a deep link the page opens with, a hover on the tab's own button) is the
 * read the pane draws from:
 *
 *   const CredentialFormPane = lazyWorkPane(
 *     WORKER_FORM_PANES["add-credential"],
 *     (module) => module.CredentialFormPane,
 *   );
 *
 * Only loaders live here, never components, so the page chunk that holds this module carries no
 * pane code. A pane moved to another module moves its loader here in the same change.
 */
const recordEvent = preloadable(() => import("./employment/history/record-event-pane"));
const amendEvent = preloadable(() => import("./employment/history/amend-event-pane"));
const credentialForm = preloadable(
  () => import("./qualification/credentials/credential-form-pane"),
);
const employerForm = preloadable(() => import("./qualification/dq-file/employer-form-pane"));
const recordCollection = preloadable(
  () => import("./qualification/drug-alcohol/record-collection-pane"),
);
const recordResult = preloadable(() => import("./qualification/drug-alcohol/record-result-pane"));
const clearinghouse = preloadable(() => import("./qualification/drug-alcohol/clearinghouse-panes"));
const assignTraining = preloadable(() => import("./development/training/assign-training-pane"));
const recordTrainingResult = preloadable(
  () => import("./development/training/record-training-result-pane"),
);
const startReview = preloadable(() => import("./development/reviews/start-review-pane"));
const editReview = preloadable(() => import("./development/reviews/edit-review-pane"));
const recordSafetyEvent = preloadable(() => import("./safety/record-safety-event-pane"));
const issueAction = preloadable(() => import("./safety/issue-action-pane"));
const recognition = preloadable(() => import("./safety/recognition-pane"));
const injury = preloadable(() => import("./safety/injury-pane"));
const assignShift = preloadable(() => import("./time/schedule/assign-shift-pane"));
const requestTimeOff = preloadable(() => import("./time/time-off/request-time-off-pane"));
const changePolicy = preloadable(() => import("./time/time-off/change-policy-pane"));
const adjustBalance = preloadable(() => import("./time/time-off/adjust-balance-pane"));
const leaveCase = preloadable(() => import("./time/leave/leave-case-pane"));
const payProfile = preloadable(() => import("./pay/pay-profile/change-profile-pane"));
const enroll = preloadable(() => import("./pay/benefits/enroll-pane"));
const uploadDocument = preloadable(() => import("./files/documents/upload-document-pane"));

export const WORKER_FORM_PANES = {
  "record-event": recordEvent,
  "amend-event": amendEvent,
  "add-credential": credentialForm,
  "renew-credential": credentialForm,
  "edit-credential": credentialForm,
  "add-employer": employerForm,
  "edit-employer": employerForm,
  "record-collection": recordCollection,
  "record-test-result": recordResult,
  "log-query": clearinghouse,
  "answer-query": clearinghouse,
  "assign-training": assignTraining,
  "record-training-result": recordTrainingResult,
  "start-review": startReview,
  "edit-review": editReview,
  "record-safety-event": recordSafetyEvent,
  "issue-action": issueAction,
  "add-recognition": recognition,
  "record-injury": injury,
  "assign-shift": assignShift,
  "request-time-off": requestTimeOff,
  "change-policy": changePolicy,
  "adjust-balance": adjustBalance,
  "open-leave-case": leaveCase,
  "assign-pay-profile": payProfile,
  "enroll-benefit": enroll,
  "upload-document": uploadDocument,
} as const satisfies Record<WorkerForm, () => Promise<unknown>>;

/** Starts reading a form's pane; a failed read is left for the pane to show when it opens. */
export function preloadWorkerForm(form: WorkerForm): void {
  WORKER_FORM_PANES[form]().catch(() => undefined);
}
