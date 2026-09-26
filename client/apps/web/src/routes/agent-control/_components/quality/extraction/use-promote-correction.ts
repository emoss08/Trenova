import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  AI_CORRECTION_LIST_KEY,
  EXTRACTION_ACCURACY_KEY,
  EXTRACTION_EVAL_CASE_LIST_KEY,
  promoteAICorrection,
} from "@/lib/graphql/extraction-eval";
import { useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { toast } from "sonner";

export type PromoteVariables = { correctionId: string; activate: boolean };

/** Freezes a correction into the evaluation set, as a candidate or already active. */
export function usePromoteCorrection() {
  const t = useT();
  const queryClient = useQueryClient();

  return useApiMutation({
    mutationFn: ({ correctionId, activate }: PromoteVariables) =>
      promoteAICorrection({ correctionId, activate }),
    onSuccess: async (created) => {
      toast.success(
        created.status === "Active"
          ? t("Added to the evaluation set as an active case")
          : t("Added to the evaluation set as a candidate"),
        { description: created.title },
      );
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: [EXTRACTION_EVAL_CASE_LIST_KEY] }),
        queryClient.invalidateQueries({ queryKey: [EXTRACTION_ACCURACY_KEY] }),
        queryClient.invalidateQueries({ queryKey: [AI_CORRECTION_LIST_KEY] }),
      ]);
    },
    resourceName: t("Evaluation case"),
  });
}
