import { http } from "@/lib/http-client";
import type {
  SearchOptions,
  SearchResponse,
  SiteSearchTab,
} from "@/types/search";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useCallback, useState } from "react";
import { useDebounce } from "./use-debounce";

/**
 * A custom hook for integrating with the search API using React Query
 */
export function useSearch() {
  const [query, setQuery] = useState("");
  const debouncedQuery = useDebounce(query, 300);
  const [activeTab, setActiveTab] = useState<SiteSearchTab>("all");
  const [selectedTypes, setSelectedTypes] = useState<string[]>([]);

  // Create the search function
  const performSearch = useCallback(
    async (options: SearchOptions) => {
      // Determine which endpoint to use based on active tab
      let endpoint = "/search";
      if (activeTab === "shipments") endpoint = "/search/shipments";
      else if (activeTab === "drivers") endpoint = "/search/drivers";
      else if (activeTab === "equipment") endpoint = "/search/equipment";
      else if (activeTab === "customers") endpoint = "/search/customers";

      // Build clean params object, only including values that exist
      const params: Record<string, string | undefined> = {
        q: options.query,
        limit: options.limit?.toString(),
        offset: options.offset?.toString(),
        highlight: options.highlight ? "true" : undefined,
      };

      // Only add types if they exist and have values
      if (options.types && options.types.length > 0) {
        params.types = options.types.join(",");
      }

      // Only add facets if they exist and have values
      if (options.facets && options.facets.length > 0) {
        params.facets = options.facets.join(",");
      }

      // Only add filter if it exists
      if (options.filter) {
        params.filter = options.filter;
      }

      // Make the request with the cleaned params
      const response = await http.get<SearchResponse>(endpoint, {
        params,
      });

      return response.data;
    },
    [activeTab],
  );

  // Use React Query to manage the search state
  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["search", debouncedQuery, activeTab, selectedTypes],
    queryFn: async () => {
      if (!debouncedQuery) {
        return { results: [], total: 0, facets: {} };
      }

      const result = await performSearch({
        query: debouncedQuery,
        types: selectedTypes.length > 0 ? selectedTypes : undefined,
        highlight: true,
        limit: 10,
      });

      return result;
    },
    // Don't auto-fetch on mount if query is empty
    enabled: !!debouncedQuery,
    // Keep previous data while fetching to prevent UI flicker
    placeholderData: keepPreviousData,
    // Stale time to prevent too many refetches
    staleTime: 1000 * 60 * 5, // 5 minutes
  });

  // Extract data from the query result
  const results = data?.results || [];
  const totalResults = data?.total || 0;
  const facets = data?.facets || {};

  // Custom search function that can be called manually
  const search = useCallback(
    async (options: SearchOptions) => {
      // Call search directly for manual searches
      return performSearch(options);
    },
    [performSearch],
  );

  // Helper function to get type-specific search
  const searchByType = useCallback(
    async (
      type: SiteSearchTab,
      searchQuery: string,
      limit = 10,
      offset = 0,
    ) => {
      setActiveTab(type === "all" ? "all" : type);

      // Only set type filter if not "all"
      const typeFilter =
        type === "all"
          ? undefined
          : [type.endsWith("s") ? type.slice(0, -1) : type];

      if (searchQuery) {
        return performSearch({
          query: searchQuery,
          types: typeFilter,
          limit,
          offset,
          highlight: true,
        });
      }

      return { results: [], total: 0, facets: {} };
    },
    [performSearch],
  );

  // Handle tab change
  const changeTab = useCallback(
    (tab: SiteSearchTab) => {
      setActiveTab(tab);

      if (query) {
        if (tab === "all") {
          // Clear type filters when "all" is selected
          setSelectedTypes([]);
        } else {
          // For specific tabs, set the corresponding type filter
          // Make sure to handle both singular and plural forms
          const type = tab.endsWith("s") ? tab.slice(0, -1) : tab;
          setSelectedTypes([type]);
        }
      }
    },
    [query],
  );

  // Toggle type filter
  const toggleTypeFilter = useCallback((type: string) => {
    setSelectedTypes((prev) => {
      const newTypes = prev.includes(type)
        ? prev.filter((t) => t !== type)
        : [...prev, type];

      return newTypes;
    });
  }, []);

  return {
    query,
    setQuery,
    results,
    isLoading,
    isError,
    errorMessage: isError
      ? (error as Error)?.message || "An error occurred"
      : "",
    totalResults,
    facets,
    activeTab,
    setActiveTab: changeTab,
    selectedTypes,
    toggleTypeFilter,
    search,
    searchByType,
    advancedSearch: search,
    clearSearch: () => setQuery(""),
    refetch,
  };
}

/**
 * A custom hook for managing recent searches
 */
export function useRecentSearches(
  storageKey = "trenova-recent-searches",
  maxItems = 5,
) {
  const [recentSearches, setRecentSearches] = useState<string[]>([]);

  // Load from localStorage
  useState(() => {
    try {
      const storedSearches = localStorage.getItem(storageKey);
      if (storedSearches) {
        const parsed = JSON.parse(storedSearches);
        if (Array.isArray(parsed)) {
          setRecentSearches(parsed);
        }
      }
    } catch (e) {
      console.error("Error parsing recent searches:", e);
      localStorage.removeItem(storageKey);
    }
  });

  // Add a search to recent
  const addRecentSearch = useCallback(
    (query: string) => {
      if (!query || query.trim() === "") return;

      const trimmedQuery = query.trim();

      setRecentSearches((prev) => {
        const newSearches = [
          trimmedQuery,
          ...prev.filter((search) => search !== trimmedQuery),
        ].slice(0, maxItems);

        localStorage.setItem(storageKey, JSON.stringify(newSearches));
        return newSearches;
      });
    },
    [storageKey, maxItems],
  );

  // Remove a search from recent
  const removeRecentSearch = useCallback(
    (query: string) => {
      setRecentSearches((prev) => {
        const newSearches = prev.filter((search) => search !== query);
        localStorage.setItem(storageKey, JSON.stringify(newSearches));
        return newSearches;
      });
    },
    [storageKey],
  );

  // Clear all recent searches
  const clearRecentSearches = useCallback(() => {
    localStorage.removeItem(storageKey);
    setRecentSearches([]);
  }, [storageKey]);

  return {
    recentSearches,
    addRecentSearch,
    removeRecentSearch,
    clearRecentSearches,
  };
}
