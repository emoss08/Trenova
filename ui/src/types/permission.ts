/* eslint-disable @typescript-eslint/prefer-literal-enum-member */
export type Resource = string;
export type ActionName = string;

export enum StandardOp {
  Create = 1 << 0, // 1
  Read = 1 << 1, // 2
  Update = 1 << 2, // 4
  Delete = 1 << 3, // 8
  Export = 1 << 4, // 16
  Import = 1 << 5, // 32
  Approve = 1 << 6, // 64
  Reject = 1 << 7, // 128
}

export const ACTION_BITS: Record<string, number> = {
  create: StandardOp.Create,
  read: StandardOp.Read,
  update: StandardOp.Update,
  delete: StandardOp.Delete,
  export: StandardOp.Export,
  import: StandardOp.Import,
  approve: StandardOp.Approve,
  reject: StandardOp.Reject,
};

export enum DataScope {
  All = "all",
  Organization = "organization",
  Own = "own",
  None = "none",
}

export enum FieldAccess {
  ReadWrite = "read_write",
  ReadOnly = "read_only",
  WriteOnly = "write_only",
  Hidden = "hidden",
}

export interface ResourceDetail {
  standardOps: number;
  extendedOps?: string[];
  dataScope: DataScope;
  quickCheck: number;
  fieldRules?: FieldRules;
}

export interface FieldRules {
  allowed: string[];
  denied: string[];
  readOnly: string[];
  masked: string[];
  conditional?: Record<string, FieldCondition>;
}

export interface FieldCondition {
  field: string;
  operator: "equals" | "not_equals" | "in" | "not_in";
  value: any;
  access: FieldAccess;
}

export interface PermissionManifest {
  version: string;
  userId: string;
  currentOrg: string;
  availableOrgs: string[];
  computedAt: number;
  expiresAt: number;
  resources: Record<Resource, number | ResourceDetail>;
  checksum: string;
}

export interface BatchPermissionCheck {
  resource: Resource;
  action: ActionName;
}

export interface BatchPermissionResult {
  resource: Resource;
  action: ActionName;
  allowed: boolean;
}

export interface SwitchOrganizationRequest {
  organizationId: string;
}

export interface SwitchOrganizationResponse {
  success: boolean;
  permissions: PermissionManifest;
}

export interface FieldAccessResponse {
  resourceType: Resource;
  fields: Record<string, FieldAccess>;
  rules?: FieldRules;
}
