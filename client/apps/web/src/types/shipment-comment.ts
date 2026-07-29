import { z } from "zod";
import { createLimitOffsetResponse } from "@trenova/shared/types/server";

export const commentTypeEnum = z.enum([
  "Internal",
  "Dispatch",
  "DriverUpdate",
  "PickupInstruction",
  "DeliveryInstruction",
  "StatusUpdate",
  "Exception",
  "CustomerUpdate",
  "Appointment",
  "Document",
  "Billing",
  "Compliance",
]);
export type CommentType = z.infer<typeof commentTypeEnum>;

export const commentVisibilityEnum = z.enum([
  "Internal",
  "Operations",
  "Customer",
  "Driver",
  "Accounting",
]);
export type CommentVisibility = z.infer<typeof commentVisibilityEnum>;

export const commentPriorityEnum = z.enum(["Low", "Normal", "High", "Urgent"]);
export type CommentPriority = z.infer<typeof commentPriorityEnum>;

export const commentSourceEnum = z.enum(["User", "System", "Integration", "AI"]);
export type CommentSource = z.infer<typeof commentSourceEnum>;

const commentUserSchema = z.object({
  id: z.string(),
  name: z.string(),
  emailAddress: z.string(),
  profilePicUrl: z.string().nullish(),
  thumbnailUrl: z.string().nullish(),
});

export type CommentUser = z.infer<typeof commentUserSchema>;

const shipmentCommentMentionSchema = z.object({
  id: z.string(),
  commentId: z.string(),
  mentionedUserId: z.string(),
  organizationId: z.string().optional(),
  businessUnitId: z.string().optional(),
  shipmentId: z.string().optional(),
  createdAt: z.number(),
  mentionedUser: commentUserSchema.nullish(),
});

export type ShipmentCommentMention = z.infer<typeof shipmentCommentMentionSchema>;

const shipmentCommentAttachmentSchema = z.object({
  documentId: z.string(),
  fileName: z.string(),
  originalName: z.string().nullish(),
  fileSize: z.number(),
  mimeType: z.string(),
  previewUrl: z.string().nullish(),
  downloadUrl: z.string(),
  createdAt: z.number(),
});

export type ShipmentCommentAttachment = z.infer<typeof shipmentCommentAttachmentSchema>;

const shipmentCommentAcknowledgmentSchema = z.object({
  id: z.string(),
  commentId: z.string(),
  userId: z.string(),
  acknowledgedAt: z.number(),
  user: commentUserSchema.nullish(),
});

export type ShipmentCommentAcknowledgment = z.infer<typeof shipmentCommentAcknowledgmentSchema>;

export const commentBodySchema = z.record(z.string(), z.any());

export type CommentBody = z.infer<typeof commentBodySchema>;

export const shipmentCommentSchema = z.object({
  id: z.string(),
  businessUnitId: z.string().optional(),
  organizationId: z.string().optional(),
  shipmentId: z.string(),
  userId: z.string().nullish(),
  parentCommentId: z.string().nullish(),
  replyCount: z.number().optional().default(0),
  comment: z.string(),
  body: commentBodySchema.nullish(),
  type: commentTypeEnum,
  visibility: commentVisibilityEnum,
  priority: commentPriorityEnum,
  source: commentSourceEnum.catch("User"),
  metadata: z.record(z.string(), z.any()).nullish(),
  editedAt: z.number().nullable(),
  pinnedAt: z.number().nullish(),
  pinnedById: z.string().nullish(),
  resolvedAt: z.number().nullish(),
  resolvedById: z.string().nullish(),
  requiresAcknowledgment: z.boolean().optional().default(false),
  deletedAt: z.number().nullish(),
  version: z.number(),
  createdAt: z.number(),
  updatedAt: z.number(),
  mentionedUserIds: z.array(z.string()).optional().default([]),
  user: commentUserSchema.nullish(),
  pinnedBy: commentUserSchema.nullish(),
  resolvedBy: commentUserSchema.nullish(),
  mentionedUsers: z.array(shipmentCommentMentionSchema).nullish(),
  acknowledgments: z.array(shipmentCommentAcknowledgmentSchema).nullish(),
  attachments: z.array(shipmentCommentAttachmentSchema).nullish(),
});

export type ShipmentComment = z.infer<typeof shipmentCommentSchema>;

export const shipmentCommentCreateSchema = z.object({
  comment: z.string().min(1).max(5000),
  body: commentBodySchema.optional(),
  parentCommentId: z.string().optional(),
  mentionedUserIds: z.array(z.string()).max(20).default([]),
  attachmentDocumentIds: z.array(z.string()).max(10).default([]),
  type: commentTypeEnum.default("Internal"),
  visibility: commentVisibilityEnum.default("Internal"),
  priority: commentPriorityEnum.default("Normal"),
  requiresAcknowledgment: z.boolean().optional(),
  clientRef: z.string().optional(),
});

export type ShipmentCommentCreateInput = z.infer<typeof shipmentCommentCreateSchema>;

export const shipmentCommentUpdateSchema = z.object({
  id: z.string(),
  comment: z.string().min(1).max(5000),
  body: commentBodySchema.optional(),
  mentionedUserIds: z.array(z.string()).max(20).default([]),
  attachmentDocumentIds: z.array(z.string()).max(10).default([]),
  type: commentTypeEnum.default("Internal"),
  visibility: commentVisibilityEnum.default("Internal"),
  priority: commentPriorityEnum.default("Normal"),
  requiresAcknowledgment: z.boolean().optional(),
  version: z.number(),
});

export type ShipmentCommentUpdateInput = z.infer<typeof shipmentCommentUpdateSchema>;

export const shipmentCommentFilterSchema = z.object({
  types: z.array(commentTypeEnum).optional(),
  priorities: z.array(commentPriorityEnum).optional(),
  authorIds: z.array(z.string()).optional(),
  mentionsUserId: z.string().optional(),
  pinnedOnly: z.boolean().optional(),
  unresolvedOnly: z.boolean().optional(),
  search: z.string().optional(),
});

export type ShipmentCommentFilter = z.infer<typeof shipmentCommentFilterSchema>;

export const shipmentCommentListResponseSchema = createLimitOffsetResponse(shipmentCommentSchema);

export type ShipmentCommentListResponse = z.infer<typeof shipmentCommentListResponseSchema>;

export const shipmentCommentCountResponseSchema = z.object({
  count: z.number(),
});

export type ShipmentCommentCountResponse = z.infer<typeof shipmentCommentCountResponseSchema>;
