import type { BadgeAccent, BadgeTone } from "@trenova/shared/types/badge";

/* Where a status sits in its lifecycle.
 *
 * Status metadata used to name a colour directly, so every new status was a
 * fresh guess: PTO "Requested" was purple, shipment "ReadyToInvoice" was pink,
 * and nothing made either wrong. Authors now name a phase and the tone follows,
 * which is what keeps two modules from colouring the same situation differently.
 *
 * Pick by asking what the operator should do, not by what the status is called:
 * a status nobody has to act on is `active`, one waiting on a person is
 * `awaiting`, one that has gone wrong is `attention` or `failed`.
 */
export const STATUS_PHASE_TONE = {
  /** Not started, still editable. Draft, New, Unassigned. */
  draft: "neutral",
  /** Accepted and waiting its turn. No one is blocked. */
  queued: "neutral",
  /** Work is under way and on track. In Transit, Processing, Received. */
  active: "info",
  /** Blocked on a person or an outside party. In Review, Submitted, Requested. */
  awaiting: "warning",
  /** Off the happy path and needs intervention. Delayed, Disputed, Sent back. */
  attention: "warning",
  /** Finished as intended. Completed, Posted, Billed. */
  complete: "success",
  /** Finished and no longer actionable, without having failed. Superseded, Archived. */
  closed: "neutral",
  /** Finished badly. Cancelled, Rejected, Expired, Failed. */
  failed: "danger",
} as const satisfies Record<string, BadgeTone>;

export type StatusPhase = keyof typeof STATUS_PHASE_TONE;

export function phaseTone(phase: StatusPhase): BadgeTone {
  return STATUS_PHASE_TONE[phase];
}

/** Status metadata: a lifecycle position, never a colour. */
export type BadgeAttrProps = {
  phase: StatusPhase;
  text: string;
  description?: string;
  icon?: React.ReactNode;
};

/* A few sets are classifications rather than lifecycles — W-2 versus 1099,
 * a duty status, a pricing method. Those members are mutually exclusive but
 * none is further along than another, so they carry a categorical accent
 * instead of a phase. Reach for this only when no phase honestly applies. */
export type BadgeClassAttrProps = {
  accent: BadgeAccent;
  text: string;
  description?: string;
  icon?: React.ReactNode;
};
