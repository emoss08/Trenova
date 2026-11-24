import type { WORKER_PORTAL_STATUS_KEY } from "./files/driver-portal/worker-portal-tab";
import type { WORKER_PAY_KEY } from "./pay/pay-profile/worker-pay-tab";
import type { BENEFIT_ENROLLMENTS_KEY, TOTAL_COMPENSATION_KEY } from "@/lib/graphql/benefits";
import type { DESK_MEMORIES_KEY } from "@/lib/graphql/desk-memories";
import type { WORKER_REVIEWS_KEY } from "@/lib/graphql/performance-review";
import type {
  PTO_BALANCE_SUMMARY_KEY,
  WORKER_PTO_ASSIGNMENTS_KEY,
  WORKER_PTO_BALANCES_KEY,
  WORKER_PTO_LEDGER_KEY,
  WORKER_PTO_REQUESTS_KEY,
} from "@/lib/graphql/pto-policy";
import type {
  AVAILABILITY_PREFERENCES_KEY,
  ROTA_KEY,
  SHIFT_ASSIGNMENTS_KEY,
} from "@/lib/graphql/scheduling";
import type { PROFILE_CHANGE_REQUESTS_KEY } from "@/lib/graphql/self-service";
import type { WORKER_CHECKLISTS_KEY } from "@/lib/graphql/worker-checklist";
import type {
  CREDENTIAL_EXPIRY_FORECAST_KEY,
  WORKER_CREDENTIAL_SUMMARY_KEY,
  WORKER_CREDENTIALS_KEY,
} from "@/lib/graphql/worker-credential";
import type {
  DRIVER_QUALIFICATION_FILE_KEY,
  OUTSTANDING_VERIFICATIONS_KEY,
} from "@/lib/graphql/worker-dqf";
import type { WORKER_DRUG_ALCOHOL_KEY } from "@/lib/graphql/worker-drug-alcohol";
import type {
  WORKER_EMPLOYMENT_EVENTS_KEY,
  WORKER_EMPLOYMENT_OPTIONS_KEY,
} from "@/lib/graphql/worker-employment";
import type { WORKER_LEAVE_KEY } from "@/lib/graphql/worker-leave";
import type {
  WORKER_DISCIPLINARY_ACTIONS_KEY,
  WORKER_DISCIPLINARY_LADDER_KEY,
  WORKER_RECOGNITIONS_KEY,
  WORKER_SAFETY_EVENTS_KEY,
  WORKER_SAFETY_SCORECARD_KEY,
} from "@/lib/graphql/worker-safety";
import type {
  WORKER_TRAINING_KEY,
  WORKER_TRAINING_SUMMARY_KEY,
} from "@/lib/graphql/worker-training";
import { worker as workerQueries } from "@/lib/queries/worker";
import type { PTO_TABLE_QUERY_KEY } from "@/routes/worker/_components/pto/use-pto-invalidation";
import { useQueryClient, type QueryKey } from "@tanstack/react-query";

/*
 * The roots of the keys each area owns. They are written out rather than imported so this
 * module, which every tab loads, does not carry every area's read module with it; each one is
 * typed as the constant its read module exports (a type-only import, erased at build), so a
 * renamed key fails the typecheck here instead of silently missing its refresh.
 */
const EMPLOYMENT_EVENTS: typeof WORKER_EMPLOYMENT_EVENTS_KEY = "worker-employment-events";
const EMPLOYMENT_OPTIONS: typeof WORKER_EMPLOYMENT_OPTIONS_KEY = "worker-employment-event-options";
const CHECKLISTS: typeof WORKER_CHECKLISTS_KEY = "worker-checklists";
const TRAINING: typeof WORKER_TRAINING_KEY = "worker-training";
const TRAINING_SUMMARY: typeof WORKER_TRAINING_SUMMARY_KEY = "worker-training-summary";
const PTO_BALANCES: typeof WORKER_PTO_BALANCES_KEY = "worker-pto-balances";
const PTO_ASSIGNMENTS: typeof WORKER_PTO_ASSIGNMENTS_KEY = "worker-pto-assignments";
const PTO_LEDGER: typeof WORKER_PTO_LEDGER_KEY = "worker-pto-ledger";
const PTO_REQUESTS: typeof WORKER_PTO_REQUESTS_KEY = "worker-pto-requests";
const PTO_BALANCE_SUMMARY: typeof PTO_BALANCE_SUMMARY_KEY = "pto-balance-summary";
const PTO_TABLE: typeof PTO_TABLE_QUERY_KEY = "worker-pto-list";
const SHIFT_ASSIGNMENTS: typeof SHIFT_ASSIGNMENTS_KEY = "shift-assignments";
const AVAILABILITY_PREFERENCES: typeof AVAILABILITY_PREFERENCES_KEY = "availability-preferences";
const ROTA: typeof ROTA_KEY = "rota";
const CREDENTIALS: typeof WORKER_CREDENTIALS_KEY = "worker-credentials";
const CREDENTIAL_SUMMARY: typeof WORKER_CREDENTIAL_SUMMARY_KEY = "worker-credential-summary";
const CREDENTIAL_FORECAST: typeof CREDENTIAL_EXPIRY_FORECAST_KEY = "credential-expiry-forecast";
const DQ_FILE: typeof DRIVER_QUALIFICATION_FILE_KEY = "driver-qualification-file";
const OUTSTANDING_VERIFICATIONS: typeof OUTSTANDING_VERIFICATIONS_KEY =
  "outstanding-employment-verifications";
