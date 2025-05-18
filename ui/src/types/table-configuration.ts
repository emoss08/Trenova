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

  // New fields
  tableConfig: TableConfig;
}

export type CreateTableConfigurationRequest = Omit<
  TableConfiguration,
  | "id"
  | "createdAt"
  | "updatedAt"
  | "version"
  | "shares"
  | "creator"
  | "tableConfig"
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

export interface ColumnVisibilityState {
  [columnId: string]: boolean;
}

export interface TableConfig {
  /** Column visibility keyed by column id */
  columnVisibility: ColumnVisibilityState;
  /** Optional page size preference */
  pageSize?: number;
  /** Optional sorting preference */
  sorting?: unknown;
  /** Optional filters the user has applied */
  filters?: unknown;
  /** Logical operator to join filters */
  joinOperator?: string;
  /** Anything else we may store later */
  [key: string]: unknown;
}
