import { z } from "zod";
import { createLimitOffsetResponse } from "./server";
import { userSchema } from "./user";

export const auditCategorySchema = z.enum(["System", "User"]);
export type AuditCategory = z.infer<typeof auditCategorySchema>;

export const auditChangeSchema = z.object({
  from: z.unknown().optional(),
  to: z.unknown().optional(),
  type: z.string().optional(),
  fieldType: z.string().optional(),
  path: z.string().optional(),
});

export type AuditChange = z.infer<typeof auditChangeSchema>;

export const auditEntrySchema = z.object({
  id: z.string(),
  userId: z.string(),
  businessUnitId: z.string(),
  organizationId: z.string(),
  timestamp: z.number(),
  changes: z.record(z.string(), z.union([auditChangeSchema, z.unknown()])).default({}),
  previousState: z.record(z.string(), z.unknown()).default({}),
  currentState: z.record(z.string(), z.unknown()).default({}),
  metadata: z.record(z.string(), z.unknown()).default({}),
  resource: z.string(),
  operation: z.string(),
  resourceId: z.string(),
  correlationId: z.string().optional(),
  userAgent: z.string().optional(),
  comment: z.string().optional(),
  ipAddress: z.string().optional(),
  category: auditCategorySchema.default("System"),
  sensitiveData: z.boolean().default(false),
  critical: z.boolean().default(false),
  user: userSchema.partial().optional(),
});

export const listByResourceIdSchema = createLimitOffsetResponse(auditEntrySchema);

export type ListByResourceIdResponse = z.infer<typeof listByResourceIdSchema>;
export type AuditEntry = z.infer<typeof auditEntrySchema>;
export type AuditChanges = AuditEntry["changes"];
