import type { DeskStartExtras } from "@/components/desk-chat/desk-home-ask";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { assistant } from "@/lib/queries/assistant";
import { assistantService } from "@/services/assistant";
import { useAssistantStore } from "@/stores/assistant-store";
import { useDeskAttachedPageStore } from "@/stores/desk-attached-page-store";
import { useDeskHandoffStore } from "@/stores/desk-handoff-store";
import type {
  AssistantPageContext,
  AssistantThread,
  CaseSubjectType,
  ThreadOrigin,
} from "@/types/assistant";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

/**
 * What a conversation is about from its first turn: a title, the case it
 * belongs to, and the page every question in it carries.
 */
export type StartBinding = {
  title?: string;
  subjectType?: CaseSubjectType;
  subjectId?: string;
  page?: AssistantPageContext;
};

type StartVariables = {
  agentId: string;
  question?: string;
  extras?: DeskStartExtras;
  binding?: StartBinding;
};

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
  const attachPage = useDeskAttachedPageStore((state) => state.attach);

  const mutation = useApiMutation({
    mutationFn: ({ agentId, binding }: StartVariables) =>
      assistantService.startThread(agentId, {
        origin,
        title: binding?.title,
        subjectType: binding?.subjectType,
        subjectId: binding?.subjectId,
      }),
    onSuccess: async (thread, { agentId, question, extras, binding }) => {
      setLastAgentId(agentId);
      if (binding?.page) {
        attachPage(thread.id, binding.page);
      }
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
        queryClient.invalidateQueries({ queryKey: assistant.threads().queryKey }),
        queryClient.invalidateQueries({ queryKey: assistant.thread._def }),
      ]);
      onStarted(thread);
    },
    resourceName: "Conversation",
  });

  const { mutate } = mutation;
  const start = useCallback(
    (agentId: string, question?: string, extras?: DeskStartExtras, binding?: StartBinding) =>
      mutate({ agentId, question, extras, binding }),
    [mutate],
  );

  return { start, isStarting: mutation.isPending };
}
