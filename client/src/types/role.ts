import { z } from "zod";
import { optionalStringSchema, timestampSchema } from "./helpers";

export const fieldSensitivitySchema = z.enum(["public", "internal", "restricted", "confidential"]);
export type FieldSensitivity = z.infer<typeof fieldSensitivitySchema>;

export const dataScopeSchema = z.enum(["own", "organization", "all"]);
export type DataScope = z.infer<typeof dataScopeSchema>;

export const operationSchema = z.enum([
  "read",
  "create",
  "update",
  "export",
  "import",
  "approve",
  "reject",
  "assign",
  "unassign",
  "archive",
  "restore",
  "submit",
  "cancel",
  "duplicate",
]);
export type Operation = z.infer<typeof operationSchema>;

export const resourcePermissionSchema = z.object({
  id: optionalStringSchema,
  roleId: optionalStringSchema,
  resource: z.string(),
  operations: z.array(operationSchema),
  dataScope: dataScopeSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
});
export type ResourcePermission = z.infer<typeof resourcePermissionSchema>;

export const coreResponsibilitySchema = z.enum(["Billing", "Operations", "Finance", "Leadership"]);
export type CoreResponsibility = z.infer<typeof coreResponsibilitySchema>;

export const roleSchema = z.object({
  id: optionalStringSchema,
  organizationId: optionalStringSchema,
  name: z.string().min(1, "Name is required").max(255),
  description: optionalStringSchema,
  coreResponsibility: coreResponsibilitySchema.nullish(),
  parentRoleIds: z.array(z.string()).nullish(),
  maxSensitivity: fieldSensitivitySchema,
  isSystem: z.boolean().optional(),
  isOrgAdmin: z.boolean().optional(),
  isBusinessUnitAdmin: z.boolean().optional(),
  createdBy: optionalStringSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  permissions: z.array(resourcePermissionSchema).optional(),
});
export type Role = z.infer<typeof roleSchema>;

export const roleSummarySchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z
    .string()
    .nullish()
    .transform((value) => value ?? ""),
  isSystem: z.boolean().optional().default(false),
  isOrgAdmin: z.boolean().optional().default(false),
  isBusinessUnitAdmin: z.boolean().optional().default(false),
});
export type RoleSummary = z.infer<typeof roleSummarySchema>;

export const roleSummaryArraySchema = z
  .array(roleSummarySchema)
  .nullish()
  .transform((value) => value ?? []);

export const userRoleAssignmentSchema = z.object({
  id: optionalStringSchema,
  userId: z.string(),
  organizationId: optionalStringSchema,
  roleId: z.string(),
  expiresAt: z.number().nullable().optional(),
  assignedBy: optionalStringSchema,
  assignedAt: timestampSchema,
  role: roleSchema.optional(),
});
export type UserRoleAssignment = z.infer<typeof userRoleAssignmentSchema>;

export const addPermissionSchema = z.object({
  resource: z.string().min(1, "Resource is required"),
  operations: z.array(operationSchema).min(1, "At least one operation is required"),
  dataScope: dataScopeSchema,
});
export type AddPermission = z.infer<typeof addPermissionSchema>;

export const createRoleSchema = roleSchema
  .omit({
    id: true,
    organizationId: true,
    createdBy: true,
    createdAt: true,
    updatedAt: true,
  })
  .extend({
    permissions: z.array(addPermissionSchema).optional(),
  });
export type CreateRole = z.infer<typeof createRoleSchema>;

export const assignRoleSchema = z.object({
  userId: z.string().min(1, "User is required"),
  expiresAt: z.number().nullable().optional(),
});
export type AssignRole = z.infer<typeof assignRoleSchema>;

export const roleImpactSchema = z.object({
  userId: z.string(),
  userName: z.string(),
  email: z.string(),
});
export type RoleImpact = z.infer<typeof roleImpactSchema>;

export const roleHierarchyEdgeSchema = z.object({
  id: z.string(),
  seniorRoleId: z.string(),
  juniorRoleId: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  createdBy: optionalStringSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  seniorRole: roleSchema.optional(),
  juniorRole: roleSchema.optional(),
});
export type RoleHierarchyEdge = z.infer<typeof roleHierarchyEdgeSchema>;

export const roleConstraintTypeSchema = z.enum(["ssd", "dsd"]);
export type RoleConstraintType = z.infer<typeof roleConstraintTypeSchema>;

export const roleConstraintSchema = z.object({
  id: optionalStringSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  name: z.string().min(1, "Name is required"),
  description: optionalStringSchema,
  type: roleConstraintTypeSchema,
  maxRoles: z.number().int().min(1),
  enabled: z.boolean().default(true),
  createdBy: optionalStringSchema,
  createdAt: timestampSchema.optional(),
  updatedAt: timestampSchema.optional(),
  roles: z.array(roleSchema).optional(),
});
export type RoleConstraint = z.infer<typeof roleConstraintSchema>;

export const saveRoleConstraintSchema = roleConstraintSchema
  .pick({
    id: true,
    name: true,
    description: true,
    type: true,
    maxRoles: true,
    enabled: true,
  })
  .extend({
    roleIds: z.array(z.string()).min(2, "Select at least two roles"),
  });
export type SaveRoleConstraint = z.infer<typeof saveRoleConstraintSchema>;
