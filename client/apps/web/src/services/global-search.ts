import { GlobalSearchDocument } from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export interface GlobalSearchHit {
  id: string;
  entityType: string;
  title: string;
  subtitle?: string;
  href: string;
  metadata?: Record<string, string>;
}

export interface GlobalSearchGroup {
  entityType: string;
  label: string;
  /** How many records match in all, which can be more than the hits returned. */
  total?: number;
  hits: GlobalSearchHit[];
}

export interface GlobalSearchResponse {
  query: string;
  groups: GlobalSearchGroup[];
}

export const globalSearchEntityTypes = ["shipment", "customer", "worker", "document"] as const;

export type GlobalSearchEntityType = (typeof globalSearchEntityTypes)[number];

export function isGlobalSearchEntityType(value: string): value is GlobalSearchEntityType {
  return (globalSearchEntityTypes as readonly string[]).includes(value);
}

export class GlobalSearchService {
  async search(
    query: string,
    limit = 5,
    entityTypes?: GlobalSearchEntityType[],
    options?: { signal?: AbortSignal },
  ): Promise<GlobalSearchResponse> {
    const response = await requestGraphQL({
      document: GlobalSearchDocument,
      operationName: "GlobalSearch",
      variables: {
        input: {
          query,
          limit,
          entityTypes: entityTypes && entityTypes.length > 0 ? entityTypes : null,
        },
      },
      signal: options?.signal,
    });
    const result = response.globalSearch;

    return {
      query: result.query,
      groups: result.groups.map((group) => ({
        entityType: group.entityType,
        label: group.label,
        total: group.total,
        hits: group.hits.map((hit) => ({
          id: hit.id,
          entityType: hit.entityType,
          title: hit.title,
          subtitle: hit.subtitle || undefined,
          href: hit.href,
          metadata: (hit.metadata as Record<string, string> | null) ?? undefined,
        })),
      })),
    };
  }
}
