import { Status } from "@/types/common";
import { ConditionType } from "@/types/roles-permissions";
import { TimeFormat } from "@/types/user";
import { z } from "zod";
import { organizationSchema } from "./organization-schema";

const fieldPermissionSchema = z.object({
  field: z.string().min(1, "Field is required"),
  action: z.string().optional(),
  scope: z.string().optional(),
  validationRules: z.record(z.any()).optional(),
  mask: z.string().optional(),
  auditLevel: z.string().optional(),
});

export type FieldPermissionSchema = z.infer<typeof fieldPermissionSchema>;

const conditionSchema = z.object({
  type: z.nativeEnum(ConditionType),
  field: z.string().min(1, "Field is required"),
  operator: z.string().min(1, "Operator is required"),
  value: z.any(),
  values: z.array(z.any()).optional(),
  description: z.string().optional(),
  errorMessage: z.string().optional(),
  priority: z.number().min(0, "Priority must be non-negative"),
  metadata: z.record(z.any()).optional(),
});

export type ConditionSchema = z.infer<typeof conditionSchema>;

const permissionSchema = z.object({
  id: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

  resource: z.string().optional(),
  action: z.string().optional(),
  scope: z.string().optional(),
  description: z.string().optional(),
  isSystemLevel: z.boolean(),
  fieldPermissions: z.array(fieldPermissionSchema).nullable().optional(),
  conditions: z.array(conditionSchema).nullable().optional(),
  dependencies: z.array(z.string()).optional(),
  customSettings: z.record(z.any()).optional(),
});

export type PermissionSchema = z.infer<typeof permissionSchema>;

export const roleSchema = z.object({
  id: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

  name: z.string().min(1, "Name is required"),
  description: z.string().min(1, "Description is required"),
  roleType: z.string().optional(),
  isSystem: z.boolean(),
  priority: z.number().min(0, "Priority must be non-negative"),
  status: z.nativeEnum(Status),
  expiresAt: z.number().optional(),

  businessUnitId: z.string().min(1, "Business unit ID is required"),
  organizationId: z.string().min(1, "Organization ID is required"),
  parentRoleId: z.string().optional(),
  permissions: z.array(permissionSchema),
});

export type RoleSchema = z.infer<typeof roleSchema>;

export const rolesPermissionsSchema = z.object({
  roles: z.array(roleSchema),
  permissions: z.array(permissionSchema),
});

export type RolesPermissionSchema = z.infer<typeof rolesPermissionsSchema>;

export const userSchema = z.object({
  id: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),
  currentOrganizationId: z
    .string()
    .min(1, "Current organization ID is required"),

  status: z.nativeEnum(Status),
  name: z
    .string()
    .min(1, "Name is required")
    .regex(
      /^[a-zA-Z]+(\s[a-zA-Z]+)*$/,
      "Name can only contain letters and spaces",
    ),
  username: z
    .string()
    .min(1, "Username is required")
    .max(20, "Username must be less than 20 characters")
    .regex(/^[a-zA-Z0-9]+$/, "Username must be alphanumeric"),
  emailAddress: z.string().email("Invalid email address"),
  profilePicUrl: z.string().optional(),
  thumbnailUrl: z.string().optional(),
  timezone: z.string().min(1, "Timezone is required"),
  timeFormat: z.nativeEnum(TimeFormat).optional(),
  isLocked: z.boolean(),
  lastLoginAt: z.number().optional(),
  mustChangePassword: z.boolean(),

  // Relationships
  currentOrganization: organizationSchema.optional(),
  organizations: organizationSchema.array().optional(),
  roles: roleSchema.array().optional(),
});

export type UserSchema = z.infer<typeof userSchema>;

export const bulkCreateUserSchema = z.object({
  users: z.array(userSchema).min(1, "At least one user is required"),
});

export type BulkCreateUserSchema = z.infer<typeof bulkCreateUserSchema>;
