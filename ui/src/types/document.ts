import { Resource } from "./audit-entry";
import { User } from "./user";

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

export function getDocumentTypeLabel(documentType: DocumentType) {
  switch (documentType) {
    case DocumentType.License:
      return "License";
    case DocumentType.Registration:
      return "Registration";
    case DocumentType.Insurance:
      return "Insurance";
    case DocumentType.Invoice:
      return "Invoice";
    case DocumentType.ProofOfDelivery:
      return "Proof of Delivery";
    case DocumentType.BillOfLading:
      return "Bill of Lading";
    case DocumentType.DriverLog:
      return "Driver Log";
    case DocumentType.MedicalCert:
      return "Medical Certificate";
    case DocumentType.Contract:
      return "Contract";
    case DocumentType.Maintenance:
      return "Maintenance";
    case DocumentType.AccidentReport:
      return "Accident Report";
    case DocumentType.TrainingRecord:
      return "Training Record";
    case DocumentType.Other:
      return "Other";
  }
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
  tags?: string[];
  // * generated presigned URL by the server. (expires in 24 hours)
  presignedURL?: string | null;
  uploadedBy?: User | null;
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
