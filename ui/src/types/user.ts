import { Status, type BaseModel } from "./common";
import { Organization } from "./organization";
import { Role } from "./roles-permissions";

export enum TimeFormat {
  TimeFormat12Hour = "12-hour",
  TimeFormat24Hour = "24-hour",
}

export interface User extends BaseModel {
  // Primary identifiers
  businessUnitId: string; // pulid.ID, required (notnull)
  currentOrganizationId: string; // pulid.ID, required (notnull)

  // Core fields
  status: Status; // property.Status, required (default: 'Active')
  name: string; // required
  username: string; // required
  emailAddress: string; // required
  profilePicUrl?: string; // optional (nullable)
  thumbnailUrl?: string; // optional (nullable)
  timezone: string; // required
  isLocked: boolean; // required (default: false)
  timeFormat: TimeFormat; // required (default: '12-hour')
  // Relationships
  currentOrganization?: Organization | null; // optional (nullable)
  organizations?: Organization[] | null; // optional (nullable)
  roles?: Role[] | null; // optional (nullable)
}
