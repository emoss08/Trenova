import { API_URL } from "@/constants/env";
import { API_ENDPOINTS, type LimitOffsetResponse } from "@/types/server";
import { type QueryKey, useQuery } from "@tanstack/react-query";
import { PaginationState } from "@tanstack/react-table";

export async function fetchData<TData extends Record<string, any>>(
  link: string,
  pageIndex: number,
  pageSize: number,
  extraSearchParams?: Record<string, any>,
): Promise<LimitOffsetResponse<TData>> {
  const fetchURL = new URL(`${API_URL}${link}`);
  fetchURL.searchParams.set("limit", pageSize.toString());
  fetchURL.searchParams.set("offset", (pageIndex * pageSize).toString());
  if (extraSearchParams) {
    Object.entries(extraSearchParams).forEach(([key, value]) =>
      fetchURL.searchParams.set(key, value),
    );
  }

  const response = await fetch(fetchURL.href, {
    credentials: "include",
  });
  if (!response.ok) {
    throw new Error("Failed to fetch data from server");
  }

  return response.json();
}

export function useDataTableQuery<TData extends Record<string, any>>(
  queryKey: QueryKey,
  link: API_ENDPOINTS,
  pagination: PaginationState,
  extraSearchParams?: Record<string, any>,
) {
  return useQuery<LimitOffsetResponse<TData>, Error>({
    queryKey: [queryKey, link, pagination, extraSearchParams],
    queryFn: async () =>
      fetchData<TData>(
        link,
        pagination.pageIndex,
        pagination.pageSize,
        extraSearchParams,
      ),
  });
}
