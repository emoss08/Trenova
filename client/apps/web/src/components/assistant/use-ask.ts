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
import type { StartedAsk } from "@/services/assistant";
import { followStream, turnFailureDetail } from "./follow-turn";
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

/** Takes back a palette question; nobody is left to tell if it fails. */
function stopAskQuietly(askId: string): void {
  void apiService.assistantService.stopAsk(askId).catch(() => undefined);
}

/**
 * One quick question at a time, from the palette. Nova answers it on a worker
 * without making a conversation, and the answer streams into a TurnState the
 * way a thread's turn does.
 *
 * The answer is read in the palette and nowhere else, so closing the palette
 * or asking something else takes the question back: a question on its way is
 * withdrawn and an answer being written is stopped, rather than leaving a
 * worker to write an answer nobody will see. Continuing in the Desk carries
 * the question and its answer into a new conversation.
 */
export function useAsk() {
  const t = useT();
  const queryClient = useQueryClient();
  const [turn, setTurn] = useState<TurnState | null>(null);
  // Letting go of the answer, and taking the question back, are two
  // different things, so they are two signals.
  const abortRef = useRef<AbortController | null>(null);
  const withdrawRef = useRef<AbortController | null>(null);
  // The question a worker has, and whether its answer has ended. Stopping has
  // to reach a worker still writing; one that has finished is kept to carry
  // into the Desk.
  const askRef = useRef<{ id: string; ended: boolean } | null>(null);
  // Whether the question on show reached a worker and ended, answered or
  // not: then it can be carried into the Desk.
  const [keepable, setKeepable] = useState(false);

  const withdrawQuestion = useCallback(() => {
    const current = askRef.current;
    askRef.current = null;
    setKeepable(false);
    if (current !== null && !current.ended) {
      stopAskQuietly(current.id);
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
      const fail = (error: unknown) => {
        unregister();
        const detail = turnFailureDetail(error, t("The connection to the assistant was lost."));
        setTurn((state) => (state ? { ...state, status: "error", error: detail } : state));
      };

      setTurn(
        initialTurnState(content, options.context ?? null, {
          mentions: options.mentions ? [...options.mentions] : undefined,
        }),
      );

      let started: StartedAsk;
      try {
        const mentions = await questionMentions(options, withdraw.signal);
        if (options.findMentions) {
          setTurn((state) => (state ? { ...state, mentions: [...mentions] } : state));
        }
        started = await apiService.assistantService.startAsk(
          content,
          { context: options.context, mentions },
          { signal: withdraw.signal },
        );
      } catch (error) {
        if (withdraw.signal.aborted || controller.signal.aborted) {
          return;
        }
        fail(error);
        return;
      }
      // Stop was pressed while the question was on its way: the worker has
      // it now and would answer nobody.
      if (withdraw.signal.aborted) {
        stopAskQuietly(started.askId);
        return;
      }

      const current = { id: started.askId, ended: false };
      askRef.current = current;
      const onEvent = (event: AssistantStreamEvent) => {
        setTurn((state) => (state ? advanceTurn(state, event) : state));
        if (event.event === "done" || event.event === "error") {
          current.ended = true;
          setKeepable(true);
        }
      };

      try {
        await followStream(
          (apply, attach) => apiService.assistantService.attachAsk(started.askId, apply, attach),
          { signal: controller.signal, onEvent },
        );
      } catch (error) {
        if (controller.signal.aborted) {
          return;
        }
        fail(error);
        return;
      }
      if (controller.signal.aborted) {
        return;
      }
      unregister();
      if (current.ended) {
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

  // A second press while the first is carrying the question waits on it,
  // so one question never opens two conversations.
  const keepingRef = useRef<Promise<string | null> | null>(null);

  /**
   * Carries the answered question into a Desk conversation, which opens with
   * the question and Nova's answer, and says which. A question with no
   * answer to carry is asked again there.
   */
  const keep = useCallback((): Promise<string | null> => {
    if (keepingRef.current !== null) {
      return keepingRef.current;
    }
    const current = askRef.current;
    if (current === null || !current.ended) {
      return Promise.resolve(null);
    }

    const keeping = (async () => {
      const kept = await apiService.assistantService.keepAsk(current.id);
      if (kept.unanswered !== "") {
        await apiService.assistantService.startTurn(kept.thread.id, kept.unanswered);
      }
      await queryClient.invalidateQueries({ queryKey: assistant.threads().queryKey });

      return kept.thread.id;
    })().finally(() => {
      keepingRef.current = null;
    });
    keepingRef.current = keeping;

    return keeping;
  }, [queryClient]);

  return { turn, isActive: isTurnActive(turn), ask, stop, reset, keep, keepable };
}
