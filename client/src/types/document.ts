import { z } from "zod";

export const documentStatusSchema = z.enum([
  "Draft",
  "Active",
  "Archived",
  "Expired",
  "Pending",
  "Rejected",
  "PendingApproval",
]);

export type DocumentStatus = z.infer<typeof documentStatusSchema>;

export const documentPreviewStatusSchema = z.enum([
  "Pending",
  "Ready",
  "Failed",
  "Unsupported",
]);

export type DocumentPreviewStatus = z.infer<typeof documentPreviewStatusSchema>;

export const documentContentStatusSchema = z.enum([
  "Pending",
  "Extracting",
  "Extracted",
  "Indexed",
  "Failed",
]);

export const documentShipmentDraftStatusSchema = z.enum([
  "Unavailable",
  "Pending",
  "Ready",
  "Failed",
]);

export type DocumentContentStatus = z.infer<typeof documentContentStatusSchema>;
export type DocumentShipmentDraftStatus = z.infer<
  typeof documentShipmentDraftStatusSchema
>;

export const documentProcessingProfileSchema = z.enum([
  "none",
  "rate_confirmation_import",
]);

export type DocumentProcessingProfile = z.infer<
  typeof documentProcessingProfileSchema
>;

export const documentSchema = z.object({
  id: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  lineageId: z.string(),
  versionNumber: z.number(),
  isCurrentVersion: z.boolean(),
  fileName: z.string(),
  originalName: z.string(),
  fileSize: z.number(),
  fileType: z.string(),
  storagePath: z.string(),
  checksumSha256: z.string().nullable().optional(),
  storageVersionId: z.string().nullable().optional(),
  storageRetentionMode: z.string().nullable().optional(),
  storageRetentionUntil: z.number().nullable().optional(),
  storageLegalHold: z.boolean().optional(),
  status: documentStatusSchema,
  description: z.string().nullable().optional(),
  resourceId: z.string(),
  resourceType: z.string(),
  processingProfile: documentProcessingProfileSchema.nullish().transform((value) => value ?? "none"),
  expirationDate: z.number().nullable().optional(),
  tags: z.array(z.string()).nullable().optional(),
  isPublic: z.boolean(),
  uploadedById: z.string(),
  approvedById: z.string().nullable().optional(),
  approvedAt: z.number().nullable().optional(),
  previewStoragePath: z.string().nullable().optional(),
  previewStatus: documentPreviewStatusSchema,
  contentStatus: documentContentStatusSchema,
  contentError: z.string().nullable().optional(),
  detectedKind: z.string().nullable().optional(),
  hasExtractedText: z.boolean(),
  shipmentDraftStatus: documentShipmentDraftStatusSchema,
  documentTypeId: z.string().nullable().optional(),
  version: z.number(),
  createdAt: z.number(),
  updatedAt: z.number(),
});

export type Document = z.infer<typeof documentSchema>;

export const uploadDocumentResponseSchema = documentSchema;

export const bulkUploadDocumentResponseSchema = z.object({
  documents: z.array(documentSchema),
  errorCount: z.number(),
  successCount: z.number(),
});

export type BulkUploadDocumentResponse = z.infer<
  typeof bulkUploadDocumentResponseSchema
>;

export const downloadUrlResponseSchema = z.object({
  url: z.string(),
});

export type DownloadUrlResponse = z.infer<typeof downloadUrlResponseSchema>;

export const documentIntelligenceFieldSchema = z.object({
  label: z.string().optional(),
  value: z.unknown().optional(),
  confidence: z.number().optional(),
  excerpt: z.string().optional(),
  evidenceExcerpt: z.string().optional(),
  pageNumber: z.number().optional(),
  reviewRequired: z.boolean().optional(),
  conflict: z.boolean().optional(),
});

export const documentIntelligenceConflictSchema = z.object({
  key: z.string().optional(),
  label: z.string().optional(),
  values: z.array(z.string()).optional().default([]),
  pageNumbers: z.array(z.number()).optional().default([]),
  evidenceExcerpt: z.string().optional(),
});

