import { z } from "zod";
import { nullableStringSchema } from "./helpers";
import { createLimitOffsetResponse } from "./server";

export const ediPartnerKindSchema = z.enum(["Internal", "External"]);
export const ediConnectionMethodSchema = z.enum(["Internal", "AS2", "SFTP", "VAN"]);
export const ediConnectionStatusSchema = z.enum([
  "PendingAcceptance",
  "Active",
  "Suspended",
  "Rejected",
  "Revoked",
]);
export const ediMappingEntityTypeSchema = z.enum([
  "Customer",
  "ServiceType",
  "ShipmentType",
  "FormulaTemplate",
  "Location",
  "Commodity",
  "AccessorialCharge",
]);
export type EDIMappingEntityType = z.infer<typeof ediMappingEntityTypeSchema>;
export const ediTransferStatusSchema = z.enum([
  "Submitted",
  "MappingRequired",
  "PendingApproval",
  "Processing",
  "Approved",
  "Rejected",
  "Expired",
  "Canceled",
  "Failed",
]);
export const ediShipmentSyncPolicySchema = z.enum([
  "ManualReview",
  "AutoOperational",
  "AutoAllSafe",
  "ReadOnly",
]);
export const ediShipmentLinkStatusSchema = z.enum(["Active", "Suspended", "Closed"]);
export const ediTransferChangeDirectionSchema = z.enum(["SourceToTarget", "TargetToSource"]);
export const ediTransferChangeStatusSchema = z.enum([
  "PendingReview",
  "Applied",
  "Rejected",
  "Failed",
  "Ignored",
]);
export const ediTransferChangeConflictStatusSchema = z.enum(["None", "Conflict", "Resolved"]);
export const ediDocumentDirectionSchema = z.enum(["Inbound", "Outbound"]);
export const ediStandardSchema = z.enum(["X12"]);
export const ediTransactionSetSchema = z.enum(["204", "210", "214", "990", "997", "999"]);
export const ediDocumentStatusSchema = z.enum(["Active", "Inactive"]);
export const ediTemplateStatusSchema = z.enum(["Draft", "Active", "Archived", "Superseded"]);
export const ediValidationModeSchema = z.enum(["Strict", "WarnOnly", "Disabled"]);
export const ediValidationSeveritySchema = z.enum(["Info", "Warning", "Error"]);
export const ediMessageStatusSchema = z.enum(["Generated", "Failed"]);

export const ediPartnerSchema = z.object({
  id: z.string(),
  businessUnitId: z.string(),
  organizationId: z.string(),
  kind: ediPartnerKindSchema,
  status: z.string(),
  code: z.string(),
  name: z.string(),
  description: z.string().nullish(),
  internalOrganizationId: z.string().nullish(),
  ediConnectionId: z.string().nullish(),
  defaultTransportId: z.string().nullish(),
  contactName: z.string().nullish(),
  contactEmail: z.string().nullish(),
  contactPhone: z.string().nullish(),
  enabledForInbound: z.boolean(),
  enabledForOutbound: z.boolean(),
  settings: z.record(z.string(), z.unknown()).nullish(),
  version: z.number().default(0),
  updatedAt: z.number().nullish(),
  internalOrganization: z
    .object({
      id: z.string(),
      name: z.string(),
    })
    .nullish(),
  connection: z.unknown().nullish(),
  defaultTransport: z.unknown().nullish(),
});

export type EDIPartner = z.infer<typeof ediPartnerSchema>;

export const ediConnectionCapabilitiesSchema = z.object({
  loadTenderOutbound: z.boolean().default(true),
  loadTenderInbound: z.boolean().default(true),
  shipmentStatus: z.boolean().default(false),
  invoice: z.boolean().default(false),
});

export type EDIConnectionCapabilities = z.infer<typeof ediConnectionCapabilitiesSchema>;

export const ediConnectionPartnerConfigSchema = z.object({
  code: z.string().default(""),
  name: z.string().default(""),
  description: z.string().default(""),
  contactName: z.string().default(""),
  contactEmail: z.string().default(""),
  contactPhone: z.string().default(""),
  enabledForInbound: z.boolean().default(true),
  enabledForOutbound: z.boolean().default(true),
  settings: z.record(z.string(), z.unknown()).default({}),
});

export type EDIConnectionPartnerConfig = z.infer<typeof ediConnectionPartnerConfigSchema>;

