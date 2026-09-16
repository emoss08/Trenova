import { useT } from "@trenova/shared/i18n/use-t";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { AssistantStreamError } from "@/services/assistant";
import type { AssistantStreamEvent, SendMessageResult } from "@/types/assistant";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { initialTurnState, isTurnActive, reduceTurn, type TurnState } from "./turn-stream";

/**
 * Drives one turn at a time for a thread: opens the stream, folds its events
 * into a TurnState the view renders, and hands over to the saved thread once
 * the server has written it.
 *
 * The transient turn is cleared only after the refetch lands, so the reply
 * never blinks out and back in between "streamed" and "saved".
 */
export function useAssistantTurn(threadId: string) {
  const t = useT();
  const queryClient = useQueryClient();

  const [turn, setTurn] = useState<TurnState | null>(null);
  const abortRef = useRef<AbortController | null>(null);

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

  const settle = useCallback(
    async (result: SendMessageResult | null) => {
      if (result?.proposalsUnrecorded) {
        toast.warning(t("The assistant proposed a change that could not be saved for approval"), {
          description: t("Nothing was changed. Ask again if you still want to make the change."),
        });
      }
      await refreshThread();
      setTurn(null);
    },
    [refreshThread, t],
  );

  const send = useCallback(
    async (content: string) => {
      abortRef.current?.abort();
      const controller = new AbortController();
      abortRef.current = controller;

      let terminal = false;
      let done: SendMessageResult | null = null;
      setTurn(initialTurnState(content));

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
        await apiService.assistantService.streamMessage(threadId, content, onEvent, controller.signal);
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
        // A refusal is complete in itself and has been saved; a server error
        // stays on screen until the person retries or dismisses it.
        setTurn((state) => {
          if (state?.status === "refused") {
            void settle(null);
          }
          return state;
        });
      }
    },
    [refreshThread, settle, t, threadId],
  );

  const stop = useCallback(() => {
    abortRef.current?.abort();
    setTurn((state) =>
      state && isTurnActive(state)
        ? { ...state, status: "error", error: t("Stopped. The assistant may still finish and save its reply.") }
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
    retry: turn ? () => send(turn.userContent) : undefined,
  };
}
