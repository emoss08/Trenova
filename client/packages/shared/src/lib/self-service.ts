/**
 * Self-service from the portal: policies people sign, and changes they ask to
 * make to their own record. Shared by the office and the driver app so the two
 * describe the same request the same way.
 */

/**
 * Whether a typed signature is the worker's own name. Case and surrounding
 * space do not make a different person; a different name does. The server
 * enforces this too — the client says so before the tap, not after.
 */
export function signatureMatches(signature: string, firstName: string, lastName: string): boolean {
  const normalise = (value: string) => value.trim().toLowerCase().split(/\s+/).join(" ");
  const typed = normalise(signature);
  if (!typed) return false;
  return typed === normalise(`${firstName} ${lastName}`);
}

export type PolicyStandingValue = "signed" | "outstanding" | "read" | "unread";

/**
 * Where a worker stands on a policy. A policy that only needs reading is a
 * lighter ask than one that needs a signature, and the card looks lighter for
 * it.
 */
export function policyStanding(policy: {
  acknowledgedAt: number | null | undefined;
  requiresSignature: boolean;
}): PolicyStandingValue {
  const done = Boolean(policy.acknowledgedAt);
  if (policy.requiresSignature) return done ? "signed" : "outstanding";
  return done ? "read" : "unread";
}

export const POLICY_STANDING_TONES: Record<PolicyStandingValue, { badge: string; label: string }> =
  {
    signed: {
      badge: "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 border-emerald-500/30",
      label: "Signed",
    },
    outstanding: {
      badge: "bg-amber-500/10 text-amber-700 dark:text-amber-300 border-amber-500/30",
      label: "Needs your signature",
    },
    read: {
      badge: "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 border-emerald-500/30",
      label: "Read",
    },
    unread: {
      badge: "bg-blue-500/10 text-blue-700 dark:text-blue-300 border-blue-500/30",
      label: "Please read",
    },
  };

type FieldChangeLike = {
  field: string;
  label: string;
  from: string;
  to: string;
};

/** A request's changes as lines a manager can scan. A cleared field says so. */
export function describeChanges(changes: readonly FieldChangeLike[]): string[] {
  const shown = (value: string) => (value.trim() === "" ? "(blank)" : value);
  return changes.map((change) => `${change.label}: ${shown(change.from)} → ${shown(change.to)}`);
}

const AUDIENCE_LABELS: Record<string, string> = {
  All: "Everyone",
  Employees: "Employees",
  Contractors: "Owner-operators",
};

export function policyAudienceLabel(audience: string): string {
  return AUDIENCE_LABELS[audience] ?? audience;
}

export const POLICY_AUDIENCE_ORDER = ["All", "Employees", "Contractors"] as const;

export type ChangeRequestTone = { badge: string; label: string };

const CHANGE_REQUEST_TONES: Record<string, ChangeRequestTone> = {
  Pending: {
    badge: "bg-amber-500/10 text-amber-700 dark:text-amber-300 border-amber-500/30",
    label: "Waiting on the office",
  },
  Approved: {
    badge: "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 border-emerald-500/30",
    label: "Approved",
  },
  Rejected: {
    badge: "bg-rose-500/10 text-rose-700 dark:text-rose-300 border-rose-500/30",
    label: "Not approved",
  },
  Withdrawn: {
    badge: "bg-muted text-muted-foreground border-transparent",
    label: "Withdrawn",
  },
};

export function changeRequestTone(status: string): ChangeRequestTone {
  return CHANGE_REQUEST_TONES[status] ?? CHANGE_REQUEST_TONES.Pending;
}
