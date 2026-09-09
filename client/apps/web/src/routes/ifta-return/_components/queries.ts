import { handleMutationError } from "@/hooks/use-api-mutation";
import {
  IFTA_PERIOD_KEY,
  IFTA_RETURN_KEY,
  IFTA_RETURN_LIST_KEY,
  fetchIftaPeriod,
  fetchIftaReturnForPeriod,
  type IftaReturn,
} from "@/lib/graphql/ifta-return";
import type { IftaPeriodKey, IftaReturnView } from "@/lib/ifta-return";
import type { QueryClient } from "@tanstack/react-query";
import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  IftaPeriodFieldsFragmentDoc,
  IftaReturnLineFieldsFragmentDoc,
} from "@trenova/graphql/generated/graphql";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { apiProblem } from "@trenova/shared/types/errors";

export function toIftaReturnView(ret: IftaReturn): IftaReturnView {
  return {
    ...ret,
    period: getFragmentData(IftaPeriodFieldsFragmentDoc, ret.period),
    lines: [...getFragmentData(IftaReturnLineFieldsFragmentDoc, ret.lines)],
  };
}

// The working copy for the quarter: the open Draft or Finalized return, else
// the most recent Filed one, else nothing at all.
export function iftaReturnForPeriodQuery(period: IftaPeriodKey) {
  return {
    queryKey: [IFTA_RETURN_KEY, "period", period.year, period.quarter] as const,
    queryFn: async ({ signal }: { signal: AbortSignal }): Promise<IftaReturnView | null> => {
      const ret = await fetchIftaReturnForPeriod(
        { year: period.year, quarter: period.quarter },
        { signal },
      );
      return ret === null ? null : toIftaReturnView(ret);
    },
  };
}

// The quarter's own bounds and due date. The return carries them once it
// exists; this is what the picker reads before one has been generated, and
// the bounds of a calendar quarter never change, so it is cached for the day.
export function iftaPeriodQuery(period: IftaPeriodKey) {
  return {
    queryKey: [IFTA_PERIOD_KEY, period.year, period.quarter] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      fetchIftaPeriod({ year: period.year, quarter: period.quarter }, { signal }),
    staleTime: 24 * 60 * 60 * 1000,
  };
}

export async function invalidateIftaReturn(
  queryClient: QueryClient,
  period: IftaPeriodKey,
): Promise<void> {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: iftaReturnForPeriodQuery(period).queryKey }),
    queryClient.invalidateQueries({ queryKey: [IFTA_RETURN_LIST_KEY] }),
  ]);
}

function isVersionMismatch(error: unknown): boolean {
  if (!(error instanceof GraphQLRequestError) && !(error instanceof ApiRequestError)) return false;
  return apiProblem.isVersionMismatchError(error.normalize());
}

/**
 * Every mutation on a return carries the version it was read at. When the
 * server refuses because someone else moved first, the toast is not enough on
 * its own — the copy on screen is stale, so it is thrown away and read again.
 */
export function invalidateOnVersionMismatch(
  error: unknown,
  queryClient: QueryClient,
  period: IftaPeriodKey,
): void {
  if (isVersionMismatch(error)) {
    void invalidateIftaReturn(queryClient, period);
  }
}

export function handleIftaReturnError(
  error: unknown,
  queryClient: QueryClient,
  period: IftaPeriodKey,
): void {
  handleMutationError({ error, resourceName: "IFTA Return" });
  invalidateOnVersionMismatch(error, queryClient, period);
}
