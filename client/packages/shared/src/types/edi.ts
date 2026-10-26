import { z } from "zod";
import { nullableArraySchema, nullableStringSchema } from "./helpers";
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
  "ServiceFailureReasonCode",
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
export const ediValidationSeveritySchema = z.enum(["Info", "Warning", "Error"]);
export const ediMessageStatusSchema = z.enum(["Generated", "Failed"]);
export const ediMessageDeliveryStatusSchema = z.enum([
  "Queued",
  "Sending",
  "Sent",
  "Failed",
  "DeadLettered",
]);
export const ediMessageAcknowledgmentStatusSchema = z.enum([
  "NotExpected",
  "Pending",
  "Accepted",
  "AcceptedWithErrors",
  "Rejected",
  "Failed",
]);
export const ediInboundFileStatusSchema = z.enum([
  "Received",
  "Parsed",
  "Processed",
  "PartiallyProcessed",
  "Quarantined",
  "Duplicate",
]);
export type EDIInboundFileStatus = z.infer<typeof ediInboundFileStatusSchema>;
export type EDITransferStatus = z.infer<typeof ediTransferStatusSchema>;
export type EDIMessageDeliveryStatus = z.infer<typeof ediMessageDeliveryStatusSchema>;
export type EDIMessageAcknowledgmentStatus = z.infer<typeof ediMessageAcknowledgmentStatusSchema>;

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
  customerId: z.string().nullish(),
  defaultTransportId: z.string().nullish(),
  defaultMappingProfileId: z.string().nullish(),
  timezone: z.string().nullish(),
  country: z.string(),
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
  customer: z.unknown().nullish(),
  mappingProfile: z.unknown().nullish(),
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
  secretState: nullableArraySchema(ediCommunicationProfileSecretStateSchema),
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
  version: z.number().default(0),
  createdAt: z.number().nullish(),
  updatedAt: z.number().nullish(),
  entries: z.array(ediMappingProfileItemSchema).default([]),
  partner: z.object({ id: z.string(), code: z.string(), name: z.string() }).nullish(),
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
  locationId: z.string().nullish(),
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

