import { User } from "./user";

export type Resource =
  | "user"
  | "business_unit"
  | "organization"
  | "worker"
  | "tractor"
  | "trailer"
  | "shipment"
  | "movement"
  | "stop"
  | "route"
  | "invoice"
  | "dispatch"
  | "report"
  | "audit_log"
  | "integration"
  | "setting"
  | "template";

export type Action =
  | "create"
  | "read"
  | "update"
  | "delete"
  | "modify_field"
  | "view_field"
  | "approve"
  | "reject"
  | "submit"
  | "cancel"
  | "assign"
  | "reassign"
  | "complete"
  | "export"
  | "import"
  | "archive"
  | "restore"
  | "manage"
  | "audit"
  | "delegate"
  | "configure";

export type Scope = "global" | "business_unit" | "organization" | "personal";

export type AuditLevel = "none" | "changes" | "access" | "full";

export type Operator =
  | "eq"
  | "neq"
  | "gt"
  | "lt"
  | "gte"
  | "lte"
  | "contains"
  | "not_contains"
  | "startsWith"
  | "endsWith"
  | "in"
  | "not_in"
  | "isNull"
  | "isNotNull"
  | "custom";

export type ConditionType = "field" | "time" | "role" | "ownership" | "custom";

export type RoleType = "system" | "organization" | "custom" | "temporary";

export type PermissionStatus = "Active" | "Inactive" | "Suspended" | "Archived";

export interface Condition {
  type: ConditionType;
  field: string;
  operator: Operator;
  value?: any; // Value can be nullable
  values?: any[]; // Optional array of values
  description?: string; // Optional description
  errorMessage?: string; // Optional custom error message
  priority: number; // Evaluation priority
  metadata?: Record<string, unknown>; // Additional metadata
}

export interface FieldPermission {
  field: string;
  actions: Action[];
  conditions?: Condition[]; // Optional conditions
  validationRules?: string; // Optional validation rules
  mask?: string; // Optional data masking
  auditLevel?: AuditLevel; // Optional audit level
}

export interface Permission {
  id: string;
  resource: Resource;
  action: Action;
  scope: Scope;
  description?: string;
  isSystemLevel: boolean;
  fieldPermissions?: FieldPermission[]; // Optional field permissions
  conditions?: Condition[]; // Optional conditions
  dependencies?: string[]; // Optional dependencies
  customSettings?: Record<string, unknown>; // Optional custom settings
  createdAt: number;
  updatedAt: number;
}

export interface Role {
  id: string;
  name: string;
  description?: string; // Optional description
  roleType: RoleType;
  isSystem: boolean;
  priority: number;
  status: "Active" | "Inactive" | "Suspended"; // Role status
  expiresAt?: number; // Nullable expiration time
  createdAt: number;
  updatedAt: number;
  businessUnitId: string;
  organizationId: string;
  parentRoleId?: string; // Nullable parent role ID
  permissions?: Permission[]; // Optional permissions
  parentRole?: Role; // Optional parent role
  childRoles?: Role[]; // Optional child roles
  metadata?: Record<string, unknown>; // Optional metadata
}

export interface RolePermission {
  roleId: string;
  permissionId: string;
  role?: Role; // Optional role
  permission?: Permission; // Optional permission
}

export interface UserRole {
  userId: string;
  roleId: string;
  user?: User; // Replace with User interface if defined elsewhere
  role?: Role; // Optional role
}

export interface PermissionGrant {
  id: string;
  status: PermissionStatus;
  expiresAt?: number; // Nullable expiration timestamp
  revokedAt?: number; // Nullable revoked timestamp
  createdAt: number;
  reason?: string; // Optional reason
  fieldOverrides?: FieldPermission[]; // Optional field overrides
  conditions?: Condition[]; // Optional conditions
  resourceId?: string; // Nullable resource ID
  auditTrail?: Record<string, unknown>; // Optional audit trail
  organizationId: string;
  businessUnitId: string;
  userId: string;
  permissionId: string;
  grantedBy: string;
  revokedBy?: string; // Nullable revoker
  user?: unknown; // Replace with User interface if defined elsewhere
  permission?: Permission; // Optional permission
  grantor?: unknown; // Replace with User interface if defined elsewhere
  revoker?: unknown; // Replace with User interface if defined elsewhere
}

export interface PermissionTemplate {
  id: string;
  name: string;
  description?: string; // Optional description
  permissions: Permission[];
  fieldSettings: FieldPermission[];
  isSystem: boolean;
  createdAt: number;
  updatedAt: number;
}
