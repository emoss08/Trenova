import type { BaseModel } from "./common";
import type { UsState } from "./us-state";

export enum OrganizationType {
  Carrier = "Carrier",
  Brokerage = "Brokerage",
  BrokerageCarrier = "Brokerage & Carrier",
}

export type BusinessUnitStatus =
  | "Active"
  | "Inactive"
  | "Pending"
  | "Suspended";

export interface Organization extends BaseModel {
  id: string;
  name: string;
  scacCode: string;
  dotNumber: string;
  logoUrl?: string; // Optional field
  orgType: OrganizationType;
  bucketName: string;
  addressLine1?: string; // Optional field
  addressLine2?: string; // Optional field
  city: string;
  postalCode?: string; // Optional field
  timezone: string;
  version: number;
  createdAt: number;
  updatedAt: number;

  businessUnitId: string;
  stateId: string;

  businessUnit?: BusinessUnit; // Optional relation
  state?: UsState; // Optional relation
  metadata?: OrganizationMetadata; // Optional relation, not exposed to API
}

export interface OrganizationMetadata {
  objectID: string;
}

export interface BusinessUnit extends BaseModel {
  name: string;
  status: BusinessUnitStatus;
  code: string;
  description: string;
  primaryContact?: string; // Optional field
  primaryEmail?: string; // Optional field
  primaryPhone?: string; // Optional field
  parentBusinessUnitId?: string; // Nullable field
  organizations?: Organization[]; // Optional relation
}
