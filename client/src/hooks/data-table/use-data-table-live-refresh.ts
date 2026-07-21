import {
  buildDataTableQueryKey,
  fetchDataTablePage,
} from "@/hooks/data-table/use-data-table-query";
import type { DataTableGraphQLConfig, DataTableQueryOptions } from "@/types/data-table";
import type { GenericLimitOffsetResponse } from "@/types/server";
import { useQueryClient } from "@tanstack/react-query";
import type { PaginationState } from "@tanstack/react-table";
import { useCallback, useEffect, useRef, useState } from "react";

type UseDataTableLiveRefreshParams<TData extends Record<string, unknown>> = {
  intervalMs: number | undefined;
  enabled: boolean;
  queryKey: string;
  graphql: DataTableGraphQLConfig<TData>;
  pagination: PaginationState;
  options: DataTableQueryOptions;
  currentResults: TData[] | undefined;
};

export function useDataTableLiveRefresh<TData extends Record<string, unknown>>({
  intervalMs,
  enabled,
  queryKey,
  graphql,
  pagination,
  options,
  currentResults,
}: UseDataTableLiveRefreshParams<TData>) {
  const queryClient = useQueryClient();
  const [staged, setStaged] = useState<GenericLimitOffsetResponse<TData> | null>(null);

  const latestRef = useRef({ pagination, options, currentResults });
  latestRef.current = { pagination, options, currentResults };

  const scopeKey = JSON.stringify({ pagination, options });
  const scopeKeyRef = useRef(scopeKey);
  if (scopeKeyRef.current !== scopeKey) {
    scopeKeyRef.current = scopeKey;
    if (staged) setStaged(null);
  }

  useEffect(() => {
    if (!enabled || !intervalMs || intervalMs <= 0) return;

    let inFlight = false;
    const tick = async () => {
      if (inFlight || document.visibilityState !== "visible") return;
      inFlight = true;
      const snapshot = latestRef.current;
      try {
        const fresh = await fetchDataTablePage<TData>({
          pageSize: snapshot.pagination.pageSize,
          options: snapshot.options,
          graphql,
        });

        const latest = latestRef.current;
        if (JSON.stringify(snapshot.options) !== JSON.stringify(latest.options)) return;

        const unchanged =
          JSON.stringify(fresh.results) === JSON.stringify(latest.currentResults ?? []);
        setStaged(unchanged ? null : fresh);
      } catch {
        // Background probe failures are non-fatal; the next tick retries.
      } finally {
        inFlight = false;
      }
    };

    const timer = setInterval(() => {
      void tick();
    }, intervalMs);
    return () => clearInterval(timer);
    // oxlint-disable-next-line exhaustive-deps
  }, [enabled, intervalMs, scopeKey, graphql]);

  const applyStaged = useCallback(() => {
    if (!staged) return;
    const { pagination: currentPagination, options: currentOptions } = latestRef.current;
    queryClient.setQueryData(
      buildDataTableQueryKey(queryKey, graphql, currentPagination, currentOptions),
      staged,
    );
    setStaged(null);
  }, [staged, queryClient, queryKey, graphql]);

  const dismissStaged = useCallback(() => {
    setStaged(null);
  }, []);

  return {
    hasPendingUpdate: staged !== null,
    applyStaged,
    dismissStaged,
  };
}
