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

export const tableDensitySchema = z.enum(["compact", "comfortable"]);

export type TableDensity = z.infer<typeof tableDensitySchema>;

export const formatRuleColorSchema = z.enum([
  "red",
  "amber",
  "green",
  "blue",
  "purple",
  "gray",
]);

export type FormatRuleColor = z.infer<typeof formatRuleColorSchema>;

export const formatRuleOperatorSchema = z.enum([
  "eq",
  "ne",
  "gt",
  "gte",
  "lt",
  "lte",
  "contains",
  "isnull",
  "isnotnull",
]);

export type FormatRuleOperator = z.infer<typeof formatRuleOperatorSchema>;

export const tableFormatRuleSchema = z.object({
  id: z.string(),
  field: z.string(),
  operator: formatRuleOperatorSchema,
  value: z.unknown().nullish(),
  color: formatRuleColorSchema,
});

export type TableFormatRule = z.infer<typeof tableFormatRuleSchema>;

export const tableColumnPinningSchema = z.object({
  left: z
    .array(z.string())
    .nullish()
    .transform((v) => v ?? []),
  right: z
    .array(z.string())
    .nullish()
    .transform((v) => v ?? []),
});

export type TableColumnPinning = z.infer<typeof tableColumnPinningSchema>;

export const tableConfigSchema = z.object({
  fieldFilters: z
    .array(fieldFilterSchema)
    .nullish()
    .transform((v) => v ?? []),
  filterGroups: z
    .array(filterGroupSchema)
    .nullish()
    .transform((v) => v ?? []),
  joinOperator: z
    .enum(["and", "or"])
    .nullish()
    .transform((v) => v ?? "and"),
  sort: z
    .array(sortFieldSchema)
    .nullish()
    .transform((v) => v ?? []),
  pageSize: z
    .number()
    .nullish()
    .transform((v) => v ?? 10),
  columnVisibility: z
    .record(z.string(), z.boolean())
    .nullish()
    .transform((v) => v ?? {}),
  columnOrder: z
    .array(z.string())
    .nullish()
    .transform((v) => v ?? []),
  columnSizing: z
    .record(z.string(), z.number())
    .nullish()
    .transform((v) => v ?? {}),
  columnPinning: tableColumnPinningSchema
    .nullish()
    .transform((v) => v ?? { left: [], right: [] }),
  density: tableDensitySchema.catch("comfortable"),
  formatRules: z
    .array(tableFormatRuleSchema)
    .nullish()
    .transform((v) => v ?? []),
});

export type TableConfig = z.infer<typeof tableConfigSchema>;

export type TableViewSource = {
  id: string;
  name: string;
};

export type ActiveTableView = TableViewSource & {
  config: TableConfig;
};

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

export const tableConfigurationUserSchema = z.object({
  id: z.string(),
  name: z.string(),
  profilePicUrl: z.string().nullish(),
});

export type TableConfigurationUser = z.infer<typeof tableConfigurationUserSchema>;

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
  isOrgDefault: z
    .boolean()
    .nullish()
    .transform((v) => v ?? false),
  version: z.number(),
  createdAt: z.number(),
  updatedAt: z.number(),
  user: tableConfigurationUserSchema.nullish(),
});

export type TableConfiguration = z.infer<typeof tableConfigurationSchema>;

export const tableConfigurationResponseSchema = createLimitOffsetResponse(
  tableConfigurationSchema,
);

export type TableConfigurationResponse = z.infer<
  typeof tableConfigurationResponseSchema
>;
