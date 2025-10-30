import type { UserSchema } from "@/lib/schemas/user-schema";
import { PermissionOperation } from "./_gen/permissions";

export type AuditEntryResponse = {
  items: AuditEntry[];
  total: number;
};

// TODO: Remove this type and convert to zod schema
export interface AuditEntry {
  id: string;
  resourceId: string;
  businessUnitId: string;
  organizationId: string;
  userId: string;
  correlationId: string;
  timestamp: number;
  changes: Record<string, any>;
  previousState: Record<string, any>;
  currentState: Record<string, any>;
  metadata: Record<string, any>;
  resource: Resource;
  operation: PermissionOperation;
  userAgent: string;
  comment: string;
  sensitiveData: boolean;
  category: string;
  critical: boolean;
  ipAddress: string;

  // * Relations
  user: UserSchema;
}

export enum Resource {
  // Core resources
  User = "user", // Represents user management resources.
  BusinessUnit = "business_unit", // Represents resources related to business units.
  Organization = "organization", // Represents resources related to organizations.
  DocumentQualityConfig = "document_quality_config", // Represents resources related to document quality config.
  ShipmentControl = "shipment_control", // Represents resources related to shipment control.
  DispatchControl = "dispatch_control", // Represents resources related to dispatch control.
  ConsolidationSettings = "consolidation_settings", // Represents resources related to consolidation settings.
  PatternConfig = "pattern_config", // Represents resources related to pattern config.
  GoogleMapsConfig = "google_maps_config", // Represents resources related to google maps config.
  BillingControl = "billing_control", // Represents resources related to billing control.
  DistanceOverride = "distance_override", // Represents resources related to distance overrides.
  Document = "document", // Represents resources related to documents.
  DedicatedLane = "dedicated_lane", // Represents resources related to dedicated lanes.
  DedicatedLaneSuggestion = "dedicated_lane_suggestion", // Represents resources related to dedicated lane suggestions.
  Role = "role", // Represents resources related to roles.
  PageFavorite = "page_favorite", // Represents resources related to page favorites.
  FiscalYear = "fiscal_year", // Represents resources related to fiscal years.

  // Operations resources
  Worker = "worker", // Represents resources related to workers.
  WorkerPTO = "worker_pto", // Represents resources related to worker PTOs.
  Consolidation = "consolidation", // Represents resources related to consolidation groups.
  Tractor = "tractor", // Represents resources for managing tractors.
  Trailer = "trailer", // Represents resources for managing trailers.
  Shipment = "shipment", // Represents resources for managing shipments.
  BillingQueue = "billing_queue", // Represents resources for managing billing queue.
  Assignment = "assignment", // Represents resources for managing assignments.
  ShipmentMove = "shipment_move", // Represents resources for managing movements.
  ShipmentComment = "shipment_comment", // Represents resources for managing shipment comments.
  ShipmentHold = "shipment_hold", // Represents resources for managing shipment holds.
  HoldReason = "hold_reason", // Represents resources for managing hold reasons.
  Stop = "stop", // Represents resources for managing stops.
  FleetCode = "fleet_code", // Represents resources for managing fleet codes.
  EquipmentType = "equipment_type", // Represents resources for managing equipment types.
  EquipmentManufacturer = "equipment_manufacturer", // Represents resources for managing equipment manufacturers.
  ShipmentType = "shipment_type", // Represents resources for managing shipment type.
  AccountType = "account_type", // Represents resources for managing account types.
  ServiceType = "service_type", // Represents resources for managing service types.
  HazardousMaterial = "hazardous_material", // Represents resources for managing hazardous materials.
  Commodity = "commodity", // Represents resources for managing commodities.
  LocationCategory = "location_category", // Represents resources for managing location categories.
  Location = "location", // Represents resources for managing locations.
  Customer = "customer", // Represents resources for managing customers.
  HazmatSegregationRule = "hazmat_segregation_rule", // Represents resources for managing hazmat segregation rules.

  // Financial resources
  Invoice = "invoice", // Represents resources related to invoices.
  AccessorialCharge = "accessorial_charge", // Represents resources related to accessorial charges.
  DocumentType = "document_type", // Represents resources related to document types.

  // Management resources
  Dispatch = "dispatch", // Represents resources for dispatch management.
  Report = "report", // Represents resources for managing reports.
  AuditEntry = "audit_entry", // Represents resources for tracking and auditing logs.
  AuditLog = "audit_log", // Represents resources for tracking and auditing logs.

  // System resources
  TableConfiguration = "table_configuration", // Represents resources for managing table configurations.
  Integration = "integration", // Represents resources for integrations with external systems.
  Setting = "setting", // Represents configuration or setting resources.
  Template = "template", // Represents resources for managing templates.
  Backup = "backup", // Represents resources for managing backups.
  Permission = "permission", // Represents resources for managing permissions.

  // Additional resources from existing enum (not in Go constants)
  Dashboard = "dashboard",
  BillingManagement = "billing_management",
  BillingClient = "billing_client",
  ConfigurationFiles = "configuration_files",
  RateManagement = "rate_management",
  SystemLog = "system_log",
  DataRetention = "data_retention",
  ResourceEditor = "resource_editor",
  Equipment = "equipment",
  Maintenance = "maintenance",
  FormulaTemplate = "formula_template",
  ShipmentManagement = "shipment_management",
  DelayCode = "delay_code",
  ChargeType = "charge_type",
  DivisionCode = "division_code",
  GlAccount = "gl_account",
  RevenueCode = "revenue_code",
  AIClassification = "ai_classification",
  EmailProfile = "email_profile",
  AILog = "ai_log",
  Variable = "variable",
  VariableFormat = "format",
  Docker = "docker",
}
