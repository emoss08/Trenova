import { useT } from "@trenova/shared/i18n/use-t";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { durableTurnsAvailable, followExistingTurn, runDurableTurn } from "./durable-turn";
import { AssistantStreamError, type ActiveTurn } from "@/services/assistant";
import type {
  AssistantPageContext,
  AssistantStreamEvent,
  SendMessageResult,
} from "@/types/assistant";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { appendToHistory, continuesHistory, type ThreadHistory } from "./thread-history";
import {
  describeTurnFailure,
  initialTurnState,
  isTurnActive,
  reduceTurn,
  type TurnContext,
  type TurnFailureCause,
  type TurnFailureKind,
  type TurnState,
} from "./turn-stream";

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
    (cause: TurnFailureCause, detail?: string) => {
      setTurn((state) => {
        if (!state || !isTurnActive(state)) {
          return state;
        }
        const message = failureMessage(describeTurnFailure(state, cause));
        return {
          ...state,
          status: "error",
          error: detail && detail !== "" ? `${message} ${detail}` : message,
        };
      });
    },
    [failureMessage],
  );

  useEffect(() => {
    return () => abortRef.current?.abort();
  }, []);

  const refreshThread = useCallback(async () => {
    await Promise.all([
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
        await Promise.all([
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
    ) => {
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
      const release = () => {
        if (abortRef.current === controller) {
          abortRef.current = null;
        }
      };

      let terminal = false;
      let done: SendMessageResult | null = null;
      setTurn(initial);

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
        await run(onEvent, controller.signal);
      } catch (error) {
        if (controller.signal.aborted) {
          return;
        }
        release();
        const detail =
          error instanceof AssistantStreamError
            ? error.message
            : t("The connection to the assistant was lost.");
        fail("failed", detail);
        void refreshThread();
        return;
      }

      if (controller.signal.aborted) {
        return;
      }
      release();

      if (!terminal) {
        // The stream closed without saying how it ended, which a proxy that
        // buffers or cuts long responses can cause. The turn may still have
        // been saved, so the thread is refreshed rather than the reply lost.
        await refreshThread();
        fail("ended");
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
    [fail, refreshThread, settle, t],
  );

  const send = useCallback(
    async (
      content: string,
      context?: AssistantPageContext | null,
      providerId = "",
      extras: TurnContext = {},
    ) => {
      const pageContext = context === undefined ? (getContext?.() ?? null) : context;
      lastContextRef.current = pageContext;
      lastProviderRef.current = providerId;
      lastContextExtrasRef.current = extras;
      turnIdRef.current = null;

      const attachments = (extras.attachments ?? []).map((item) => item.documentId);
      await follow(initialTurnState(content, pageContext, extras), async (onEvent, signal) => {
        if (await durableTurnsAvailable()) {
          await runDurableTurn({
            threadId,
            content,
            context: pageContext,
            providerId,
            attachmentDocumentIds: attachments,
            mentions: extras.mentions ?? [],
            signal,
            onTurnStarted: (id) => {
              turnIdRef.current = id;
            },
            onEvent,
          });
          return;
        }
        await apiService.assistantService.streamMessage(threadId, content, onEvent, signal, {
          context: pageContext,
          providerId,
          attachmentDocumentIds: attachments,
          mentions: extras.mentions ?? [],
        });
      });
    },
    [follow, getContext, threadId],
  );

  /**
   * Picks up a reply the conversation is already producing: one this page
   * lost when it was closed or reloaded, or one the application started,
   * such as the agent reporting what came of a decision. Nothing happens
   * when the conversation is quiet or this view is already following a turn.
   */
  const rejoin = useCallback(async () => {
    if (abortRef.current !== null && !abortRef.current.signal.aborted) {
      return;
    }

    let active: ActiveTurn | null;
    try {
      active = await apiService.assistantService.activeTurn(threadId);
    } catch {
      return;
    }
    // A question sent while the lookup was out owns the view now.
    if (active === null || (abortRef.current !== null && !abortRef.current.signal.aborted)) {
      return;
    }

    const followUp = active.origin === "DecisionFollowUp";
    const turnId = active.id;
    turnIdRef.current = turnId;
    await follow(
      initialTurnState(followUp ? "" : (active.input ?? ""), null, { followUp }),
      (onEvent, signal) => followExistingTurn(turnId, { signal, onEvent }),
    );
  }, [follow, threadId]);

  const stop = useCallback(() => {
    // A turn on a worker has to be told. Aborting the reader used to stop the
    // model, because the model was running on the request being aborted; with
    // the work moved off it, abandoning the reader leaves the turn running and
    // billing for an answer nobody will read.
    const running = turnIdRef.current;
    if (running !== null) {
      turnIdRef.current = null;
      void apiService.assistantService.stopTurn(running).catch(() => undefined);
    }

    abortRef.current?.abort();
    abortRef.current = null;
    // The server saves what had run when the stream was cut; the refetch
    // shows it under the notice rather than leaving it to the next turn.
    void refreshThread();
    fail("stopped");
  }, [fail, refreshThread]);

  const dismiss = useCallback(async () => {
    await refreshThread();
    setTurn(null);
  }, [refreshThread]);

  return {
    turn,
    isActive: isTurnActive(turn),
    send,
    rejoin,
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