export const ediConnectionSchema = z.object({
  id: z.string(),
  businessUnitId: z.string(),
  sourceOrganizationId: z.string(),
  targetOrganizationId: z.string(),
  sourcePartnerId: z.string().nullish(),
  targetPartnerId: z.string().nullish(),
  method: ediConnectionMethodSchema,
  status: ediConnectionStatusSchema,
  capabilities: ediConnectionCapabilitiesSchema,
  sourcePartnerConfig: ediConnectionPartnerConfigSchema,
  targetPartnerConfig: ediConnectionPartnerConfigSchema,
  requestedById: z.string().nullish(),
  requestedAt: z.number(),
  acceptedAt: z.number().nullish(),
  rejectedAt: z.number().nullish(),
  rejectionReason: z.string().nullish(),
  suspendedAt: z.number().nullish(),
  revokedAt: z.number().nullish(),
  version: z.number().default(0),
  createdAt: z.number(),
  updatedAt: z.number(),
  sourceOrganization: z.object({ id: z.string(), name: z.string() }).nullish(),
  targetOrganization: z.object({ id: z.string(), name: z.string() }).nullish(),
  sourcePartner: ediPartnerSchema.nullish(),
  targetPartner: ediPartnerSchema.nullish(),
});

export type EDIConnection = z.infer<typeof ediConnectionSchema>;

export const ediCommunicationProfileSecretStateSchema = z.object({
  key: z.string(),
  hasValue: z.boolean(),
});

export const ediCommunicationProfileSchema = z.object({
  id: z.string(),
  businessUnitId: z.string(),
  organizationId: z.string(),
  ediConnectionId: z.string().nullish(),
  ediPartnerId: z.string().nullish(),
  method: ediConnectionMethodSchema,
  status: z.string(),
  name: z.string(),
  description: z.string().nullish(),
  config: z.record(z.string(), z.unknown()).default({}),
  secretState: z.array(ediCommunicationProfileSecretStateSchema).default([]),
  version: z.number().default(0),
  createdAt: z.number(),
  updatedAt: z.number(),
  partner: ediPartnerSchema.nullish(),
});

export type EDICommunicationProfile = z.infer<typeof ediCommunicationProfileSchema>;

export const ediMappingProfileItemSchema = z.object({
  id: z.string().optional(),
  businessUnitId: z.string().optional(),
  organizationId: z.string().optional(),
  ediPartnerId: z.string().optional(),
  mappingProfileId: z.string().optional(),
  entityType: ediMappingEntityTypeSchema,
  sourceId: z.string(),
  sourceLabel: z.string().nullish(),
  targetId: z.string(),
  targetLabel: z.string().nullish(),
});

export type EDIMappingProfileItem = z.infer<typeof ediMappingProfileItemSchema>;

export const ediMappingProfileSchema = z.object({
  id: z.string(),
  businessUnitId: z.string(),
  organizationId: z.string(),
  ediPartnerId: z.string(),
  name: z.string(),
  description: nullableStringSchema,
  entries: z.array(ediMappingProfileItemSchema).default([]),
});

export type EDIMappingProfile = z.infer<typeof ediMappingProfileSchema>;

export const ediMappingResolutionSchema = z.object({
  entityType: ediMappingEntityTypeSchema,
  sourceId: z.string(),
  sourceLabel: z.string().nullish(),
  targetId: z.string().nullish(),
  targetLabel: z.string().nullish(),
  resolved: z.boolean(),
});

export type EDIMappingResolution = z.infer<typeof ediMappingResolutionSchema>;

export const ediMappingPreviewSchema = z.object({
  resolved: z.array(ediMappingResolutionSchema),
  unresolved: z.array(ediMappingResolutionSchema),
  all: z.array(ediMappingResolutionSchema),
});

export type EDIMappingPreview = z.infer<typeof ediMappingPreviewSchema>;

const nullableNumberSchema = z.number().nullish();

export const loadTenderStopSchema = z.object({
  locationId: z.string(),
  locationLabel: z.string().nullish(),
  locationName: z.string().nullish(),
  locationCode: z.string().nullish(),
  locationAddressLine1: z.string().nullish(),
  locationAddressLine2: z.string().nullish(),
  locationCity: z.string().nullish(),
  locationStateCode: z.string().nullish(),
  locationPostalCode: z.string().nullish(),
  type: z.string(),
  scheduleType: z.string(),
  sequence: z.number(),
  pieces: nullableNumberSchema,
  weight: nullableNumberSchema,
  scheduledWindowStart: nullableNumberSchema,
  scheduledWindowEnd: nullableNumberSchema,
  addressLine: z.string().nullish(),
});

