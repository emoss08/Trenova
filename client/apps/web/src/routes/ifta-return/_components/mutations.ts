import { generateIftaReturn, recomputeIftaReturn } from "@/lib/graphql/ifta-return";
import type { IftaPeriodKey } from "@/lib/ifta-return";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { handleIftaReturnError, invalidateIftaReturn } from "./queries";

export function useGenerateIftaReturn(period: IftaPeriodKey) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => generateIftaReturn({ year: period.year, quarter: period.quarter }),
    onSuccess: async () => {
      await invalidateIftaReturn(queryClient, period);
    },
    onError: (error) => handleIftaReturnError(error, queryClient, period),
  });
}

export function useRecomputeIftaReturn(period: IftaPeriodKey) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, version }: { id: string; version: number }) =>
      recomputeIftaReturn(id, version),
    onSuccess: async () => {
      await invalidateIftaReturn(queryClient, period);
    },
    onError: (error) => handleIftaReturnError(error, queryClient, period),
  });
}
