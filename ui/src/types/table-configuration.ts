import type { BaseModelWithOrganization, Status } from "./common";
import { User } from "./user";

export enum ShareType {
  User = "User",
  Role = "Role",
  Team = "Team",
}

export enum Visibility {
  Private = "Private",
  Public = "Public",
  Shared = "Shared",
}

export interface TableConfiguration extends BaseModelWithOrganization {
  // Core Fields
  status: Status;
  name: string;
  description?: string;
  tableIdentifier: string;
  filterConfig: Record<string, any>;
  visibility: Visibility;
  isDefault: boolean;

  // Relationships (optional due to `omitempty`)
  shares?: ConfigurationShare[];
  creator?: User;
}

export type CreateTableConfigurationRequest = Omit<
  TableConfiguration,
  "id" | "createdAt" | "updatedAt" | "version" | "shares" | "creator"
>;

// ConfigurationShare interface
export interface ConfigurationShare {
  // Primary identifiers
  id: string;
  configurationId: string;
  businessUnitId: string;
  organizationId: string;
  sharedWithId: string;
  shareType: ShareType; // required

  // Metadata
  createdAt: number; // required

  // Relationships (optional due to `omitempty`)
  shareWithUser?: User;
  configuration?: TableConfiguration;
}