export const documentIntelligenceStopSchema = z.object({
  sequence: z.number(),
  role: z.string(),
  name: z.string().optional().default(""),
  addressLine1: z.string().optional().default(""),
  addressLine2: z.string().optional().default(""),
  city: z.string().optional().default(""),
  state: z.string().optional().default(""),
  postalCode: z.string().optional().default(""),
  date: z.string().optional().default(""),
  timeWindow: z.string().optional().default(""),
  appointmentRequired: z.boolean().optional().default(false),
  pageNumber: z.number().optional(),
  evidenceExcerpt: z.string().optional(),
  confidence: z.number().optional(),
  reviewRequired: z.boolean().optional(),
});

export const documentIntelligenceSchema = z
  .object({
    kind: z.string().optional(),
    overallConfidence: z.number().optional(),
    reviewStatus: z.string().optional(),
    classifierSource: z.string().optional(),
    providerFingerprint: z.string().optional(),
    classificationReason: z.string().optional(),
    missingFields: z.array(z.string()).optional().default([]),
    signals: z.array(z.string()).optional().default([]),
    conflicts: z
      .array(documentIntelligenceConflictSchema)
      .optional()
      .default([]),
    fields: z
      .record(z.string(), documentIntelligenceFieldSchema)
      .optional()
      .default({}),
    stops: z.array(documentIntelligenceStopSchema).optional().default([]),
    rawExcerpt: z.string().optional(),
  })
  .passthrough();

export const documentIntelligenceAIDiagnosticsSchema = z
  .object({
    fallbackAnalysis: documentIntelligenceSchema.optional(),
    candidateAnalysis: documentIntelligenceSchema.nullable().optional(),
    acceptanceStatus: z
      .enum(["not_attempted", "accepted", "rejected"])
      .optional(),
    rejectionReason: z.string().optional(),
  })
  .passthrough();

export const documentStructuredDataSchema = z
  .object({
    schemaVersion: z.number().optional(),
    intelligence: documentIntelligenceSchema.optional(),
    aiDiagnostics: documentIntelligenceAIDiagnosticsSchema.optional(),
  })
  .passthrough();

export const documentContentSchema = z.object({
  id: z.string(),
  documentId: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  status: documentContentStatusSchema,
  contentText: z.string().nullable().optional(),
  pageCount: z.number(),
  sourceKind: z.string().nullable().optional(),
  detectedLanguage: z.string().nullable().optional(),
  detectedDocumentKind: z.string().nullable().optional(),
  classificationConfidence: z.number(),
  structuredData: documentStructuredDataSchema.default({}),
  failureCode: z.string().nullable().optional(),
  failureMessage: z.string().nullable().optional(),
  version: z.number(),
  createdAt: z.number(),
  updatedAt: z.number(),
  lastExtractedAt: z.number().nullable().optional(),
  pages: z
    .array(
      z.object({
        id: z.string(),
        documentContentId: z.string(),
        documentId: z.string(),
        organizationId: z.string(),
        businessUnitId: z.string(),
        pageNumber: z.number(),
        sourceKind: z.string(),
        extractedText: z.string().nullable().optional(),
        ocrConfidence: z.number(),
        preprocessingApplied: z.boolean(),
        width: z.number(),
        height: z.number(),
        metadata: z.record(z.string(), z.unknown()).default({}),
        version: z.number(),
        createdAt: z.number(),
        updatedAt: z.number(),
      }),
    )
    .optional()
    .default([]),
});

export const documentShipmentDraftSchema = z.object({
  id: z.string(),
  documentId: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  status: documentShipmentDraftStatusSchema,
  documentKind: z.string().nullable().optional(),
  confidence: z.number(),
  draftData: documentIntelligenceSchema,
  failureCode: z.string().nullable().optional(),
  failureMessage: z.string().nullable().optional(),
  attachedShipmentId: z.string().nullable().optional(),
  attachedAt: z.number().nullable().optional(),
  attachedById: z.string().nullable().optional(),
  version: z.number(),
  createdAt: z.number(),
  updatedAt: z.number(),
});

export const documentUploadStrategySchema = z.enum(["single", "multipart"]);
export const documentUploadSessionStatusSchema = z.enum([
  "Initiated",
  "Uploading",
  "Uploaded",
  "Verifying",
  "Finalizing",
  "Paused",
  "Completing",
  "Completed",
  "Available",
  "Quarantined",
  "Failed",
  "Canceled",
  "Expired",
]);