export const loadTenderPayloadSchema = z.object({
  purposeCode: z.string().nullish(),
  tenderOfferId: z.string().nullish(),
  shipmentId: z.string().nullish(),
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
  totalChargeAmount: z.unknown().nullish(),
  ratingDetail: z.record(z.string(), z.unknown()).nullish(),
  moves: nullableArraySchema(loadTenderMoveSchema),
  commodities: nullableArraySchema(loadTenderCommoditySchema),
  additionalCharges: nullableArraySchema(loadTenderChargeSchema),
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
const shipmentStatusPayloadSchema = z
  .object({
    shipmentId: z.string().nullish(),
    bol: z.string().nullish(),
    proNumber: z.string().nullish(),
    statusCode: z.string().nullish(),
    statusReasonCode: z.string().nullish(),
    eventDate: nullableNumberSchema,
    eventTime: nullableNumberSchema,
    eventTimeCode: z.string().nullish(),
    stopId: z.string().nullish(),
    stopType: z.string().nullish(),
    stopSequence: nullableNumberSchema,
    locationId: z.string().nullish(),
    locationName: z.string().nullish(),
    locationCode: z.string().nullish(),
    addressLine: z.string().nullish(),
    city: z.string().nullish(),
    stateCode: z.string().nullish(),
    postalCode: z.string().nullish(),
    countryCode: z.string().nullish(),
    appointmentNumber: z.string().nullish(),
    scheduledWindowStart: nullableNumberSchema,
    scheduledWindowEnd: nullableNumberSchema,
    actualArrival: nullableNumberSchema,
    actualDeparture: nullableNumberSchema,
    equipmentNumber: z.string().nullish(),
    equipmentType: z.string().nullish(),
    exceptionCode: z.string().nullish(),
    reasonCode: z.string().nullish(),
    reasonDescription: z.string().nullish(),
    lateMinutes: nullableNumberSchema,
    serviceFailureId: z.string().nullish(),
    serviceFailureNumber: z.string().nullish(),
    serviceFailureReasonCodeId: z.string().nullish(),
    serviceFailureReasonCode: z.string().nullish(),
    references: z.record(z.string(), z.string()).nullish(),
  })
  .catchall(z.unknown());
const tenderResponsePayloadSchema = z.object({}).catchall(z.unknown());
const functionalAcknowledgmentPayloadSchema = z
  .object({ diagnostics: z.array(ediAcknowledgmentDiagnosticSchema).nullish() })
  .catchall(z.unknown());
const implementationAcknowledgmentPayloadSchema = functionalAcknowledgmentPayloadSchema;

export const ediDocumentPayloadSchema = z.object({
  transactionSet: ediTransactionSetSchema.nullish(),
  purposeCode: z.string().nullish(),
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
  sourceBusinessUnitId: z.string().nullish(),
  targetOrganizationId: z.string(),
  targetBusinessUnitId: z.string().nullish(),
  sourcePartnerId: z.string(),
  targetPartnerId: z.string(),
  sourceShipmentId: z.string().nullish(),
  targetShipmentId: z.string().nullish(),
  inboundMessageId: z.string().nullish(),
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
  submittedById: z.string().nullish(),
  approvedById: z.string().nullish(),
  rejectedById: z.string().nullish(),
  canceledById: z.string().nullish(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),
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


export const ediDocumentPreviewSchema = z.object({
  rawX12: z.string(),
  segmentCount: z.number(),
  x12Version: z.string(),
  interchangeControlNumber: z.string(),
  groupControlNumber: z.string(),
  transactionControlNumber: z.string(),
  diagnostics: z.array(ediDiagnosticSchema).default([]),
  partnerName: z.string().nullish(),
  ladingVersionId: z.string().nullish(),
  map: z.string().nullish(),
  document: z.unknown().nullish(),
});

export type EDIDocumentPreview = z.infer<typeof ediDocumentPreviewSchema>;

export const ediConnectionCheckSchema = z.object({
  name: z.string(),
  status: z.enum(["passed", "warning", "failed"]),
  message: z.string().nullish(),
});

export type EDIConnectionCheck = z.infer<typeof ediConnectionCheckSchema>;

export const ediConnectionTestResultSchema = z.object({
  success: z.boolean(),
  checks: z.array(ediConnectionCheckSchema),
});

export type EDIConnectionTestResult = z.infer<typeof ediConnectionTestResultSchema>;

/** What a pickup asked for by hand found: the files it staged and how many it skipped as already read. */
export const ediPollResultSchema = z.object({
  profileId: z.string(),
  stagedFileIds: z
    .array(z.string())
    .nullish()
    .transform((ids) => ids ?? []),
  skippedFiles: z.number(),
});

export type EDIPollResult = z.infer<typeof ediPollResultSchema>;

export const ediCertificateSummarySchema = z.object({
  subject: z.string(),
  issuer: z.string(),
  serialNumber: z.string(),
  notBefore: z.number(),
  notAfter: z.number(),
  expiresInDays: z.number(),
  expired: z.boolean(),
  sha256Fingerprint: z.string(),
});

export type EDICertificateSummary = z.infer<typeof ediCertificateSummarySchema>;

export const ediPartnerReadinessItemSchema = z.object({
  key: z.string(),
  label: z.string(),
  complete: z.boolean(),
});

export const ediPartnerReadinessSchema = z.object({
  partnerId: z.string(),
  ready: z.boolean(),
  completedCount: z.number(),
  totalCount: z.number(),
  items: z.array(ediPartnerReadinessItemSchema),
});

export type EDIPartnerReadiness = z.infer<typeof ediPartnerReadinessSchema>;

export const ediBulkActionResultSchema = z.object({
  succeeded: z.array(z.string()),
  failed: z.array(
    z.object({
      id: z.string(),
      error: z.string(),
    }),
  ),
});

export type EDIBulkActionResult = z.infer<typeof ediBulkActionResultSchema>;

export const ediMessageSchema = z.object({
  id: z.string(),
  businessUnitId: z.string().nullish(),
  organizationId: z.string().nullish(),
  ediPartnerId: z.string(),
  documentTypeId: z.string(),
  ladingVersionId: z.string().nullish(),
  flowInstanceId: z.string().nullish(),
  shipmentId: z.string().nullish(),
  invoiceId: z.string().nullish(),
  transferId: z.string().nullish(),
  inboundFileId: z.string().nullish(),
  direction: ediDocumentDirectionSchema,
  standard: ediStandardSchema,
  transactionSet: ediTransactionSetSchema,
  x12Version: z.string(),
  status: ediMessageStatusSchema,
  interchangeControlNumber: z.string(),
  groupControlNumber: z.string(),
  transactionControlNumber: z.string(),
  segmentCount: z.number(),
  rawX12: z.string(),
  payloadSnapshot: ediDocumentPayloadSchema.nullish(),
  deliveryStatus: z.preprocess(
    (value) => (value === "" ? null : value),
    ediMessageDeliveryStatusSchema.nullish(),
  ),
  deliveryRemotePath: z.string().nullish(),
  deliveryAttempts: z.number().default(0),
  deliveryLastAttemptAt: nullableNumberSchema,
  deliverySentAt: nullableNumberSchema,
  rawPurgedAt: nullableNumberSchema,
  deliveryLastError: z.string().nullish(),
  ackStatus: z.preprocess(
    (value) => (value === "" ? null : value),
    ediMessageAcknowledgmentStatusSchema.nullish(),
  ),
  ackMessageId: z.string().nullish(),
  ackReceivedAt: nullableNumberSchema,
  ackLastError: z.string().nullish(),
  generatedById: z.string().nullish(),
  generatedAt: z.number(),
  diagnosticCount: z.number().default(0),
  partner: ediPartnerSchema.nullish(),
  documentType: ediDocumentTypeSchema.nullish(),
  validationErrors: z.array(ediDiagnosticSchema).default([]),
});

export type EDIMessage = z.infer<typeof ediMessageSchema>;

export const ediInboundFileSchema = z.object({
  id: z.string(),
  businessUnitId: z.string().nullish(),
  organizationId: z.string().nullish(),
  communicationProfileId: z.string(),
  ediPartnerId: z.string().nullish(),
  method: ediConnectionMethodSchema,
  remotePath: z.string(),
  fileName: z.string(),
  checksum: z.string(),
  sizeBytes: z.number().default(0),
  rawContent: z.string().nullish(),
  interchangeControlNumber: z.string().nullish(),
  isaSenderQualifier: z.string().nullish(),
  isaSenderId: z.string().nullish(),
  isaReceiverQualifier: z.string().nullish(),
  isaReceiverId: z.string().nullish(),
  status: ediInboundFileStatusSchema,
  failureReason: z.string().nullish(),
  transactionCount: z.number().default(0),
  receivedAt: z.number(),
  processedAt: nullableNumberSchema,
  rawPurgedAt: nullableNumberSchema,
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),
  partner: ediPartnerSchema.nullish(),
  communicationProfile: ediCommunicationProfileSchema.nullish(),
  messages: z.array(ediMessageSchema).nullish(),
});

export type EDIInboundFile = z.infer<typeof ediInboundFileSchema>;

export const ediInspectionDiagnosticSourceSchema = z.enum([
  "inspection",
  "render",
  "validation",
  "transform",
  "starlark",
  "condition",
  "source_context",
  "partner_setting",
]);

export const ediInspectionDiagnosticSchema = ediDiagnosticSchema.extend({
  source: ediInspectionDiagnosticSourceSchema,
  segmentIndex: z.number().default(0),
});

export type EDIInspectionDiagnostic = z.infer<typeof ediInspectionDiagnosticSchema>;

export const ediX12SeparatorsSchema = z.object({
  element: z.string(),
  segment: z.string(),
  component: z.string(),
  repetition: z.string(),
  source: z.enum(["isa", "envelope", "fallback"]),
  hasConflict: z.boolean().default(false),
});

export const ediX12ComponentSchema = z.object({
  position: z.number(),
  value: z.string(),
  empty: z.boolean(),
});

export const ediX12ElementSchema = z.object({
  position: z.number(),
  label: z.string(),
  value: z.string(),
  empty: z.boolean(),
  required: z.boolean(),
  known: z.boolean(),
  startOffset: z.number(),
  endOffset: z.number(),
  components: z.array(ediX12ComponentSchema).default([]),
});

export const ediX12SegmentSchema = z.object({
  index: z.number(),
  transactionIndex: z.number().default(0),
  segmentId: z.string(),
  name: z.string(),
  type: z.string(),
  loop: z.string().nullish(),
  raw: z.string(),
  rawWithTerminator: z.string(),
  startOffset: z.number(),
  endOffset: z.number(),
  elements: z.array(ediX12ElementSchema).default([]),
  malformed: z.boolean(),
});

export type EDIX12Segment = z.infer<typeof ediX12SegmentSchema>;

export const ediInspectSummarySchema = z.object({
  segmentCount: z.number(),
  groupCount: z.number(),
  transactionCount: z.number(),
  errorCount: z.number(),
  warningCount: z.number(),
  infoCount: z.number(),
});

export const ediX12EnvelopeInspectionSchema = z.object({
  isaControlNumber: z.string().nullish(),
  ieaControlNumber: z.string().nullish(),
  expectedGroups: z.number().default(0),
  actualGroups: z.number(),
});

export const ediX12FunctionalGroupSchema = z.object({
  index: z.number(),
  functionalIdCode: z.string().nullish(),
  gsControlNumber: z.string().nullish(),
  geControlNumber: z.string().nullish(),
  expectedCount: z.number().default(0),
  actualCount: z.number(),
  startSegmentIndex: z.number(),
  endSegmentIndex: z.number().default(0),
});

export const ediX12TransactionSchema = z.object({
  index: z.number(),
  transactionSet: z.string().nullish(),
  stControlNumber: z.string().nullish(),
  seControlNumber: z.string().nullish(),
  expectedSegments: z.number().default(0),
  actualSegments: z.number(),
  startSegmentIndex: z.number(),
  endSegmentIndex: z.number().default(0),
});

/** What the inspector reads of an interchange: its envelope, sets and findings. */
const ediX12InspectionOverviewSchema = z.object({
  transactionSet: z.union([ediTransactionSetSchema, z.literal("")]).nullish(),
  x12Version: z.string().nullish(),
  separators: ediX12SeparatorsSchema,
  summary: ediInspectSummarySchema,
  envelope: ediX12EnvelopeInspectionSchema,
  groups: z.array(ediX12FunctionalGroupSchema).default([]),
  transactions: z.array(ediX12TransactionSchema).default([]),
  diagnostics: z.array(ediInspectionDiagnosticSchema).default([]),
});

export const ediX12InspectionSchema = ediX12InspectionOverviewSchema.extend({
  rawX12: z.string(),
  segments: z.array(ediX12SegmentSchema).default([]),
  formatted: z.string(),
});

export type EDIX12Inspection = z.infer<typeof ediX12InspectionSchema>;

/**
 * An inbound file's interchange as the file card reads it: the overview
 * without the segments, formatted copy or raw text it already holds, and
 * whether the file opened with an ISA at all.
 */
export const ediInboundFileInspectionSchema = ediX12InspectionOverviewSchema.extend({
  interchange: z.boolean(),
});

export type EDIInboundFileInspection = z.infer<typeof ediInboundFileInspectionSchema>;

export const ediMessageInspectionSchema = z.object({
  message: ediMessageSchema,
  inspection: ediX12InspectionSchema,
  provenance: z.object({
    messageId: z.string(),
    ladingVersionId: z.string().nullish(),
    flowInstanceId: z.string().nullish(),
    generatedAt: z.number(),
    generatedById: z.string().nullish(),
  }),
});

export type EDIMessageInspection = z.infer<typeof ediMessageInspectionSchema>;

/** The envelope an inspection reads a file against, as edi.X12EnvelopeSettings marshals it. */
export type EDIX12Envelope = {
  interchangeSenderQualifier: string;
  interchangeSenderId: string;
  interchangeReceiverQualifier: string;
  interchangeReceiverId: string;
  applicationSenderCode: string;
  applicationReceiverCode: string;
  interchangeUsageIndicator: string;
  elementSeparator: string;
  segmentTerminator: string;
  componentSeparator: string;
  repetitionSeparator: string;
};

export type InspectX12Request = {
  rawX12: string;
  transactionSet?: z.infer<typeof ediTransactionSetSchema>;
  x12Version?: string;
  envelope?: Partial<EDIX12Envelope> | null;
  diagnostics?: EDIDiagnostic[];
};


export const ediConnectionListSchema = createLimitOffsetResponse(ediConnectionSchema);
export const ediShipmentLinkListSchema = createLimitOffsetResponse(ediShipmentLinkSchema);
export const ediTransferChangeListSchema = createLimitOffsetResponse(ediTransferChangeSchema);
export const ediMessageListSchema = createLimitOffsetResponse(ediMessageSchema);
export const ediTransferListSchema = createLimitOffsetResponse(ediTransferSchema);

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

export type UpsertEDIPartnerRequest = {
  kind?: z.infer<typeof ediPartnerKindSchema>;
  status?: string;
  code: string;
  name: string;
  description?: string;
  internalOrganizationId?: string;
  ediConnectionId?: string;
  customerId?: string;
  defaultTransportId?: string;
  defaultMappingProfileId?: string;
  timezone?: string;
  country?: string;
  contactName?: string;
  contactEmail?: string;
  contactPhone?: string;
  enabledForInbound?: boolean;
  enabledForOutbound?: boolean;
  settings?: Record<string, unknown>;
  version?: number;
};

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


export type PreviewEDIDocumentRequest = {
  ediPartnerId: string;
  map?: string;
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
  ediPartnerId?: string;
};

export type ApproveEDITransferRequest = {
  mappings: EDIMappingProfileItem[];
};

export type RejectEDITransferRequest = {
  reason: string;
};
