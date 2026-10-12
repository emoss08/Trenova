import { useT } from "@trenova/shared/i18n/use-t";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import {
  followTurn,
  runTurn,
  stopTurnQuietly,
  turnFailureDetail,
  turnLimitOf,
} from "./follow-turn";
import type { ActiveTurn } from "@/services/assistant";
import type {
  AssistantEntityRef,
  AssistantPageContext,
  AssistantStreamEvent,
  AssistantSurface,
  QueuedMessageList,
  SendMessageResult,
} from "@/types/assistant";
import { useRealtimeStore } from "@/stores/realtime-store";
import type { AssistantLiveTurnList } from "@/types/assistant";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { listSaysQuiet } from "./active-turns";
import { appendToHistory, continuesHistory, type ThreadHistory } from "./thread-history";
import { registerTurnReader } from "./turn-readers";
import {
  describeTurnFailure,
  initialTurnState,
  isTurnActive,
  advanceTurn,
  type NextTurn,
  type TurnContext,
  type TurnFailureCause,
  type TurnFailureKind,
  type TurnLimit,
  type TurnState,
} from "./turn-stream";

/** A send not yet taken by a worker, and whether Stop was pressed on it. */
type PendingSend = { stopped: boolean };

/** The reply the queue started, as the live view follows it. */
function activeFromNext(next: NextTurn, threadId: string): ActiveTurn {
  return { id: next.turnId, threadId, status: "Running", origin: "Person", input: next.input };
}

/**
 * Drives one turn at a time for a thread: opens the stream, folds its events
 * into a TurnState the view renders, and hands over to the saved thread once
 * the server has written it.
 *
 * The transient turn is cleared only after the refetch lands, so the reply
 * never blinks out and back in between "streamed" and "saved".
 */