export type LoadTenderStop = z.infer<typeof loadTenderStopSchema>;

export const loadTenderMoveSchema = z.object({
  loaded: z.boolean(),
  sequence: z.number(),
  distance: nullableNumberSchema,
  stops: z.array(loadTenderStopSchema).default([]),
});

export type LoadTenderMove = z.infer<typeof loadTenderMoveSchema>;

export const loadTenderCommoditySchema = z.object({
  commodityId: z.string(),
  commodityLabel: z.string().nullish(),
  commodityName: z.string().nullish(),
  commodityDescription: z.string().nullish(),
  weight: z.number(),
  pieces: z.number(),
});

export type LoadTenderCommodity = z.infer<typeof loadTenderCommoditySchema>;

export const loadTenderChargeSchema = z.object({
  accessorialChargeId: z.string(),
  accessorialLabel: z.string().nullish(),
  accessorialCode: z.string().nullish(),
  accessorialDescription: z.string().nullish(),
  method: z.string(),
  amount: z.unknown(),
  unit: z.number(),
});

export type LoadTenderCharge = z.infer<typeof loadTenderChargeSchema>;

const loadTenderPayloadSchema = z.object({
  shipmentId: z.string(),
  businessUnitId: z.string().nullish(),
  organizationId: z.string().nullish(),
  serviceTypeId: z.string().nullish(),
  serviceTypeLabel: z.string().nullish(),
  shipmentTypeId: z.string().nullish(),
  shipmentTypeLabel: z.string().nullish(),
  customerId: z.string().nullish(),
  customerLabel: z.string().nullish(),
  formulaTemplateId: z.string().nullish(),
  formulaTemplateLabel: z.string().nullish(),
  bol: z.string().nullish(),
  pieces: z.number().nullish(),
  weight: z.number().nullish(),
  temperatureMin: nullableNumberSchema,
  temperatureMax: nullableNumberSchema,
  freightChargeAmount: z.unknown().nullish(),
  otherChargeAmount: z.unknown().nullish(),
  baseRate: z.unknown().nullish(),
  totalChargeAmount: z.unknown().nullish(),
  ratingUnit: nullableNumberSchema,
  ratingDetail: z.record(z.string(), z.unknown()).nullish(),
  moves: z.array(loadTenderMoveSchema).default([]),
  commodities: z.array(loadTenderCommoditySchema).default([]),
  additionalCharges: z.array(loadTenderChargeSchema).default([]),
  requiredMappingEntityIds: z.record(z.string(), z.array(z.string())).nullish(),
});

export type LoadTenderPayload = z.infer<typeof loadTenderPayloadSchema>;

export const ediTransferSchema = z.object({
  id: z.string(),
  sourceOrganizationId: z.string(),
  targetOrganizationId: z.string(),
  sourcePartnerId: z.string(),
  targetPartnerId: z.string(),
  sourceShipmentId: z.string(),
  targetShipmentId: z.string().nullish(),
  status: ediTransferStatusSchema,
  tenderPayload: loadTenderPayloadSchema,
  mappingSnapshot: z.array(ediMappingResolutionSchema).default([]),
  rejectionReason: z.string().nullish(),
  failureReason: z.string().nullish(),
  approvalWorkflowId: z.string().nullish(),
  approvalWorkflowRunId: z.string().nullish(),
  submittedAt: z.number(),
  approvedAt: z.number().nullish(),
  processingStartedAt: z.number().nullish(),
  processedAt: z.number().nullish(),
  rejectedAt: z.number().nullish(),
  canceledAt: z.number().nullish(),
  sourcePartner: ediPartnerSchema.nullish(),
  targetPartner: ediPartnerSchema.nullish(),
});

export type EDITransfer = z.infer<typeof ediTransferSchema>;

export const ediShipmentLinkSchema = z.object({
  id: z.string(),
  businessUnitId: z.string(),
  sourceOrganizationId: z.string(),
  targetOrganizationId: z.string(),
  sourceShipmentId: z.string(),
  targetShipmentId: z.string(),
  tenderTransferId: z.string(),
  originatingMessageId: z.string().nullish(),
  syncPolicy: ediShipmentSyncPolicySchema,
  fieldOwnership: z.record(z.string(), z.string()).default({}),
  status: ediShipmentLinkStatusSchema,
  version: z.number().default(0),
  createdAt: z.number(),
  updatedAt: z.number(),
});

