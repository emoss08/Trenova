import { preloadable } from "@/lib/lazy-component";
import type { WorkerView } from "@/routes/worker/use-worker-state";

/**
 * Where the code of each view inside a tab lives, read once and shared. The tab builds each view
 * from its loader, and the page warms them: the view under the pointer or focus in the section
 * strip, and every view of the tab showing once the browser is idle.
 *
 *   const Credentials = lazyComponent(WORKER_VIEW_MODULES.credentials, (module) => module.default);
 *
 * Only loaders live here, never components. A view moved to another module moves its loader here
 * in the same change.
 */
export const WORKER_VIEW_MODULES = {
  history: preloadable(() => import("./employment/history/worker-timeline-tab")),
  checklists: preloadable(() => import("./employment/checklists/worker-checklist-tab")),
  credentials: preloadable(() => import("./qualification/credentials/worker-credentials-tab")),
  "dq-file": preloadable(() => import("./qualification/dq-file/worker-dqf-tab")),
  "drug-and-alcohol": preloadable(() => import("./qualification/drug-alcohol/worker-testing-tab")),
  training: preloadable(() => import("./development/training/worker-training-tab")),
  reviews: preloadable(() => import("./development/reviews/worker-reviews-tab")),
  "hours-of-service": preloadable(() => import("./time/hours/hours-of-service-view")),
  schedule: preloadable(() => import("./time/schedule/schedule-view")),
  "time-off": preloadable(() => import("./time/time-off/time-off-view")),
  leave: preloadable(() => import("./time/leave/leave-view")),
  "pay-profile": preloadable(() => import("./pay/pay-profile/worker-pay-tab")),
  benefits: preloadable(() => import("./pay/benefits/worker-benefits-section")),
  documents: preloadable(() => import("./files/documents/worker-documents")),
  "driver-portal": preloadable(() => import("./files/driver-portal/worker-portal-tab")),
  "agent-memory": preloadable(() => import("./files/agent-memory/worker-memories")),
} as const satisfies Record<WorkerView, () => Promise<unknown>>;

/** Starts reading a view's code; a failed read is left for the view to show when it is drawn. */
export function preloadWorkerView(view: WorkerView): void {
  WORKER_VIEW_MODULES[view]().catch(() => undefined);
}