export const documentUploadPartSchema = z.object({
  partNumber: z.number(),
  etag: z.string().optional().default(""),
  size: z.number(),
});

export const documentUploadSessionSchema = z.object({
  id: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  documentId: z.string().nullable().optional(),
  lineageId: z.string().nullable().optional(),
  resourceId: z.string(),
  resourceType: z.string(),
  processingProfile: documentProcessingProfileSchema.nullish().transform((value) => value ?? "none"),
  documentTypeId: z.string().nullable().optional(),
  originalName: z.string(),
  contentType: z.string(),
  fileSize: z.number(),
  storagePath: z.string(),
  storageProviderUploadId: z.string().optional().default(""),
  strategy: documentUploadStrategySchema,
  status: documentUploadSessionStatusSchema,
  description: z.string().nullable().optional(),
  tags: z
    .array(z.string())
    .nullish()
    .transform((value) => value ?? []),
  uploadedParts: z
    .array(documentUploadPartSchema)
    .nullish()
    .transform((value) => value ?? []),
  partSize: z.number(),
  failureCode: z.string().nullable().optional(),
  failureMessage: z.string().nullable().optional(),
  expiresAt: z.number(),
  lastActivityAt: z.number(),
  version: z.number(),
  createdAt: z.number(),
  updatedAt: z.number(),
});

export const documentUploadSessionStateSchema = z.object({
  session: documentUploadSessionSchema,
  parts: z
    .array(documentUploadPartSchema)
    .nullish()
    .transform((value) => value ?? []),
});

export const documentUploadPartTargetSchema = z.object({
  partNumber: z.number(),
  url: z.string(),
});

export type DocumentUploadSession = z.infer<typeof documentUploadSessionSchema>;
export type DocumentUploadSessionStatus = z.infer<
  typeof documentUploadSessionStatusSchema
>;
export type DocumentUploadStrategy = z.infer<
  typeof documentUploadStrategySchema
>;
export type DocumentUploadPart = z.infer<typeof documentUploadPartSchema>;
export type DocumentUploadSessionState = z.infer<
  typeof documentUploadSessionStateSchema
>;
export type DocumentUploadPartTarget = z.infer<
  typeof documentUploadPartTargetSchema
>;
export type DocumentContent = z.infer<typeof documentContentSchema>;
export type DocumentIntelligenceField = z.infer<
  typeof documentIntelligenceFieldSchema
>;
export type DocumentIntelligenceConflict = z.infer<
  typeof documentIntelligenceConflictSchema
>;
export type DocumentIntelligenceStop = z.infer<
  typeof documentIntelligenceStopSchema
>;
export type DocumentIntelligence = z.infer<typeof documentIntelligenceSchema>;
export type DocumentIntelligenceAIDiagnostics = z.infer<
  typeof documentIntelligenceAIDiagnosticsSchema
>;
export type DocumentStructuredData = z.infer<
  typeof documentStructuredDataSchema
>;
export type DocumentShipmentDraft = z.infer<typeof documentShipmentDraftSchema>;

export const documentPacketItemStatusSchema = z.enum([
  "Missing",
  "Complete",
  "ExpiringSoon",
  "Expired",
  "NeedsReview",
]);

export const documentPacketStatusSchema = z.enum([
  "Complete",
  "Incomplete",
  "ExpiringSoon",
  "Expired",
  "NeedsReview",
]);

export const documentPacketItemSchema = z.object({
  documentTypeId: z.string(),
  documentTypeCode: z.string(),
  documentTypeName: z.string(),
  required: z.boolean(),
  allowMultiple: z.boolean(),
  displayOrder: z.number(),
  expirationRequired: z.boolean(),
  expirationWarningDays: z.number(),
  status: documentPacketItemStatusSchema,
  documentCount: z.number(),
  currentDocumentIds: z.array(z.string()),
});

export const documentPacketSummarySchema = z.object({
  resourceId: z.string(),
  resourceType: z.string(),
  status: documentPacketStatusSchema,
  totalRules: z.number(),
  satisfiedRules: z.number(),
  missingRequired: z.number(),
  expiringSoon: z.number(),
  expired: z.number(),
  needsReview: z.number(),
  items: z.array(documentPacketItemSchema),
});

