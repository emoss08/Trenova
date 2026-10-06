import type { DeskStartExtras } from "@/components/desk-chat/desk-home-ask";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useAssistantStore } from "@/stores/assistant-store";
import { useDeskHandoffStore } from "@/stores/desk-handoff-store";
import type { AssistantThread, ThreadOrigin } from "@/types/assistant";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

type StartVariables = { agentId: string; question?: string; extras?: DeskStartExtras };

/**
 * Starts a conversation, optionally with the question that prompted it and
 * the files, records and model chosen alongside. The Desk and the assistant
 * panel start conversations the same way and differ only in where they go
 * next, which `onStarted` says.
 *
 * The question is handed over rather than sent here: only the conversation
 * knows the thread is empty and that its history has loaded, which is what
 * keeps a reload from asking the same question twice. The files ride the
 * same way and upload once the conversation is open.
 */
export function useStartConversation({
  origin,
  onStarted,
}: {
  origin: ThreadOrigin;
  onStarted: (thread: AssistantThread) => void;
}) {
  const queryClient = useQueryClient();
  const setLastAgentId = useAssistantStore((state) => state.setLastAgentId);
  const setOpeningQuestion = useAssistantStore((state) => state.setOpeningQuestion);
  const setHandoff = useDeskHandoffStore((state) => state.setHandoff);

  const mutation = useApiMutation({
    mutationFn: ({ agentId }: StartVariables) =>
      apiService.assistantService.startThread(agentId, { origin }),
    onSuccess: async (thread, { agentId, question, extras }) => {
      setLastAgentId(agentId);
      if (extras && (extras.files.length > 0 || extras.mentions.length > 0 || extras.providerId)) {
        setHandoff({
          threadId: thread.id,
          files: extras.files,
          mentions: extras.mentions,
          providerId: extras.providerId,
        });
      }
      if (question !== undefined && question !== "") {
        setOpeningQuestion({ threadId: thread.id, text: question });
      }
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queries.assistant.threads().queryKey }),
        queryClient.invalidateQueries({ queryKey: queries.assistant.thread._def }),
      ]);
      onStarted(thread);
    },
    resourceName: "Conversation",
  });

  const { mutate } = mutation;
  const start = useCallback(
    (agentId: string, question?: string, extras?: DeskStartExtras) =>
      mutate({ agentId, question, extras }),
    [mutate],
  );

  return { start, isStarting: mutation.isPending };
}
