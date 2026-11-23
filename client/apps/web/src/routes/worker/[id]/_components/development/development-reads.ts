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
import { previewPendingRefetchMs } from "@/lib/resource-documents";
import { queryOptions } from "@tanstack/react-query";

/**
 * The Development tab's reads, one definition each, so the views and the panes share one cache
 * entry per read and the invalidation hooks name the same keys.
 */
export function trainingSummaryQuery(workerId: string) {
  return queryOptions({
    queryKey: [WORKER_TRAINING_SUMMARY_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerTrainingSummary(workerId, { signal }),
    refetchInterval: (query) =>
      previewPendingRefetchMs((query.state.data?.items ?? []).map((item) => item.record?.document)),
  });
}

export function trainingRecordsQuery(workerId: string) {
  return queryOptions({
    queryKey: [WORKER_TRAINING_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerTrainingRecords(workerId, true, { signal }),
    refetchInterval: (query) =>
      previewPendingRefetchMs((query.state.data ?? []).map((record) => record.document)),
  });
}

export function workerReviewsQuery(workerId: string) {
  return queryOptions({
    queryKey: [WORKER_REVIEWS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerPerformanceReviews(workerId, { signal }),
  });
}
