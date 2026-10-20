import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  AGENT_CONTROL_QUERY_KEY,
  updateAgentControl,
  type AgentControl,
} from "@/lib/graphql/agent-control";
import { aiControl } from "@/lib/queries/ai-control";
import { invalidateProposalViews } from "@/lib/proposal-cache";
import { assistant } from "@/lib/queries/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQueryClient } from "@tanstack/react-query";
import { controlInput } from "../agent-control-options";

/** Pauses every agent (true) or resumes them (false), organization-wide. */
export function usePauseAgents(control: AgentControl | undefined) {
  const t = useT();
  const queryClient = useQueryClient();

  return useApiMutation({
    mutationFn: (shadowMode: boolean) => {
      if (!control) {
        throw new Error("The organization's settings have not loaded");
      }
      return updateAgentControl(controlInput(control, { shadowMode }));
    },
    resourceName: t("Organization-wide settings"),
    onSuccess: async (saved) => {
      queryClient.setQueryData(AGENT_CONTROL_QUERY_KEY, saved);
      // Every waiting proposal is held or freed with the pause, so every
      // approval surface, the Desk queue and its badge included, is read again.
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: aiControl._def }),
        queryClient.invalidateQueries({ queryKey: assistant._def }),
        invalidateProposalViews(queryClient),
      ]);
    },
  });
}
