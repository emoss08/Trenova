import { api } from "@/lib/api";

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

export type GlobalSearchEntityType = "shipment" | "customer" | "worker" | "document";

export class GlobalSearchService {
  async search(
    query: string,
    limit = 5,
    entityTypes?: GlobalSearchEntityType[],
  ): Promise<GlobalSearchResponse> {
    const searchParams = new URLSearchParams();
    searchParams.set("query", query);
    searchParams.set("limit", String(limit));
    if (entityTypes && entityTypes.length > 0) {
      searchParams.set("entityTypes", entityTypes.join(","));
    }

    return api.get<GlobalSearchResponse>(`/search/global/?${searchParams.toString()}`);
  }
}