const DRUG_ALCOHOL: typeof WORKER_DRUG_ALCOHOL_KEY = "worker-drug-alcohol";
const REVIEWS: typeof WORKER_REVIEWS_KEY = "worker-reviews";
const SAFETY_EVENTS: typeof WORKER_SAFETY_EVENTS_KEY = "worker-safety-events";
const SAFETY_SCORECARD: typeof WORKER_SAFETY_SCORECARD_KEY = "worker-safety-scorecard";
const DISCIPLINARY_ACTIONS: typeof WORKER_DISCIPLINARY_ACTIONS_KEY = "worker-disciplinary-actions";
const DISCIPLINARY_LADDER: typeof WORKER_DISCIPLINARY_LADDER_KEY = "worker-disciplinary-ladder";
const RECOGNITIONS: typeof WORKER_RECOGNITIONS_KEY = "worker-recognitions";
const LEAVE: typeof WORKER_LEAVE_KEY = "worker-leave";
const PAY: typeof WORKER_PAY_KEY = "worker-pay";
const BENEFIT_ENROLLMENTS: typeof BENEFIT_ENROLLMENTS_KEY = "benefit-enrollments";
const TOTAL_COMPENSATION: typeof TOTAL_COMPENSATION_KEY = "total-compensation";
const PORTAL_STATUS: typeof WORKER_PORTAL_STATUS_KEY = "worker-portal-status";
const DESK_MEMORIES: typeof DESK_MEMORIES_KEY = "desk-memories";
const PROFILE_CHANGES: typeof PROFILE_CHANGE_REQUESTS_KEY = "profile-change-requests";

/** The query roster rows are read under; a write that moves a column on the roster refreshes it. */
const ROSTER_KEY = ["worker-list"] as const;

type WriteReach = {
  /** What the area's own reads are keyed by, exactly. */
  keys: (workerId: string) => readonly QueryKey[];
  /** The write moves a figure the page's overview read carries (header, tab marks, strips). */
  overview: boolean;
  /** The write moves a field of the worker record (status, profile summary, mirrored dates). */
  record: boolean;
  /** The write moves a roster column (status, compliance, qualification, training health). */
  roster: boolean;
};

/**
 * Every area that writes from the worker page, what it owns and how far a write reaches. The
 * overview carries a headline for every area (`WorkerOverview`), so every write refreshes it; the
 * record only where the server mirrors the change onto the worker or their profile; the roster
 * where a roster column follows. One row per area; a new area adds its row here.
 */
