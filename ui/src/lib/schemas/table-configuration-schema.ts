import { Resource } from "@/types/audit-entry";
import { Visibility } from "@/types/table-configuration";
import * as z from "zod/v4";
import {
  nullableIntegerSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

const filterFieldSchema = z.object({
  field: z.string(),
  operator: z.string(),
  value: z.any(),
});

const sortFieldSchema = z.object({
  field: z.string(),
  direction: z.string(),
});

const tableConfigSchema = z.object({
  columnVisibility: z.record(z.string(), z.boolean()),
  pageSize: nullableIntegerSchema,
  sort: z.array(sortFieldSchema).optional(),
  filters: z.array(filterFieldSchema).optional(),
  joinOperator: optionalStringSchema,
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
});

export type TableConfigurationSchema = z.infer<typeof tableConfigurationSchema>;
