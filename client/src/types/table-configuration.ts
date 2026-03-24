import { z } from "zod";
import {
  fieldFilterSchema,
  filterGroupSchema,
  sortFieldSchema,
} from "./data-table";
import { createLimitOffsetResponse } from "./server";

export const configurationVisibilitySchema = z.enum([
  "Private",
  "Public",
  "Shared",
]);

export type ConfigurationVisibility = z.infer<
  typeof configurationVisibilitySchema
>;

export const tableConfigSchema = z.object({
  fieldFilters: z.array(fieldFilterSchema).default([]),
  filterGroups: z.array(filterGroupSchema).default([]),
  joinOperator: z.enum(["and", "or"]).default("and"),
  sort: z.array(sortFieldSchema).default([]),
  pageSize: z.number().default(10),
  columnVisibility: z.record(z.string(), z.boolean()).default({}),
  columnOrder: z.array(z.string()).default([]),
});

export type TableConfig = z.infer<typeof tableConfigSchema>;

export const tableConfigurationFormSchema = z.object({
  name: z.string().min(1, "Name is required").max(255),
  description: z.string().default(""),
  resource: z.string().min(1, "Resource is required"),
  tableConfig: tableConfigSchema,
  visibility: configurationVisibilitySchema.default("Private"),
  isDefault: z.boolean().default(false),
});

export type TableConfigurationFormValues = z.input<
  typeof tableConfigurationFormSchema
>;

export const tableConfigurationSchema = z.object({
  id: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  userId: z.string(),
  name: z.string(),
  description: z.string(),
  resource: z.string(),
  tableConfig: tableConfigSchema,
  visibility: configurationVisibilitySchema,
  isDefault: z.boolean(),
  version: z.number(),
  createdAt: z.number(),
  updatedAt: z.number(),
});

export type TableConfiguration = z.infer<typeof tableConfigurationSchema>;

export const tableConfigurationResponseSchema = createLimitOffsetResponse(
  tableConfigurationSchema,
);

export type TableConfigurationResponse = z.infer<
  typeof tableConfigurationResponseSchema
>;
