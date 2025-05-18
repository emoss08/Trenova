import { Visibility } from "@/types/table-configuration";
import { z } from "zod";

const tableConfigSchema = z.object({
  columnVisibility: z.record(z.boolean()),
  pageSize: z.number().optional(),
  sorting: z.any().optional(),
  filters: z.any().optional(),
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
  tableIdentifier: z.string().min(1, "Table identifier is required"),
  tableConfig: tableConfigSchema,
  visibility: z.nativeEnum(Visibility),
  isDefault: z.boolean(),
});

export type TableConfigurationSchema = z.infer<typeof tableConfigurationSchema>;
