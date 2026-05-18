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
export const ediTemplateStatusSchema = z.enum([
  "Draft",
  "Certified",
  "Active",
  "Deprecated",
  "Archived",
  "Superseded",
]);
export type EDITemplateStatus = z.infer<typeof ediTemplateStatusSchema>;
export const ediValidationModeSchema = z.enum(["Strict", "WarnOnly", "Disabled"]);
export const ediValidationSeveritySchema = z.enum(["Info", "Warning", "Error"]);
export const ediMessageStatusSchema = z.enum(["Generated", "Failed"]);
export const ediSourceContextDataTypeSchema = z.enum([
  "string",
  "number",
  "integer",
  "boolean",
  "timestamp",
  "date",
  "decimal",
  "object",
  "array",
  "unknown",
]);
export const ediSourceContextKindSchema = z.enum([
  "shipment",
  "repeat",
  "partner",
  "runtime",
  "mapping",
  "organization",
  "customer",
  "location",
  "commodity",
  "charge",
  "envelope",
]);
export const ediSourceContextFieldStatusSchema = z.enum(["Active", "Deprecated", "Future"]);
export const ediPartnerSettingDataTypeSchema = z.enum([
  "string",
  "number",
  "integer",
  "boolean",
  "decimal",
  "enum",
  "object",
  "array",
  "map",
  "secret",
  "unknown",
]);
export const ediPartnerSettingStatusSchema = z.enum(["Active", "Deprecated", "Future"]);
export const ediScriptLanguageSchema = z.enum(["Starlark"]);
export const ediTemplateElementSourceSchema = z.enum([
  "constant",
  "fieldPath",
  "partnerSetting",
  "mapping",
  "runtime",
  "repeat",
  "transform",
  "starlark",
]);
export type EDITemplateElementSource = z.infer<typeof ediTemplateElementSourceSchema>;

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

const ediAcknowledgmentDiagnosticSchema = z.object({
  segmentId: z.string().nullish(),
  segmentPosition: z.number().nullish(),
  elementPosition: z.number().nullish(),
  errorCode: z.string().nullish(),
  message: z.string().nullish(),
});

const freightInvoicePayloadSchema = z.object({}).catchall(z.unknown());
const shipmentStatusPayloadSchema = z.object({}).catchall(z.unknown());
const tenderResponsePayloadSchema = z.object({}).catchall(z.unknown());
const functionalAcknowledgmentPayloadSchema = z
  .object({ diagnostics: z.array(ediAcknowledgmentDiagnosticSchema).nullish() })
  .catchall(z.unknown());
const implementationAcknowledgmentPayloadSchema = functionalAcknowledgmentPayloadSchema;

export const ediDocumentPayloadSchema = z.object({
  transactionSet: ediTransactionSetSchema.nullish(),
  loadTender: loadTenderPayloadSchema.nullish(),
  shipment: loadTenderPayloadSchema.nullish(),
  invoice: freightInvoicePayloadSchema.nullish(),
  shipmentStatus: shipmentStatusPayloadSchema.nullish(),
  tenderResponse: tenderResponsePayloadSchema.nullish(),
  functionalAck: functionalAcknowledgmentPayloadSchema.nullish(),
  implementationAck: implementationAcknowledgmentPayloadSchema.nullish(),
});

export type EDIDocumentPayload = z.infer<typeof ediDocumentPayloadSchema>;

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

export const ediTemplateElementBaseSourceSchema = z.object({
  source: ediTemplateElementSourceSchema,
  value: z.string().nullish(),
  fieldPath: z.string().nullish(),
  partnerSettingPath: z.string().nullish(),
  mappingEntityType: ediMappingEntityTypeSchema.nullish(),
  mappingSourcePath: z.string().nullish(),
  runtimeKey: z.string().nullish(),
  repeatPath: z.string().nullish(),
  default: z.string().nullish(),
});
export type EDITemplateElementBaseSource = z.infer<typeof ediTemplateElementBaseSourceSchema>;

export const ediTemplateTransformStepSchema = z.object({
  operation: z.string(),
  arguments: z.record(z.string(), z.unknown()).default({}),
});
export type EDITemplateTransformStep = z.infer<typeof ediTemplateTransformStepSchema>;

