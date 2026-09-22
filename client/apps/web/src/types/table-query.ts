import { filterOperatorSchema, sortDirectionSchema } from "@trenova/shared/types/data-table";
import { z } from "zod";

/*
What the server returns when a table is asked a question in words.

The filters are the ones that compiled; the unresolved entries are what the
question asked for that the table's fields cannot express. Both matter and
neither may be dropped — a filter that silently did not apply is the worst
outcome available here, because the table comes back looking answered and the
one condition the person cared about is the one that went missing.
*/

// The operators and directions are the table's own: one the table cannot apply
// fails the parse, loudly, rather than reaching the filter builder as a chip
// that does nothing.
const fieldFilterSchema = z.object({
  field: z.string(),
  operator: filterOperatorSchema,
  value: z.unknown(),
});

const sortFieldSchema = z.object({
  field: z.string(),
  direction: sortDirectionSchema,
});

export const unresolvedTermSchema = z.object({
  phrase: z.string(),
  reason: z.string(),
});

export const composedTableQuerySchema = z.object({
  query: z.string().default(""),
  fieldFilters: z
    .array(fieldFilterSchema)
    .nullish()
    .transform((value) => value ?? []),
  sort: z
    .array(sortFieldSchema)
    .nullish()
    .transform((value) => value ?? []),
  explanation: z.string().default(""),
  terms: z
    .array(z.string())
    .nullish()
    .transform((value) => value ?? []),
  unresolved: z
    .array(unresolvedTermSchema)
    .nullish()
    .transform((value) => value ?? []),
});

export type ComposedTableQuery = z.infer<typeof composedTableQuerySchema>;
export type UnresolvedTerm = z.infer<typeof unresolvedTermSchema>;

export const tableCatalogueFieldSchema = z.object({
  name: z.string(),
  kind: z.string(),
  note: z.string().default(""),
  values: z
    .array(z.string())
    .nullish()
    .transform((value) => value ?? []),
  sortable: z.boolean().default(false),
  operators: z
    .array(z.string())
    .nullish()
    .transform((value) => value ?? []),
});

export const tableCatalogueResourceSchema = z.object({
  resource: z.string(),
  entity: z.string(),
  summary: z.string().default(""),
  fields: z
    .array(tableCatalogueFieldSchema)
    .nullish()
    .transform((value) => value ?? []),
});

export const tableCatalogueSchema = z.object({
  results: z
    .array(tableCatalogueResourceSchema)
    .nullish()
    .transform((value) => value ?? []),
  count: z.number().default(0),
});

export type TableCatalogueResource = z.infer<typeof tableCatalogueResourceSchema>;
