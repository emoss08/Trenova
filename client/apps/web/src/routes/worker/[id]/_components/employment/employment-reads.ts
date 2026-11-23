import {
  fetchActiveWorkerChecklistTemplates,
  fetchWorkerChecklists,
  WORKER_CHECKLIST_TEMPLATES_KEY,
  WORKER_CHECKLISTS_KEY,
} from "@/lib/graphql/worker-checklist";
import {
  fetchWorkerEmploymentEventOptions,
  fetchWorkerEmploymentEvents,
  WORKER_EMPLOYMENT_EVENTS_KEY,
  WORKER_EMPLOYMENT_OPTIONS_KEY,
} from "@/lib/graphql/worker-employment";
import { queryOptions } from "@tanstack/react-query";

/**
 * The Employment tab's reads, one definition each, so the lifecycle route, the views and the
 * panes share one cache entry per read and the invalidation hooks name the same keys.
 */
export function employmentEventsQuery(workerId: string) {
  return queryOptions({
    queryKey: [WORKER_EMPLOYMENT_EVENTS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerEmploymentEvents(workerId, undefined, { signal }),
  });
}

export function employmentOptionsQuery(workerId: string) {
  return queryOptions({
    queryKey: [WORKER_EMPLOYMENT_OPTIONS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerEmploymentEventOptions(workerId, { signal }),
  });
}

export function workerChecklistsQuery(workerId: string) {
  return queryOptions({
    queryKey: [WORKER_CHECKLISTS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerChecklists(workerId, true, { signal }),
  });
}

export function checklistTemplatesQuery() {
  return queryOptions({
    queryKey: [WORKER_CHECKLIST_TEMPLATES_KEY],
    queryFn: ({ signal }) => fetchActiveWorkerChecklistTemplates({ signal }),
    staleTime: 5 * 60 * 1000,
  });
}
