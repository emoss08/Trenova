import { generateIftaReturn, recomputeIftaReturn } from "@/lib/graphql/ifta-return";
import { quarterLabel, type IftaPeriodKey } from "@/lib/ifta-return";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { handleIftaReturnError, invalidateIftaReturn } from "./queries";

export function useGenerateIftaReturn(period: IftaPeriodKey) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => generateIftaReturn({ year: period.year, quarter: period.quarter }),
    onSuccess: async () => {
      toast.success(`${quarterLabel(period)} return generated`, {
        description:
          "The draft is computed from the miles, fuel and rates on file now. Recompute it whenever late data lands.",
      });
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
      toast.success("Return recomputed", {
        description: "Every line was rebuilt from the miles, fuel and rates on file now.",
      });
      await invalidateIftaReturn(queryClient, period);
    },
    onError: (error) => handleIftaReturnError(error, queryClient, period),
  });
}
