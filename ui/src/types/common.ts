import type { IconDefinition } from "@fortawesome/pro-regular-svg-icons";
import type { BusinessUnit, Organization } from "./organization";

export enum Status {
  Active = "Active",
  Inactive = "Inactive",
}

export interface BaseModel {
  // Primary identifiers
  id: string; // corresponds to pulid.ID

  // Metadata and versioning
  version: number; // required (default: 0)
  createdAt: number; // required (epoch timestamp)
  updatedAt: number; // required (epoch timestamp)
}

export interface BaseModelWithOrganization extends BaseModel {
  // Primary identifiers
  organizationId: string; // pulid.ID, required
  businessUnitId: string; // pulid.ID, required

  // Relationships (optional due to `omitempty`)
  organization?: Organization;
  businessUnit?: BusinessUnit;
}

export interface ChoiceProps<T extends string | boolean | number> {
  value: T;
  label: string;
  color?: string;
  description?: string;
  icon?: IconDefinition;
}

export enum Gender {
  Male = "Male",
  Female = "Female",
}

export type HasField<T, K extends keyof T> = K extends keyof T ? true : false;
