import { useT } from "@trenova/shared/i18n/use-t";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { AssistantStreamError } from "@/services/assistant";
import type {
  AssistantPageContext,
  AssistantStreamEvent,
  SendMessageResult,
} from "@/types/assistant";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { appendToHistory, type ThreadHistory } from "./thread-history";
import { initialTurnState, isTurnActive, reduceTurn, type TurnState } from "./turn-stream";

/**
 * Drives one turn at a time for a thread: opens the stream, folds its events
 * into a TurnState the view renders, and hands over to the saved thread once
 * the server has written it.
 *
 * The transient turn is cleared only after the refetch lands, so the reply
 * never blinks out and back in between "streamed" and "saved".
 */
export function useAssistantTurn(threadId: string, getContext?: () => AssistantPageContext | null) {
  const t = useT();
  const queryClient = useQueryClient();

  const [turn, setTurn] = useState<TurnState | null>(null);
  const abortRef = useRef<AbortController | null>(null);
  const lastContextRef = useRef<AssistantPageContext | null>(null);

  useEffect(() => {
    return () => abortRef.current?.abort();
  }, []);

  const refreshThread = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: queries.assistant.messages(threadId).queryKey }),
      queryClient.invalidateQueries({ queryKey: queries.assistant.proposals(threadId).queryKey }),
      queryClient.invalidateQueries({ queryKey: queries.assistant.threads().queryKey }),
    ]);
  }, [queryClient, threadId]);

  // A finished turn is appended to the history the thread already holds
  // rather than refetched: the result carries the rows the server wrote, and a
  // thread with several pages loaded would otherwise fetch every one of them
  // after each reply. Only when nothing is cached does the query fetch fresh.
  const absorbTurn = useCallback(
    async (result: SendMessageResult | null) => {
      const key = queries.assistant.messages(threadId).queryKey;
      const cached = queryClient.getQueryData<ThreadHistory>(key);
      if (result && cached && cached.pages.length > 0) {
        queryClient.setQueryData<ThreadHistory>(key, (history) =>
          appendToHistory(history, result.messages),
        );
        await Promise.all([
          queryClient.invalidateQueries({
            queryKey: queries.assistant.proposals(threadId).queryKey,
          }),
          queryClient.invalidateQueries({ queryKey: queries.assistant.threads().queryKey }),
        ]);
        return;
      }

      await refreshThread();
    },
    [queryClient, refreshThread, threadId],
  );

  const settle = useCallback(
    async (result: SendMessageResult | null) => {
      if (result?.proposalsUnrecorded) {
        toast.warning(t("The assistant proposed a change that could not be saved for approval"), {
          description: t("Nothing was changed. Ask again if you still want to make the change."),
        });
      }
      await absorbTurn(result);
      setTurn(null);
    },
    [absorbTurn, t],
  );

  const send = useCallback(
    async (content: string, context?: AssistantPageContext | null, providerId = "") => {
      abortRef.current?.abort();
      const controller = new AbortController();
      abortRef.current = controller;

      const pageContext = context === undefined ? (getContext?.() ?? null) : context;
      lastContextRef.current = pageContext;

      let terminal = false;
      let done: SendMessageResult | null = null;
      setTurn(initialTurnState(content, pageContext));

      const onEvent = (event: AssistantStreamEvent) => {
        setTurn((state) => (state ? reduceTurn(state, event) : state));
        if (event.event === "done") {
          terminal = true;
          done = event.data;
        } else if (event.event === "refused" || event.event === "error") {
          terminal = true;
        }
      };

      try {
        await apiService.assistantService.streamMessage(
          threadId,
          content,
          onEvent,
          controller.signal,
          pageContext,
          providerId,
        );
      } catch (error) {
        if (controller.signal.aborted) {
          return;
        }
        const message =
          error instanceof AssistantStreamError
            ? error.message
            : t("The connection to the assistant was lost.");
        setTurn((state) => (state ? { ...state, status: "error", error: message } : state));
        return;
      }

      if (controller.signal.aborted) {
        return;
      }

      if (!terminal) {
        // The stream closed without saying how it ended, which a proxy that
        // buffers or cuts long responses can cause. The turn may still have
        // been saved, so the thread is refreshed rather than the reply lost.
        await refreshThread();
        setTurn((state) =>
          state && state.status !== "done"
            ? { ...state, status: "error", error: t("The reply ended before it was finished.") }
            : state,
        );
        return;
      }

      if (done !== null) {
        await settle(done);
      } else {
        // A refusal is complete in itself and has been saved. A server error
        // stays on screen until the person retries or dismisses it — but the
        // thread is refreshed underneath it, because the server now keeps what
        // ran before the failure (the question, the lookups, any write a tool
        // made) and that record belongs in view rather than behind a banner
        // implying nothing happened.
        setTurn((state) => {
          if (state?.status === "refused") {
            void settle(null);
          } else if (state?.status === "error") {
            void refreshThread();
          }
          return state;
        });
      }
    },
    [getContext, refreshThread, settle, t, threadId],
  );

  const stop = useCallback(() => {
    abortRef.current?.abort();
    setTurn((state) =>
      state && isTurnActive(state)
        ? {
            ...state,
            status: "error",
            error: t("Stopped. What was said so far has been kept in the thread."),
          }
        : state,
    );
  }, [t]);

  const dismiss = useCallback(async () => {
    await refreshThread();
    setTurn(null);
  }, [refreshThread]);

  return {
    turn,
    isActive: isTurnActive(turn),
    send,
    stop,
    dismiss,
    retry: turn ? () => send(turn.userContent, lastContextRef.current) : undefined,
  };
}
