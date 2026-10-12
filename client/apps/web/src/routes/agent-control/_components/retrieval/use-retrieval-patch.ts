import {
  AI_RETRIEVAL_FAILED_LIST_KEY,
  updateAIRetrievalSettings,
  type AIRetrievalSettingsPatch,
} from "@/lib/graphql/ai-retrieval";
import { queries } from "@/lib/queries";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { toast } from "sonner";

type RetrievalChange = {
  patch: AIRetrievalSettingsPatch;
  /** What the toast says once it is saved. */
  done: string;
};

/**
 * Saves one retrieval setting the moment it changes. Only the field changed is sent, so
 * a switch never overwrites a budget someone else raised meanwhile.
 */
export function useRetrievalPatch() {
  const t = useT();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ patch }: RetrievalChange) => updateAIRetrievalSettings(patch),
    onSuccess: async (status, { done }) => {
      queryClient.setQueryData(queries.aiRetrieval.status().queryKey, status);
      await queryClient.invalidateQueries({ queryKey: [AI_RETRIEVAL_FAILED_LIST_KEY] });
      toast.success(done);
    },
    onError: () => {
      toast.error(t("The retrieval settings could not be saved. Reload the page and try again."));
    },
  });
}
