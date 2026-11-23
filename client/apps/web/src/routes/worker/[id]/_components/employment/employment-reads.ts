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
import type { WorkerEmploymentEventKind } from "@trenova/graphql/generated/graphql";

/**
 * The events that open, close, hold or settle an employment cycle: everything but transfers,
 * promotions and rate changes, which can be many. A worker has a handful of these per
 * employment, so the newest 100 always hold the current cycle and its milestones.
 */
export const EMPLOYMENT_CYCLE_KINDS: readonly WorkerEmploymentEventKind[] = [
  "Hired",
  "Rehired",
  "ProbationEnded",
  "Suspended",
  "Reinstated",
  "LeaveStarted",
  "LeaveEnded",
  "Terminated",
];

/** The most the server returns in one page. */
const MAX_PAGE = 100;

/**
 * The Employment tab's reads, one definition each, so the lifecycle route, the views and the
 * panes share one cache entry per read. Every key sits under `[root, workerId]`, the prefix the
 * area's row in `WORKER_WRITE_REACH` refreshes after a write.
 */
export function employmentCycleEventsQuery(workerId: string) {
  return queryOptions({
    queryKey: [WORKER_EMPLOYMENT_EVENTS_KEY, workerId, "cycle"],
    queryFn: async ({ signal }) => {
      const page = await fetchWorkerEmploymentEvents(
        { workerId, kinds: EMPLOYMENT_CYCLE_KINDS, first: MAX_PAGE },
        { signal },
      );
      return (page.edges ?? []).map((edge) => edge.node);
    },
  });
}

/** The newest event of one kind, or null: the server reads newest first, so one row is enough. */
export function latestEmploymentEventQuery(workerId: string, kind: WorkerEmploymentEventKind) {
  return queryOptions({
    queryKey: [WORKER_EMPLOYMENT_EVENTS_KEY, workerId, "latest", kind],
    queryFn: async ({ signal }) => {
      const page = await fetchWorkerEmploymentEvents(
        { workerId, kinds: [kind], first: 1 },
        { signal },
      );
      return page.edges?.[0]?.node ?? null;
    },
  });
}

export function employmentOptionsQuery(workerId: string) {
  return queryOptions({
    queryKey: [WORKER_EMPLOYMENT_OPTIONS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerEmploymentEventOptions(workerId, { signal }),
  });
}

/**
 * Every checklist, open first then closed newest first, in one list the server caps at the
 * newest 100 (`SERVER_LIST_CAP`). The lifecycle route and the Checklists view share it.
 */
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
