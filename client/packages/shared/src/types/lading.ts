import { z } from "zod";
import type { LadingCheck, LadingFile, LadingSample } from "@trenova/shared/edi/lading";
import type { CatalogMessage } from "@trenova/shared/i18n/message";

export const ladingVersionStatusSchema = z.enum([
  "Draft",
  "Certified",
  "Active",
  "Shadow",
  "Superseded",
  "Archived",
]);
export type LadingVersionStatus = z.infer<typeof ladingVersionStatusSchema>;

export const flowInstanceStatusSchema = z.enum(["Open", "Held", "Escalated", "Closed"]);
export type FlowInstanceStatus = z.infer<typeof flowInstanceStatusSchema>;

export const ladingReplayStatusSchema = z.enum(["Running", "Completed", "Failed"]);
export type LadingReplayStatus = z.infer<typeof ladingReplayStatusSchema>;

const catalogMessageSchema = z.custom<CatalogMessage>(
  (value) => typeof value === "object" && value !== null && "message" in value,
);
const ladingFileSchema = z.object({ path: z.string(), source: z.string() });
const ladingSampleSchema = z.object({ path: z.string(), content: z.string() });
const manifestSchema = z.custom<LadingCheck>((value) => typeof value === "object" && value !== null);

export const ladingReadinessSchema = z.object({
  errors: z.number(),
  failing: z.number(),
  tests: z.number(),
  uncoveredFlowClauses: z.number(),
  ready: z.boolean(),
  checkedAt: z.number().nullish(),
});
export type LadingVersionReadiness = z.infer<typeof ladingReadinessSchema>;

export const ladingVersionSchema = z.object({
  id: z.string(),
  projectId: z.string(),
  number: z.number(),
  status: ladingVersionStatusSchema,
  parentVersionId: z.string().nullish(),
  files: z.array(ladingFileSchema).nullish().transform((files): LadingFile[] => files ?? []),
  samples: z
    .array(z.object({ path: z.string(), digest: z.string(), size: z.number() }))
    .nullish()
    .transform((samples) => samples ?? []),
  manifest: manifestSchema,
  readiness: ladingReadinessSchema.nullish(),
  notes: z.string().nullish(),
  errorCount: z.number().default(0),
  warningCount: z.number().default(0),
  shadowUntil: z.number().nullish(),
  certifiedAt: z.number().nullish(),
  activatedAt: z.number().nullish(),
  supersededAt: z.number().nullish(),
  archivedAt: z.number().nullish(),
  version: z.number(),
  createdAt: z.number(),
  updatedAt: z.number(),
});
export type LadingVersion = z.infer<typeof ladingVersionSchema>;

export const ladingVersionDetailSchema = z.object({
  version: ladingVersionSchema,
  samples: z
    .array(ladingSampleSchema)
    .nullish()
    .transform((samples): LadingSample[] => samples ?? []),
});
export type LadingVersionDetail = z.infer<typeof ladingVersionDetailSchema>;

export const ladingProjectSchema = z.object({
  id: z.string(),
  ediPartnerId: z.string(),
  activeVersionId: z.string().nullish(),
  shadowVersionId: z.string().nullish(),
  settings: z
    .record(z.string(), z.unknown())
    .nullish()
    .transform((settings) => settings ?? {}),
  nextVersionNumber: z.number(),
  version: z.number(),
  createdAt: z.number(),
  updatedAt: z.number(),
  partner: z.object({ id: z.string(), code: z.string(), name: z.string() }).nullish(),
});
export type LadingProject = z.infer<typeof ladingProjectSchema>;

export const ladingVersionListSchema = z.object({
  items: z
    .array(ladingVersionSchema)
    .nullish()
    .transform((items) => items ?? []),
});

const replaySummarySchema = z.object({
  total: z.number(),
  same: z.number(),
  differ: z.number(),
  newlyRejected: z.number(),
  newlyHeld: z.number(),
  newlyDeclined: z.number(),
  failed: z.number(),
  fields: z.record(z.string(), z.number()).nullish(),
});

const differenceSchema = z.object({ path: z.string(), before: z.unknown(), after: z.unknown() });
const replayOutcomeSchema = z.object({
  kind: z.string(),
  message: z.string(),
  rule: z.string().nullish(),
});

