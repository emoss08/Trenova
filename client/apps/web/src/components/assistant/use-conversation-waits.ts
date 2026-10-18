import { useT } from "@trenova/shared/i18n/use-t";
import { agentExtension } from "@/lib/queries/agent-extension";
import { assistant } from "@/lib/queries/assistant";
import { apiService } from "@/services/api";
import type { AgentWait, AgentWaitList } from "@/types/assistant";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";
import { toast } from "@trenova/shared/components/ui/toast";
import { turnFailureDetail } from "./follow-turn";

const NO_WAITS: AgentWait[] = [];

/**
 * What the conversation's agent is waiting on. The server holds each wait and
 * picks the work up as a new turn when it ends; this only shows them and lets
 * the person cancel one, or stop the web watch one follows, which ends its
 * wait. Moved by the "agent_waits" realtime event.
 */
export function useConversationWaits(threadId: string, enabled: boolean) {
  const t = useT();
  const queryClient = useQueryClient();
  const key = assistant.waits(threadId).queryKey;
  const query = useQuery({ ...assistant.waits(threadId), enabled });
  const all = query.data?.items ?? NO_WAITS;

  const open = useMemo(() => all.filter((wait) => wait.status === "Waiting"), [all]);
  const byId = useMemo(() => new Map(all.map((wait) => [wait.id, wait])), [all]);

  const close = useCallback(
    async (wait: AgentWait, end: () => Promise<unknown>, failure: string) => {
      queryClient.setQueryData<AgentWaitList>(key, (current) => ({
        items: (current?.items ?? []).map((known) =>
          known.id === wait.id ? { ...known, status: "Cancelled" } : known,
        ),
      }));
      try {
        await end();
      } catch (error) {
        toast.error(failure, {
          description: turnFailureDetail(error, t("Try again in a moment.")),
        });
      } finally {
        void queryClient.invalidateQueries({ queryKey: key });
      }
    },
    [key, queryClient, t],
  );

  const cancel = useCallback(
    (wait: AgentWait) =>
      close(
        wait,
        () => apiService.assistantService.cancelWait(threadId, wait.id),
        t("The wait could not be cancelled"),
      ),
    [close, t, threadId],
  );

  const stopWatch = useCallback(
    async (wait: AgentWait) => {
      const watchId = wait.webWatch?.watchId;
      if (!watchId) {
        return;
      }
      await close(
        wait,
        () => apiService.agentExtensionService.stopWebWatch(watchId),
        t("The web watch could not be stopped"),
      );
      void queryClient.invalidateQueries({ queryKey: agentExtension.webWatches._def });
    },
    [close, queryClient, t],
  );

  return { open, byId, cancel, stopWatch };
}

export type ConversationWaits = ReturnType<typeof useConversationWaits>;

const WAIT_ID = /Wait id: (awt_[A-Za-z0-9]+)/u;

/** The wait a turn's opening note picks up, named at the end of the note. */
export function waitIdOfNote(content: string): string | null {
  return WAIT_ID.exec(content)?.[1] ?? null;
}
