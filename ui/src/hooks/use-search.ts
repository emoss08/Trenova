import { api } from "@/services/api";
import { SearchEntityType, type SearchRequest } from "@/types/search";
import { useQuery } from "@tanstack/react-query";
import { useCallback, useMemo, useState } from "react";
import { useDebounce } from "./use-debounce";

export function useSearch(activeTab: SearchEntityType = SearchEntityType.All) {
  const [searchQuery, setSearchQuery] = useState("");
  const debouncedSearchQuery = useDebounce(searchQuery, 300);
  const cleanedDebouncedQuery = useMemo(() => {
    // Remove trailing @mention token (e.g., "@shipments") from the query used for API
    return (debouncedSearchQuery || "").replace(/@\S*$/, "").trim();
  }, [debouncedSearchQuery]);

  const performSearch = useCallback(async (req: SearchRequest) => {
    const response = await api.search.search(req);
    return response;
  }, []);

  const selectedEntityTypes: SearchEntityType[] =
    activeTab === SearchEntityType.All
      ? [SearchEntityType.Shipment, SearchEntityType.Customer]
      : [activeTab as SearchEntityType];

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["search", cleanedDebouncedQuery, activeTab, selectedEntityTypes],
    queryFn: async () => {
      if (!cleanedDebouncedQuery) {
        return {
          hits: [],
          total: 0,
          offset: 0,
          limit: 0,
          processingTimeMs: 0,
          query: "",
        };
      }

      const request: SearchRequest = {
        query: cleanedDebouncedQuery,
        entityTypes: selectedEntityTypes,
        limit: 10,
        offset: 0,
      };

      // Debug: outgoing request
      console.debug("[useSearch] request", {
        activeTab,
        request,
      });

      const response = await performSearch(request);

      // Debug: response summary
      console.debug("[useSearch] response", {
        hits: response?.hits?.length ?? 0,
        total: response?.total,
        processingTimeMs: response?.processingTimeMs,
      });

      return response;
    },
    enabled: !!cleanedDebouncedQuery,
  });

  return {
    searchResults: data?.hits,
    searchQuery,
    setSearchQuery,
    isLoading,
    isError,
    error,
    refetch,
  };
}
