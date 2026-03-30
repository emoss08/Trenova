import { z } from "zod";
import {
  documentContentSchema,
  documentSchema,
  documentShipmentDraftSchema,
  documentUploadSessionSchema,
} from "./document";

export const workflowReferenceSchema = z.object({
  kind: z.string(),
  workflowId: z.string(),
});

export type WorkflowReference = z.infer<typeof workflowReferenceSchema>;

export const documentOperationsDiagnosticsSchema = z.object({
  document: documentSchema,
  versions: z.array(documentSchema),
  sessions: z.array(documentUploadSessionSchema),
  content: documentContentSchema.nullable().optional(),
  shipmentDraft: documentShipmentDraftSchema.nullable().optional(),
  lastErrors: z.array(z.string()),
  workflowRefs: z.array(workflowReferenceSchema),
});

export type DocumentOperationsDiagnostics = z.infer<typeof documentOperationsDiagnosticsSchema>;
