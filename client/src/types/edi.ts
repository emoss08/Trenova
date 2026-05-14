import { z } from "zod";
import { createLimitOffsetResponse } from "./server";

export const ediPartnerKindSchema = z.enum(["Internal", "External"]);
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
  "Canceled",
  "Failed",
]);

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
});

export type EDIPartner = z.infer<typeof ediPartnerSchema>;

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
  description: z.string().nullish(),
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

const loadTenderPayloadSchema = z.object({
  shipmentId: z.string(),
  bol: z.string().nullish(),
  pieces: z.number().nullish(),
  weight: z.number().nullish(),
  moves: z.array(z.unknown()).default([]),
  commodities: z.array(z.unknown()).default([]),
  additionalCharges: z.array(z.unknown()).default([]),
});

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

export const ediPartnerListSchema = createLimitOffsetResponse(ediPartnerSchema);
export const ediTransferListSchema = createLimitOffsetResponse(ediTransferSchema);
export const ediPartnerSelectOptionListSchema = createLimitOffsetResponse(ediPartnerSchema);

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
