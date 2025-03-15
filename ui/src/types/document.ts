import { Resource } from "./audit-entry";

export enum DocumentType {
  License = "License",
  Registration = "Registration",
  Insurance = "Insurance",
  Invoice = "Invoice",
  ProofOfDelivery = "ProofOfDelivery",
  BillOfLading = "BillOfLading",
  DriverLog = "DriverLog",
  MedicalCert = "MedicalCert",
  Contract = "Contract",
  Maintenance = "Maintenance",
  AccidentReport = "AccidentReport",
  TrainingRecord = "TrainingRecord",
  Other = "Other",
}

export enum DocumentStatus {
  Draft = "Draft",
  Active = "Active",
  Archived = "Archived",
  Expired = "Expired",
  Rejected = "Rejected",
  PendingApproval = "PendingApproval",
}

export type Document = {
  id: string;
  fileName: string;
  originalName: string;
  fileType: string;
  fileSize: number;
  documentType: DocumentType;
  resourceType: Resource;
  resourceId: string;
  createdAt: number;
  status: DocumentStatus;
  description?: string;
  tags?: string[];
};

export type ResourceFolder = {
  resourceType: Resource;
  resourceId: string;
  resourceName: string;
  documentCount: number;
};

export type DocumentBreadcrumbItem = {
  label: string;
  path: string;
};

export type VersionInfo = {
  versionId: string;
  lastModified: string;
  createdBy: string;
  comment?: string;
  size: number;
  checksum: string;
  isLatest: boolean;
  classification?: string;
};
