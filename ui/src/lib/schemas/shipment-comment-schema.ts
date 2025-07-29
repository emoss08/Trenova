/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import * as z from "zod/v4";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";
import { userSchema } from "./user-schema";

export const shipmentCommentMentionSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  mentionedUserId: optionalStringSchema,
  mentionedUser: userSchema.nullish(),
});

export const shipmentCommentSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  userId: optionalStringSchema,
  shipmentId: optionalStringSchema,

  comment: z.string().min(1, {
    error: "Comment is required",
  }),
  isHighPriority: z.boolean().default(false),

  // Relationships
  user: userSchema.nullish(),
  mentionedUsers: shipmentCommentMentionSchema.array().nullish(),
});

export type ShipmentCommentSchema = z.infer<typeof shipmentCommentSchema>;
export type ShipmentCommentMentionSchema = z.infer<
  typeof shipmentCommentMentionSchema
>;
