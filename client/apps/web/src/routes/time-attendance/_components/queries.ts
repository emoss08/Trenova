import {
  fetchOpenTimeEntries,
  fetchTimesheets,
  OPEN_ENTRIES_KEY,
  TIMESHEETS_KEY,
} from "@/lib/graphql/timesheet";
import { ACTIVE_QUEUE_STATUSES } from "@/lib/time-attendance";
import { addRotaWeeks } from "@trenova/shared/lib/scheduling";

export const OPEN_ENTRIES_REFRESH_MS = 60_000;

/** Weeks handed over and waiting on a manager. The overview reads it; so does the queue's caption. */
export function awaitingApprovalQuery() {
  return {
    queryKey: [TIMESHEETS_KEY, "awaiting"] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      fetchTimesheets({ statuses: ["Submitted"] }, { signal }),
  };
}

/** Every sheet whose week is the one in progress, whatever state it is in. */
export function thisWeekQuery(weekStart: number) {
  return {
    queryKey: [TIMESHEETS_KEY, "week", weekStart] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      fetchTimesheets({ from: weekStart, to: addRotaWeeks(weekStart, 1) }, { signal }),
  };
}

/** Approved weeks no payroll run has picked up yet, whatever period they fall in. */
export function unpaidApprovedQuery() {
  return {
    queryKey: [TIMESHEETS_KEY, "unpaid"] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      fetchTimesheets({ statuses: ["Approved"], unexportedOnly: true }, { signal }),
  };
}

export function openEntriesQuery(teamOnly: boolean) {
  return {
    queryKey: [OPEN_ENTRIES_KEY, teamOnly] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      fetchOpenTimeEntries({ teamOnly }, { signal }),
    refetchInterval: OPEN_ENTRIES_REFRESH_MS,
  };
}

/**
 * The queue's live segments in one request. Paid weeks are history and grow
 * without bound, so they are read on their own when that view is opened.
 */
export function activeQueueQuery(args: { teamOnly: boolean; workerId: string | null }) {
  return {
    queryKey: [TIMESHEETS_KEY, "queue", "active", args.teamOnly, args.workerId] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      fetchTimesheets(
        {
          statuses: [...ACTIVE_QUEUE_STATUSES],
          teamOnly: args.teamOnly,
          workerId: args.workerId,
          limit: 500,
        },
        { signal },
      ),
  };
}

export function paidQueueQuery(args: { teamOnly: boolean; workerId: string | null }) {
  return {
    queryKey: [TIMESHEETS_KEY, "queue", "locked", args.teamOnly, args.workerId] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      fetchTimesheets(
        { statuses: ["Locked"], teamOnly: args.teamOnly, workerId: args.workerId },
        { signal },
      ),
  };
}

/** One person's sheet for the week in progress: where their hours stand against overtime. */
export function workerWeekQuery(workerId: string, weekStart: number) {
  return {
    queryKey: [TIMESHEETS_KEY, "worker-week", workerId, weekStart] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      fetchTimesheets({ workerId, from: weekStart, to: addRotaWeeks(weekStart, 1) }, { signal }),
  };
}
