/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

export enum RoleType {
  System = "System",
  Organization = "Organization",
  Custom = "Custom",
  Temporary = "Temporary",
}

export enum Action {
  // Standard CRUD
  Create = "create", // Create a new resource.
  Read = "read", // Read or view a resource.
  Update = "update", // Update an existing resource.
  Delete = "delete", // Delete an existing resource.

  // Field-level actions
  ModifyField = "modify_field", // Modify specific fields in a resource.
  ViewField = "view_field", // View specific fields in a resource.

  // Workflow actions
  Approve = "approve", // Approve an action or resource.
  Reject = "reject", // Reject an action or resource.
  Submit = "submit", // Submit an action or resource for approval.
  Cancel = "cancel", // Cancel an action or resource.
  Assign = "assign", // Assign a resource to a user or group.
  Reassign = "reassign", // Reassign a resource to a different user or group.
  Complete = "complete", // Mark a resource or action as completed.
  Duplicate = "duplicate", // Duplicate a resource.

  // Configuration actions
  ManageDefaults = "manage_defaults", // Manage default table configurations.
  Share = "share", // Share a table configuration with others.

  // Data actions
  Export = "export", // Export data from the system.
  Import = "import", // Import data into the system.
  Archive = "archive", // Archive a resource.
  Restore = "restore", // Restore an archived resource.

  // Administrative actions
  Manage = "manage", // Perform administrative actions, including full access.
  Audit = "audit", // Audit actions for compliance and review.
  Delegate = "delegate", // Delegate permissions or responsibilities to others.
  Configure = "configure", // Configure system settings or resources.

  // Shipment related actions
  Split = "split", // Split a shipment.
  ReadyToBill = "ready_to_bill", // Mark a shipment as ready to bill.
  ReleaseToBilling = "release_to_billing", // Release a shipment to billing.

  // Billing queue related actions
  BulkTransfer = "bulk_transfer", // Bulk transfer shipments to the billing queue.
  ReviewInvoice = "review_invoice", // Review an invoice.
  PostInvoice = "post_invoice", // Post an invoice.

  // AI related actions
  Classify = "classify", // Classify a resource.
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
