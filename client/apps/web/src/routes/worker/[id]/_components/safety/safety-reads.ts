import { fetchWorkerInjuries, WORKER_INJURIES_KEY } from "@/lib/graphql/worker-injury";
import {
  fetchDefaultSafetyPoints,
  fetchWorkerDisciplinaryActions,
  fetchWorkerDisciplinaryLadder,
  fetchWorkerRecognitions,
  fetchWorkerSafetyEvents,
  fetchWorkerSafetyScorecard,
  WORKER_DISCIPLINARY_ACTIONS_KEY,
  WORKER_DISCIPLINARY_LADDER_KEY,
  WORKER_RECOGNITIONS_KEY,
  WORKER_SAFETY_EVENTS_KEY,
  WORKER_SAFETY_SCORECARD_KEY,
} from "@/lib/graphql/worker-safety";
import { previewPendingRefetchMs } from "@/lib/resource-documents";
import { HOUR_MS } from "@/lib/time-units";
import { queryOptions } from "@tanstack/react-query";
import type {
  InspectionResult,
  SafetyEventKind,
  SafetySeverity,
} from "@trenova/shared/types/worker-safety";
import { useWorkerList } from "../use-worker-list";

/**
 * The Safety tab's reads, one definition each, so the tab and its panes share one cache entry per
 * read and `useWorkerInvalidation(workerId, "safety")` names the same keys. The events and the
 * recognition are capped lists read whole; actions and injuries are cursor connections read a
 * page at a time.
 * Every figure (the score, the points, the open count, the
 * stair) comes from the scorecard and the ladder, never from the rows loaded so far.
 */
export function safetyScorecardQuery(workerId: string) {
  return queryOptions({
    queryKey: [WORKER_SAFETY_SCORECARD_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerSafetyScorecard(workerId, { signal }),
  });
}

export function disciplinaryLadderQuery(workerId: string) {
  return queryOptions({
    queryKey: [WORKER_DISCIPLINARY_LADDER_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerDisciplinaryLadder(workerId, { signal }),
  });
}

export type SafetyPointsFacts = {
  kind: SafetyEventKind;
  severity: SafetySeverity;
  preventable: boolean;
  inspectionResult: InspectionResult | null;
};

/** The house scale's points for what happened; the same for every worker, so kept an hour. */
export function defaultSafetyPointsQuery(facts: SafetyPointsFacts) {
  const inspectionResult = facts.kind === "Inspection" ? facts.inspectionResult : null;
  return queryOptions({
    queryKey: [
      "default-safety-points",
      facts.kind,
      facts.severity,
      facts.preventable,
      inspectionResult,
    ],
    queryFn: ({ signal }) =>
      fetchDefaultSafetyPoints(
        {
          kind: facts.kind,
          severity: facts.severity,
          preventable: facts.preventable,
          inspectionResult,
        },
        { signal },
      ),
    staleTime: HOUR_MS,
  });
}

/**
 * The worker's events, newest first, as one capped list (the server returns the newest
 * `SERVER_LIST_CAP`). Polls while a report's preview is still being drawn, and only then.
 */
export function safetyEventsQuery(workerId: string) {
  return queryOptions({
    queryKey: [WORKER_SAFETY_EVENTS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerSafetyEvents(workerId, {}, { signal }),
    refetchInterval: (query) =>
      previewPendingRefetchMs(
        query,
        (query.state.data ?? []).map((event) => event.document),
      ),
  });
}

/**
 * Only the open and under-review events, newest first, from the server: what the Open filter
 * shows once the whole list came back full, so an open event past the newest 100 is not missed.
 * Under the events key, so a write refreshes it with the list.
 */
export function openSafetyEventsQuery(workerId: string) {
  return queryOptions({
    queryKey: [WORKER_SAFETY_EVENTS_KEY, workerId, "open"],
    queryFn: ({ signal }) => fetchWorkerSafetyEvents(workerId, { openOnly: true }, { signal }),
    refetchInterval: (query) =>
      previewPendingRefetchMs(
        query,
        (query.state.data ?? []).map((event) => event.document),
      ),
  });
}

export function useDisciplinaryActionList(workerId: string) {
  return useWorkerList({
    root: WORKER_DISCIPLINARY_ACTIONS_KEY,
    workerId,
    fetchPage: ({ first, after, signal }) =>
      fetchWorkerDisciplinaryActions({ workerId, first, after }, { signal }),
  });
}

/** The worker's recognitions, newest first, as one capped list (the newest 100). */
export function recognitionsQuery(workerId: string) {
  return queryOptions({
    queryKey: [WORKER_RECOGNITIONS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerRecognitions(workerId, { signal }),
  });
}

export function useInjuryList(workerId: string, enabled: boolean) {
  return useWorkerList({
    root: WORKER_INJURIES_KEY,
    workerId,
    enabled,
    fetchPage: ({ first, after, signal }) =>
      fetchWorkerInjuries({ workerId, first, after }, { signal }),
  });
}
