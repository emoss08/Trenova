import { useT } from "@trenova/shared/i18n/use-t";
import { assistant } from "@/lib/queries/assistant";
import { apiService } from "@/services/api";
import type {
  AssistantEntityRef,
  AssistantPageContext,
  AssistantStreamEvent,
} from "@/types/assistant";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";
import { runTurn, stopTurnQuietly, turnFailureDetail } from "./follow-turn";
import { registerTurnReader } from "./turn-readers";
import { initialTurnState, isTurnActive, advanceTurn, type TurnState } from "./turn-stream";

type FindMentions = (signal: AbortSignal) => Promise<readonly AssistantEntityRef[]>;

/**
 * The records a question is about: named outright, or found while it is on
 * its way. Finding them is part of asking, so it is cancelled with the
 * question, and a search that fails sends the question without them rather
 * than losing it.
 */
export type AskQuestionOptions = { context?: AssistantPageContext | null } & (
  | { mentions?: readonly AssistantEntityRef[]; findMentions?: never }
  | { mentions?: never; findMentions: FindMentions }
);

async function questionMentions(
  options: AskQuestionOptions,
  signal: AbortSignal,
): Promise<readonly AssistantEntityRef[]> {
  if (!options.findMentions) {
    return options.mentions ?? [];
  }
  let found: readonly AssistantEntityRef[];
  try {
    found = await options.findMentions(signal);
  } catch (error) {
    if (signal.aborted) {
      throw error;
    }
    found = [];
  }
  signal.throwIfAborted();
  return found;
}

/**
 * One quick question at a time, from the palette. The answer streams into a
 * TurnState like a thread's turn does, and the thread it was answered on is
 * kept only when the person opens it in the Desk to continue.
 *
 * The answer is read in the palette and nowhere else: the server neither
 * announces it nor lists it in the Desk. So closing the palette or asking
 * something else takes the question back — a question on its way is
 * withdrawn and a turn under way is stopped — rather than leaving a worker to
 * write an answer nobody will see. A kept question is a Desk conversation, and
 * its turn runs on there.
 */
export function useAsk() {
  const t = useT();
  const queryClient = useQueryClient();
  const [turn, setTurn] = useState<TurnState | null>(null);
  // Letting go of the answer, and taking the question back, are two
  // different things, so they are two signals.
  const abortRef = useRef<AbortController | null>(null);
  const withdrawRef = useRef<AbortController | null>(null);
  // The turn a worker is answering. Stopping has to reach it: closing the
  // reader leaves the turn running and billing.
  const turnIdRef = useRef<string | null>(null);

  // Takes back whatever question is out: the turn a worker is answering is
  // stopped, a question on its way is withdrawn, and the reader lets go.
  const withdrawQuestion = useCallback(() => {
    const running = turnIdRef.current;
    turnIdRef.current = null;
    if (running !== null) {
      stopTurnQuietly(running);
    }
    withdrawRef.current?.abort();
    withdrawRef.current = null;
    abortRef.current?.abort();
    abortRef.current = null;
  }, []);

  useEffect(() => withdrawQuestion, [withdrawQuestion]);

  const ask = useCallback(
    async (content: string, options: AskQuestionOptions = {}) => {
      withdrawQuestion();
      const controller = new AbortController();
      abortRef.current = controller;
      const withdraw = new AbortController();
      withdrawRef.current = withdraw;
      const unregister = registerTurnReader(controller);

      setTurn(
        initialTurnState(content, options.context ?? null, {
          mentions: options.mentions ? [...options.mentions] : undefined,
        }),
      );

      let terminal = false;
      const onEvent = (event: AssistantStreamEvent) => {
        setTurn((state) => (state ? advanceTurn(state, event) : state));
        if (event.event === "done" || event.event === "error") {
          terminal = true;
        }
      };

      turnIdRef.current = null;
      try {
        await runTurn(
          async (signal) => {
            const mentions = await questionMentions(options, signal);
            if (options.findMentions) {
              setTurn((state) => (state ? { ...state, mentions: [...mentions] } : state));
            }
            return apiService.assistantService.startAsk(
              content,
              { context: options.context, mentions },
              { signal },
            );
          },
          {
            signal: controller.signal,
            withdrawSignal: withdraw.signal,
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
        unregister();
        const detail = turnFailureDetail(error, t("The connection to the assistant was lost."));
        setTurn((state) => (state ? { ...state, status: "error", error: detail } : state));
        return;
      }
      if (controller.signal.aborted) {
        return;
      }
      unregister();
      if (terminal) {
        // Nothing is left to stop once the answer has ended.
        turnIdRef.current = null;
        return;
      }
      setTurn((state) =>
        state && isTurnActive(state)
          ? { ...state, status: "error", error: t("The reply was cut off before it finished.") }
          : state,
      );
    },
    [t, withdrawQuestion],
  );

  const stop = useCallback(() => {
    withdrawQuestion();
    setTurn((state) =>
      state && isTurnActive(state) ? { ...state, status: "error", error: t("Stopped.") } : state,
    );
  }, [t, withdrawQuestion]);

  // Puts the question away when the palette closes or moves on from it.
  const reset = useCallback(() => {
    withdrawQuestion();
    setTurn(null);
  }, [withdrawQuestion]);

  /**
   * Lists the question as a conversation and says where it is. The thread
   * exists already; keeping it is what makes the rail show it, and from then
   * on its turn belongs to the Desk, so putting the palette away no longer
   * stops it.
   */
  const keep = useCallback(async (): Promise<string | null> => {
    const threadId = turn?.thread?.id ?? turn?.result?.thread.id ?? null;
    if (threadId === null) {
      return null;
    }
    await apiService.assistantService.updateThread(threadId, { keep: true });
    turnIdRef.current = null;
    withdrawRef.current = null;
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: assistant.threads().queryKey }),
      queryClient.invalidateQueries({ queryKey: assistant.activeTurns().queryKey }),
    ]);

    return threadId;
  }, [queryClient, turn?.result?.thread.id, turn?.thread?.id]);

  return { turn, isActive: isTurnActive(turn), ask, stop, reset, keep };
}
