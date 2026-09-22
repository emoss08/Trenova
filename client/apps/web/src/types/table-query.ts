import { filterOperatorSchema } from "@trenova/shared/types/data-table";
import { z } from "zod";

/*
What the server returns when a table is asked a question in words.

The filters are the ones that compiled; the unresolved entries are what the
question asked for that the table's fields cannot express. Both matter and
neither may be dropped — a filter that silently did not apply is the worst
outcome available here, because the table comes back looking answered and the
one condition the person cared about is the one that went missing.
*/

/*
The operator is checked against the grid's own vocabulary rather than accepted
as any string. This is the parse boundary between a model's answer and the
table, and a plain z.string() validated nothing at it: whatever the composer
returned went into the grid's filter list unexamined.

The composer cannot currently produce anything outside this set — filtercatalog's
operatorsByKind offers eighteen operators and every one of them is here. dbtype
declares more (counteq, lastmonth, like, thismonth and the rest) that it never
offers. So widening operatorsByKind without widening filterOperatorSchema to
match would start rejecting legitimate answers right here, which is the failure
this comment exists to prevent.
*/
const fieldFilterSchema = z.object({
  field: z.string(),
  operator: filterOperatorSchema,
  value: z.unknown(),
});

const sortFieldSchema = z.object({
  field: z.string(),
  direction: z.string(),
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
