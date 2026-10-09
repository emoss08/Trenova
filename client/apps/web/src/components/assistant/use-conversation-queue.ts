import { useT } from "@trenova/shared/i18n/use-t";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { QueueOutcome } from "@/services/assistant";
import type {
  AssistantEntityRef,
  AssistantPageContext,
  AssistantSurface,
  QueuedMessage,
  QueuedMessageList,
} from "@/types/assistant";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import type { ComposerPayload } from "./composer-types";
import { turnFailureDetail } from "./follow-turn";

const NO_ITEMS: QueuedMessage[] = [];

/** How a message typed while the agent works is handed over. */
export type QueueMode = "steer" | "queue";

export type ConversationQueueOptions = {
  /** The conversation can take messages at all; a read-only one has no queue. */
  enabled: boolean;
  /** Messages the reply under way has already read, which leave the list at once. */
  steered: ReadonlySet<string>;
  /** The page, model and records the message is sent with. */
  context: () => AssistantPageContext | null;
  surface?: AssistantSurface;
  providerId: string;
  /** A message the queue sent as a reply of its own, for the view to follow. */
  onStarted: (turnId: string, content: string, mentions: AssistantEntityRef[]) => void;
};

/**
 * What the person has left for a conversation while its agent works: kept on
 * the server, so it survives a reload or another tab, and moved by the
 * "assistant_queue" realtime event. Each change is applied to the cache as it
 * is asked for and put right by the server's answer.
 */
export function useConversationQueue(threadId: string, options: ConversationQueueOptions) {
  const t = useT();
  const queryClient = useQueryClient();
  const { enabled, steered, context, surface, providerId, onStarted } = options;
  const key = queries.assistant.queue(threadId).queryKey;
  const query = useQuery({ ...queries.assistant.queue(threadId), enabled });

  const items = useMemo(() => {
    const all = query.data?.items ?? NO_ITEMS;
    return steered.size === 0 ? all : all.filter((item) => !steered.has(item.id));
  }, [query.data, steered]);

  const write = useCallback(
    (update: (items: QueuedMessage[]) => QueuedMessage[]) =>
      queryClient.setQueryData<QueuedMessageList>(key, (current) => ({
        items: update(current?.items ?? []),
      })),
    [key, queryClient],
  );

  const refresh = useCallback(
    () => queryClient.invalidateQueries({ queryKey: key }),
    [key, queryClient],
  );

  const failed = useCallback(
    (title: string, error: unknown) => {
      toast.error(title, {
        description: turnFailureDetail(error, t("Try again in a moment.")),
      });
      void refresh();
    },
    [refresh, t],
  );

  const settle = useCallback(
    (outcome: QueueOutcome, content: string, mentions: AssistantEntityRef[]) => {
      const item = outcome.item;
      if (item) {
        write((current) => [...current.filter((known) => known.id !== item.id), item]);
      }
      if (outcome.held !== "") {
        toast.warning(t("The message is waiting"), { description: outcome.held });
      }
      if (outcome.turn) {
        onStarted(outcome.turn.turnId, content, mentions);
      }
      void refresh();
    },
    [onStarted, refresh, t, write],
  );

  const add = useCallback(
    async (content: string, payload: ComposerPayload, mode: QueueMode) => {
      const mentions = [...payload.mentions];
      try {
        const outcome = await apiService.assistantService.enqueue(threadId, content, {
          context: context(),
          surface,
          providerId,
          mentions,
          attachmentDocumentIds:
            mode === "queue" ? payload.attachments.map((item) => item.documentId) : [],
          steer: mode === "steer",
        });
        settle(outcome, content, mentions);
        return true;
      } catch (error) {
        failed(
          mode === "steer"
            ? t("The message could not reach the agent")
            : t("The message could not be queued"),
          error,
        );
        return false;
      }
    },
    [context, failed, providerId, settle, surface, t, threadId],
  );

  const edit = useCallback(
    async (item: QueuedMessage, content: string) => {
      write((current) =>
        current.map((known) => (known.id === item.id ? { ...known, content } : known)),
      );
      try {
        const saved = await apiService.assistantService.editQueued(threadId, item, content);
        write((current) => current.map((known) => (known.id === saved.id ? saved : known)));
        return true;
      } catch (error) {
        failed(t("The message could not be changed"), error);
        return false;
      }
    },
    [failed, t, threadId, write],
  );

  const remove = useCallback(
    async (item: QueuedMessage) => {
      write((current) => current.filter((known) => known.id !== item.id));
      try {
        await apiService.assistantService.removeQueued(threadId, item.id);
      } catch (error) {
        failed(t("The message could not be removed"), error);
      }
    },
    [failed, t, threadId, write],
  );

  const move = useCallback(
    async (item: QueuedMessage, offset: -1 | 1) => {
      const all = query.data?.items ?? NO_ITEMS;
      const from = all.findIndex((known) => known.id === item.id);
      const to = from + offset;
      if (from === -1 || to < 0 || to >= all.length) {
        return;
      }
      const ordered = [...all];
      [ordered[from], ordered[to]] = [ordered[to], ordered[from]];
      write(() => ordered);
      try {
        const saved = await apiService.assistantService.reorderQueue(
          threadId,
          ordered.map((known) => known.id),
        );
        write(() => saved.items);
      } catch (error) {
        failed(t("The order could not be saved"), error);
      }
    },
    [failed, query.data, t, threadId, write],
  );

  const sendNow = useCallback(
    async (item: QueuedMessage) => {
      try {
        const outcome = await apiService.assistantService.sendQueued(threadId, item.id);
        if (outcome.turn) {
          write((current) => current.filter((known) => known.id !== item.id));
        }
        settle(outcome, item.content, item.request.mentions);
      } catch (error) {
        failed(t("The message could not be sent"), error);
      }
    },
    [failed, settle, t, threadId, write],
  );

  return { items, add, edit, remove, move, sendNow };
}

export type ConversationQueue = ReturnType<typeof useConversationQueue>;
