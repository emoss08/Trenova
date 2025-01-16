import type { BaseModel } from "./common";
import type { Organization } from "./organization";

type BusinessUnitStatus = "Active" | "Inactive" | "Pending" | "Suspended";

export interface BusinessUnit extends BaseModel {
  name: string;
  status: BusinessUnitStatus;
  code: string;
  description: string;
  primaryContact: string;
  primaryEmail: string;
  primaryPhone: string;
  parentBusinessUnitId?: string;
  organizations: Organization[];
}