export type EDIShipmentLink = z.infer<typeof ediShipmentLinkSchema>;

export const ediTransferChangeSchema = z.object({
  id: z.string(),
  businessUnitId: z.string(),
  shipmentLinkId: z.string(),
  direction: ediTransferChangeDirectionSchema,
  changeType: z.string(),
  status: ediTransferChangeStatusSchema,
  conflictStatus: ediTransferChangeConflictStatusSchema,
  conflictReason: z.string().nullish(),
  idempotencyKey: z.string(),
  sourceShipmentVersion: z.number(),
  targetShipmentVersion: z.number(),
  payload: z.record(z.string(), z.unknown()).default({}),
  diff: z.record(z.string(), z.unknown()).default({}),
  reviewedById: z.string().nullish(),
  reviewedAt: z.number().nullish(),
  appliedById: z.string().nullish(),
  appliedAt: z.number().nullish(),
  failureReason: z.string().nullish(),
  version: z.number().default(0),
  createdAt: z.number(),
  updatedAt: z.number(),
});

export type EDITransferChange = z.infer<typeof ediTransferChangeSchema>;

export const ediTemplateElementSchema = z.object({
  position: z.number(),
  name: z.string(),
  source: z.string(),
  value: z.string().nullish(),
  fieldPath: z.string().nullish(),
  partnerSettingPath: z.string().nullish(),
  expression: z.string().nullish(),
  runtimeKey: z.string().nullish(),
  repeatPath: z.string().nullish(),
  default: z.string().nullish(),
  condition: z.string().nullish(),
  implementationGuideNote: z.string().nullish(),
  validation: z
    .object({
      required: z.boolean().default(false),
      maxLength: z.number().default(0),
      minLength: z.number().default(0),
      code: z.string().nullish(),
      message: z.string().nullish(),
    })
    .default({ required: false, maxLength: 0, minLength: 0 }),
});

export type EDITemplateElement = z.infer<typeof ediTemplateElementSchema>;

export const ediTemplateSegmentSchema = z.object({
  id: z.string(),
  templateVersionId: z.string(),
  segmentId: z.string(),
  name: z.string(),
  sequence: z.number(),
  loopId: z.string().nullish(),
  repeatPath: z.string().nullish(),
  condition: z.string().nullish(),
  required: z.boolean(),
  maxUse: z.number(),
  elements: z.array(ediTemplateElementSchema).default([]),
  usageNotes: z.string().nullish(),
});

export type EDITemplateSegment = z.infer<typeof ediTemplateSegmentSchema>;

export const ediTemplateVersionSchema = z.object({
  id: z.string(),
  templateId: z.string(),
  versionNumber: z.number(),
  x12Version: z.string(),
  functionalGroupId: z.string(),
  status: ediTemplateStatusSchema,
  isActive: z.boolean(),
  notes: z.string().nullish(),
  segments: z.array(ediTemplateSegmentSchema).default([]),
});

export type EDITemplateVersion = z.infer<typeof ediTemplateVersionSchema>;

export const ediDocumentTypeSchema = z.object({
  id: z.string(),
  code: z.string(),
  name: z.string(),
  standard: ediStandardSchema,
  transactionSet: ediTransactionSetSchema,
  direction: ediDocumentDirectionSchema,
  defaultVersion: z.string(),
  status: ediDocumentStatusSchema,
});

export type EDIDocumentType = z.infer<typeof ediDocumentTypeSchema>;

export const ediTemplateSchema = z.object({
  id: z.string(),
  businessUnitId: z.string(),
  organizationId: z.string(),
  documentTypeId: z.string(),
  name: z.string(),
  description: z.string().nullish(),
  direction: ediDocumentDirectionSchema,
  standard: ediStandardSchema,
  transactionSet: ediTransactionSetSchema,
  status: ediTemplateStatusSchema,
  activeVersion: ediTemplateVersionSchema.nullish(),
  versions: z.array(ediTemplateVersionSchema).default([]),
});

export type EDITemplate = z.infer<typeof ediTemplateSchema>;

export const ediX12EnvelopeSettingsSchema = z.object({
  interchangeSenderId: z.string().default("TRENOVA"),
  interchangeReceiverId: z.string().default("PARTNER"),
  applicationSenderCode: z.string().default("TRENOVA"),
  applicationReceiverCode: z.string().default("PARTNER"),
  interchangeUsageIndicator: z.string().default("T"),
  elementSeparator: z.string().default("*"),
  segmentTerminator: z.string().default("~"),
  componentSeparator: z.string().default(">"),
  repetitionSeparator: z.string().default("^"),
});

