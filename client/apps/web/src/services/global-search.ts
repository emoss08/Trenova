import { api } from "@trenova/shared/lib/api";

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
    const searchParams = new URLSearchParams();
    searchParams.set("query", query);
    searchParams.set("limit", String(limit));
    if (entityTypes && entityTypes.length > 0) {
      searchParams.set("entityTypes", entityTypes.join(","));
    }

    return api.get<GlobalSearchResponse>(`/search/global/?${searchParams.toString()}`, {
      signal: options?.signal,
    });
  }
}
