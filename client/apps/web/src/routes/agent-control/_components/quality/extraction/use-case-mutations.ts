import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  deleteExtractionEvalCase,
  EXTRACTION_ACCURACY_KEY,
  EXTRACTION_EVAL_CASE_DETAIL_KEY,
  EXTRACTION_EVAL_CASE_LIST_KEY,
  updateExtractionEvalCase,
} from "@/lib/graphql/extraction-eval";
import type {
  ExtractionEvalCaseStatus,
  UpdateExtractionEvalCaseInput,
} from "@trenova/graphql/generated/graphql";
import { useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { toast } from "sonner";
import { CASE_STATUS_MOVED } from "./extraction-model";

export type CaseUpdate = { id: string; input: UpdateExtractionEvalCaseInput };

/** Saving, moving and deleting a case, each refreshing what depends on it. */
export function useCaseMutations() {
  const t = useT();
  const queryClient = useQueryClient();

  const refresh = async (id: string) => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: [EXTRACTION_EVAL_CASE_LIST_KEY] }),
      queryClient.invalidateQueries({ queryKey: [EXTRACTION_EVAL_CASE_DETAIL_KEY, id] }),
      queryClient.invalidateQueries({ queryKey: [EXTRACTION_ACCURACY_KEY] }),
    ]);
  };

  const update = useApiMutation({
    mutationFn: ({ id, input }: CaseUpdate) => updateExtractionEvalCase(id, input),
    onSuccess: async (updated, { input }) => {
      toast.success(
        input.status
          ? t(CASE_STATUS_MOVED[input.status as ExtractionEvalCaseStatus])
          : t("Changes have been saved"),
      );
      await refresh(updated.id);
    },
    resourceName: t("Evaluation case"),
  });

  const remove = useApiMutation({
    mutationFn: (id: string) => deleteExtractionEvalCase(id),
    onSuccess: async (_deleted, id) => {
      toast.success(t("Case deleted"), {
        description: t("Its frozen document text and every result that scored it are gone."),
      });
      await refresh(id);
    },
    resourceName: t("Evaluation case"),
  });

  return { update, remove };
}
