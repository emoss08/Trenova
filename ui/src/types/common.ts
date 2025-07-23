/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import type {
  FieldFilter,
  SortField,
} from "@/lib/schemas/table-configuration-schema";
import type { IconDefinition } from "@fortawesome/pro-regular-svg-icons";
import type { ReactNode } from "react";

export enum Status {
  Active = "Active",
  Inactive = "Inactive",
}

export interface ChoiceProps<T extends string | boolean | number> {
  value: T;
  label: string;
  color?: string;
  description?: string;
  icon?: IconDefinition | ReactNode;
  disabled?: boolean;
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
