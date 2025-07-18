import { Resource } from "@/types/audit-entry";
import { Visibility } from "@/types/table-configuration";
import * as z from "zod/v4";
import {
  nullableIntegerSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";
import { userSchema } from "./user-schema";

export const LogicalOperatorSchema = z.enum(["and", "or"]);

export const SortDirectionSchema = z.enum(["asc", "desc"]);

export const FilterOperatorSchema = z.enum([
  "eq",
  "ne",
  "gt",
  "gte",
  "lt",
  "lte",
  "contains",
  "startswith",
  "endswith",
  "like",
  "ilike",
  "in",
  "notin",
  "isnull",
  "isnotnull",
  "daterange",
]);

const filterFieldSchema = z.object({
  field: z.string(),
  operator: FilterOperatorSchema,
  value: z.any(),
});

const sortFieldSchema = z.object({
  field: z.string(),
  direction: SortDirectionSchema,
});

export const FilterStateSchema = z.object({
  filters: z.array(filterFieldSchema),
  sort: z.array(sortFieldSchema),
  globalSearch: z.string(),
  logicalOperators: z.array(LogicalOperatorSchema).optional(),
});

const tableConfigSchema = z.object({
  columnVisibility: z.record(z.string(), z.boolean()),
  pageSize: nullableIntegerSchema,
  sort: z.array(sortFieldSchema).optional(),
  filters: z.array(filterFieldSchema).optional(),
  joinOperator: optionalStringSchema,
});

export const shareConfigurationSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  shareType: z.string(),
  shareWithId: optionalStringSchema,
  configurationId: optionalStringSchema,
});

export const tableConfigurationSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  // * Code Fields
  name: z.string().min(1, { error: "Name is required" }),
  description: optionalStringSchema,
  resource: z.enum(Resource),
  tableConfig: tableConfigSchema,
  visibility: z.enum(Visibility),
  isDefault: z.boolean(),

  creator: userSchema.nullish(),
  shares: z.array(shareConfigurationSchema).nullish(),
});

export type ShareConfigurationSchema = z.infer<typeof shareConfigurationSchema>;
export type TableConfigurationSchema = z.infer<typeof tableConfigurationSchema>;
export type FilterStateSchema = z.infer<typeof FilterStateSchema>;

export type FilterFieldSchema = z.infer<typeof filterFieldSchema>;
export type SortFieldSchema = z.infer<typeof sortFieldSchema>;

export type LogicalOperator = z.infer<typeof LogicalOperatorSchema>;
export type SortDirection = z.infer<typeof SortDirectionSchema>;
export type FilterOperator = z.infer<typeof FilterOperatorSchema>;
export type FieldFilter = z.infer<typeof filterFieldSchema>;
export type SortField = z.infer<typeof sortFieldSchema>;
