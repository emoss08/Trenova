import { API_BASE_URL } from "@/lib/constants";
import type { FieldFilter, FilterGroup, SortField } from "@/types/data-table";
import type { API_ENDPOINTS, GenericLimitOffsetResponse } from "@/types/server";
import { useQuery } from "@tanstack/react-query";
import type { PaginationState } from "@tanstack/react-table";

export type DataTableQueryOptions = {
  query?: string;
  fieldFilters?: FieldFilter[];
  filterGroups?: FilterGroup[];
  sort?: SortField[];
  extraSearchParams?: Record<string, unknown>;
};

export async function fetchData<TData extends Record<string, unknown>>(
  link: string,
  pageIndex: number,
  pageSize: number,
  options?: DataTableQueryOptions,
): Promise<GenericLimitOffsetResponse<TData>> {
  const fetchURL = new URL(`${API_BASE_URL}${link}`, window.location.origin);
  fetchURL.searchParams.set("limit", pageSize.toString());
  fetchURL.searchParams.set("offset", (pageIndex * pageSize).toString());

  if (options?.query) {
    fetchURL.searchParams.set("query", options.query);
  }

  if (options?.fieldFilters && options.fieldFilters.length > 0) {
    fetchURL.searchParams.set(
      "fieldFilters",
      JSON.stringify(options.fieldFilters),
    );
  }

  if (options?.filterGroups && options.filterGroups.length > 0) {
    fetchURL.searchParams.set(
      "filterGroups",
      JSON.stringify(options.filterGroups),
    );
  }

  if (options?.sort && options.sort.length > 0) {
    fetchURL.searchParams.set("sort", JSON.stringify(options.sort));
  }

  if (options?.extraSearchParams) {
    Object.entries(options.extraSearchParams).forEach(([key, value]) => {
      if (value !== undefined && value !== null) {
        if (typeof value === "string") {
          fetchURL.searchParams.set(key, value);
        } else if (typeof value === "boolean" || typeof value === "number") {
          fetchURL.searchParams.set(key, String(value));
        } else if (Array.isArray(value) || typeof value === "object") {
          fetchURL.searchParams.set(key, JSON.stringify(value));
        }
      }
    });
  }

  const response = await fetch(fetchURL.href, {
    credentials: "include",
  });
  if (!response.ok) {
    throw new Error("Failed to fetch data from server");
  }

  return response.json();
}

export function useDataTableQuery<TData extends Record<string, unknown>>(
  queryKey: string,
  link: API_ENDPOINTS,
  pagination: PaginationState,
  options?: DataTableQueryOptions,
) {
  return useQuery<GenericLimitOffsetResponse<TData>, Error>({
    queryKey: [queryKey, link, pagination, options],
    queryFn: async () =>
      fetchData<TData>(
        link,
        pagination.pageIndex,
        pagination.pageSize,
        options,
      ),
    // structuralSharing: false,
  });
}