export type EDIX12EnvelopeSettings = z.infer<typeof ediX12EnvelopeSettingsSchema>;

export const ediAcknowledgmentConfigSchema = z.object({
  expected: z.boolean().default(false),
  type: z.string().default("None"),
  slaInMinutes: z.number().default(0),
  missingAckSeverity: ediValidationSeveritySchema.default("Warning"),
});

export const ediPartnerDocumentProfileSchema = z.object({
  id: z.string(),
  businessUnitId: z.string(),
  organizationId: z.string(),
  ediPartnerId: z.string(),
  documentTypeId: z.string(),
  templateId: z.string(),
  templateVersionId: z.string().nullish(),
  name: z.string(),
  status: ediDocumentStatusSchema,
  direction: ediDocumentDirectionSchema,
  standard: ediStandardSchema,
  transactionSet: ediTransactionSetSchema,
  x12VersionOverride: z.string().nullish(),
  functionalGroupId: z.string(),
  envelope: ediX12EnvelopeSettingsSchema,
  acknowledgment: ediAcknowledgmentConfigSchema,
  validationMode: ediValidationModeSchema,
  partnerSettings: z.record(z.string(), z.unknown()).default({}),
  version: z.number().default(0),
  createdAt: z.number(),
  updatedAt: z.number(),
  partner: ediPartnerSchema.nullish(),
  template: ediTemplateSchema.nullish(),
  templateVersion: ediTemplateVersionSchema.nullish(),
});

export type EDIPartnerDocumentProfile = z.infer<typeof ediPartnerDocumentProfileSchema>;

export const ediDiagnosticSchema = z.object({
  severity: ediValidationSeveritySchema,
  code: z.string(),
  segmentId: z.string().nullish(),
  elementPosition: z.number().default(0),
  path: z.string().nullish(),
  message: z.string(),
});

export type EDIDiagnostic = z.infer<typeof ediDiagnosticSchema>;

export const ediDocumentPreviewSchema = z.object({
  rawX12: z.string(),
  segmentCount: z.number(),
  x12Version: z.string(),
  interchangeControlNumber: z.string(),
  groupControlNumber: z.string(),
  transactionControlNumber: z.string(),
  diagnostics: z.array(ediDiagnosticSchema).default([]),
  profile: ediPartnerDocumentProfileSchema.nullish(),
  templateVersion: ediTemplateVersionSchema.nullish(),
});

export type EDIDocumentPreview = z.infer<typeof ediDocumentPreviewSchema>;

export const ediMessageSchema = z.object({
  id: z.string(),
  ediPartnerId: z.string(),
  documentTypeId: z.string(),
  partnerDocumentProfileId: z.string(),
  templateId: z.string(),
  templateVersionId: z.string(),
  shipmentId: z.string().nullish(),
  transferId: z.string().nullish(),
  direction: ediDocumentDirectionSchema,
  standard: ediStandardSchema,
  transactionSet: ediTransactionSetSchema,
  x12Version: z.string(),
  status: ediMessageStatusSchema,
  validationMode: ediValidationModeSchema,
  interchangeControlNumber: z.string(),
  groupControlNumber: z.string(),
  transactionControlNumber: z.string(),
  segmentCount: z.number(),
  rawX12: z.string(),
  generatedAt: z.number(),
  partnerDocumentProfile: ediPartnerDocumentProfileSchema.nullish(),
  validationErrors: z.array(ediDiagnosticSchema).default([]),
});

export type EDIMessage = z.infer<typeof ediMessageSchema>;

export const ediTestCaseSchema = z.object({
  id: z.string(),
  partnerDocumentProfileId: z.string(),
  name: z.string(),
  description: z.string().nullish(),
  payload: loadTenderPayloadSchema,
  expectedWarnings: z.number(),
  expectedErrors: z.number(),
});

export type EDITestCase = z.infer<typeof ediTestCaseSchema>;

