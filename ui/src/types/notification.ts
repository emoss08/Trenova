export type NotificationQueryParams = {
  limit?: number;
  offset?: number;
  unreadOnly?: boolean;
};

export type NotificationChannel = "user" | "role" | "global";

export type NotificationPriority = "critical" | "high" | "medium" | "low";

export enum EventType {
  JobUnknown = "job.unknown",
  EventJobReportExport = "job.report.export_complete",
  JobShipmentDuplicate = "job.shipment.duplicate_complete",
  JobPatternAnalysis = "job.analysis.pattern_complete",
  JobComplianceCheck = "job.compliance.check_complete",
  JobBillingProcess = "job.billing.process_complete",

  // System Events
  SystemMaintenance = "system.maintenance.scheduled",
  SystemAlert = "system.alert.critical",

  // Business Events
  ShipmentStatusChange = "business.shipment.status_change",
  WorkerComplianceExpired = "business.worker.compliance_expired",
  CustomerDocumentReview = "business.customer.document_review",

  // Entity Update Events
  EntityUpdated = "entity.updated",
  ShipmentUpdated = "entity.shipment.updated",
  WorkerUpdated = "entity.worker.updated",
  CustomerUpdated = "entity.customer.updated",
  TractorUpdated = "entity.tractor.updated",
  TrailerUpdated = "entity.trailer.updated",
  LocationUpdated = "entity.location.updated",

  // Batch Events
  BatchSummary = "batch.summary",
}

export enum RateLimitPeriod {
  Minute = "minute",
  Hour = "hour",
  Day = "day",
}

export enum UpdateType {
  Any = "any",
  StatusChange = "status_change",
  Assignment = "assignment",
  DateChange = "date_change",
  LocationChange = "location_change",
  DocumentUpload = "document_upload",
  PriceChange = "price_change",
  ComplianceChange = "compliance_change",
  FieldChange = "field_change",
}

export enum Priority {
  Critical = "critical",
  High = "high",
  Medium = "medium",
  Low = "low",
}

export enum Channel {
  Global = "global",
  User = "user",
  Role = "role",
}

export enum DeliveryStatus {
  Pending = "pending",
  Delivered = "delivered",
  Failed = "failed",
  Expired = "expired",
}

export enum NotificationResources {
  Shipment = "shipment",
  Worker = "worker",
  Customer = "customer",
  Tractor = "tractor",
  Trailer = "trailer",
  Location = "location",
  Commodity = "commodity",
}

// Update type display names
export const UPDATE_TYPE_LABELS: Record<UpdateType, string> = {
  status_change: "Status Changes",
  assignment: "Assignments",
  location_change: "Location Changes",
  document_upload: "Document Uploads",
  price_change: "Price Changes",
  compliance_change: "Compliance Changes",
  field_change: "Field Changes",
  date_change: "Date Changes",
  any: "General Updates",
};

// Priority colors and icons
export const PRIORITY_CONFIG = {
  critical: {
    color: "text-red-600",
    bgColor: "bg-red-50",
    borderColor: "border-red-200",
    icon: "AlertTriangle",
  },
  high: {
    color: "text-orange-600",
    bgColor: "bg-orange-50",
    borderColor: "border-orange-200",
    icon: "AlertCircle",
  },
  medium: {
    color: "text-yellow-600",
    bgColor: "bg-yellow-50",
    borderColor: "border-yellow-200",
    icon: "Info",
  },
  low: {
    color: "text-blue-600",
    bgColor: "bg-blue-50",
    borderColor: "border-blue-200",
    icon: "Bell",
  },
} as const;
