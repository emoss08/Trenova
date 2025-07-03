import type {
  FieldFilter,
  SortField,
} from "@/lib/schemas/table-configuration-schema";
import type { IconDefinition } from "@fortawesome/pro-regular-svg-icons";

export enum Status {
  Active = "Active",
  Inactive = "Inactive",
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

export type QueryOptions = {
  id?: string;
  tenantOpts?: {
    buId: string;
    orgId: string;
    userId: string;
  };
  query?: string;
  sort?: SortField[];
  filters?: FieldFilter[];
  limit?: number;
  offset?: number;
};