export const ediPartnerListSchema = createLimitOffsetResponse(ediPartnerSchema);
export const ediConnectionListSchema = createLimitOffsetResponse(ediConnectionSchema);
export const ediCommunicationProfileListSchema = createLimitOffsetResponse(
  ediCommunicationProfileSchema,
);
export const ediTransferListSchema = createLimitOffsetResponse(ediTransferSchema);
export const ediMappingProfileListSchema = createLimitOffsetResponse(ediMappingProfileSchema);
export const ediShipmentLinkListSchema = createLimitOffsetResponse(ediShipmentLinkSchema);
export const ediTransferChangeListSchema = createLimitOffsetResponse(ediTransferChangeSchema);
export const ediPartnerSelectOptionListSchema = createLimitOffsetResponse(ediPartnerSchema);
export const ediTemplateListSchema = createLimitOffsetResponse(ediTemplateSchema);
export const ediPartnerDocumentProfileListSchema = createLimitOffsetResponse(
  ediPartnerDocumentProfileSchema,
);
export const ediMessageListSchema = createLimitOffsetResponse(ediMessageSchema);
export const ediTestCaseListSchema = createLimitOffsetResponse(ediTestCaseSchema);

export const createInternalPartnerPairRequestSchema = z.object({
  targetOrganizationId: z.string().min(1),
  sourceCode: z.string().min(1),
  sourceName: z.string().min(1),
  sourceDescription: z.string().optional().default(""),
  sourceContactName: z.string().optional().default(""),
  sourceContactEmail: z.string().optional().default(""),
  sourceContactPhone: z.string().optional().default(""),
  sourceEnabledForInbound: z.boolean().default(true),
  sourceEnabledForOutbound: z.boolean().default(true),
  sourceSettings: z.record(z.string(), z.unknown()).default({}),
  targetCode: z.string().min(1),
  targetName: z.string().min(1),
  targetDescription: z.string().optional().default(""),
  targetContactName: z.string().optional().default(""),
  targetContactEmail: z.string().optional().default(""),
  targetContactPhone: z.string().optional().default(""),
  targetEnabledForInbound: z.boolean().default(true),
  targetEnabledForOutbound: z.boolean().default(true),
  targetSettings: z.record(z.string(), z.unknown()).default({}),
});

export type CreateInternalPartnerPairRequest = z.infer<
  typeof createInternalPartnerPairRequestSchema
>;

export const createEDIConnectionRequestSchema = z.object({
  targetOrganizationId: z.string().min(1),
  method: ediConnectionMethodSchema.default("Internal"),
  capabilities: ediConnectionCapabilitiesSchema.default({
    loadTenderOutbound: true,
    loadTenderInbound: true,
    shipmentStatus: false,
    invoice: false,
  }),
  sourcePartnerConfig: ediConnectionPartnerConfigSchema,
  targetPartnerConfig: ediConnectionPartnerConfigSchema,
});

export type CreateEDIConnectionRequest = z.infer<typeof createEDIConnectionRequestSchema>;

export type EDIConnectionActionRequest = {
  reason?: string;
};

export type UpsertEDICommunicationProfileRequest = {
  ediConnectionId?: string;
  ediPartnerId?: string;
  method: z.infer<typeof ediConnectionMethodSchema>;
  status?: string;
  name: string;
  description?: string;
  config: Record<string, unknown>;
  secrets?: Record<string, string>;
  version?: number;
};

export type UpsertEDIPartnerDocumentProfileRequest = {
  ediPartnerId: string;
  templateId?: string;
  templateVersionId?: string;
  name: string;
  status: z.infer<typeof ediDocumentStatusSchema>;
  x12VersionOverride?: string;
  functionalGroupId: string;
  envelope: EDIX12EnvelopeSettings;
  acknowledgment: z.infer<typeof ediAcknowledgmentConfigSchema>;
  validationMode: z.infer<typeof ediValidationModeSchema>;
  partnerSettings: Record<string, unknown>;
  version?: number;
};

export type PreviewEDIDocumentRequest = {
  partnerDocumentProfileId?: string;
  ediPartnerId?: string;
  shipmentId?: string;
  transferId?: string;
  payload?: LoadTenderPayload;
};

export type GenerateEDIDocumentRequest = PreviewEDIDocumentRequest;

export const internalPartnerPairSchema = z.object({
  sourcePartner: ediPartnerSchema,
  targetPartner: ediPartnerSchema,
});

export type InternalPartnerPair = z.infer<typeof internalPartnerPairSchema>;

export type SubmitLoadTenderRequest = {
  sourceShipmentId: string;
  ediPartnerId: string;
};

export type ApproveEDITransferRequest = {
  mappings: EDIMappingProfileItem[];
};

export type RejectEDITransferRequest = {
  reason: string;
};