export const ediTemplateElementSchema = z.object({
  position: z.number(),
  name: z.string(),
  source: ediTemplateElementSourceSchema,
  value: z.string().nullish(),
  fieldPath: z.string().nullish(),
  partnerSettingPath: z.string().nullish(),
  mappingEntityType: ediMappingEntityTypeSchema.nullish(),
  mappingSourcePath: z.string().nullish(),
  runtimeKey: z.string().nullish(),
  repeatPath: z.string().nullish(),
  baseSource: ediTemplateElementBaseSourceSchema.nullish(),
  transformPipeline: z.array(ediTemplateTransformStepSchema).default([]),
  starlarkFunction: z.string().nullish(),
  starlarkScript: z.string().nullish(),
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

export const ediTemplateScriptLibrarySchema = z.object({
  id: z.string(),
  businessUnitId: z.string().optional(),
  organizationId: z.string().optional(),
  templateVersionId: z.string(),
  name: z.string(),
  description: z.string().nullish(),
  language: ediScriptLanguageSchema,
  script: z.string(),
  status: ediTemplateStatusSchema,
  version: z.number().default(0),
  createdAt: z.number().nullish(),
  updatedAt: z.number().nullish(),
  functionNames: z.array(z.string()).default([]),
});

export type EDITemplateScriptLibrary = z.infer<typeof ediTemplateScriptLibrarySchema>;

export const ediTemplateVersionSchema = z.object({
  id: z.string(),
  businessUnitId: z.string().optional(),
  organizationId: z.string().optional(),
  templateId: z.string(),
  sourceVersionId: z.string().nullish(),
  versionNumber: z.number(),
  x12Version: z.string(),
  functionalGroupId: z.string(),
  status: ediTemplateStatusSchema,
  isActive: z.boolean(),
  notes: z.string().nullish(),
  certificationNotes: z.string().nullish(),
  activationNotes: z.string().nullish(),
  archiveNotes: z.string().nullish(),
  deprecatedNotes: z.string().nullish(),
  supersededNotes: z.string().nullish(),
  certifiedAt: z.number().nullish(),
  activatedAt: z.number().nullish(),
  archivedAt: z.number().nullish(),
  deprecatedAt: z.number().nullish(),
  supersededAt: z.number().nullish(),
  version: z.number().default(0),
  createdAt: z.number().nullish(),
  updatedAt: z.number().nullish(),
  segments: z.array(ediTemplateSegmentSchema).default([]),
  scriptLibraries: z.array(ediTemplateScriptLibrarySchema).default([]),
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
  version: z.number().default(0),
  createdAt: z.number().nullish(),
  updatedAt: z.number().nullish(),
  activeVersion: ediTemplateVersionSchema.nullish(),
  versions: z.array(ediTemplateVersionSchema).default([]),
});

export type EDITemplate = z.infer<typeof ediTemplateSchema>;

export const ediSourceContextFieldSchema = z.object({
  id: z.string(),
  schemaId: z.string(),
  path: z.string(),
  sourceKind: ediSourceContextKindSchema,
  dataType: ediSourceContextDataTypeSchema,
  repeated: z.boolean(),
  repeatPath: z.string().nullish(),
  parentPath: z.string().nullish(),
  displayName: z.string(),
  description: z.string().nullish(),
  status: ediSourceContextFieldStatusSchema,
  createdAt: z.number().nullish(),
  updatedAt: z.number().nullish(),
});

export type EDISourceContextField = z.infer<typeof ediSourceContextFieldSchema>;

export const ediSourceContextSchemaSchema = z.object({
  id: z.string(),
  businessUnitId: z.string().nullish(),
  organizationId: z.string().nullish(),
  standard: ediStandardSchema,
  transactionSet: ediTransactionSetSchema,
  direction: ediDocumentDirectionSchema,
  x12Version: z.string(),
  contextKey: z.string(),
  schemaVersion: z.number(),
  name: z.string(),
  description: z.string().nullish(),
  status: ediSourceContextFieldStatusSchema,
  createdAt: z.number().nullish(),
  updatedAt: z.number().nullish(),
  fields: z.array(ediSourceContextFieldSchema).default([]),
});

export type EDISourceContextSchema = z.infer<typeof ediSourceContextSchemaSchema>;

export const ediPartnerSettingFieldSchema = z.object({
  id: z.string(),
  schemaId: z.string(),
  path: z.string(),
  label: z.string(),
  description: z.string().nullish(),
  dataType: ediPartnerSettingDataTypeSchema,
  required: z.boolean(),
  nullable: z.boolean(),
  defaultValue: z.unknown().nullish(),
  allowedValues: z.array(z.string()).default([]),
  secret: z.boolean(),
  groupKey: z.string().nullish(),
  displayOrder: z.number().default(0),
  validationPattern: z.string().nullish(),
  minLength: z.number().default(0),
  maxLength: z.number().default(0),
  usageNotes: z.string().nullish(),
  status: ediPartnerSettingStatusSchema,
  createdAt: z.number().nullish(),
  updatedAt: z.number().nullish(),
});

export type EDIPartnerSettingField = z.infer<typeof ediPartnerSettingFieldSchema>;

export const ediPartnerSettingSchemaSchema = z.object({
  id: z.string(),
  businessUnitId: z.string().nullish(),
  organizationId: z.string().nullish(),
  documentTypeId: z.string().nullish(),
  standard: ediStandardSchema,
  transactionSet: ediTransactionSetSchema,
  direction: ediDocumentDirectionSchema,
  x12Version: z.string(),
  schemaVersion: z.number(),
  name: z.string(),
  description: z.string().nullish(),
  status: ediPartnerSettingStatusSchema,
  createdAt: z.number().nullish(),
  updatedAt: z.number().nullish(),
  fields: z.array(ediPartnerSettingFieldSchema).default([]),
});

export type EDIPartnerSettingSchema = z.infer<typeof ediPartnerSettingSchemaSchema>;

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
  documentType: ediDocumentTypeSchema.nullish(),
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
  suggestedFix: z.string().nullish(),
});

