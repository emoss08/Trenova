import { z } from "zod";

/**
 * Training health lives in its own leaf module because two things need it that
 * cannot both import each other: the training types (which need the worker's
 * driver type) and the worker profile (which caches the health for the
 * roster). Importing this from either is safe; it depends on nothing.
 */
export const workerTrainingHealthSchema = z.enum([
  "Current",
  "Scheduled",
  "DueSoon",
  "Overdue",
  "ExpiringSoon",
  "Expired",
  "Failed",
  "Missing",
]);
export type WorkerTrainingHealth = z.infer<typeof workerTrainingHealthSchema>;

export const WORKER_TRAINING_HEALTH_LABELS: Record<WorkerTrainingHealth, string> = {
  Current: "Current",
  Scheduled: "Scheduled",
  DueSoon: "Due soon",
  Overdue: "Overdue",
  ExpiringSoon: "Expiring soon",
  Expired: "Expired",
  Failed: "Failed",
  Missing: "Missing",
};
