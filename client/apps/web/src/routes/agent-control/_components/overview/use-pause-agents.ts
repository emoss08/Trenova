import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  AGENT_CONTROL_QUERY_KEY,
  updateAgentControl,
  type AgentControl,
} from "@/lib/graphql/agent-control";
import { queries } from "@/lib/queries";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
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
    onSuccess: async (saved, shadowMode) => {
      queryClient.setQueryData(AGENT_CONTROL_QUERY_KEY, saved);
      await queryClient.invalidateQueries({ queryKey: queries.aiControl._def });
      toast.success(
        shadowMode
          ? t("All agents paused · they keep running in shadow")
          : t("Agents resumed · held proposals are back in Desk"),
      );
    },
  });
}