export function useAssistantTurn(
  threadId: string,
  getContext?: () => AssistantPageContext | null,
  onConversationEvent?: (event: AssistantStreamEvent) => void,
  surface?: AssistantSurface,
) {
  const t = useT();
  const queryClient = useQueryClient();
  // What a turn says about the conversation rather than the reply — how
  // full its context is, a compaction it set off — goes to whoever keeps
  // that. The latest callback is read when an event arrives, so a turn
  // followed across renders never calls a stale one.
  const conversationEventRef = useRef(onConversationEvent);
  useEffect(() => {
    conversationEventRef.current = onConversationEvent;
  }, [onConversationEvent]);

  const [turn, setTurn] = useState<TurnState | null>(null);
  const abortRef = useRef<AbortController | null>(null);
  // The turn a worker is producing, when one is. Stopping needs it: with the
  // work off this request, aborting the reader stops nothing.
  const turnIdRef = useRef<string | null>(null);
  const lastContextRef = useRef<AssistantPageContext | null>(null);
  // The model the last send asked for, so a retry asks the same one rather
  // than silently falling back to automatic.
  const lastProviderRef = useRef("");
  // What the last send handed over, so a retry carries the same files and
  // records.
  const lastContextExtrasRef = useRef<TurnContext>({});
  // A send between the click and the worker taking the question. The ref
  // refuses a second send in the same frame, before any render could disable
  // the composer; the state is what disables it. Stop marks the pending send
  // so a question still on its way is withdrawn rather than asked.
  const startingRef = useRef<PendingSend | null>(null);
  const [starting, setStarting] = useState(false);

  const endStarting = useCallback((pending: PendingSend) => {
    if (startingRef.current === pending) {
      startingRef.current = null;
      setStarting(false);
    }
  }, []);

  const failureMessage = useCallback(
    (kind: TurnFailureKind) => {
      switch (kind) {
        case "stopped-before-start":
          return t("Stopped before a reply started.");
        case "stopped":
          return t("Stopped. What was said so far has been kept in the thread.");
        case "failed-before-start":
          return t("This reply failed before it started.");
        default:
          return t("The reply was cut off before it finished. What arrived has been kept.");
      }
    },
    [t],
  );

  const fail = useCallback(
    (
      cause: TurnFailureCause,
      detail?: string,
      limit: TurnLimit | null = null,
      rateLimited: number | null = null,
    ) => {
      setTurn((state) => {
        if (!state || !isTurnActive(state)) {
          return state;
        }
        const message = failureMessage(describeTurnFailure(state, cause));
        return {
          ...state,
          status: "error",
          error: detail && detail !== "" ? `${message} ${detail}` : message,
          limit,
          stopped: cause === "stopped",
          rateLimited,
        };
      });
    },
    [failureMessage],
  );

  useEffect(() => {
    return () => abortRef.current?.abort();
  }, []);

  // Every ending — saved, refused, failed, stopped — also moves the "writing"
  // markers, so they clear with the reply on screen rather than a round-trip
  // after it.
  const refreshThread = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: queries.assistant.activeTurns().queryKey }),
      queryClient.invalidateQueries({ queryKey: queries.assistant.messages(threadId).queryKey }),
      queryClient.invalidateQueries({ queryKey: queries.assistant.proposals(threadId).queryKey }),
      queryClient.invalidateQueries({ queryKey: queries.assistant.plans(threadId).queryKey }),
      queryClient.invalidateQueries({ queryKey: queries.assistant.artifacts(threadId).queryKey }),
      queryClient.invalidateQueries({ queryKey: queries.assistant.threads().queryKey }),
    ]);
  }, [queryClient, threadId]);

  // The proposals a finished turn raised are put in the cache before the
  // refetch, so their cards appear with the reply rather than a round-trip
  // later. The refetch then brings the hold state and anything else.
  const seedProposals = useCallback(
    (result: SendMessageResult) => {
      const proposals = result.proposals ?? [];
      if (proposals.length === 0) {
        return;
      }
      const key = queries.assistant.proposals(threadId).queryKey;
      queryClient.setQueryData<{ results: SendMessageResult["proposals"] & object }>(
        key,
        (cached) => {
          const known = new Set((cached?.results ?? []).map((proposal) => proposal.id));
          const fresh = proposals.filter((proposal) => !known.has(proposal.id));
          return { results: [...(cached?.results ?? []), ...fresh] };
        },
      );
    },
    [queryClient, threadId],
  );

  // A finished turn is appended to the history the thread already holds
  // rather than refetched: the result carries the rows the server wrote, and a
  // thread with several pages loaded would otherwise fetch every one of them
  // after each reply. It is appended only when it continues the page: a turn
  // whose numbers skip ahead means something was saved that the client never
  // saw, an aborted turn most often, and that is fetched rather than papered
  // over. Nothing cached also fetches fresh.
  //
  // Once the saved rows are in the history the reply is on screen from the
  // cache, so the lists that only move with it (markers, proposals, plans,
  // artifacts, the sidebar) refresh behind it rather than holding the
  // streaming copy up, and the reply shown twice, until the slowest returns.
  const absorbTurn = useCallback(
    async (result: SendMessageResult | null) => {
      const key = queries.assistant.messages(threadId).queryKey;
      const cached = queryClient.getQueryData<ThreadHistory>(key);
      if (result) {
        seedProposals(result);
      }
      if (result && continuesHistory(cached, result.messages)) {
        queryClient.setQueryData<ThreadHistory>(key, (history) =>
          appendToHistory(history, result.messages),
        );
        void Promise.all([
          queryClient.invalidateQueries({
            queryKey: queries.assistant.activeTurns().queryKey,
          }),
          queryClient.invalidateQueries({
            queryKey: queries.assistant.proposals(threadId).queryKey,
          }),
          queryClient.invalidateQueries({ queryKey: queries.assistant.plans(threadId).queryKey }),
          queryClient.invalidateQueries({
            queryKey: queries.assistant.artifacts(threadId).queryKey,
          }),
          queryClient.invalidateQueries({ queryKey: queries.assistant.threads().queryKey }),
        ]);
        return;
      }

      await refreshThread();
    },
    [queryClient, refreshThread, seedProposals, threadId],
  );

  // A reply that failed after the question reached the agent is saved with a
  // note saying why, and the conversation draws that note as the same card.
  // Once the refetched conversation holds it, the live copy is dropped, or
  // the question and its card would show twice.
  const handOverIfSaved = useCallback(
    async (startedAt: number) => {
      await refreshThread();
      const history = queryClient.getQueryData<ThreadHistory>(
        queries.assistant.messages(threadId).queryKey,
      );
      const newest = history?.pages[0]?.results.at(-1);
      if (newest?.failure && newest.createdAt >= Math.floor(startedAt / 1000) - 5) {
        setTurn((state) => (state?.status === "error" ? null : state));
      }
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

  /**
   * Follows one turn from its first event to the saved thread: folds its
   * events into the view, and hands over to the refetched history once it
   * ends. A question the person asks and a reply they rejoin are the same
   * thing from here on; only how the events arrive differs.
   */
  const follow = useCallback(
    async (
      initial: TurnState,
      run: (onEvent: (event: AssistantStreamEvent) => void, signal: AbortSignal) => Promise<void>,
    ): Promise<NextTurn | null> => {
      // Sending over a reply still arriving cuts it off. The server keeps
      // what had run by then, so the thread is refetched to show it rather
      // than the cut-off turn vanishing under the new question.
      // Only a stream still open is interrupted. A finished turn used to
      // leave its controller behind, so every send after the first looked
      // like an interruption and refetched the whole thread under itself.
      const interrupted = abortRef.current !== null && !abortRef.current.signal.aborted;
      abortRef.current?.abort();
      if (interrupted) {
        void refreshThread();
      }
      const controller = new AbortController();
      abortRef.current = controller;
      // Signing out lets go of every reader at once; see turn-readers.
      const unregister = registerTurnReader(controller);
      const release = () => {
        unregister();
        if (abortRef.current === controller) {
          abortRef.current = null;
        }
      };

      let terminal = false;
      let ended = false;
      let done: SendMessageResult | null = null;
      let next: NextTurn | null = null;
      setTurn(initial);

      const onEvent = (event: AssistantStreamEvent) => {
        if (event.event === "context" || event.event === "compaction_started") {
          conversationEventRef.current?.(event);
          return;
        }
        if (event.event === "next_turn") {
          next = {
            turnId: event.data.turnId,
            queuedId: event.data.queuedId,
            input: event.data.input,
          };
        }
        setTurn((state) => (state ? advanceTurn(state, event) : state));
        if (event.event === "done") {
          terminal = true;
          ended = true;
          done = event.data;
        } else if (event.event === "refused" || event.event === "error") {
          terminal = true;
        }
      };

      try {
        await run(onEvent, controller.signal);
      } catch (error) {
        if (controller.signal.aborted) {
          return null;
        }
        release();
        fail(
          "failed",
          turnFailureDetail(error, t("The connection to the assistant was lost.")),
          turnLimitOf(error),
          error instanceof ApiRequestError && error.status === 429 ? (error.retryAfter ?? 5) : null,
        );
        void refreshThread();
        return null;
      }

      if (controller.signal.aborted) {
        return null;
      }
      release();

      if (!terminal) {
        // The stream closed without saying how it ended, which a proxy that
        // buffers or cuts long responses can cause. The turn may still have
        // been saved, so the thread is refreshed rather than the reply lost.
        await refreshThread();
        fail("ended");
        return null;
      }

      // An ending rebuilt from the turn's record carries no result; settling
      // on it refetches the conversation, which holds what was saved.
      if (ended) {
        await settle(done);
        // The queue started the next reply as this one was saved: it is
        // followed straight on, so the conversation never looks idle between.
        return next;
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
            void handOverIfSaved(state.startedAt);
          }
          return state;
        });
      }
      return null;
    },
    [fail, handOverIfSaved, refreshThread, settle, t],
  );

  // The records a queued message named, read from the queue the view holds,
  // so the reply the queue started shows them on its question from the start.
  const queuedMentions = useCallback(
    (queuedId: string): AssistantEntityRef[] => {
      const queue = queryClient.getQueryData<QueuedMessageList>(
        queries.assistant.queue(threadId).queryKey,
      );
      return queue?.items.find((item) => item.id === queuedId)?.request.mentions ?? [];
    },
    [queryClient, threadId],
  );

  /**
   * Follows a reply the server is producing for this conversation: one this
   * page lost when it was closed or reloaded, or one the application started,
   * such as the agent reporting what came of a decision.
   */
  const followActive = useCallback(
    async (active: ActiveTurn, context: TurnContext = {}) => {
      let current: ActiveTurn | null = active;
      let extras = context;
      while (current !== null) {
        const turnId = current.id;
        // A follow-up or a wait picked up carries no words of the person's own.
        const followUp = current.origin === "DecisionFollowUp" || current.origin === "WaitResolved";
        turnIdRef.current = turnId;
        const next: NextTurn | null = await follow(
          initialTurnState(followUp ? "" : (current.input ?? ""), null, { ...extras, followUp }),
          (onEvent, signal) => followTurn(turnId, { signal, onEvent }),
        );
        current = next === null ? null : activeFromNext(next, threadId);
        extras = next === null ? {} : { mentions: queuedMentions(next.queuedId) };
      }
    },
    [follow, queuedMentions, threadId],
  );

  /** The reply the server is producing for this conversation, if any. */
  const activeTurn = useCallback(async (): Promise<ActiveTurn | null> => {
    try {
      return await apiService.assistantService.activeTurn(threadId);
    } catch {
      return null;
    }
  }, [threadId]);

  const following = useCallback(
    () => abortRef.current !== null && !abortRef.current.signal.aborted,
    [],
  );

  /**
   * Picks up the reply the conversation is producing, when this view is not
   * already following one. Nothing happens when the conversation is quiet.
   */
  const rejoin = useCallback(async () => {
    // A send under way follows whatever the conversation is producing itself.
    if (following() || startingRef.current !== null) {
      return;
    }
    const active = await activeTurn();
    // A compaction is not a reply: it is handed to whoever shows it, which
    // follows it on its own stream.
    if (active?.origin === "Compaction") {
      conversationEventRef.current?.({
        event: "compaction_started",
        data: {
          turnId: active.id,
          threadId,
          auto: false,
          before: 0,
          after: 0,
          autoCompactOff: false,
        },
      });
      return;
    }
    // A question sent while the lookup was out owns the view now.
    if (active === null || following() || startingRef.current !== null) {
      return;
    }
    await followActive(active);
  }, [activeTurn, followActive, following, threadId]);

  const send = useCallback(
    async (
      content: string,
      context?: AssistantPageContext | null,
      providerId = "",
      extras: TurnContext = {},
    ) => {
      // One question at a time: a second Enter or click before the first
      // reached a worker used to send the message twice, and the server either
      // refused the second or queued it behind the first.
      if (startingRef.current !== null) {
        return;
      }
      const pending: PendingSend = { stopped: false };
      startingRef.current = pending;
      setStarting(true);

      const pageContext = context === undefined ? (getContext?.() ?? null) : context;
      lastContextRef.current = pageContext;
      lastProviderRef.current = providerId;
      lastContextExtrasRef.current = extras;
      const initial = initialTurnState(content, pageContext, extras);

      try {
        // A reply the server is producing that this view has not picked up —
        // the agent answering a decision made elsewhere, most often — is
        // followed to its end first. Asking over it used to fail with "already
        // working on a reply" while nothing on screen said anything was.
        //
        // The question is on screen from the click, not from the end of that
        // check, and the check is skipped when the live list the realtime
        // connection keeps current already says the conversation is quiet.
        if (!following()) {
          setTurn(initial);
          const quiet = listSaysQuiet(
            queryClient.getQueryState<AssistantLiveTurnList>(
              queries.assistant.activeTurns().queryKey,
            ),
            threadId,
            useRealtimeStore.getState().connectionState === "connected",
          );
          const running = quiet ? null : await activeTurn();
          if (
            running !== null &&
            running.origin !== "Compaction" &&
            !following() &&
            !pending.stopped
          ) {
            await followActive(running);
          }
        }

        // Stopped before the question left: it is kept on screen as stopped,
        // so it can be sent again, and never reaches the server.
        if (pending.stopped) {
          setTurn({
            ...initial,
            status: "error",
            error: failureMessage("stopped-before-start"),
          });
          return;
        }

        turnIdRef.current = null;
        const attachments = (extras.attachments ?? []).map((item) => item.documentId);
        const next = await follow(initial, (onEvent, signal) =>
          runTurn(
            (startSignal) =>
              apiService.assistantService.startTurn(
                threadId,
                content,
                {
                  context: pageContext,
                  surface,
                  providerId,
                  attachmentDocumentIds: attachments,
                  mentions: extras.mentions ?? [],
                  directedAgentId: extras.directedAgentId,
                },
                { signal: startSignal },
              ),
            {
              signal,
              onTurnStarted: (started) => {
                turnIdRef.current = started.turnId;
                endStarting(pending);
                void queryClient.invalidateQueries({
                  queryKey: queries.assistant.activeTurns().queryKey,
                });
              },
              onEvent,
              findWithdrawn: async () => {
                const active = await activeTurn();
                return active !== null &&
                  active.origin !== "DecisionFollowUp" &&
                  active.origin !== "WaitResolved" &&
                  active.origin !== "Compaction" &&
                  active.input === content
                  ? active.id
                  : null;
              },
            },
          ),
        );
        if (next !== null) {
          endStarting(pending);
          await followActive(activeFromNext(next, threadId), {
            mentions: queuedMentions(next.queuedId),
          });
        }
      } finally {
        endStarting(pending);
      }
    },
    [
      activeTurn,
      endStarting,
      failureMessage,
      follow,
      followActive,
      following,
      getContext,
      queryClient,
      queuedMentions,
      surface,
      threadId,
    ],
  );

  /**
   * Follows a reply the queue started at the person's request: a message sent
   * from the queue, or one queued while the conversation was free. Nothing
   * happens when this view already follows a reply; it reaches this one
   * through the queue when it ends.
   */
  const followStarted = useCallback(
    async (turnId: string, content: string, mentions: AssistantEntityRef[] = []) => {
      if (following() || startingRef.current !== null) {
        return;
      }
      void queryClient.invalidateQueries({ queryKey: queries.assistant.activeTurns().queryKey });
      await followActive(
        { id: turnId, threadId, status: "Running", origin: "Person", input: content },
        {
          mentions,
        },
      );
    },
    [followActive, following, queryClient, threadId],
  );

  const stop = useCallback(() => {
    const pending = startingRef.current;
    if (pending !== null) {
      pending.stopped = true;
      endStarting(pending);
    }

    // A turn on a worker has to be told: abandoning the reader leaves the turn
    // running and billing for an answer nobody will read.
    const running = turnIdRef.current;
    if (running !== null) {
      turnIdRef.current = null;
      stopTurnQuietly(running);
    }

    abortRef.current?.abort();
    abortRef.current = null;
    // The server saves what had run when the stream was cut; the refetch
    // shows it under the notice rather than leaving it to the next turn.
    void refreshThread();
    fail("stopped");
  }, [endStarting, fail, refreshThread]);

  const dismiss = useCallback(async () => {
    await refreshThread();
    setTurn(null);
  }, [refreshThread]);

  return {
    turn,
    isActive: starting || isTurnActive(turn),
    send,
    rejoin,
    followStarted,
    stop,
    dismiss,
    retry: turn
      ? () =>
          send(
            turn.userContent,
            lastContextRef.current,
            lastProviderRef.current,
            lastContextExtrasRef.current,
          )
      : undefined,
  };
}
