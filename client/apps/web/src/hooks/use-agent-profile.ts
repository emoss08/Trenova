import {
  AGENT_PROFILE_KEY,
  addAgentProfileGuidance,
  agentProfileQueryKey,
  fetchAgentProfile,
  regenerateAgentProfile,
  retireAgentProfileGuidance,
  type AgentProfileOwner,
} from "@/lib/graphql/agent-profile";
import { describeApiError } from "@/lib/api-error-message";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { toast } from "sonner";

/**
 * A profile agents are told, and what a person may do to it: give a line of guidance,
 * retire one, or ask for it to be written again. The write-up itself is never edited
 * here; guidance is how a person steers it, and the next version honours every line.
 */
export function useAgentProfile(owner: AgentProfileOwner, options?: { enabled?: boolean }) {
  const t = useT();
  const queryClient = useQueryClient();
  const enabled = options?.enabled ?? true;

  const profileQuery = useQuery({
    queryKey: agentProfileQueryKey(owner),
    queryFn: ({ signal }) => fetchAgentProfile(owner, { signal }),
    enabled,
  });

  const refresh = () => queryClient.invalidateQueries({ queryKey: [AGENT_PROFILE_KEY] });

  const addGuidance = useMutation({
    mutationFn: (content: string) => addAgentProfileGuidance(owner, content),
    onSuccess: async () => {
      toast.success(t("Guidance added; the profile will be written again with it shortly"));
      await refresh();
    },
    onError: (error) => {
      toast.error(describeApiError(error, t("The guidance could not be added")));
    },
  });

  const retireGuidance = useMutation({
    mutationFn: (id: string) => retireAgentProfileGuidance(owner, id),
    onSuccess: async () => {
      toast.success(t("Guidance retired; the profile will be written again without it"));
      await refresh();
    },
    onError: (error) => {
      toast.error(describeApiError(error, t("The guidance could not be retired")));
    },
  });

  const regenerate = useMutation({
    mutationFn: () => regenerateAgentProfile(owner),
    onSuccess: () => {
      toast.success(t("The profile is being written again; it will update here when it's done"));
    },
    onError: (error) => {
      toast.error(describeApiError(error, t("The profile could not be written again")));
    },
  });

  return { profileQuery, addGuidance, retireGuidance, regenerate };
}
