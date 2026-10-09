import { useT } from "@trenova/shared/i18n/use-t";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { AgentWait, AgentWaitList } from "@/types/assistant";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { turnFailureDetail } from "./follow-turn";

const NO_WAITS: AgentWait[] = [];

/**
 * What the conversation's agent is waiting on. The server holds each wait and
 * picks the work up as a new turn when it ends; this only shows them and lets
 * the person cancel one. Moved by the "agent_waits" realtime event.
 */
export function useConversationWaits(threadId: string, enabled: boolean) {
  const t = useT();
  const queryClient = useQueryClient();
  const key = queries.assistant.waits(threadId).queryKey;
  const query = useQuery({ ...queries.assistant.waits(threadId), enabled });
  const all = query.data?.items ?? NO_WAITS;

  const open = useMemo(() => all.filter((wait) => wait.status === "Waiting"), [all]);
  const byId = useMemo(() => new Map(all.map((wait) => [wait.id, wait])), [all]);

  const cancel = useCallback(
    async (wait: AgentWait) => {
      queryClient.setQueryData<AgentWaitList>(key, (current) => ({
        items: (current?.items ?? []).map((known) =>
          known.id === wait.id ? { ...known, status: "Cancelled" } : known,
        ),
      }));
      try {
        await apiService.assistantService.cancelWait(threadId, wait.id);
      } catch (error) {
        toast.error(t("The wait could not be cancelled"), {
          description: turnFailureDetail(error, t("Try again in a moment.")),
        });
      } finally {
        void queryClient.invalidateQueries({ queryKey: key });
      }
    },
    [key, queryClient, t, threadId],
  );

  return { open, byId, cancel };
}

export type ConversationWaits = ReturnType<typeof useConversationWaits>;

const WAIT_ID = /Wait id: (awt_[A-Za-z0-9]+)/u;

/** The wait a turn's opening note picks up, named at the end of the note. */
export function waitIdOfNote(content: string): string | null {
  return WAIT_ID.exec(content)?.[1] ?? null;
}