export type EDIDiagnostic = z.infer<typeof ediDiagnosticSchema>;

export const ediTemplateValidationResponseSchema = z.object({
  diagnostics: z.array(ediDiagnosticSchema).default([]),
});

export type EDITemplateValidationResponse = z.infer<typeof ediTemplateValidationResponseSchema>;

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
  businessUnitId: z.string().nullish(),
  organizationId: z.string().nullish(),
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
  payloadSnapshot: ediDocumentPayloadSchema.nullish(),
  generatedById: z.string().nullish(),
  generatedAt: z.number(),
  diagnosticCount: z.number().default(0),
  partner: ediPartnerSchema.nullish(),
  documentType: ediDocumentTypeSchema.nullish(),
  partnerDocumentProfile: ediPartnerDocumentProfileSchema.nullish(),
  template: ediTemplateSchema.nullish(),
  templateVersion: ediTemplateVersionSchema.nullish(),
  validationErrors: z.array(ediDiagnosticSchema).default([]),
});

export type EDIMessage = z.infer<typeof ediMessageSchema>;

export const ediTestCaseSchema = z.object({
  id: z.string(),
  partnerDocumentProfileId: z.string(),
  name: z.string(),
  description: z.string().nullish(),
  payload: ediDocumentPayloadSchema,
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
export const ediSourceContextSchemaListSchema = createLimitOffsetResponse(
  ediSourceContextSchemaSchema,
);
export const ediSourceContextFieldListSchema = createLimitOffsetResponse(
  ediSourceContextFieldSchema,
);
export const ediPartnerSettingSchemaListSchema = createLimitOffsetResponse(
  ediPartnerSettingSchemaSchema,
);
export const ediPartnerSettingFieldListSchema = createLimitOffsetResponse(
  ediPartnerSettingFieldSchema,
);
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

export type CreateEDITemplateRequest = {
  documentTypeId: string;
  name: string;
  description?: string;
  direction: z.infer<typeof ediDocumentDirectionSchema>;
  standard: z.infer<typeof ediStandardSchema>;
  transactionSet: z.infer<typeof ediTransactionSetSchema>;
  x12Version: string;
  functionalGroupId: string;
  notes?: string;
  segments?: EDITemplateSegment[];
  scriptLibraries?: EDITemplateScriptLibrary[];
};

export type UpdateEDITemplateRequest = {
  name: string;
  description?: string;
  status?: EDITemplateStatus;
  version?: number;
};

export type CreateEDITemplateDraftRequest = {
  sourceVersionId?: string;
  notes?: string;
};

export type UpdateEDITemplateVersionRequest = {
  x12Version: string;
  functionalGroupId: string;
  notes?: string;
  version?: number;
};

export type ReplaceEDITemplateSegmentsRequest = {
  segments: EDITemplateSegment[];
  version?: number;
};

export type ReplaceEDITemplateScriptLibrariesRequest = {
  scriptLibraries: EDITemplateScriptLibrary[];
  version?: number;
};

export type EDITemplateActionRequest = {
  notes?: string;
};

export type PreviewEDIDocumentRequest = {
  partnerDocumentProfileId?: string;
  ediPartnerId?: string;
  shipmentId?: string;
  transferId?: string;
  invoiceId?: string;
  shipmentEventId?: string;
  sourceMessageId?: string;
  transactionSet?: z.infer<typeof ediTransactionSetSchema>;
  direction?: z.infer<typeof ediDocumentDirectionSchema>;
  payload?: EDIDocumentPayload;
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

export const createTemplateDraftSchema = z.object({
  documentTypeId: z.string(),
  name: z.string(),
  description: z.string(),
  direction: ediDocumentDirectionSchema.default("Outbound"),
  transactionSet: ediTransactionSetSchema.default("204"),
  x12Version: z.string(),
  functionalGroupId: z.string(),
  notes: z.string(),
});

export type CreateTemplateDraft = z.infer<typeof createTemplateDraftSchema>;
