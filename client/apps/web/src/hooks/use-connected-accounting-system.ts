import {
  ACCOUNTING_SYSTEMS,
  hasLiveAccountingConnection,
  resolveAccountingSystem,
  type AccountingSystemCandidate,
} from "@/lib/accounting-sync";
import type { AccountingConnection } from "@/lib/graphql/accounting-sync";
import { queries } from "@/lib/queries";
import type { AccountingSystem } from "@trenova/graphql/generated/graphql";
import { useQueries, type QueryFunction, type UseQueryResult } from "@tanstack/react-query";

export type AccountingSystemSource = "status" | "mappingSummary" | "syncSummary";

export type ConnectedAccountingSystem = {
  system: AccountingSystem | null;
  connected: boolean;
  isLoading: boolean;
  isError: boolean;
  retry: () => void;
};

type ConnectionCarrier = { connection: AccountingConnection | null };

type ConnectionQuery<TSource extends AccountingSystemSource> = {
  queryKey: readonly ["accountingSync", TSource, AccountingSystem];
  queryFn: QueryFunction<ConnectionCarrier, readonly ["accountingSync", TSource, AccountingSystem]>;
};

const SOURCES: {
  [TSource in AccountingSystemSource]: (system: AccountingSystem) => ConnectionQuery<TSource>;
} = {
  status: queries.accountingSync.status,
  mappingSummary: queries.accountingSync.mappingSummary,
  syncSummary: queries.accountingSync.syncSummary,
};

function combineResolution(
  results: UseQueryResult<ConnectionCarrier>[],
): ConnectedAccountingSystem {
  const retry = () => {
    for (const result of results) {
      if (result.isError) {
        void result.refetch();
      }
    }
  };

  if (results.some((result) => result.isPending)) {
    return {
      system: null,
      connected: false,
      isLoading: results.some((result) => result.isLoading),
      isError: false,
      retry,
    };
  }

  const candidates: AccountingSystemCandidate[] = [];
  results.forEach((result, index) => {
    if (result.data) {
      candidates.push({ system: ACCOUNTING_SYSTEMS[index], connection: result.data.connection });
    }
  });
  const live = candidates.find((candidate) => hasLiveAccountingConnection(candidate.connection));
  if (live) {
    return { system: live.system, connected: true, isLoading: false, isError: false, retry };
  }
  if (results.some((result) => result.isError)) {
    return { system: null, connected: false, isLoading: false, isError: true, retry };
  }
  return {
    system: resolveAccountingSystem(candidates),
    connected: false,
    isLoading: false,
    isError: false,
    retry,
  };
}

export function useConnectedAccountingSystem(
  source: AccountingSystemSource,
  enabled = true,
): ConnectedAccountingSystem {
  const factory = SOURCES[source];
  return useQueries({
    queries: ACCOUNTING_SYSTEMS.map((system) => ({ ...factory(system), enabled })),
    combine: combineResolution,
  });
}