export const WORKER_WRITE_REACH = {
  // Events move status, fleet and dispatchability, and their effects reach checklists, training
  // and the time off book.
  employment: {
    keys: (id) => [
      [EMPLOYMENT_EVENTS, id],
      [EMPLOYMENT_OPTIONS, id],
      [CHECKLISTS, id],
      [TRAINING, id],
      [TRAINING_SUMMARY, id],
      [PTO_BALANCES, id],
      [PTO_ASSIGNMENTS, id],
      [PTO_REQUESTS, id],
      [PTO_TABLE],
    ],
    overview: true,
    record: true,
    roster: true,
  },
  // Requests, decisions, policy changes, adjustments and accruals: the worker's book, the
  // organisation's balance summary, PTO table, upcoming list and chart (the roster's time off
  // views), the overview's time off line and the roster's time off column.
  "time-off": {
    keys: (id) => [
      [PTO_BALANCES, id],
      [PTO_LEDGER, id],
      [PTO_ASSIGNMENTS, id],
      [PTO_REQUESTS, id],
      [PTO_BALANCE_SUMMARY],
      [PTO_TABLE],
      workerQueries.listUpcomingPTO._def,
      workerQueries.ptoChartData._def,
    ],
    overview: true,
    record: false,
    roster: true,
  },
  // Shifts and availability: the worker's own reads, the organisation's rota, and the
  // overview's next shift.
  schedule: {
    keys: (id) => [[SHIFT_ASSIGNMENTS, id], [AVAILABILITY_PREFERENCES, id], [ROTA]],
    overview: true,
    record: false,
    roster: false,
  },
  // Open checklist count; a finished onboarding item can move qualification readiness.
  checklists: { keys: (id) => [[CHECKLISTS, id]], overview: true, record: true, roster: true },
  // Licence, medical and hazmat dates are mirrored onto the profile; the forecast is org-wide.
  credentials: {
    keys: (id) => [[CREDENTIAL_SUMMARY, id], [CREDENTIALS, id], [CREDENTIAL_FORECAST]],
    overview: true,
    record: true,
    roster: true,
  },
  // The file is derived from the evidence and decides the qualified flag.
  "dq-file": {
    keys: (id) => [[DQ_FILE, id], [OUTSTANDING_VERIFICATIONS]],
    overview: true,
    record: true,
    roster: true,
  },
  // The testing standing is mirrored onto the profile and decides assignment.
  testing: { keys: (id) => [[DRUG_ALCOHOL, id]], overview: true, record: true, roster: true },
  // Training health and the next due date are mirrored onto the profile.
  training: {
    keys: (id) => [
      [TRAINING, id],
      [TRAINING_SUMMARY, id],
    ],
    overview: true,
    record: true,
    roster: true,
  },
  // Open and last review, next review date: overview only.
  reviews: { keys: (id) => [[REVIEWS, id]], overview: true, record: false, roster: false },
  // Score and rating are mirrored onto the profile; discipline can suspend, which is an event.
  safety: {
    keys: (id) => [
      [SAFETY_EVENTS, id],
      [SAFETY_SCORECARD, id],
      [DISCIPLINARY_ACTIONS, id],
      [DISCIPLINARY_LADDER, id],
      [RECOGNITIONS, id],
      [EMPLOYMENT_EVENTS, id],
    ],
    overview: true,
    record: true,
    roster: true,
  },
  // A leave of absence changes status and shows on the employment timeline.
  leave: {
    keys: (id) => [
      [LEAVE, id],
      [EMPLOYMENT_EVENTS, id],
    ],
    overview: true,
    record: true,
    roster: true,
  },
  // The pay headline; every pay read is under one root.
  "pay-profile": { keys: () => [[PAY]], overview: true, record: false, roster: false },
  benefits: {
    keys: (id) => [
      [BENEFIT_ENROLLMENTS, id],
      [TOTAL_COMPENSATION, id],
    ],
    overview: true,
    record: false,
    roster: false,
  },
  // Portal access and last sign-in.
  portal: { keys: (id) => [[PORTAL_STATUS, id]], overview: true, record: false, roster: false },
  // The memory count.
  "agent-memory": {
    keys: (id) => [[DESK_MEMORIES, "record", "Worker", id]],
    overview: true,
    record: false,
    roster: false,
  },
  // The Profile form writes the record itself.
  profile: { keys: () => [], overview: true, record: true, roster: true },
  // The photo is on the record (the header) and the roster's rows; nothing the overview counts.
  photo: { keys: () => [], overview: false, record: true, roster: true },
  // The waiting count; an approval writes the listed fields onto the record.
  "profile-change": {
    keys: (id) => [[PROFILE_CHANGES, id]],
    overview: true,
    record: true,
    roster: true,
  },
} as const satisfies Record<string, WriteReach>;

export type WorkerWriteArea = keyof typeof WORKER_WRITE_REACH;

/** The exact keys a write in `area` refreshes, for one worker. */
export function workerWriteKeys(workerId: string, area: WorkerWriteArea): readonly QueryKey[] {
  const reach: WriteReach = WORKER_WRITE_REACH[area];
  return [
    ...reach.keys(workerId),
    ...(reach.overview ? [workerQueries.overview(workerId).queryKey] : []),
    ...(reach.record ? [workerQueries.record(workerId).queryKey] : []),
    ...(reach.roster ? [ROSTER_KEY] : []),
  ];
}

/**
 * What a write from the worker page calls once it has landed: refreshes exactly what the area's
 * row in `WORKER_WRITE_REACH` names, and settles when every active read has come back, so a
 * control can hold until the screen shows the change.
 *
 *   const invalidate = useWorkerInvalidation(workerId, "credentials");
 *   onSuccess: () => invalidate(),
 */
export function useWorkerInvalidation(
  workerId: string,
  area: WorkerWriteArea,
): () => Promise<void> {
  const queryClient = useQueryClient();
  return async () => {
    await Promise.all(
      workerWriteKeys(workerId, area).map((queryKey) =>
        queryClient.invalidateQueries({ queryKey }),
      ),
    );
  };
}
