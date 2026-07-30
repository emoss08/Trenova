import { z } from "zod";
import { stringArraySchema } from "./helpers";
import { roleSummaryArraySchema } from "./role";

export const Operation = {
  Read: 1 << 0,
  Create: 1 << 1,
  Update: 1 << 2,
  Delete: 1 << 3,
  Export: 1 << 4,
  Import: 1 << 5,
  Approve: 1 << 8,
  Reject: 1 << 9,
  Assign: 1 << 10,
  Unassign: 1 << 11,
  Archive: 1 << 12,
  Restore: 1 << 13,
  Submit: 1 << 14,
  Cancel: 1 << 15,
  Duplicate: 1 << 16,
  Close: 1 << 17,
  Lock: 1 << 18,
  Unlock: 1 << 19,
  Activate: 1 << 20,
  Reopen: 1 << 21,
  Pin: 1 << 22,
  Unpin: 1 << 23,
  Resolve: 1 << 24,
  Manage: 1 << 25,
} as const;

export type OperationType = (typeof Operation)[keyof typeof Operation];

export const FieldSensitivity = z.enum(["public", "internal", "restricted", "confidential"]);

export type FieldSensitivityType = z.infer<typeof FieldSensitivity>;

export const orgSummarySchema = z.object({
  id: z.string(),
  name: z.string(),
});

export type OrgSummary = z.infer<typeof orgSummarySchema>;

export const permissionManifestSchema = z.object({
  version: z.string(),
  userId: z.string(),
  organizationId: z.string(),
  activeRoleIds: stringArraySchema,
  authorizedRoleIds: stringArraySchema,
  activeRoles: roleSummaryArraySchema,
  authorizedRoles: roleSummaryArraySchema,
  requiresRoleActivation: z.boolean().optional().default(false),
  maxSensitivity: FieldSensitivity,
  permissions: z.record(z.string(), z.number()),
  routeAccess: z.record(z.string(), z.boolean()),
  availableOrgs: z.array(orgSummarySchema),
  checksum: z.string(),
  expiresAt: z.number(),
});

export type PermissionManifest = z.infer<typeof permissionManifestSchema>;

export const permissionVersionSchema = z.object({
  checksum: z.string(),
  expiresAt: z.number(),
});

export type PermissionVersion = z.infer<typeof permissionVersionSchema>;

// Resource is generated from the Go permission resource enum — see
// types/generated/permission-resources.ts. Re-exported here so the long-standing
// `@trenova/shared/types/permission` import path stays the one place callers reach for.
export { Resource } from "@trenova/shared/types/generated/permission-resources";
export type { ResourceType } from "@trenova/shared/types/generated/permission-resources";
