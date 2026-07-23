import { z } from "zod";

export const workerSyncFailureSchema = z.object({
  workerId: z.string(),
  worker: z.string(),
  operation: z.string(),
  message: z.string(),
});

export const workerSyncReadinessResponseSchema = z.object({
  totalWorkers: z.number().int(),
  activeWorkers: z.number().int(),
  syncedActiveWorkers: z.number().int(),
  unsyncedActiveWorkers: z.number().int(),
  allActiveWorkersSynced: z.boolean(),
  lastCalculatedAt: z.number().int(),
});
export const workerSyncResultSchema = z.object({
  totalWorkers: z.number().int().nonnegative(),
  activeWorkers: z.number().int().nonnegative(),
  createdDrivers: z.number().int().nonnegative(),
  updatedMappings: z.number().int().nonnegative(),
  failed: z.number().int().nonnegative(),
  failures: z.array(workerSyncFailureSchema).optional(),
});

export const workerSyncSummarySchema = z.object({
  workflowId: z.string().min(1),
  runId: z.string().min(1),
  startedAt: z.number().int().nonnegative(),
  closedAt: z.number().int().nonnegative(),
  durationSeconds: z.number().int().nonnegative(),
  result: workerSyncResultSchema,
});

const logLevelEnum = z.enum(["info", "warn", "error", "success", "debug"]);

export const workerSyncLogLineSchema = z.object({
  id: z.string().min(1),
  ts: z.number().int().nonnegative(),
  level: logLevelEnum,
  source: z.string().min(1),
  message: z.string().min(1),
});

export const testSamsaraConnectionResponse = z.object({
  provider: z.string(),
  success: z.boolean(),
  checkedAt: z.number().int(),
});

export const workerSyncDriftSchema = z.object({
  workerId: z.string(),
  workerName: z.string(),
  driftType: z.string(),
  message: z.string(),
  localExternalId: z.string().optional(),
  remoteDriverId: z.string().optional(),
  detectedAt: z.number().int(),
});

export const workerSyncDriftResponseSchema = z.object({
  drifts: z.array(workerSyncDriftSchema),
  totalDrifts: z.number().int(),
  workersWithDrift: z.number().int(),
  missingMapping: z.number().int(),
  missingRemoteDriver: z.number().int(),
  mappingMismatch: z.number().int(),
  remoteDeactivated: z.number().int(),
  lastCalculatedAt: z.number().int(),
});

export const repairWorkerSyncDriftResponseSchema = z.object({
  requestedWorkers: z.number().int(),
  repairedWorkers: z.number().int(),
  failedWorkers: z.number().int(),
  failures: z.array(workerSyncFailureSchema).optional(),
});

export const repairWorkerSyncDriftRequestSchema = z.object({
  workerIds: z.array(z.string()).default([]),
});

export const syncWorkflowStartResponseSchema = z.object({
  workflowId: z.string(),
  runId: z.string(),
  taskQueue: z.string(),
  status: z.string(),
  submittedAt: z.number().int(),
});

export const syncWorkflowStatusResponseSchema = z.object({
  workflowId: z.string(),
  runId: z.string(),
  taskQueue: z.string(),
  status: z.string(),
  startedAt: z.number().int().nonnegative().optional(),
  closedAt: z.number().int().nonnegative().optional(),
  result: workerSyncResultSchema.optional(),
  error: z.string().optional(),
});

export type SyncWorkflowStatusResponse = z.infer<typeof syncWorkflowStatusResponseSchema>;
export type RepairWorkerSyncDriftRequest = z.infer<typeof repairWorkerSyncDriftRequestSchema>;
export type RepairWorkerSyncDriftResponse = z.infer<typeof repairWorkerSyncDriftResponseSchema>;
export type WorkerSyncDriftResponse = z.infer<typeof workerSyncDriftResponseSchema>;
export type workerSyncReadinessResponse = z.infer<typeof workerSyncReadinessResponseSchema>;
export type TestSamsaraConnectionResponse = z.infer<typeof testSamsaraConnectionResponse>;
export type WorkerSyncSummary = z.infer<typeof workerSyncSummarySchema>;
export type WorkerSyncLogLine = z.infer<typeof workerSyncLogLineSchema>;
export type WorkerSyncLogLevel = z.infer<typeof logLevelEnum>;
export type WorkerSyncFailure = z.infer<typeof workerSyncFailureSchema>;
export type WorkerSyncResult = z.infer<typeof workerSyncResultSchema>;
export type WorkerSyncFailureDetails = z.infer<typeof workerSyncFailureSchema>;
export type SyncWorkflowStartResponse = z.infer<typeof syncWorkflowStartResponseSchema>;
