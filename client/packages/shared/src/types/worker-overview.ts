import { z } from "zod";

/**
 * The one-word verdict on a worker. Blocked means they cannot be put on a load
 * today, AtRisk means something has already lapsed, Watch means something is
 * about to. Mirrors the Go `worker.Standing`.
 */
export const workerStandingSchema = z.enum(["Good", "Watch", "AtRisk", "Blocked"]);
export type WorkerStanding = z.infer<typeof workerStandingSchema>;

export const WORKER_STANDING_LABELS: Record<WorkerStanding, string> = {
  Good: "Good standing",
  Watch: "Worth watching",
  AtRisk: "At risk",
  Blocked: "Cannot be assigned",
};

export const concernSeveritySchema = z.enum(["Critical", "Warning", "Info"]);
export type ConcernSeverity = z.infer<typeof concernSeveritySchema>;

/**
 * Concern codes are a stable contract with the server. They are keys for
 * icons and grouping, never parsed for meaning, and the server owns the prose.
 */
export const CONCERN_CODES = [
  "not_employed",
  "not_assignable",
  "credentials_expired",
  "credentials_missing",
  "credentials_expiring",
  "training_overdue",
  "training_missing",
  "training_expired",
  "training_due",
  "safety_rating",
  "out_of_service",
  "active_discipline",
  "open_safety_events",
  "checklist_overdue",
  "checklist_open",
  "review_overdue",
] as const;

export type ConcernCode = (typeof CONCERN_CODES)[number];

/**
 * The worker-panel tab a concern points at. The server sends the tab id, so
 * the overview can hand the reader straight to the place that fixes it.
 */
export type ConcernTab =
  | "credentials"
  | "training"
  | "safety"
  | "checklist"
  | "reviews"
  | "timeline";
