export enum RoleType {
  System = "System",
  Organization = "Organization",
  Custom = "Custom",
  Temporary = "Temporary",
}

export enum Action {
  Create = "create",
  Read = "read",
  Update = "update",
  Delete = "delete",
  ModifyField = "modify_field",
  ViewField = "view_field",
  Approve = "approve",
  Reject = "reject",
  Submit = "submit",
  Cancel = "cancel",
  Assign = "assign",
  Reassign = "reassign",
  Complete = "complete",
  Export = "export",
  Import = "import",
  Archive = "archive",
  Restore = "restore",
  Manage = "manage",
  Audit = "audit",
  Delegate = "delegate",
  Configure = "configure",
}

export enum Scope {
  Global = "global",
  BusinessUnit = "business_unit",
  Organization = "organization",
  Personal = "personal",
}

export enum AuditLevel {
  None = "none",
  Changes = "changes",
  Access = "access",
  Full = "full",
}

export enum ConditionType {
  Field = "field",
  Time = "time",
  Role = "role",
  Ownership = "ownership",
  Custom = "custom",
}

export enum PermissionStatus {
  Active = "Active",
  Inactive = "Inactive",
  Suspended = "Suspended",
  Archived = "Archived",
}
