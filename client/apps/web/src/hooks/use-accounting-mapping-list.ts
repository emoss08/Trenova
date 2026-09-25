import { accountingMappingFilterKey } from "@/lib/accounting-sync";
import { fetchAccountingMappings } from "@/lib/graphql/accounting-sync";
import { queries } from "@/lib/queries";
import type {
  AccountingMappingFilterInput,
  AccountingSystem,
} from "@trenova/graphql/generated/graphql";
import { useInfiniteQuery } from "@tanstack/react-query";
import { useMemo } from "react";

export function useAccountingMappingList({
  system,
  filter,
  pageSize,
  enabled = true,
}: {
  system: AccountingSystem;
  filter: AccountingMappingFilterInput;
  pageSize: number;
  enabled?: boolean;
}) {
  const query = useInfiniteQuery({
    queryKey: queries.accountingSync.mappings(system, accountingMappingFilterKey(filter), pageSize)
      .queryKey,
    queryFn: ({ pageParam, signal }) =>
      fetchAccountingMappings(
        { integrationType: system, filter, first: pageSize, after: pageParam },
        { signal },
      ),
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) => (lastPage.hasNextPage ? lastPage.endCursor : undefined),
    enabled,
  });

  const mappings = useMemo(
    () => query.data?.pages.flatMap((page) => page.mappings) ?? [],
    [query.data?.pages],
  );

  return {
    mappings,
    isLoading: query.isLoading,
    isError: query.isError,
    hasNextPage: query.hasNextPage,
    isFetchingNextPage: query.isFetchingNextPage,
    fetchNextPage: query.fetchNextPage,
  };
}
