import { z } from "zod";
import { createLimitOffsetResponse } from "./server";

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
  mentionedUser: commentUserSchema.optional(),
});

export type ShipmentCommentMention = z.infer<typeof shipmentCommentMentionSchema>;

export const shipmentCommentSchema = z.object({
  id: z.string(),
  businessUnitId: z.string().optional(),
  organizationId: z.string().optional(),
  shipmentId: z.string(),
  userId: z.string(),
  comment: z.string(),
  type: commentTypeEnum,
  visibility: commentVisibilityEnum,
  priority: commentPriorityEnum,
  source: commentSourceEnum.catch("User"),
  metadata: z.record(z.string(), z.any()).optional(),
  editedAt: z.number().nullable(),
  version: z.number(),
  createdAt: z.number(),
  updatedAt: z.number(),
  mentionedUserIds: z.array(z.string()).optional().default([]),
  user: commentUserSchema.optional(),
  mentionedUsers: z.array(shipmentCommentMentionSchema).optional(),
});

export type ShipmentComment = z.infer<typeof shipmentCommentSchema>;

export const shipmentCommentCreateSchema = z.object({
  comment: z.string().min(1).max(5000),
  mentionedUserIds: z.array(z.string()).max(20).default([]),
  type: commentTypeEnum.default("Internal"),
  visibility: commentVisibilityEnum.default("Internal"),
  priority: commentPriorityEnum.default("Normal"),
});

export type ShipmentCommentCreateInput = z.infer<typeof shipmentCommentCreateSchema>;

export const shipmentCommentUpdateSchema = shipmentCommentCreateSchema.extend({
  id: z.string(),
  version: z.number(),
});

export type ShipmentCommentUpdateInput = z.infer<typeof shipmentCommentUpdateSchema>;

export const shipmentCommentListResponseSchema = createLimitOffsetResponse(shipmentCommentSchema);

export type ShipmentCommentListResponse = z.infer<typeof shipmentCommentListResponseSchema>;

export const shipmentCommentCountResponseSchema = z.object({
  count: z.number(),
});

export type ShipmentCommentCountResponse = z.infer<typeof shipmentCommentCountResponseSchema>;