export const ladingReplaySchema = z.object({
  id: z.string(),
  versionId: z.string(),
  againstVersionId: z.string(),
  map: z.string(),
  periodStart: z.number(),
  periodEnd: z.number(),
  status: ladingReplayStatusSchema,
  summary: replaySummarySchema.nullish(),
  items: z
    .array(
      z.object({
        messageId: z.string(),
        same: z.boolean(),
        differences: z.array(differenceSchema).nullish(),
        newOutcomes: z.array(replayOutcomeSchema).nullish(),
        goneOutcomes: z.array(replayOutcomeSchema).nullish(),
        error: z.string().nullish(),
      }),
    )
    .nullish()
    .transform((items) => items ?? []),
  error: z.string().nullish(),
  finishedAt: z.number().nullish(),
  createdAt: z.number(),
});
export type LadingReplay = z.infer<typeof ladingReplaySchema>;

export const ladingReplayListSchema = z.object({
  items: z
    .array(ladingReplaySchema)
    .nullish()
    .transform((items) => items ?? []),
});

export const ladingShadowReportSchema = z.object({
  summary: z.object({ total: z.number(), same: z.number(), failed: z.number() }).nullish(),
  results: z
    .array(
      z.object({
        id: z.string(),
        messageId: z.string(),
        transactionSet: z.string(),
        same: z.boolean(),
        differences: z.array(differenceSchema).nullish(),
        newOutcomes: z.array(replayOutcomeSchema).nullish(),
        goneOutcomes: z.array(replayOutcomeSchema).nullish(),
        error: z.string().nullish(),
        createdAt: z.number(),
      }),
    )
    .nullish()
    .transform((results) => results ?? []),
});
export type LadingShadowReport = z.infer<typeof ladingShadowReportSchema>;

const flowActionSchema = z.object({
  kind: z.string(),
  set: z.string().nullish(),
  rejected: z.boolean().nullish(),
  message: catalogMessageSchema.nullish(),
  due: z.number().nullish(),
  proposal: z.string().nullish(),
  map: z.string().nullish(),
});

export const flowInstanceDetailSchema = z.object({
  instance: z.object({
    id: z.string(),
    ediPartnerId: z.string(),
    versionId: z.string().nullish(),
    flow: z.string(),
    key: z.string(),
    status: flowInstanceStatusSchema,
    reason: catalogMessageSchema.nullish(),
    wakeAt: z.number().nullish(),
    shipmentId: z.string().nullish(),
    transferId: z.string().nullish(),
    lastEventAt: z.number(),
    heldAt: z.number().nullish(),
    resolvedAt: z.number().nullish(),
    version: z.number(),
    createdAt: z.number(),
  }),
  summary: z.object({
    key: z.string(),
    load: z.object({ exists: z.boolean(), state: z.string() }),
    awaitingResponse: z.number().nullish(),
    responseDue: z.number().nullish(),
    escalated: z.boolean(),
    declined: z.boolean(),
    held: z.boolean(),
    declineReason: z.string().nullish(),
    pending: z
      .array(
        z.object({
          id: z.string(),
          set: z.string(),
          control: z.string(),
          due: z.number(),
          resends: z.number(),
        }),
      )
      .nullish()
      .transform((pending) => pending ?? []),
  }),
  events: z
    .array(
      z.object({
        id: z.string(),
        kind: z.string(),
        actions: z
          .array(flowActionSchema)
          .nullish()
          .transform((actions) => actions ?? []),
        error: catalogMessageSchema.nullish(),
        messageId: z.string().nullish(),
        createdAt: z.number(),
      }),
    )
    .nullish()
    .transform((events) => events ?? []),
});
export type FlowInstanceDetail = z.infer<typeof flowInstanceDetailSchema>;

export type CreateLadingProjectRequest = {
  partnerId: string;
  role: "carrier" | "broker";
  sets: string[];
  them?: [string, string];
  us?: [string, string];
  group?: [string, string];
  usage?: "test" | "production";
  acknowledge?: string;
  ackWithin?: number;
};

export type SaveLadingDraftRequest = {
  files: LadingFile[];
  samples: LadingSample[];
  notes: string;
  version: number;
};

export type StartLadingReplayRequest = {
  againstVersionId?: string;
  map: string;
  from: number;
  to: number;
};
