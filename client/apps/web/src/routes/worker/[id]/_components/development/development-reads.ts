import {
  fetchWorkerPerformanceReviews,
  WORKER_REVIEWS_KEY,
} from "@/lib/graphql/performance-review";
import {
  fetchWorkerTrainingRecords,
  fetchWorkerTrainingSummary,
  WORKER_TRAINING_KEY,
  WORKER_TRAINING_SUMMARY_KEY,
} from "@/lib/graphql/worker-training";
import { previewPendingRefetchMs, type PreviewedDocument } from "@/lib/resource-documents";
import { queryOptions } from "@tanstack/react-query";
import type { ConnectionPage } from "../use-worker-list";

/**
 * The Development tab's reads, one definition each, so the views and the panes share one cache
 * entry per read and `useWorkerInvalidation` refreshes the same keys.
 */

/**
 * The worker's standing and one slot per course. Polls while a slot's certificate preview is
 * still being drawn; TanStack stops it when the view is hidden (no observer) or the browser tab is.
 */
export function trainingSummaryQuery(workerId: string) {
  return queryOptions({
    queryKey: [WORKER_TRAINING_SUMMARY_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerTrainingSummary(workerId, { signal }),
    refetchInterval: (query) =>
      previewPendingRefetchMs(
        query,
        (query.state.data?.items ?? []).map((item) => item.record?.document),
      ),
  });
}

/**
 * Every training record, open and closed, a page at a time (`useWorkerList`): the history folded
 * under the courses table reads it, and the tab reads the same pages to open a record the address
 * names. Polls, bounded, while a loaded record's certificate preview is still being drawn.
 */
export function trainingRecordsList(workerId: string) {
  return {
    root: WORKER_TRAINING_KEY,
    workerId,
    fetchPage: ({ first, after, signal }: ConnectionPage) =>
      fetchWorkerTrainingRecords(
        { workerId, includeClosed: true, first, after: after ?? undefined },
        { signal },
      ),
    refetchInterval: (rows: { document?: PreviewedDocument }[], query: object) =>
      previewPendingRefetchMs(
        query,
        rows.map((row) => row.document),
      ),
  };
}

export function workerReviewsQuery(workerId: string) {
  return queryOptions({
    queryKey: [WORKER_REVIEWS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerPerformanceReviews(workerId, { signal }),
  });
}
