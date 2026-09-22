import {
  composedTableQuerySchema,
  tableCatalogueSchema,
  type ComposedTableQuery,
} from "@/types/table-query";
import type { FieldFilter, SortField } from "@trenova/shared/types/data-table";
import { api } from "@trenova/shared/lib/api";
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
    const response = await api.post(
      `/tables/${encodeURIComponent(resource)}/compose/`,
      payload,
      options,
    );

    return safeParse(composedTableQuerySchema, response, "Composed query");
  }

  /** Which tables can be asked in words, and in which terms. */
  public async catalogue(options?: { signal?: AbortSignal }) {
    const response = await api.get("/tables/", options);

    return safeParse(tableCatalogueSchema, response, "Table catalogue");
  }
}
