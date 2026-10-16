import {
  AI_RETRIEVAL_FAILED_LIST_KEY,
  updateAIRetrievalSettings,
  type AIRetrievalSettingsPatch,
} from "@/lib/graphql/ai-retrieval";
import { aiRetrieval } from "@/lib/queries/ai-retrieval";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { toast } from "@trenova/shared/components/ui/toast";

/**
 * Saves one retrieval setting the moment it changes. Only the field changed is sent, so
 * a switch never overwrites a budget someone else raised meanwhile.
 */
export function useRetrievalPatch() {
  const t = useT();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (patch: AIRetrievalSettingsPatch) => updateAIRetrievalSettings(patch),
    onSuccess: async (status) => {
      queryClient.setQueryData(aiRetrieval.status().queryKey, status);
      await queryClient.invalidateQueries({ queryKey: [AI_RETRIEVAL_FAILED_LIST_KEY] });
    },
    onError: () => {
      toast.error(t("The retrieval settings could not be saved. Reload the page and try again."));
    },
  });
}
