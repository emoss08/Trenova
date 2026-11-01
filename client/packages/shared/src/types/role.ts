import { z } from "zod";
import { optionalStringSchema, timestampSchema } from "./helpers";
import { translate } from "@trenova/shared/i18n/runtime";
import { DATA_SCOPES, PERMISSION_OPERATIONS } from "./generated/permission-resources";

export const fieldSensitivitySchema = z.enum(["public", "internal", "restricted", "confidential"]);
export type FieldSensitivity = z.infer<typeof fieldSensitivitySchema>;

export const dataScopeSchema = z.enum(DATA_SCOPES);
export type DataScope = z.infer<typeof dataScopeSchema>;

export const operationSchema = z.enum(PERMISSION_OPERATIONS);
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
  name: z
    .string()
    .min(1, { error: () => translate("Name is required") })
    .max(255),
  description: optionalStringSchema,
  coreResponsibility: coreResponsibilitySchema.nullish(),
  parentRoleIds: z.array(z.string()).nullish(),
  maxSensitivity: fieldSensitivitySchema,
  isSystem: z.boolean().optional(),
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
  // Left as `number | undefined` rather than defaulted to 0: the permission manifest is
  // returned unparsed by permission-api and is also rehydrated from localStorage, so a
  // summary can genuinely reach the UI without this field. Callers must distinguish
  // "no permissions" from "the server did not say".
  permissionCount: z
    .number()
    .nullish()
    .transform((value) => value ?? undefined)
    .optional(),
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
  role: z
    .object({
      id: optionalStringSchema,
      name: z.string(),
      description: z.string().nullish(),
      isSystem: z.boolean().optional(),
    })
    .nullish(),
});
export type UserRoleAssignment = z.infer<typeof userRoleAssignmentSchema>;

export const addPermissionSchema = z.object({
  resource: z.string().min(1, { error: () => translate("Resource is required") }),
  operations: z
    .array(operationSchema)
    .min(1, { error: () => translate("At least one operation is required") }),
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
  userId: z.string().min(1, { error: () => translate("User is required") }),
  expiresAt: z.number().nullable().optional(),
});
export type AssignRole = z.infer<typeof assignRoleSchema>;

export const roleImpactSchema = z.object({
  userId: z.string(),
  userName: z.string(),
  organizationId: z.string(),
  orgName: z.string(),
  assignmentType: z.string(),
});
export type RoleImpact = z.infer<typeof roleImpactSchema>;
