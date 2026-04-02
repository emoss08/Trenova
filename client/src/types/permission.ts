import { z } from "zod";

export const Operation = {
  Read: 1 << 0,
  Create: 1 << 1,
  Update: 1 << 2,
  Export: 1 << 3,
  Import: 1 << 4,
  Approve: 1 << 8,
  Reject: 1 << 9,
  Assign: 1 << 10,
  Unassign: 1 << 11,
  Archive: 1 << 12,
  Restore: 1 << 13,
  Submit: 1 << 14,
  Cancel: 1 << 15,
  Duplicate: 1 << 16,
} as const;

export type OperationType = (typeof Operation)[keyof typeof Operation];

export const FieldSensitivity = z.enum([
  "public",
  "internal",
  "restricted",
  "confidential",
]);

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
  isPlatformAdmin: z.boolean(),
  isOrgAdmin: z.boolean(),
  isBusinessUnitAdmin: z.boolean().optional().default(false),
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

export const Resource = {
  Organization: "organization",
  BusinessUnit: "business_unit",
  User: "user",
  Role: "role",
  AuditLog: "audit_log",
  TableConfiguration: "table_configuration",
  SequenceConfig: "sequence_config",
  Integration: "integration",
  EquipmentType: "equipment_type",
  EquipmentManufacturer: "equipment_manufacturer",
  Trailer: "trailer",
  Tractor: "tractor",
  FleetCode: "fleet_code",
  Worker: "worker",
  WorkerPTO: "worker_pto",
  Shipment: "shipment",
  ShipmentControl: "shipment_control",
  DocumentControl: "document_control",
  ShipmentType: "shipment_type",
  ShipmentMove: "shipment_move",
  ShipmentStop: "shipment_stop",
  DataEntryControl: "data_entry_control",
  DispatchControl: "dispatch_control",
  Invoice: "invoice",
  AccessorialCharge: "accessorial_charge",
  ChargeType: "charge_type",
  RevenueCode: "revenue_code",
  FormulaTemplate: "formula_template",
  Customer: "customer",
  CustomerContact: "customer_contact",
  Location: "location",
  LocationCategory: "location_category",
  Commodity: "commodity",
  HazardousMaterial: "hazardous_material",
  HazmatSegregationRule: "hazmat_segregation_rule",
  AccountingControl: "accounting_control",
  BillingControl: "billing_control",
  AccountType: "account_type",
  GeneralLedgerAccount: "general_ledger_account",
  DivisionCode: "division_code",
  Qualification: "qualification",
  DocumentClassification: "document_classification",
  DocumentType: "document_type",
  DistanceOverride: "distance_override",
  HoldReason: "hold_reason",
  ServiceType: "service_type",
  DelayCode: "delay_code",
  ReasonCode: "reason_code",
  CommentType: "comment_type",
  Tag: "tag",
  Report: "report",
  Dashboard: "dashboard",
  CustomFieldDefinition: "custom_field_definition",
  TableChangeAlert: "table_change_alert",
  FiscalYear: "fiscal_year",
  FiscalPeriod: "fiscal_period",
} as const;

export type ResourceType = (typeof Resource)[keyof typeof Resource];
