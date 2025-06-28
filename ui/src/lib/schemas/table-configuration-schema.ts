import { Resource } from "@/types/audit-entry";
import { Visibility } from "@/types/table-configuration";
import { z } from "zod";

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
  columnVisibility: z.record(z.boolean()),
  pageSize: z.number().optional(),
  sort: z.array(sortFieldSchema).optional(),
  filters: z.array(filterFieldSchema).optional(),
  joinOperator: z.string().optional(),
});

export const tableConfigurationSchema = z.object({
  id: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

  // * Code Fields
  name: z.string().min(1, "Name is required"),
  description: z.string().optional(),
  resource: z.nativeEnum(Resource),
  tableConfig: tableConfigSchema,
  visibility: z.nativeEnum(Visibility),
  isDefault: z.boolean(),
});

export type TableConfigurationSchema = z.infer<typeof tableConfigurationSchema>;
