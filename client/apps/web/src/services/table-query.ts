import {
  composedTableQuerySchema,
  tableCatalogueSchema,
  type ComposedTableQuery,
} from "@/types/table-query";
import {
  ComposeTableQueryDocument,
  TableQueryCatalogueDocument,
} from "@trenova/graphql/generated/graphql";
import type { FieldFilter, SortField } from "@trenova/shared/types/data-table";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";

export type ComposeTableQueryPayload = {
  prompt: string;
  /** What the table already shows, so a follow-up narrows rather than replaces. */
  current: {
    query: string;
    fieldFilters: FieldFilter[];
    sort: SortField[];
  };
};

/**
 * The catalogue's key, written out rather than taken from the query factory.
 *
 * It belongs to no table and is invalidated by nothing — which tables answer
 * to a description changes only when the server ships — so the factory would
 * be carrying a key that never needs targeting. Holding it here keeps the
 * toolbar off a module most data-table tests mock a slice of.
 */
export const TABLE_CATALOGUE_KEY = ["table-catalogue"] as const;

export class TableQueryService {
  public async compose(
    resource: string,
    payload: ComposeTableQueryPayload,
    options?: { signal?: AbortSignal },
  ): Promise<ComposedTableQuery> {
    const response = await requestGraphQL({
      document: ComposeTableQueryDocument,
      operationName: "ComposeTableQuery",
      variables: {
        input: {
          resource,
          prompt: payload.prompt,
          current: {
            query: payload.current.query,
            fieldFilters: payload.current.fieldFilters.map((filter) => ({
              field: filter.field,
              operator: filter.operator,
              value: filter.value,
            })),
            sort: payload.current.sort.map((sort) => ({
              field: sort.field,
              direction: sort.direction,
            })),
          },
        },
      },
      signal: options?.signal,
    });

    return safeParse(composedTableQuerySchema, response.composeTableQuery, "Composed query");
  }

  /** Which tables can be asked in words, and in which terms. */
  public async catalogue(options?: { signal?: AbortSignal }) {
    const response = await requestGraphQL({
      document: TableQueryCatalogueDocument,
      operationName: "TableQueryCatalogue",
      signal: options?.signal,
    });
    const results = response.tableQueryCatalogue;

    return safeParse(tableCatalogueSchema, { results, count: results.length }, "Table catalogue");
  }
}
