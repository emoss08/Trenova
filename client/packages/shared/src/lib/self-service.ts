/**
 * Self-service from the portal: policies people sign, and changes they ask to
 * make to their own record. Shared by the office and the driver app so the two
 * describe the same request the same way.
 */

import type { BadgeVariant } from "@trenova/shared/types/badge";

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

export type StandingTone = { variant: BadgeVariant; label: string };

export const POLICY_STANDING_TONES: Record<PolicyStandingValue, StandingTone> = {
  signed: { variant: "active", label: "Signed" },
  outstanding: { variant: "warning", label: "Needs your signature" },
  read: { variant: "active", label: "Read" },
  unread: { variant: "info", label: "Please read" },
};

const STANDING_RANK: Record<PolicyStandingValue, number> = {
  outstanding: 0,
  unread: 1,
  signed: 2,
  read: 3,
};

/**
 * Policies in the order a driver wants them: what still needs doing first,
 * signatures before reading, and what is done underneath. Stable within a
 * group, so the carrier's own ordering survives.
 */
export function orderPoliciesForSigning<
  T extends { acknowledgedAt: number | null | undefined; requiresSignature: boolean },
>(policies: readonly T[]): T[] {
  return policies
    .map((policy, index) => ({ policy, index, rank: STANDING_RANK[policyStanding(policy)] }))
    .sort((a, b) => a.rank - b.rank || a.index - b.index)
    .map((row) => row.policy);
}

/** The signed share of everybody a policy applies to. Nobody in scope is 0, not a division by nothing. */
export function compliancePercent(signed: number, outstanding: number): number {
  const total = signed + outstanding;
  if (total <= 0) return 0;
  return Math.round((signed / total) * 100);
}

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

export type ChangeRequestTone = { variant: BadgeVariant; label: string };

const CHANGE_REQUEST_TONES: Record<string, ChangeRequestTone> = {
  Pending: { variant: "warning", label: "Waiting on the office" },
  Approved: { variant: "active", label: "Approved" },
  Rejected: { variant: "inactive", label: "Not approved" },
  Withdrawn: { variant: "secondary", label: "Withdrawn" },
};

export function changeRequestTone(status: string): ChangeRequestTone {
  return CHANGE_REQUEST_TONES[status] ?? CHANGE_REQUEST_TONES.Pending;
}