export type DocumentPacketSummary = z.infer<typeof documentPacketSummarySchema>;
export type DocumentPacketItem = z.infer<typeof documentPacketItemSchema>;

export interface UploadDocumentParams {
  file: File;
  resourceId: string;
  resourceType: string;
  processingProfile?: DocumentProcessingProfile;
  description?: string;
  tags?: string[];
  documentTypeId?: string;
  lineageId?: string;
}

export interface BulkUploadDocumentParams {
  files: File[];
  resourceId: string;
  resourceType: string;
  lineageId?: string;
}

export interface CreateDocumentUploadSessionParams {
  resourceId: string;
  resourceType: string;
  processingProfile?: DocumentProcessingProfile;
  fileName: string;
  fileSize: number;
  contentType: string;
  description?: string;
  tags?: string[];
  documentTypeId?: string;
  lineageId?: string;
}

export const importAssistantActionSchema = z.object({
  type: z.string(),
  fieldKey: z.string(),
  value: z.string(),
  metadata: z.record(z.string(), z.unknown()).optional(),
});

export const importAssistantSuggestionSchema = z.object({
  label: z.string(),
  prompt: z.string(),
  type: z.enum(["prompt", "input", "action", "date"]).default("prompt"),
  placeholder: z.string().optional(),
  action: z.string().optional(),
  submitLabel: z.string().optional(),
});

export const importAssistantToolCallRecordSchema = z.object({
  name: z.string(),
  callId: z.string().optional(),
  status: z.string(),
  input: z.string(),
  output: z.string(),
});

export const importAssistantChatMessageSchema = z.object({
  id: z.string(),
  role: z.enum(["user", "assistant"]),
  text: z.string(),
  toolCalls: z.array(importAssistantToolCallRecordSchema).nullish().transform((v) => v ?? []),
  suggestions: z.array(importAssistantSuggestionSchema).nullish().transform((v) => v ?? []),
  createdAt: z.number(),
});

export const importAssistantChatResponseSchema = z.object({
  message: z.string(),
  conversationId: z.string(),
  actions: z.array(importAssistantActionSchema).nullish().transform((v) => v ?? []),
  suggestions: z.array(importAssistantSuggestionSchema).nullish().transform((v) => v ?? []),
  toolCalls: z.array(importAssistantToolCallRecordSchema).nullish().transform((v) => v ?? []),
});

export const conversationStatusSchema = z.enum(["Active", "Completed", "Superseded"]);
export type ConversationStatus = z.infer<typeof conversationStatusSchema>;

export const conversationStatusReasonSchema = z.enum(["", "reextract", "shipment_created", "manual_restart"]);
export type ConversationStatusReason = z.infer<typeof conversationStatusReasonSchema>;

export const importAssistantChatHistoryResponseSchema = z.object({
  documentId: z.string(),
  conversationId: z.string().optional(),
  status: conversationStatusSchema.default("Active"),
  statusReason: conversationStatusReasonSchema.default(""),
  turnCount: z.number().default(0),
  lastMessageAt: z.number().nullable().optional(),
  updatedAt: z.number().default(0),
  messages: z.array(importAssistantChatMessageSchema).nullish().transform((v) => v ?? []),
});

export type ImportAssistantAction = z.infer<typeof importAssistantActionSchema>;
export type ImportAssistantSuggestion = z.infer<typeof importAssistantSuggestionSchema>;
export type ImportAssistantToolCallRecord = z.infer<typeof importAssistantToolCallRecordSchema>;
export type ImportAssistantChatMessage = z.infer<typeof importAssistantChatMessageSchema>;
export type ImportAssistantChatResponse = z.infer<typeof importAssistantChatResponseSchema>;
export type ImportAssistantChatHistoryResponse = z.infer<typeof importAssistantChatHistoryResponseSchema>;

export interface ImportAssistantChatParams {
  message: string;
  conversationId?: string;
  reconciliationState: Record<string, unknown>;
  requiredFields: Record<string, string>;
  stops?: Array<Record<string, unknown>>;
  shipmentData?: Record<string, unknown>;
}
