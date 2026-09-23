import { useT } from "@trenova/shared/i18n/use-t";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type {
  AssistantEntityRef,
  AssistantPageContext,
  AssistantStreamEvent,
} from "@/types/assistant";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";
import { runTurn, stopTurnQuietly, turnFailureDetail } from "./follow-turn";
import { initialTurnState, isTurnActive, reduceTurn, type TurnState } from "./turn-stream";

/**
 * One quick question at a time, from anywhere. The answer streams into a
 * TurnState like a thread's turn does, and the thread it was answered on is
 * kept so the question can be opened in the Desk and continued.
 */
export function useAsk() {
  const t = useT();
  const queryClient = useQueryClient();
  const [turn, setTurn] = useState<TurnState | null>(null);
  const abortRef = useRef<AbortController | null>(null);
  // The turn a worker is answering. Stopping has to reach it: closing the
  // reader leaves the turn running and billing.
  const turnIdRef = useRef<string | null>(null);

  useEffect(() => () => abortRef.current?.abort(), []);

  const ask = useCallback(
    async (
      content: string,
      options: { context?: AssistantPageContext | null; mentions?: AssistantEntityRef[] } = {},
    ) => {
      abortRef.current?.abort();
      const controller = new AbortController();
      abortRef.current = controller;

      setTurn(initialTurnState(content, options.context ?? null, { mentions: options.mentions }));

      let terminal = false;
      const onEvent = (event: AssistantStreamEvent) => {
        setTurn((state) => (state ? reduceTurn(state, event) : state));
        if (event.event === "done" || event.event === "error") {
          terminal = true;
        }
      };

      turnIdRef.current = null;
      try {
        await runTurn(
          (signal) => apiService.assistantService.startAsk(content, options, { signal }),
          {
            signal: controller.signal,
            onTurnStarted: (started) => {
              turnIdRef.current = started.turnId;
              // The thread is named before the answer, so the question can be
              // kept whatever happens to the answer.
              if (started.thread) {
                onEvent({ event: "thread", data: started.thread });
              }
            },
            onEvent,
          },
        );
      } catch (error) {
        if (controller.signal.aborted) {
          return;
        }
        const detail = turnFailureDetail(error, t("The connection to the assistant was lost."));
        setTurn((state) => (state ? { ...state, status: "error", error: detail } : state));
        return;
      }
      if (controller.signal.aborted) {
        return;
      }
      if (!terminal) {
        setTurn((state) =>
          state && isTurnActive(state)
            ? { ...state, status: "error", error: t("The reply was cut off before it finished.") }
            : state,
        );
      }
    },
    [t],
  );

  const stop = useCallback(() => {
    const running = turnIdRef.current;
    if (running !== null) {
      turnIdRef.current = null;
      stopTurnQuietly(running);
    }
    abortRef.current?.abort();
    abortRef.current = null;
    setTurn((state) =>
      state && isTurnActive(state) ? { ...state, status: "error", error: t("Stopped.") } : state,
    );
  }, [t]);

  const reset = useCallback(() => {
    turnIdRef.current = null;
    abortRef.current?.abort();
    abortRef.current = null;
    setTurn(null);
  }, []);

  /**
   * Lists the question as a conversation and says where it is. The thread
   * exists already; keeping it is what makes the rail show it.
   */
  const keep = useCallback(async (): Promise<string | null> => {
    const threadId = turn?.thread?.id ?? turn?.result?.thread.id ?? null;
    if (threadId === null) {
      return null;
    }
    await apiService.assistantService.updateThread(threadId, { keep: true });
    await queryClient.invalidateQueries({ queryKey: queries.assistant.threads().queryKey });

    return threadId;
  }, [queryClient, turn?.result?.thread.id, turn?.thread?.id]);

  return { turn, isActive: isTurnActive(turn), ask, stop, reset, keep };
}
