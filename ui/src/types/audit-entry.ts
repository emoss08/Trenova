import { BaseModel } from "./common";
import { User } from "./user";

export interface AuditEntry extends BaseModel {
  id: string;
  resourceId: string;
  userId: string;
  businessUnitId: string;
  organizationId: string;
  correlationId: string;
  timestamp: number;
  changes: Record<string, any>;
  previousState: Record<string, any>;
  currentState: Record<string, any>;
  metadata: Record<string, any>;
  resource: Resource;
  action: AuditEntryAction;
  userAgent: string;
  comment: string;
  sensitiveData: boolean;
  category: string;
  critical: boolean;
  ipAddress: string;

  // * Relations
  user: User;
}

export enum AuditEntryAction {
  Create = "create",
  Update = "update",
  Delete = "delete",
  View = "view",
  Approve = "approve",
  Reject = "reject",
  Submit = "submit",
  Cancel = "cancel",
  Assign = "assign",
  Reassign = "reassign",
  Complete = "complete",
  Duplicate = "duplicate",
  ManageDefaults = "manage_defaults",
  Share = "share",
  Export = "export",
  Import = "import",
  Archive = "archive",
  Restore = "restore",
  Manage = "manage",
  Audit = "audit",
  Delegate = "delegate",
  Configure = "configure",
  Split = "split",
}

export enum Resource {
  Dashboard = "dashboard",
  BillingManagement = "billing_management",
  BillingClient = "billing_client",
  ConfigurationFiles = "configuration_files",
  RateManagement = "rate_management",
  SystemLog = "system_log",
  ShipmentControl = "shipment_control",
  DataRetention = "data_retention",
  Equipment = "equipment",
  Maintenance = "maintenance",
  User = "user",
  FormulaTemplate = "formula_template",
  ShipmentManagement = "shipment_management",
  Route = "route",
  CommentType = "comment_type",
  DelayCode = "delay_code",
  BusinessUnit = "business_unit",
  Organization = "organization",
  DocumentQualityConfig = "document_quality_config",
  ChargeType = "charge_type",
  DivisionCode = "division_code",
  GlAccount = "gl_account",
  RevenueCode = "revenue_code",
  AccessorialCharge = "accessorial_charge",
  DocumentClassification = "document_classification",
  Worker = "worker",
  Tractor = "tractor",
  Trailer = "trailer",
  Shipment = "shipment",
  Assignment = "assignment",
  ShipmentMove = "shipment_move",
  Stop = "stop",
  FleetCode = "fleet_code",
  EquipmentType = "equipment_type",
  EquipmentManufacturer = "equipment_manufacturer",
  ShipmentType = "shipment_type",
  ServiceType = "service_type",
  HazardousMaterial = "hazardous_material",
  Commodity = "commodity",
  LocationCategory = "location_category",
  Location = "location",
  Customer = "customer",
  Invoice = "invoice",
  Dispatch = "dispatch",
  Report = "report",
  AuditEntries = "audit_entries",
  TableConfiguration = "table_configuration",
  Integration = "integration",
  Setting = "setting",
  Template = "template",
  Backup = "backup",
}
