import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useAssistantStore } from "@/stores/assistant-store";
import type {
  AssistantLiveTurnList,
  AssistantThread,
  assistantThreadListSchema,
} from "@/types/assistant";
import type { QueryClient } from "@tanstack/react-query";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { toast } from "sonner";
import type { z } from "zod";
import { isViewingThread, type ReplyReady, type ViewingState } from "./reply-ready";

type ThreadList = z.infer<typeof assistantThreadListSchema>;

export type AnnounceReplyReadyOptions = {
  reply: ReplyReady;
  /** The notification carrying it, marked read once the person has seen the reply. */
  notificationId: string | null;
  queryClient: QueryClient;
  navigate: (to: string) => void;
  t: TranslateFn;
  /** Where the person is. Read from the page and the panel when not given. */
  viewing?: () => ViewingState;
};

export type ReplyReadyOutcome = "shown" | "suppressed";

/**
 * Tells the person a reply they walked away from is finished, with a way
 * straight to it.
 *
 * The notification itself says nothing about which conversation: it travels
 * on a channel every member of the organization receives, so its words are
 * generic by design. What the notice names is read here, from the
 * conversation the reply belongs to.
 *
 * The server writes the notice only for a reply nobody was reading, but a
 * person can open the conversation in the moment between the two. When they
 * are looking at it already the notice would be about the thing in front of
 * them, so nothing is shown and the notice is marked read — they have seen
 * the reply.
 *
 * A failed reply takes the danger tone because it needs the person to ask
 * again; a completed or refused one is simply news, and stays neutral.
 */
export async function announceReplyReady(
  options: AnnounceReplyReadyOptions,
): Promise<ReplyReadyOutcome> {
  const { reply, queryClient, t } = options;

  // Read before the caches are refreshed: the list of live replies still
  // names the conversation this reply was written in, and is the one place
  // that knows the title of a palette question nobody kept.
  const cachedTitle = cachedConversationTitle(queryClient, reply.threadId);

  // The reply is over wherever it is shown: the markers clear and the
  // conversation reads what was saved rather than what a reader last saw.
  void Promise.all([
    queryClient.invalidateQueries({ queryKey: queries.assistant.activeTurns().queryKey }),
    queryClient.invalidateQueries({ queryKey: queries.assistant.threads().queryKey }),
    queryClient.invalidateQueries({
      queryKey: queries.assistant.messages(reply.threadId).queryKey,
    }),
  ]);

  const conversation = cachedTitle ?? (await fetchConversationTitle(queryClient, reply.threadId));

  if (isViewingThread(reply.threadId, (options.viewing ?? currentViewing)())) {
    markSeen(options);
    return "suppressed";
  }

  const toastOptions = {
    // One toast per reply, however many times the notice is delivered.
    id: `assistant-reply-${reply.turnId || reply.threadId}`,
    description:
      conversation === null
        ? undefined
        : conversation === ""
          ? t("Untitled conversation")
          : conversation,
    action: {
      label: t("Open"),
      onClick: () => {
        markSeen(options);
        options.navigate(reply.link);
      },
    },
  };
  const title = replyTitle(t, reply);

  if (reply.status === "Failed") {
    toast.error(title, toastOptions);
  } else {
    toast(title, toastOptions);
  }

  return "shown";
}

function replyTitle(t: TranslateFn, reply: ReplyReady): string {
  switch (reply.status) {
    case "Failed":
      return t("Your reply could not finish");
    case "Refused":
      return t("The assistant declined to answer");
    default:
      return t("Your reply is ready");
  }
}

/**
 * The conversation's title from what this tab already holds: the list, the
 * conversation read on its own, or the live reply that was writing in it.
 * Empty when the conversation is untitled; null when nothing here knows it.
 */
export function cachedConversationTitle(queryClient: QueryClient, threadId: string): string | null {
  const listed = queryClient
    .getQueryData<ThreadList>(queries.assistant.threads().queryKey)
    ?.items.find((thread) => thread.id === threadId);
  if (listed) {
    return listed.title;
  }

  const single = queryClient.getQueryData<AssistantThread>(
    queries.assistant.thread(threadId).queryKey,
  );
  if (single) {
    return single.title;
  }

  const live = queryClient
    .getQueryData<AssistantLiveTurnList>(queries.assistant.activeTurns().queryKey)
    ?.items.find((turn) => turn.threadId === threadId);

  return live ? live.threadTitle : null;
}

/**
 * Reads the conversation when nothing here holds it — a palette question
 * asked in another tab, most often. A conversation that cannot be read is
 * announced without its name rather than not at all.
 */
async function fetchConversationTitle(
  queryClient: QueryClient,
  threadId: string,
): Promise<string | null> {
  try {
    const thread = await queryClient.fetchQuery({
      ...queries.assistant.thread(threadId),
      retry: false,
    });
    return thread.title;
  } catch {
    return null;
  }
}

function currentViewing(): ViewingState {
  const { open, activeThreadId } = useAssistantStore.getState();

  return {
    pathname: typeof window === "undefined" ? "" : window.location.pathname,
    panelOpen: open,
    panelThreadId: activeThreadId,
  };
}

function markSeen({ notificationId, queryClient }: AnnounceReplyReadyOptions): void {
  if (notificationId === null || notificationId === "") {
    return;
  }
  apiService.notificationService
    .markRead([notificationId])
    .then(() => queryClient.invalidateQueries({ queryKey: queries.notification._def }))
    .catch(() => undefined);
}
