import { useApiMutation } from "@/hooks/use-api-mutation";
import { reviewInboundMessage, type InboundMessageDetail } from "@/lib/graphql/inbox";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useAssistantStore } from "@/stores/assistant-store";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { useCallback } from "react";
import { useNavigate } from "react-router";
import { toast } from "sonner";

export type ReviewOutcome = "Actioned" | "Ignored";

/**
 * What a person does to a message, in one place, so the keys and the buttons
 * cannot drift: marking it handled or ignored, and asking the desk about it.
 *
 * `onReviewed` is told when a message has been decided, so the page can
 * move on to the next one the way a triage run does.
 */
export function useInboxActions({ onReviewed }: { onReviewed: (id: string) => void }) {
  const t = useT();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const refresh = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: queries.inbox._def }),
      queryClient.invalidateQueries({ queryKey: queries.watchtower._def }),
    ]);
  }, [queryClient]);

  const reviewMutation = useApiMutation({
    mutationFn: ({ id, status, note }: { id: string; status: ReviewOutcome; note: string }) =>
      reviewInboundMessage(id, { status, note: note.trim() === "" ? null : note.trim() }),
    onSuccess: async (updated) => {
      toast.success(
        updated.status === "Actioned" ? t("Marked as handled") : t("Marked as ignored"),
      );
      onReviewed(updated.id);
      await refresh();
    },
    resourceName: "Message",
  });

  // Asking opens a conversation on the message itself, so the agent starts
  // from the mail rather than from "which message?". The agent is the one the
  // person last talked to, on the same rule the rest of the Desk uses.
  const agentsQuery = useQuery(queries.assistant.agents(true, true));
  const lastAgentId = useAssistantStore((state) => state.lastAgentId);
  const askAgent =
    agentsQuery.data?.find((agent) => agent.id === lastAgentId) ?? agentsQuery.data?.[0] ?? null;

  const askMutation = useApiMutation({
    mutationFn: (message: Pick<InboundMessageDetail, "id" | "subject">) => {
      if (askAgent === null) {
        throw new Error(t("No agents are available to ask"));
      }

      return apiService.assistantService.startThread(askAgent.id, {
        origin: "Desk",
        title: message.subject === "" ? t("About an inbound message") : message.subject,
        subjectType: "InboundMessage",
        subjectId: message.id,
      });
    },
    onSuccess: (thread) => navigate(`/desk/t/${thread.id}`),
    resourceName: "Conversation",
  });

  return {
    review: reviewMutation.mutate,
    ask: askMutation.mutate,
    canAsk: askAgent !== null,
    refresh,
    busy: reviewMutation.isPending || askMutation.isPending,
  };
}
